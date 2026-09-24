package platform

import (
	"github.com/bookly-kbtu/backend/internal/infrastructure/postgres"
)

type Service struct {
	db *postgres.DB
}

func New(db *postgres.DB) *Service {
	return &Service{db: db}
}
