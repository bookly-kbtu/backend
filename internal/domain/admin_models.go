package domain

import (
	"time"

	"github.com/google/uuid"
)

type AdminUser struct {
	ID        uuid.UUID `db:"id"`
	Phone     *string   `db:"phone"`
	Status    string    `db:"status"`
	Roles     []string
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type MarketStats struct {
	CitiesCount   int `db:"cities_count" json:"cities_count"`
	FirmsCount    int `db:"firms_count" json:"firms_count"`
	ServicesCount int `db:"services_count" json:"services_count"`
	MastersCount  int `db:"masters_count" json:"masters_count"`
}
