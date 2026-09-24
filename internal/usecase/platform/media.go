package platform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
)

var errMediaDisabled = fmt.Errorf("%w: media storage is not configured", domain.ErrUnavailable)

// Upload is an image received from a client, not yet validated.
type Upload struct {
	Body io.Reader
	Size int64
}

// OpenMedia returns a stored file for public delivery.
func (s *Service) OpenMedia(ctx context.Context, key string) (*domain.MediaObject, error) {
	if s.media == nil {
		return nil, errMediaDisabled
	}
	if key == "" || strings.Contains(key, "..") {
		return nil, domain.ErrNotFound
	}
	return s.media.Get(ctx, key)
}

func (s *Service) SetClientAvatar(ctx context.Context, userID uuid.UUID, file Upload) (*domain.ClientProfile, error) {
	const query = `
		INSERT INTO client_profiles (user_id, avatar_url) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET avatar_url = EXCLUDED.avatar_url`
	if err := s.replaceAvatar(ctx, userID, file, `SELECT avatar_url FROM client_profiles WHERE user_id = $1`, query); err != nil {
		return nil, err
	}
	return s.GetClientProfile(ctx, userID)
}

func (s *Service) DeleteClientAvatar(ctx context.Context, userID uuid.UUID) (*domain.ClientProfile, error) {
	if err := s.clearAvatar(ctx, userID, `SELECT avatar_url FROM client_profiles WHERE user_id = $1`,
		`UPDATE client_profiles SET avatar_url = NULL WHERE user_id = $1`); err != nil {
		return nil, err
	}
	return s.GetClientProfile(ctx, userID)
}

func (s *Service) SetMasterAvatar(ctx context.Context, userID uuid.UUID, file Upload) (*domain.MasterProfile, error) {
	if _, err := s.GetMasterProfile(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.replaceAvatar(ctx, userID, file, `SELECT avatar_url FROM master_profiles WHERE user_id = $1`,
		`UPDATE master_profiles SET avatar_url = $2 WHERE user_id = $1`); err != nil {
		return nil, err
	}
	return s.GetMasterProfile(ctx, userID)
}

func (s *Service) DeleteMasterAvatar(ctx context.Context, userID uuid.UUID) (*domain.MasterProfile, error) {
	if err := s.clearAvatar(ctx, userID, `SELECT avatar_url FROM master_profiles WHERE user_id = $1`,
		`UPDATE master_profiles SET avatar_url = NULL WHERE user_id = $1`); err != nil {
		return nil, err
	}
	return s.GetMasterProfile(ctx, userID)
}

// replaceAvatar stores the image, points the profile at it, then drops the old file.
func (s *Service) replaceAvatar(ctx context.Context, userID uuid.UUID, file Upload, selectOld, update string) error {
	if s.media == nil {
		return errMediaDisabled
	}
	body, contentType, ext, err := readImage(file)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("avatars/%s/%s.%s", userID, uuid.New(), ext)
	if err := s.media.Put(ctx, key, bytes.NewReader(body), int64(len(body)), contentType); err != nil {
		return err
	}
	old := s.currentURL(ctx, selectOld, userID)
	if err := postgres.Exec(ctx, s.db.Q(ctx), update, userID, s.mediaBaseURL+"/"+key); err != nil {
		_ = s.media.Delete(context.WithoutCancel(ctx), key)
		return err
	}
	s.deleteOwned(ctx, old)
	return nil
}

func (s *Service) clearAvatar(ctx context.Context, userID uuid.UUID, selectOld, update string) error {
	old := s.currentURL(ctx, selectOld, userID)
	if err := postgres.Exec(ctx, s.db.Q(ctx), update, userID); err != nil {
		return err
	}
	s.deleteOwned(ctx, old)
	return nil
}

func (s *Service) currentURL(ctx context.Context, query string, userID uuid.UUID) *string {
	var url *string
	_ = postgres.Get(ctx, s.db.Q(ctx), &url, query, userID)
	return url
}

// deleteOwned removes a file we stored earlier. URLs set by other means are left alone.
// Failure only leaves an orphaned object, so it does not fail the request.
func (s *Service) deleteOwned(ctx context.Context, url *string) {
	if s.media == nil || url == nil {
		return
	}
	key, ok := strings.CutPrefix(*url, s.mediaBaseURL+"/")
	if !ok {
		return
	}
	_ = s.media.Delete(context.WithoutCancel(ctx), key)
}

// readImage buffers the upload and trusts only the sniffed content type,
// never the client-supplied one.
func readImage(file Upload) ([]byte, string, string, error) {
	if file.Size > domain.MaxImageBytes {
		return nil, "", "", fmt.Errorf("%w: image must be at most %d MB", domain.ErrValidation, domain.MaxImageBytes>>20)
	}
	body, err := io.ReadAll(io.LimitReader(file.Body, domain.MaxImageBytes+1))
	if err != nil {
		return nil, "", "", fmt.Errorf("read upload: %w", err)
	}
	if len(body) > domain.MaxImageBytes {
		return nil, "", "", fmt.Errorf("%w: image must be at most %d MB", domain.ErrValidation, domain.MaxImageBytes>>20)
	}
	contentType := http.DetectContentType(body)
	ext, ok := domain.ImageExtension(contentType)
	if !ok {
		return nil, "", "", fmt.Errorf("%w: only JPEG, PNG and WebP images are allowed", domain.ErrValidation)
	}
	return body, contentType, ext, nil
}
