package api

import (
	"errors"
	"log/slog"
	"net/http"

	"deps-api/internal/utils"

	"github.com/google/uuid"
)

type HTTPError struct {
	Status  int    `json:"-"`
	Message string `json:"error"`
	ErrorID string `json:"error_id,omitempty"`
}

func (e *HTTPError) Error() string { return e.Message }

func ErrNotFound(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusNotFound, Message: msg}
}

func ErrBadRequest(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusBadRequest, Message: msg}
}

func ErrUnprocessable(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusUnprocessableEntity, Message: msg}
}

func ErrConflict(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusConflict, Message: msg}
}

func ErrUnauthorized(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusUnauthorized, Message: msg}
}

func errInternal(id string) *HTTPError {
	return &HTTPError{Status: http.StatusInternalServerError, Message: "internal server error", ErrorID: id}
}

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		errorID := uuid.New().String()
		slog.ErrorContext(r.Context(), "unexpected error",
			"error_id", errorID,
			"status_code", http.StatusInternalServerError,
			"message", "internal server error",
			"url", r.URL.Path,
			"method", r.Method,
			"ip", utils.GetRequestIP(r),
			"error", err,
		)
		JSON(w, http.StatusInternalServerError, errInternal(errorID))
		return
	}

	slog.ErrorContext(r.Context(), "http error",
		"status_code", httpErr.Status,
		"message", httpErr.Message,
		"url", r.URL.Path,
		"method", r.Method,
		"ip", utils.GetRequestIP(r),
	)
	JSON(w, httpErr.Status, httpErr)
}
