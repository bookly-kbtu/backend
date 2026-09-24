package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrValidation      = errors.New("validation error")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("conflict")
	ErrTooManyRequests = errors.New("too many requests")
)
