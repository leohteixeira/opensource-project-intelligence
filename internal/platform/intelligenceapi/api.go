// Package intelligenceapi exposes deterministic metric, health, contributor, and comparison HTTP operations.
package intelligenceapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/alert"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
	"github.com/leohteixeira/opensource-project-intelligence/internal/comparison"
	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/intelligencestore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/policy"
	"github.com/leohteixeira/opensource-project-intelligence/internal/radar"
	"github.com/leohteixeira/opensource-project-intelligence/internal/topic"
	"github.com/leohteixeira/opensource-project-intelligence/internal/trend"
)

const maxBodyBytes = 64 << 10

type IDSource interface {
	Next(context.Context) (int64, error)
}

type Handler struct {
	httpapi.ServerInterface
	store  *intelligencestore.Store
	ids    IDSource
	logger *slog.Logger
	now    func() time.Time
	model  ModelIdentity
}

type ModelIdentity struct {
	Provider string
	Model    string
}

type Option func(*Handler) error

// WithModelIdentity installs the operator-selected provider/model identity.
// User requests never carry either value.
func WithModelIdentity(identity ModelIdentity) Option {
	return func(handler *Handler) error {
		identity.Provider = strings.TrimSpace(identity.Provider)
		identity.Model = strings.TrimSpace(identity.Model)
		if identity.Provider == "" || identity.Model == "" {
			return analysis.ErrProviderUnavailable
		}
		handler.model = identity
		return nil
	}
}

func New(store *intelligencestore.Store, ids IDSource, logger *slog.Logger,
	options ...Option) (*Handler, error) {
	if store == nil || ids == nil || logger == nil {
		return nil, errors.New("intelligence store, ID source, and logger are required")
	}
	handler := &Handler{store: store, ids: ids, logger: logger, now: time.Now}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("intelligence API option is required")
		}
		if err := option(handler); err != nil {
			return nil, err
		}
	}
	return handler, nil
}

func Routes(handler *Handler) http.Handler {
	generatedMux := http.NewServeMux()
	generated := httpapi.HandlerFromMux(handler, generatedMux)
	mux := http.NewServeMux()
	for _, pattern := range []string{
		"GET /api/v1/projects/{project_id}/metrics",
		"GET /api/v1/projects/{project_id}/metrics/{metric_name}",
		"GET /api/v1/projects/{project_id}/health",
		"GET /api/v1/projects/{project_id}/contributors",
		"POST /api/v1/comparisons", "GET /api/v1/comparisons/{comparison_id}",
		"GET /api/v1/projects/{project_id}/adoption",
		"GET /api/v1/projects/{project_id}/security",
		"GET /api/v1/projects/{project_id}/topics",
		"POST /api/v1/projects/{project_id}/topics/{topic_id}/corrections",
		"GET /api/v1/projects/{project_id}/releases",
		"GET /api/v1/projects/{project_id}/releases/{release_id}",
		"POST /api/v1/projects/{project_id}/crawls",
		"POST /api/v1/projects/{project_id}/knowledge/search",
		"POST /api/v1/projects/{project_id}/queries",
		"GET /api/v1/analysis-runs/{run_id}",
		"POST /api/v1/analysis-runs/{run_id}/reruns",
		"POST /api/v1/analysis-runs/{run_id}/feedback",
		"POST /api/v1/analysis-series/{series_id}/selection",
		"GET /api/v1/projects/{project_id}/trends",
		"GET /api/v1/projects/{project_id}/recommendation",
		"GET /api/v1/policies", "POST /api/v1/policies",
		"POST /api/v1/policies/{policy_id}/versions",
		"GET /api/v1/policies/{policy_id}/versions/{version}",
		"POST /api/v1/policies/{policy_id}/versions/{version}/activation",
		"GET /api/v1/radar", "POST /api/v1/radar/{project_id}/override",
		"DELETE /api/v1/radar/{project_id}/override",
		"POST /api/v1/alert-rules", "PATCH /api/v1/alert-rules/{rule_id}",
		"GET /api/v1/alerts", "POST /api/v1/alerts/{alert_id}/read",
		"POST /api/v1/alerts/{alert_id}/transition",
	} {
		mux.Handle(pattern, generated)
	}
	return mux
}

