package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

func Recover(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"panic", fmt.Sprint(r),
					"path", c.Path(),
					"stack", string(debug.Stack()),
				)
				err = fiber.ErrInternalServerError
			}
		}()
		return c.Next()
	}
}
