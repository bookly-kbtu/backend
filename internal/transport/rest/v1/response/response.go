package response

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/bookly-kbtu/backend/internal/domain"
)

type ErrorBody struct {
	Error string `json:"error"`
}

func OK(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusOK).JSON(data)
}

func Created(c *fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(data)
}

func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorBody{Error: message})
}

// ErrorHandler maps domain and fiber errors to JSON responses.
func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return Error(c, fiberErr.Code, fiberErr.Message)
		}

		switch {
		case errors.Is(err, domain.ErrNotFound):
			return Error(c, fiber.StatusNotFound, err.Error())
		case errors.Is(err, domain.ErrValidation):
			return Error(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrUnauthorized):
			return Error(c, fiber.StatusUnauthorized, err.Error())
		case errors.Is(err, domain.ErrForbidden):
			return Error(c, fiber.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrConflict):
			return Error(c, fiber.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrTooManyRequests):
			return Error(c, fiber.StatusTooManyRequests, err.Error())
		}

		logger.Error("unhandled error", "error", err, "path", c.Path())
		return Error(c, fiber.StatusInternalServerError, "internal server error")
	}
}
