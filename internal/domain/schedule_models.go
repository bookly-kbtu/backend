package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkingHour struct {
	ID         uuid.UUID `db:"id"`
	MasterID   uuid.UUID `db:"master_id"`
	LocationID uuid.UUID `db:"location_id"`
	Weekday    int       `db:"weekday"`
	StartTime  string    `db:"start_time"`
	EndTime    string    `db:"end_time"`
	IsActive   bool      `db:"is_active"`
}

type WorkingHourInput struct {
	LocationID uuid.UUID
	Weekday    int
	StartTime  string
	EndTime    string
}

type ScheduleExceptionRecord struct {
	ID             uuid.UUID  `db:"id"`
	MasterID       uuid.UUID  `db:"master_id"`
	LocationID     *uuid.UUID `db:"location_id"`
	ExceptionKind  string     `db:"exception_kind"`
	StartsOn       time.Time  `db:"starts_on"`
	EndsOn         time.Time  `db:"ends_on"`
	LocalStartTime *string    `db:"local_start_time"`
	LocalEndTime   *string    `db:"local_end_time"`
	Reason         *string    `db:"reason"`
}

type ScheduleExceptionInput struct {
	LocationID     *uuid.UUID
	ExceptionKind  string
	StartsOn       time.Time
	EndsOn         time.Time
	LocalStartTime *string
	LocalEndTime   *string
	Reason         *string
}

type Slot struct {
	StartsAt time.Time `db:"starts_at" json:"starts_at"`
	EndsAt   time.Time `db:"ends_at" json:"ends_at"`
}
