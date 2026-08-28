// Package operationsapi exposes bounded assistant and export operations.
package operationsapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis/agent"
	exportartifact "github.com/leohteixeira/opensource-project-intelligence/internal/export"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/exportstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

const maxBodyBytes = 64 << 10

var errMalformedRequest = errors.New("malformed request")

type Assistant interface {
	Propose(context.Context, access.Principal, string, string) (agent.Proposal, error)
	Confirm(context.Context, access.Principal, int64, string, string) (agent.Proposal, error)
}

type Exports interface {
	Create(context.Context, access.Principal, exportartifact.Request, string) (exportstore.Export, error)
	Get(context.Context, access.Principal, int64) (exportstore.Export, error)
	Download(context.Context, access.Principal, int64) (exportstore.Export, []byte, error)
}

type Handler struct {
	httpapi.ServerInterface
	assistant Assistant
	exports   Exports
	logger    *slog.Logger
}

func New(assistant Assistant, exports Exports, logger *slog.Logger) (*Handler, error) {
	if logger == nil {
		return nil, errors.New("operations logger is required")
	}
	return &Handler{assistant: assistant, exports: exports, logger: logger}, nil
}

func Routes(handler *Handler) http.Handler {
	generatedMux := http.NewServeMux()
	generated := httpapi.HandlerFromMux(handler, generatedMux)
	mux := http.NewServeMux()
	for _, pattern := range []string{
		"POST /api/v1/assistant/proposals",
		"POST /api/v1/assistant/proposals/{proposal_id}/confirmation",
		"POST /api/v1/exports", "GET /api/v1/exports/{export_id}",
		"GET /api/v1/exports/{export_id}/download",
	} {
		mux.Handle(pattern, generated)
	}
	return mux
}

func (h *Handler) PostApiV1AssistantProposals(
	w http.ResponseWriter,
	r *http.Request,
	params httpapi.PostApiV1AssistantProposalsParams,
) {
	if h.assistant == nil {
		h.problem(w, r, agent.ErrUnavailable)
		return
	}
	var body struct {
		Message string `json:"message"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.assistant.Propose(r.Context(), accessapi.Principal(r.Context()), body.Message,
		params.IdempotencyKey)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) PostApiV1AssistantProposalsProposalIdConfirmation(
	w http.ResponseWriter,
	r *http.Request,
	rawID string,
	params httpapi.PostApiV1AssistantProposalsProposalIdConfirmationParams,
) {
	if h.assistant == nil {
		h.problem(w, r, agent.ErrUnavailable)
		return
	}
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		ConfirmationToken string `json:"confirmation_token"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.assistant.Confirm(r.Context(), accessapi.Principal(r.Context()), id,
		body.ConfirmationToken, params.IdempotencyKey)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) PostApiV1Exports(
	w http.ResponseWriter,
	r *http.Request,
	params httpapi.PostApiV1ExportsParams,
) {
	if h.exports == nil {
		h.problem(w, r, agent.ErrUnavailable)
		return
	}
	var request exportartifact.Request
	if err := decodeJSON(w, r, &request); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.exports.Create(r.Context(), accessapi.Principal(r.Context()), request,
		params.IdempotencyKey)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) GetApiV1ExportsExportId(w http.ResponseWriter, r *http.Request, rawID string) {
	if h.exports == nil {
		h.problem(w, r, agent.ErrUnavailable)
		return
	}
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.exports.Get(r.Context(), accessapi.Principal(r.Context()), id)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) GetApiV1ExportsExportIdDownload(w http.ResponseWriter, r *http.Request, rawID string) {
	if h.exports == nil {
		h.problem(w, r, agent.ErrUnavailable)
		return
	}
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, body, err := h.exports.Download(r.Context(), accessapi.Principal(r.Context()), id)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("Content-Type", value.MediaType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+exportstore.Filename(value)+`"`)
	w.Header().Set("Digest", "sha-256="+value.SHA256)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		h.logger.Warn("write export download", slog.Any("error", err))
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: malformed body", errMalformedRequest)
	}
	if decoder.Decode(&struct{}{}) == nil {
		return fmt.Errorf("%w: multiple documents", errMalformedRequest)
	}
	return nil
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, agent.ErrInvalid
	}
	return id, nil
}

func (h *Handler) problem(w http.ResponseWriter, r *http.Request, err error) {
	status, code, title, detail := http.StatusInternalServerError, "internal_error", "Internal server error", "The operation could not be completed."
	switch {
	case errors.Is(err, errMalformedRequest):
		status, code, title, detail = http.StatusBadRequest, "invalid_request", "Invalid request", "Correct the malformed request body."
	case errors.Is(err, access.ErrAuthenticationRequired):
		status, code, title, detail = http.StatusUnauthorized, "authentication_required", "Authentication required", "Sign in to continue."
	case errors.Is(err, access.ErrAccessPending):
		status, code, title, detail = http.StatusForbidden, "access_pending", "Access pending", "An Admin must approve this membership."
	case errors.Is(err, access.ErrPermissionDenied):
		status, code, title, detail = http.StatusForbidden, "permission_denied", "Permission denied", "Your role cannot perform this operation."
	case errors.Is(err, project.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		status, code, title, detail = http.StatusNotFound, "resource_not_found", "Resource not found", "The requested resource is unavailable."
	case errors.Is(err, exportartifact.ErrExpired):
		status, code, title, detail = http.StatusGone, "export_expired", "Export expired", "Request a new export."
	case errors.Is(err, agent.ErrExpired):
		status, code, title, detail = http.StatusGone, "resource_expired", "Proposal expired", "Create a fresh proposal."
	case errors.Is(err, agent.ErrIdempotencyKey), errors.Is(err, exportartifact.ErrIdempotencyKey):
		status, code, title, detail = http.StatusConflict, "idempotency_key_reused", "Idempotency key reused", "Generate a new key for the changed request."
	case errors.Is(err, agent.ErrStateChanged):
		status, code, title, detail = http.StatusPreconditionFailed, "version_conflict", "Version conflict", "Fetch current state and review a new proposal."
	case errors.Is(err, agent.ErrAlreadyUsed), errors.Is(err, exportartifact.ErrNotReady):
		status, code, title, detail = http.StatusConflict, "state_conflict", "State conflict", "Reload current state before retrying."
	case errors.Is(err, exportartifact.ErrBusy), errors.Is(err, exportartifact.ErrTooLarge),
		errors.Is(err, agent.ErrQuotaExceeded), errors.Is(err, agent.ErrRunLimit):
		status, code, title, detail = http.StatusTooManyRequests, "rate_limited", "Capacity exhausted", "Retry after reducing scope or after current work completes."
	case errors.Is(err, agent.ErrUnavailable):
		status, code, title, detail = http.StatusServiceUnavailable, "ai_provider_unavailable", "AI provider unavailable", "Deterministic product capabilities remain available."
	case errors.Is(err, agent.ErrActionNotAllowed):
		status, code, title, detail = http.StatusUnprocessableEntity, "action_not_allowed", "Action not allowed", "Use the conventional authorized surface."
	case errors.Is(err, agent.ErrInvalid), errors.Is(err, exportartifact.ErrInvalidRequest):
		status, code, title, detail = http.StatusBadRequest, "invalid_request", "Invalid request", "Review the typed scope, format, and cutoff."
	}
	httpx.WriteProblem(w, h.logger, accessapi.RequestID(r.Context()),
		httpx.NewProblem(status, code, title, detail, err))
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	httpx.WriteJSON(w, h.logger, status, value)
}
