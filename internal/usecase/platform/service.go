package platform

import (
	"github.com/bookly-kbtu/backend/internal/domain"
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
)

type Service struct {
	db *postgres.DB
	// media is nil when uploads are not configured.
	media        domain.MediaStorage
	mediaBaseURL string
}

func New(db *postgres.DB, media domain.MediaStorage, mediaBaseURL string) *Service {
	return &Service{db: db, media: media, mediaBaseURL: mediaBaseURL}
}
