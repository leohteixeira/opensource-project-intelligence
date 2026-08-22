package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// ProblemError preserves an internal cause while exposing only frozen safe transport fields.
type ProblemError struct {
	Status   int
	Code     string
	Title    string
	Detail   string
	Instance string
	Errors   []FieldError
	cause    error
}

// FieldError identifies one invalid request field without exposing its rejected value.
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// NewProblem constructs a safe typed transport error.
func NewProblem(status int, code, title, detail string, cause error) *ProblemError {
	return &ProblemError{Status: status, Code: code, Title: title, Detail: detail, cause: cause}
}

func (e *ProblemError) Error() string {
	if e.cause == nil {
		return e.Code
	}
	return fmt.Sprintf("%s: %v", e.Code, e.cause)
}

// Unwrap keeps the original cause available to internal error handling.
func (e *ProblemError) Unwrap() error { return e.cause }

type problemDocument struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Detail    string       `json:"detail,omitempty"`
	Code      string       `json:"code"`
	Instance  string       `json:"instance,omitempty"`
	RequestID string       `json:"request_id"`
	Errors    []FieldError `json:"errors,omitempty"`
}

// WriteProblem serializes a typed error without including its wrapped cause.
func WriteProblem(
	w http.ResponseWriter,
	logger *slog.Logger,
	requestID string,
	err error,
) {
	var typed *ProblemError
	if !errors.As(err, &typed) {
		typed = NewProblem(
			http.StatusInternalServerError,
			"internal_error",
			"Internal server error",
			"The request could not be completed.",
			err,
		)
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(typed.Status)
	document := problemDocument{
		Type:      "https://opensource-project-intelligence.local/problems/" + typed.Code,
		Title:     typed.Title,
		Status:    typed.Status,
		Detail:    typed.Detail,
		Code:      typed.Code,
		Instance:  typed.Instance,
		RequestID: requestID,
		Errors:    typed.Errors,
	}
	if encodeErr := json.NewEncoder(w).Encode(document); encodeErr != nil {
		logger.Error("cannot encode the problem response", slog.Any("error", encodeErr))
	}
}
