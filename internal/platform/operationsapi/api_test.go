package operationsapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis/agent"
	exportartifact "github.com/leohteixeira/opensource-project-intelligence/internal/export"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/exportstore"
)

type fakeAssistant struct {
	proposal   agent.Proposal
	err        error
	confirm    agent.Proposal
	confirmErr error
}

func (fake fakeAssistant) Propose(context.Context, access.Principal, string, string) (agent.Proposal, error) {
	return fake.proposal, fake.err
}

func (fake fakeAssistant) Confirm(context.Context, access.Principal, int64, string, string) (agent.Proposal, error) {
	return fake.confirm, fake.confirmErr
}

type fakeExports struct {
	value       exportstore.Export
	body        []byte
	createErr   error
	getErr      error
	downloadErr error
}

func (fake fakeExports) Create(context.Context, access.Principal, exportartifact.Request, string) (exportstore.Export, error) {
	return fake.value, fake.createErr
}

func (fake fakeExports) Get(context.Context, access.Principal, int64) (exportstore.Export, error) {
	return fake.value, fake.getErr
}

func (fake fakeExports) Download(context.Context, access.Principal, int64) (exportstore.Export, []byte, error) {
	return fake.value, fake.body, fake.downloadErr
}

func operationHandler(t *testing.T, assistant fakeAssistant, exports fakeExports) http.Handler {
	t.Helper()
	handler, err := New(assistant, exports, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return Routes(handler)
}

func perform(handler http.Handler, method, target, body, key string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func validExportBody() string {
	return `{"project_ids":[42],"resource":"metrics","format":"csv","locale":"en",` +
		`"window_from":"2026-05-01T00:00:00Z","window_to":"2026-08-01T00:00:00Z",` +
		`"cutoff":"2026-08-01T00:00:00Z"}`
}

func TestIT289PostExportsReturnsAcceptedJob(t *testing.T) {
	handler := operationHandler(t, fakeAssistant{}, fakeExports{value: exportstore.Export{ID: 9, JobID: 10, State: "queued"}})
	response := perform(handler, http.MethodPost, "/api/v1/exports", validExportBody(), "export-1")
	if response.Code != http.StatusAccepted || !strings.Contains(response.Body.String(), `"job_id":"10"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestIT290PostExportsMapsValidationAndAuthorizationFailures(t *testing.T) {
	for _, err := range []error{exportartifact.ErrInvalidRequest, access.ErrPermissionDenied} {
		handler := operationHandler(t, fakeAssistant{}, fakeExports{createErr: err})
		response := perform(handler, http.MethodPost, "/api/v1/exports", validExportBody(), "export-1")
		if response.Code < 400 {
			t.Fatalf("error %v returned %d", err, response.Code)
		}
	}
}

func TestIT291GetExportMetadataReturnsLifecycleAndChecksum(t *testing.T) {
	value := exportstore.Export{ID: 9, State: "succeeded", SHA256: strings.Repeat("a", 64), SizeBytes: 5}
	response := perform(operationHandler(t, fakeAssistant{}, fakeExports{value: value}), http.MethodGet,
		"/api/v1/exports/9", "", "")
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"sha256"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestIT292GetExportMetadataEnforcesAuthorization(t *testing.T) {
	response := perform(operationHandler(t, fakeAssistant{}, fakeExports{getErr: access.ErrPermissionDenied}),
		http.MethodGet, "/api/v1/exports/9", "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestIT293DownloadReturnsArtifactAndRejectsExpiry(t *testing.T) {
	value := exportstore.Export{ID: 9, State: "succeeded", MediaType: "text/csv", SHA256: strings.Repeat("a", 64),
		Request: exportartifact.Request{Resource: "metrics", Format: exportartifact.CSV}}
	handler := operationHandler(t, fakeAssistant{}, fakeExports{value: value, body: []byte("proof")})
	response := perform(handler, http.MethodGet, "/api/v1/exports/9/download", "", "")
	if response.Code != http.StatusOK || response.Body.String() != "proof" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	expired := operationHandler(t, fakeAssistant{}, fakeExports{downloadErr: exportartifact.ErrExpired})
	if got := perform(expired, http.MethodGet, "/api/v1/exports/9/download", "", "").Code; got != http.StatusGone {
		t.Fatalf("expired status = %d", got)
	}
}

func TestIT294DownloadEnforcesAuthorization(t *testing.T) {
	response := perform(operationHandler(t, fakeAssistant{}, fakeExports{downloadErr: access.ErrPermissionDenied}),
		http.MethodGet, "/api/v1/exports/9/download", "", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestIT295PostAssistantProposalReturnsPreviewOrValidationFailure(t *testing.T) {
	proposal := agent.Proposal{ID: 7, Status: agent.AwaitingConfirmation, Action: agent.ActionRepositoryAdd,
		ExpiresAt: time.Date(2026, 8, 22, 12, 10, 0, 0, time.UTC)}
	response := perform(operationHandler(t, fakeAssistant{proposal: proposal}, fakeExports{}), http.MethodPost,
		"/api/v1/assistant/proposals", `{"message":"add repository"}`, "proposal-1")
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), "awaiting_confirmation") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
	invalid := operationHandler(t, fakeAssistant{err: agent.ErrActionNotAllowed}, fakeExports{})
	if got := perform(invalid, http.MethodPost, "/api/v1/assistant/proposals", `{"message":"delete"}`, "proposal-2").Code; got != http.StatusUnprocessableEntity {
		t.Fatalf("invalid status = %d", got)
	}
}

func TestIT296PostAssistantProposalEnforcesIdentityAndScope(t *testing.T) {
	handler := operationHandler(t, fakeAssistant{err: access.ErrPermissionDenied}, fakeExports{})
	response := perform(handler, http.MethodPost, "/api/v1/assistant/proposals", `{"message":"add"}`, "proposal")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestIT297PostConfirmationReturnsExecutionReceipt(t *testing.T) {
	receipt := agent.Proposal{ID: 7, Status: agent.Executed, Result: agent.Result{RepositoryID: 12, AuditEventID: 13}}
	handler := operationHandler(t, fakeAssistant{confirm: receipt}, fakeExports{})
	response := perform(handler, http.MethodPost, "/api/v1/assistant/proposals/7/confirmation",
		`{"confirmation_token":"secret"}`, "confirm-1")
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"audit_event_id":"13"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestIT298PostConfirmationMapsExpiredReplayAndChangedState(t *testing.T) {
	for err, want := range map[error]int{agent.ErrExpired: http.StatusGone, agent.ErrAlreadyUsed: http.StatusConflict,
		agent.ErrStateChanged: http.StatusPreconditionFailed, errors.New("write failed"): http.StatusInternalServerError} {
		handler := operationHandler(t, fakeAssistant{confirmErr: err}, fakeExports{})
		response := perform(handler, http.MethodPost, "/api/v1/assistant/proposals/7/confirmation",
			`{"confirmation_token":"secret"}`, "confirm-1")
		if response.Code != want {
			t.Fatalf("error %v status = %d, want %d", err, response.Code, want)
		}
	}
}
