package apierr

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	CodeInvalidInput        = "invalid_input"
	CodeNotFound            = "not_found"
	CodeConflict            = "conflict"
	CodeUnauthorized        = "unauthorized"
	CodeForbidden           = "forbidden"
	CodeInternalError       = "internal_error"
	CodeInvalidTransition   = "invalid_transition"
	CodeInsufficientStock   = "insufficient_stock"
	CodeAlreadyFulfilled    = "already_fulfilled"
	CodeTransactionCanceled = "transaction_canceled"
)

func New(c *fiber.Ctx, status int, code string, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// BadRequest returns 400 with the given code and message.
func BadRequest(c *fiber.Ctx, code string, message string) error {
	return New(c, http.StatusBadRequest, code, message)
}

// NotFound returns 404.
func NotFound(c *fiber.Ctx, message string) error {
	return New(c, http.StatusNotFound, CodeNotFound, message)
}

// Conflict returns 409.
func Conflict(c *fiber.Ctx, code string, message string) error {
	return New(c, http.StatusConflict, code, message)
}

// Internal returns 500 — never expose internal error details to the client.
func Internal(c *fiber.Ctx) error {
	return New(c, http.StatusInternalServerError, CodeInternalError, "an unexpected error occurred")
}

// Unauthorized returns 401.
func Unauthorized(c *fiber.Ctx) error {
	return New(c, http.StatusUnauthorized, CodeUnauthorized, "authentication required")
}

// Forbidden returns 403.
func Forbidden(c *fiber.Ctx) error {
	return New(c, http.StatusForbidden, CodeForbidden, "you do not have permission to perform this action")
}
