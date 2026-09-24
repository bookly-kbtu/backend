package otp

import (
	"context"
	"log/slog"

	"github.com/bookly-kbtu/backend/internal/domain"
)

// StaticGenerator returns the same code from config for every challenge.
// MVP stub: replace with a random generator once a real sender is wired.
type StaticGenerator struct {
	code string
}

func NewStaticGenerator(code string) *StaticGenerator {
	return &StaticGenerator{code: code}
}

func (g *StaticGenerator) Generate() (string, error) {
	return g.code, nil
}

// LogSender only logs the fact of sending. Never logs the code.
// Replace with SMS / WhatsApp / Telegram Gateway providers.
type LogSender struct {
	log *slog.Logger
}

func NewLogSender(log *slog.Logger) *LogSender {
	return &LogSender{log: log}
}

func (s *LogSender) Send(ctx context.Context, destination string, channel domain.OTPChannel, _ string) error {
	s.log.InfoContext(ctx, "otp code sent (stub)", "destination", maskPhone(destination), "channel", channel)
	return nil
}

func maskPhone(phone string) string {
	if len(phone) <= 4 {
		return "****"
	}
	return phone[:len(phone)-4] + "****"
}
