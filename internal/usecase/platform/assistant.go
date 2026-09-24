package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

const assistantColumns = `id,transcript,parsed_intent,status,selected_master_id,selected_service_id,selected_booking_id,created_at,updated_at`

func (s *Service) CreateAssistantRequest(ctx context.Context, userID uuid.UUID, transcript string, intent domain.AssistantIntent) (*domain.AssistantRequest, error) {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" || len(transcript) > 8000 || len(intent.Query) > 200 || (intent.MaxPrice != nil && *intent.MaxPrice < 0) {
		return nil, fmt.Errorf("%w: transcript required (max 8000 bytes), query max 200 bytes, price nonnegative", domain.ErrValidation)
	}
	if strings.TrimSpace(intent.Query) == "" && intent.CategoryID == nil {
		return nil, fmt.Errorf("%w: structured intent requires query or category_id", domain.ErrValidation)
	}
	payload, err := json.Marshal(intent)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	if err = postgres.Exec(ctx, s.db.Q(ctx), `INSERT INTO assistant_requests(id,user_id,transcript,parsed_intent,status) VALUES($1,$2,$3,$4,'parsed')`, id, userID, transcript, payload); err != nil {
		return nil, err
	}
	return s.AssistantRequest(ctx, userID, id)
}

func (s *Service) AssistantRequest(ctx context.Context, userID, id uuid.UUID) (*domain.AssistantRequest, error) {
	var result domain.AssistantRequest
	err := postgres.Get(ctx, s.db.Q(ctx), &result, `SELECT `+assistantColumns+` FROM assistant_requests WHERE user_id=$1 AND id=$2`, userID, id)
	return &result, err
}

func (s *Service) AssistantRequests(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.AssistantRequest, error) {
	limit, offset, err := page(limit, offset)
	if err != nil {
		return nil, err
	}
	return postgres.List[domain.AssistantRequest](ctx, s.db.Q(ctx), `SELECT `+assistantColumns+` FROM assistant_requests WHERE user_id=$1 ORDER BY created_at DESC,id LIMIT $2 OFFSET $3`, userID, limit, offset)
}

func (s *Service) AssistantCandidates(ctx context.Context, userID, id uuid.UUID, limit, offset int) ([]domain.AssistantCandidate, error) {
	limit, offset, err := page(limit, offset)
	if err != nil {
		return nil, err
	}
	request, err := s.AssistantRequest(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	var intent domain.AssistantIntent
	if err = json.Unmarshal(request.ParsedIntent, &intent); err != nil {
		return nil, err
	}
	result, err := postgres.List[domain.AssistantCandidate](ctx, s.db.Q(ctx), `SELECT m.user_id AS master_id,m.display_name,ms.id AS service_id,ms.name AS service_name,ms.price_amount,ms.currency,ms.duration_minutes
 FROM master_services ms JOIN master_profiles m ON m.user_id=ms.master_id JOIN users u ON u.id=m.user_id JOIN service_categories c ON c.id=ms.category_id
 WHERE ms.is_active AND m.is_active AND c.is_active AND u.status='active' AND m.user_id<>$4
 AND ($1='' OR ms.name ILIKE '%'||$1||'%' OR c.name ILIKE '%'||$1||'%' OR m.display_name ILIKE '%'||$1||'%')
 AND ($2::uuid IS NULL OR ms.category_id=$2) AND ($3::bigint IS NULL OR (ms.currency='KZT' AND ms.price_amount<=$3))
 ORDER BY ms.price_amount,ms.id LIMIT $5 OFFSET $6`, strings.TrimSpace(intent.Query), intent.CategoryID, intent.MaxPrice, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	err = postgres.Exec(ctx, s.db.Q(ctx), `UPDATE assistant_requests SET status='searched' WHERE id=$1 AND user_id=$2 AND status='parsed'`, id, userID)
	return result, err
}

func (s *Service) SelectAssistantService(ctx context.Context, userID, id, masterID, serviceID uuid.UUID) (*domain.AssistantRequest, error) {
	if _, err := s.GetPublicMaster(ctx, masterID); err != nil {
		return nil, err
	}
	service, err := s.GetMasterService(ctx, masterID, serviceID)
	if err != nil {
		return nil, err
	}
	if !service.IsActive || masterID == userID {
		return nil, domain.ErrValidation
	}
	err = s.db.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.lockAssistant(ctx, userID, id); err != nil {
			return err
		}
		request, err := s.AssistantRequest(ctx, userID, id)
		if err != nil {
			return err
		}
		if request.SelectedBookingID != nil {
			return domain.ErrConflict
		}
		return s.execOne(ctx, `UPDATE assistant_requests SET selected_master_id=$3,selected_service_id=$4,status='selected' WHERE user_id=$1 AND id=$2`, userID, id, masterID, serviceID)
	})
	if err != nil {
		return nil, err
	}
	return s.AssistantRequest(ctx, userID, id)
}

func (s *Service) lockAssistant(ctx context.Context, userID, id uuid.UUID) error {
	var locked uuid.UUID
	return postgres.Get(ctx, s.db.Q(ctx), &locked, `SELECT id FROM assistant_requests WHERE user_id=$1 AND id=$2 FOR UPDATE`, userID, id)
}

func (s *Service) BookAssistant(ctx context.Context, userID, id, locationID uuid.UUID, startsAt time.Time, comment *string) (*domain.Booking, error) {
	var booking *domain.Booking
	err := s.db.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.lockAssistant(ctx, userID, id); err != nil {
			return err
		}
		request, err := s.AssistantRequest(ctx, userID, id)
		if err != nil {
			return err
		}
		if request.SelectedBookingID != nil {
			booking, err = s.GetBookingForUser(ctx, *request.SelectedBookingID, userID)
			return err
		}
		if request.SelectedMasterID == nil || request.SelectedServiceID == nil {
			return fmt.Errorf("%w: select a service first", domain.ErrConflict)
		}
		booking, err = s.CreateBooking(ctx, userID, domain.CreateBookingInput{MasterID: *request.SelectedMasterID, MasterServiceID: *request.SelectedServiceID, MasterLocationID: locationID, StartsAt: startsAt, ClientComment: comment})
		if err != nil {
			return err
		}
		return s.execOne(ctx, `UPDATE assistant_requests SET selected_booking_id=$3,status='booked' WHERE user_id=$1 AND id=$2`, userID, id, booking.ID)
	})
	return booking, err
}
