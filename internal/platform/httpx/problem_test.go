package httpx_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
)

func TestUT271ProblemSerializationDoesNotLeakCause(t *testing.T) {
	t.Parallel()

	const (
		requestID = "request-732684512931872768"
		secret    = "wrapped-cause-must-not-be-exposed"
	)
	err := httpx.NewProblem(
		http.StatusConflict,
		"stale_version",
		"The resource changed",
		"Refresh the resource and retry with its current version.",
		errors.New(secret),
	)
	err.Instance = "/api/v1/projects"
	err.Errors = []httpx.FieldError{{Field: "repository_url", Code: "private_network_target"}}
	recorder := httptest.NewRecorder()

	httpx.WriteProblem(recorder, slog.New(slog.NewTextHandler(io.Discard, nil)), requestID, err)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusConflict)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`"code":"stale_version"`,
		`"status":409`,
		`"instance":"/api/v1/projects"`,
		`"errors":[{"field":"repository_url","code":"private_network_target"}]`,
		requestID,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body = %q, want %q", body, want)
		}
	}
	if strings.Contains(body, secret) {
		t.Fatalf("body leaked wrapped cause: %q", body)
	}
}
