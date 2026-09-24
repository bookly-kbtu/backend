package token

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/bookly-kbtu/backend/internal/domain"
)

func TestIssueParse(t *testing.T) {
	m := NewManager("test-secret-test-secret-test-secret", time.Minute)
	in := Claims{UserID: uuid.New(), SessionID: uuid.New(), Roles: []domain.Role{domain.RoleClient}}

	raw, _, err := m.Issue(in, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	got, err := m.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != in.UserID || got.SessionID != in.SessionID || len(got.Roles) != 1 {
		t.Fatalf("claims mismatch: %+v", got)
	}
}

func TestParseRejects(t *testing.T) {
	m := NewManager("test-secret-test-secret-test-secret", time.Minute)
	other := NewManager("other-secret-other-secret-other-secret", time.Minute)
	claims := Claims{UserID: uuid.New(), SessionID: uuid.New()}

	expired, _, _ := m.Issue(claims, time.Now().Add(-time.Hour))
	foreign, _, _ := other.Issue(claims, time.Now())

	for name, raw := range map[string]string{"expired": expired, "wrong secret": foreign, "garbage": "abc"} {
		if _, err := m.Parse(raw); !errors.Is(err, domain.ErrUnauthorized) {
			t.Errorf("%s: error = %v; want ErrUnauthorized", name, err)
		}
	}
}