func (h *Handler) GetApiV1ProjectsProjectIdMetrics(w http.ResponseWriter, r *http.Request, rawID string, params httpapi.GetApiV1ProjectsProjectIdMetricsParams) {
	projectID, window, ok := h.requestWindow(w, r, rawID, params.Window, params.Cutoff)
	if !ok {
		return
	}
	values, err := h.store.Metrics(r.Context(), accessapi.Principal(r.Context()), projectID, window)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if params.Dimension != nil && strings.TrimSpace(*params.Dimension) != "" {
		filtered := values[:0]
		for _, value := range values {
			if metricDimension(value.Definition.Name) == strings.ToLower(strings.TrimSpace(*params.Dimension)) {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if offset > len(values) {
		offset = len(values)
	}
	limit := pageLimit(params.Limit)
	end := min(len(values), offset+limit)
	response := map[string]any{"items": values[offset:end], "has_more": end < len(values), "window": window}
	if end < len(values) {
		response["next_cursor"] = encodeCursor(end)
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetApiV1ProjectsProjectIdMetricsMetricName(w http.ResponseWriter, r *http.Request, rawID, name string, params httpapi.GetApiV1ProjectsProjectIdMetricsMetricNameParams) {
	projectID, window, ok := h.requestWindow(w, r, rawID, params.Window, params.Cutoff)
	if !ok {
		return
	}
	value, err := h.store.Metric(r.Context(), accessapi.Principal(r.Context()), projectID, name, window)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) GetApiV1ProjectsProjectIdHealth(w http.ResponseWriter, r *http.Request, rawID string, params httpapi.GetApiV1ProjectsProjectIdHealthParams) {
	projectID, window, ok := h.requestWindow(w, r, rawID, params.Window, params.Cutoff)
	if !ok {
		return
	}
	value, err := h.store.Health(r.Context(), accessapi.Principal(r.Context()), projectID, window)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) GetApiV1ProjectsProjectIdContributors(w http.ResponseWriter, r *http.Request, rawID string, params httpapi.GetApiV1ProjectsProjectIdContributorsParams) {
	projectID, window, ok := h.requestWindow(w, r, rawID, params.Window, nil)
	if !ok {
		return
	}
	limit := pageLimit(params.Limit)
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.Contributors(r.Context(), accessapi.Principal(r.Context()), projectID, window, limit, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func encodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeCursor(raw *string) (int, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return 0, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(*raw)
	if err != nil {
		return 0, metric.ErrInvalid
	}
	offset, err := strconv.Atoi(string(decoded))
	if err != nil || offset < 0 {
		return 0, metric.ErrInvalid
	}
	return offset, nil
}

func metricDimension(name string) string {
	switch name {
	case "release_frequency":
		return "activity"
	case "active_contributors":
		return "community"
	case "top_three_author_share":
		return "concentration"
	case "median_pr_merge_time":
		return "stability"
	default:
		return "maintenance"
	}
}

func (h *Handler) PostApiV1Comparisons(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
		h.problem(w, r, metric.ErrInvalid)
		return
	}
	var body struct {
		ProjectIDs []string `json:"project_ids"`
		Window     string   `json:"window"`
		Cutoff     string   `json:"cutoff"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	cutoff, err := time.Parse(time.RFC3339, body.Cutoff)
	if err != nil {
		h.problem(w, r, metric.ErrInvalid)
		return
	}
	window, err := metric.ResolveWindow(body.Window, cutoff)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	projects := make([]comparison.Project, 0, len(body.ProjectIDs))
	for _, rawID := range body.ProjectIDs {
		projectID, parseErr := parseID(rawID)
		if parseErr != nil {
			h.problem(w, r, parseErr)
			return
		}
		metrics, loadErr := h.store.Metrics(r.Context(), accessapi.Principal(r.Context()), projectID, window)
		if loadErr != nil {
			h.problem(w, r, loadErr)
			return
		}
		projects = append(projects, comparison.Project{ID: projectID, Resolved: true, Metrics: metrics})
	}
	id, err := h.ids.Next(r.Context())
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := comparison.Materialize(id, projects, window)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err = h.store.SaveComparison(r.Context(), accessapi.Principal(r.Context()), value)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) GetApiV1ComparisonsComparisonId(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.Comparison(r.Context(), accessapi.Principal(r.Context()), id)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) requestWindow(w http.ResponseWriter, r *http.Request, rawID string, rawWindow, rawCutoff *string) (int64, metric.Window, bool) {
	projectID, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return 0, metric.Window{}, false
	}
	windowName := "90d"
	if rawWindow != nil && strings.TrimSpace(*rawWindow) != "" {
		windowName = *rawWindow
	}
	cutoff := h.now().UTC().Truncate(24 * time.Hour)
	if rawCutoff != nil && strings.TrimSpace(*rawCutoff) != "" {
		cutoff, err = time.Parse(time.RFC3339, *rawCutoff)
		if err != nil {
			h.problem(w, r, metric.ErrInvalid)
			return 0, metric.Window{}, false
		}
	}
	window, err := metric.ResolveWindow(windowName, cutoff)
	if err != nil {
		h.problem(w, r, err)
		return 0, metric.Window{}, false
	}
	return projectID, window, true
}

func pageLimit(raw *int32) int {
	if raw == nil || *raw <= 0 {
		return 50
	}
	return min(int(*raw), 200)
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, metric.ErrInvalid
	}
	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: malformed body", metric.ErrInvalid)
	}
	if decoder.Decode(&struct{}{}) == nil {
		return fmt.Errorf("%w: multiple documents", metric.ErrInvalid)
	}
	return nil
}

func (h *Handler) problem(w http.ResponseWriter, r *http.Request, err error) {
	status, code, title, detail := http.StatusInternalServerError, "internal_error", "Internal server error", "The request could not be completed."
	switch {
	case errors.Is(err, access.ErrAuthenticationRequired):
		status, code, title, detail = http.StatusUnauthorized, "authentication_required", "Authentication required", "Sign in to continue."
	case errors.Is(err, access.ErrAccessPending), errors.Is(err, access.ErrPermissionDenied):
		status, code, title, detail = http.StatusForbidden, "permission_denied", "Permission denied", "Your role cannot inspect this intelligence."
	case errors.Is(err, access.ErrNotFound), errors.Is(err, pgx.ErrNoRows):
		status, code, title, detail = http.StatusNotFound, "not_found", "Resource not found", "The requested resource is unavailable."
	case errors.Is(err, access.ErrVersionConflict):
		status, code, title, detail = http.StatusConflict, "version_conflict", "Version conflict", "Reload the current version and retry the attributed change."
	case errors.Is(err, analysis.ErrProviderUnavailable):
		status, code, title, detail = http.StatusServiceUnavailable, "ai_provider_unavailable", "AI provider unavailable", "The operator has not configured an available model provider. Deterministic intelligence remains available."
	case errors.Is(err, metric.ErrInvalid), errors.Is(err, comparison.ErrInvalid),
		errors.Is(err, knowledge.ErrInvalid), errors.Is(err, analysis.ErrInvalidRun),
		errors.Is(err, analysis.ErrSchema), errors.Is(err, analysis.ErrEvidence),
		errors.Is(err, analysis.ErrSelection), errors.Is(err, topic.ErrInvalid),
		errors.Is(err, trend.ErrInvalid), errors.Is(err, policy.ErrInvalid),
		errors.Is(err, radar.ErrInvalid), errors.Is(err, alert.ErrInvalid):
		status, code, title, detail = http.StatusBadRequest, "invalid_request", "Invalid request", "Use two to five distinct Projects, a supported window, and one exact cutoff."
	}
	typed := httpx.NewProblem(status, code, title, detail, err)
	httpx.WriteProblem(w, h.logger, accessapi.RequestID(r.Context()), typed)
}

func parseVersionHeader(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 4 || raw[0] != '"' || raw[1] != 'v' || raw[len(raw)-1] != '"' {
		return 0, analysis.ErrInvalidRun
	}
	version, err := strconv.ParseInt(raw[2:len(raw)-1], 10, 64)
	if err != nil || version < 0 {
		return 0, analysis.ErrInvalidRun
	}
	return version, nil
}

func (h *Handler) writeVersionedJSON(w http.ResponseWriter, status int, version int64, value any) {
	w.Header().Set("ETag", strconv.Quote("v"+strconv.FormatInt(version, 10)))
	h.writeJSON(w, status, value)
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	httpx.WriteJSON(w, h.logger, status, value)
}
