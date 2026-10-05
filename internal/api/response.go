package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorDetail represents structured error information.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the standard error payload.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// JSON writes a JSON response with status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("failed to encode JSON response", "err", err)
		}
	}
}

// Error writes a structured JSON error response.
func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// InternalError writes a generic 500 error response without leaking details.
func InternalError(w http.ResponseWriter, err error, logMessage string) {
	slog.Error(logMessage, "err", err)
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected internal server error occurred.")
}

// BadRequest writes a 400 bad request error.
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "BAD_REQUEST", message)
}

// Unauthorized writes a 401 unauthorized error.
func Unauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Authentication required or invalid credentials."
	}
	Error(w, http.StatusUnauthorized, "UNAUTHORIZED", message)
}

// Forbidden writes a 403 forbidden error.
func Forbidden(w http.ResponseWriter, message string) {
	if message == "" {
		message = "You do not have permission to perform this action."
	}
	Error(w, http.StatusForbidden, "FORBIDDEN", message)
}

// NotFound writes a 404 not found error.
func NotFound(w http.ResponseWriter, message string) {
	if message == "" {
		message = "The requested resource was not found."
	}
	Error(w, http.StatusNotFound, "NOT_FOUND", message)
}
