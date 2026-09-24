package domain

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	DefaultCurrency    = "KZT"
	MaxServiceDuration = 12 * 60 // minutes
)

var (
	slugPattern     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
)

// RequireText trims s and fails when it is empty.
func RequireText(field, s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("%w: %s is required", ErrValidation, field)
	}
	return s, nil
}

func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: invalid slug %q", ErrValidation, slug)
	}
	return nil
}

func ValidateCurrency(currency string) error {
	if !currencyPattern.MatchString(currency) {
		return fmt.Errorf("%w: invalid currency %q", ErrValidation, currency)
	}
	return nil
}

// ValidatePrice checks price in minor units (tiyn for KZT).
func ValidatePrice(amount int64) error {
	if amount < 0 {
		return fmt.Errorf("%w: price must not be negative", ErrValidation)
	}
	return nil
}

func ValidateServiceDuration(minutes int) error {
	if minutes <= 0 || minutes > MaxServiceDuration {
		return fmt.Errorf("%w: duration must be between 1 and %d minutes", ErrValidation, MaxServiceDuration)
	}
	return nil
}
