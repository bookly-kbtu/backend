package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

func Logger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()

		// Run the error handler first so the logged status is the real one.
		if err != nil {
			if handlerErr := c.App().ErrorHandler(c, err); handlerErr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		logger.Info("http request",
			"request_id", c.Locals(RequestIDKey),
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration", time.Since(start),
			"ip", c.IP(),
		)
		return nil
	}
}
