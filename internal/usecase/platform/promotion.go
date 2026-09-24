package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
	"github.com/google/uuid"
)

// Promotion is an explicit admin copy into an existing account. Imported hours
// do not describe weekdays reliably, so availability is configured by the owner.
func (s *Service) PromoteFirm(ctx context.Context, adminID, firmID uuid.UUID, in domain.PromoteFirmInput) (*domain.Promotion, error) {
	if in.UserID == uuid.Nil || len(in.Services) > 100 {
		return nil, domain.ErrValidation
	}
	if in.TimeZone == "" {
		in.TimeZone = "Asia/Almaty"
	}
	if _, err := time.LoadLocation(in.TimeZone); err != nil {
		return nil, fmt.Errorf("%w: invalid time_zone", domain.ErrValidation)
	}
	result := &domain.Promotion{MasterID: in.UserID}
	err := s.db.WithinTx(ctx, func(ctx context.Context) error {
		var userID uuid.UUID
		if err := postgres.Get(ctx, s.db.Q(ctx), &userID, `SELECT id FROM users WHERE id=$1 AND status='active' FOR UPDATE`, in.UserID); err != nil {
			return err
		}
		var firm struct {
			Name        string   `db:"name"`
			Address     *string  `db:"address_text"`
			Description *string  `db:"description"`
			AvatarURL   *string  `db:"avatar_url"`
			Latitude    *float64 `db:"latitude"`
			Longitude   *float64 `db:"longitude"`
		}
		if err := postgres.Get(ctx, s.db.Q(ctx), &firm, `SELECT name,address_text,description,avatar_url,ST_Y(location::geometry) AS latitude,ST_X(location::geometry) AS longitude FROM source_firms WHERE id=$1 FOR UPDATE`, firmID); err != nil {
			return err
		}
		var exists bool
		if err := postgres.Get(ctx, s.db.Q(ctx), &exists, `SELECT EXISTS(SELECT 1 FROM source_firm_promotions WHERE source_firm_id=$1)`, firmID); err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("%w: firm already promoted", domain.ErrConflict)
		}
		if firm.Address == nil {
			return fmt.Errorf("%w: imported firm has no address", domain.ErrValidation)
		}
		if err := postgres.Exec(ctx, s.db.Q(ctx), `INSERT INTO master_profiles(user_id,display_name,description,avatar_url) VALUES($1,$2,$3,$4) ON CONFLICT(user_id) DO NOTHING`, in.UserID, firm.Name, firm.Description, firm.AvatarURL); err != nil {
			return err
		}
		if err := postgres.Exec(ctx, s.db.Q(ctx), `INSERT INTO user_roles(user_id,role_code) VALUES($1,'master') ON CONFLICT DO NOTHING`, in.UserID); err != nil {
			return err
		}
		loc, err := s.CreateMasterLocation(ctx, in.UserID, domain.LocationInput{Name: firm.Name, AddressText: *firm.Address, Latitude: firm.Latitude, Longitude: firm.Longitude, TimeZone: in.TimeZone})
		if err != nil {
			return err
		}
		result.LocationID = loc.ID
		seen := map[uuid.UUID]bool{}
		for _, item := range in.Services {
			if seen[item.SourceServiceID] {
				return fmt.Errorf("%w: duplicate source service", domain.ErrValidation)
			}
			seen[item.SourceServiceID] = true
			var source struct {
				Name        string  `db:"name"`
				Description *string `db:"description"`
				Price       *int64  `db:"price_min_amount"`
				Duration    *int    `db:"duration_minutes"`
				Currency    *string `db:"currency"`
			}
			if err := postgres.Get(ctx, s.db.Q(ctx), &source, `SELECT name,description,price_min_amount,duration_minutes,currency FROM source_services WHERE id=$1 AND source_firm_id=$2`, item.SourceServiceID, firmID); err != nil {
				return err
			}
			if item.PriceAmount != nil {
				source.Price = item.PriceAmount
			}
			if item.DurationMinutes != nil {
				source.Duration = item.DurationMinutes
			}
			if source.Price == nil || source.Duration == nil {
				return fmt.Errorf("%w: price_amount and duration_minutes required for incomplete source services", domain.ErrValidation)
			}
			currency := domain.DefaultCurrency
			if source.Currency != nil {
				currency = *source.Currency
			}
			var active bool
			if err := postgres.Get(ctx, s.db.Q(ctx), &active, `SELECT is_active FROM service_categories WHERE id=$1`, item.CategoryID); err != nil {
				return err
			}
			if !active {
				return domain.ErrValidation
			}
			if _, err := s.CreateMasterService(ctx, in.UserID, domain.ServiceInput{CategoryID: item.CategoryID, Name: source.Name, Description: source.Description, PriceAmount: *source.Price, Currency: currency, DurationMinutes: *source.Duration}); err != nil {
				return err
			}
		}
		return postgres.Exec(ctx, s.db.Q(ctx), `INSERT INTO source_firm_promotions(source_firm_id,master_id,location_id,promoted_by) VALUES($1,$2,$3,$4)`, firmID, in.UserID, loc.ID, adminID)
	})
	return result, mapDBError(err)
}
