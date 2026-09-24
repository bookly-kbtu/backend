package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID       `db:"id" json:"id"`
	BookingID *uuid.UUID      `db:"booking_id" json:"booking_id"`
	Type      string          `db:"notification_type" json:"notification_type"`
	Payload   json.RawMessage `db:"payload" json:"payload" swaggertype:"object"`
	ReadAt    *time.Time      `db:"read_at" json:"read_at"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
}
