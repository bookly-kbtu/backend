package domain

import (
	"fmt"
	"slices"
)

// Enum values live in code, not in DB CHECK constraints: adding a value
// is a code change, never a migration. Usecases must parse/validate input
// through Parse* functions before writing to the database.

func parseEnum[T ~string](field, raw string, allowed []T) (T, error) {
	value := T(raw)
	if !slices.Contains(allowed, value) {
		return "", fmt.Errorf("%w: invalid %s %q", ErrValidation, field, raw)
	}
	return value, nil
}
