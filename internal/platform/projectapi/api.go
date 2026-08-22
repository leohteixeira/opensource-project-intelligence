// Package projectapi exposes Portfolio, Project, source, and durable Job HTTP operations.
package projectapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
	"github.com/leohteixeira/opensource-project-intelligence/internal/job"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/projectstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/project"
)

const maxBodyBytes = 64 << 10

type Handler struct {
	httpapi.ServerInterface

	store   *projectstore.Store
	cursors *access.CursorCodec
	logger  *slog.Logger
	urls    URLValidator
	wakeups Wakeups
}

type URLValidator interface {
	Validate(context.Context, string) (*url.URL, error)
}

// Wakeups is optional disposable acceleration. A notification never carries
// authority; the handler always re-reads PostgreSQL before emitting an event.
type Wakeups interface {
	Subscribe(context.Context, string) (<-chan struct{}, error)
}

func New(store *projectstore.Store, cursors *access.CursorCodec, logger *slog.Logger) (*Handler, error) {
	return NewWithURLValidator(store, cursors, logger, collector.PublicURLPolicy{})
}

func NewWithURLValidator(
	store *projectstore.Store,
	cursors *access.CursorCodec,
	logger *slog.Logger,
	urls URLValidator,
) (*Handler, error) {
	if store == nil || cursors == nil || logger == nil || urls == nil {
		return nil, errors.New("project store, cursor codec, and logger are required")
	}
	return &Handler{store: store, cursors: cursors, logger: logger, urls: urls}, nil
}

func (h *Handler) UseWakeups(wakeups Wakeups) error {
	if wakeups == nil {
		return errors.New("Job wake-up adapter is required")
	}
	h.wakeups = wakeups
	return nil
}

func Routes(handler *Handler) http.Handler {
	generatedMux := http.NewServeMux()
	generated := httpapi.HandlerFromMux(handler, generatedMux)
	mux := http.NewServeMux()
	for _, pattern := range []string{
		"GET /api/v1/portfolio",
		"GET /api/v1/projects", "POST /api/v1/projects",
		"GET /api/v1/projects/{project_id}", "PATCH /api/v1/projects/{project_id}",
		"POST /api/v1/projects/{project_id}/transition", "POST /api/v1/projects/{project_id}/deletion",
		"GET /api/v1/projects/{project_id}/repositories", "POST /api/v1/projects/{project_id}/repositories",
		"PATCH /api/v1/projects/{project_id}/repositories/{repository_id}",
		"DELETE /api/v1/projects/{project_id}/repositories/{repository_id}",
		"GET /api/v1/projects/{project_id}/sources", "POST /api/v1/projects/{project_id}/sources",
		"PATCH /api/v1/projects/{project_id}/sources/{source_id}",
		"DELETE /api/v1/projects/{project_id}/sources/{source_id}",
		"GET /api/v1/projects/{project_id}/associations",
		"POST /api/v1/projects/{project_id}/associations/{association_id}/correction",
		"POST /api/v1/projects/{project_id}/syncs", "POST /api/v1/projects/{project_id}/history-requests",
		"GET /api/v1/projects/{project_id}/jobs", "GET /api/v1/jobs/{job_id}",
		"GET /api/v1/jobs/{job_id}/events", "POST /api/v1/jobs/{job_id}/cancellation",
	} {
		mux.Handle(pattern, generated)
	}
	return mux
}

func (h *Handler) GetApiV1Portfolio(w http.ResponseWriter, r *http.Request, _ httpapi.GetApiV1PortfolioParams) {
	value, err := h.store.Portfolio(r.Context(), accessapi.Principal(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) GetApiV1Projects(w http.ResponseWriter, r *http.Request, params httpapi.GetApiV1ProjectsParams) {
	filter, offset, err := h.projectFilter(params)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	values, err := h.store.ListProjects(r.Context(), accessapi.Principal(r.Context()), filter)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "projects", projectFilterKey(params), offset, filter.Limit, values)
}

func (h *Handler) PostApiV1Projects(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RepositoryURL string `json:"repository_url"`
		HistoryDays   int    `json:"history_days"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	validatedURL, err := h.urls.Validate(r.Context(), body.RepositoryURL)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.Register(r.Context(), accessapi.Principal(r.Context()), validatedURL.String(),
		body.HistoryDays, r.Header.Get("Idempotency-Key"), accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Project.Version))
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) GetApiV1ProjectsProjectId(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.GetProject(r.Context(), accessapi.Principal(r.Context()), id)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Version))
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) PatchApiV1ProjectsProjectId(w http.ResponseWriter, r *http.Request, rawID string) {
	id, expectedVersion, ok := h.mutationIdentity(w, r, rawID)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.UpdateProject(r.Context(), accessapi.Principal(r.Context()), id,
		expectedVersion, body.Name, body.Description, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Version))
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) PostApiV1ProjectsProjectIdTransition(w http.ResponseWriter, r *http.Request, rawID string) {
	id, expectedVersion, ok := h.mutationIdentity(w, r, rawID)
	if !ok {
		return
	}
	var body struct {
		To     project.State `json:"to"`
		Reason string        `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.Transition(r.Context(), accessapi.Principal(r.Context()), id,
		expectedVersion, body.To, body.Reason, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Project.Version))
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) PostApiV1ProjectsProjectIdDeletion(w http.ResponseWriter, r *http.Request, rawID string) {
	id, expectedVersion, ok := h.mutationIdentity(w, r, rawID)
	if !ok {
		return
	}
	var body struct {
		Confirmation string `json:"confirmation"`
		Reason       string `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.RequestDeletion(r.Context(), accessapi.Principal(r.Context()), id,
		expectedVersion, body.Confirmation, body.Reason, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Project.Version))
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) GetApiV1ProjectsProjectIdRepositories(
	w http.ResponseWriter,
	r *http.Request,
	rawID string,
	params httpapi.GetApiV1ProjectsProjectIdRepositoriesParams,
) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filter, offset, filters, err := h.pageFilter("repositories", "", params.Cursor, params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	values, err := h.store.ListRepositories(r.Context(), accessapi.Principal(r.Context()), id, filter)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "repositories", filters, offset, filter.Limit, values)
}

func (h *Handler) PostApiV1ProjectsProjectIdRepositories(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		URL  string                 `json:"url"`
		Role project.RepositoryRole `json:"role"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	validatedURL, err := h.urls.Validate(r.Context(), body.URL)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.AddRepository(r.Context(), accessapi.Principal(r.Context()), id,
		validatedURL.String(), body.Role, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Version))
	h.writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) PatchApiV1ProjectsProjectIdRepositoriesRepositoryId(
	w http.ResponseWriter,
	r *http.Request,
	rawProjectID, rawRepositoryID string,
) {
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	repositoryID, err := parseID(rawRepositoryID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Role project.RepositoryRole `json:"role"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.ChangeRepositoryRole(r.Context(), accessapi.Principal(r.Context()),
		projectID, repositoryID, version, body.Role, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Version))
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteApiV1ProjectsProjectIdRepositoriesRepositoryId(
	w http.ResponseWriter,
	r *http.Request,
	rawProjectID, rawRepositoryID string,
) {
	projectID, repositoryID, err := parsePair(rawProjectID, rawRepositoryID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if err := h.store.RemoveRepository(r.Context(), accessapi.Principal(r.Context()),
		projectID, repositoryID, version, accessapi.RequestID(r.Context())); err != nil {
		h.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetApiV1ProjectsProjectIdSources(
	w http.ResponseWriter,
	r *http.Request,
	rawID string,
	params httpapi.GetApiV1ProjectsProjectIdSourcesParams,
) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filters := pointer(params.Kind) + "|" + pointer(params.State)
	filter, offset, filters, err := h.pageFilter("sources", filters, params.Cursor, params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filter.Kind, filter.State = pointer(params.Kind), pointer(params.State)
	values, err := h.store.ListSources(r.Context(), accessapi.Principal(r.Context()), id, filter)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "sources", filters, offset, filter.Limit, values)
}

func (h *Handler) PostApiV1ProjectsProjectIdSources(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Kind project.SourceKind `json:"kind"`
		URL  string             `json:"url"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	validatedURL, err := h.urls.Validate(r.Context(), body.URL)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.AddSource(r.Context(), accessapi.Principal(r.Context()), id,
		body.Kind, validatedURL.String(), accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Version))
	h.writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) PatchApiV1ProjectsProjectIdSourcesSourceId(
	w http.ResponseWriter,
	r *http.Request,
	rawProjectID, rawSourceID string,
) {
	projectID, sourceID, err := parsePair(rawProjectID, rawSourceID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		State project.SourceState `json:"state"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.UpdateSource(r.Context(), accessapi.Principal(r.Context()), projectID,
		sourceID, version, body.State, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Version))
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) DeleteApiV1ProjectsProjectIdSourcesSourceId(
	w http.ResponseWriter,
	r *http.Request,
	rawProjectID, rawSourceID string,
) {
	projectID, sourceID, err := parsePair(rawProjectID, rawSourceID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.RemoveSource(r.Context(), accessapi.Principal(r.Context()), projectID,
		sourceID, version, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) GetApiV1ProjectsProjectIdAssociations(
	w http.ResponseWriter,
	r *http.Request,
	rawID string,
	params httpapi.GetApiV1ProjectsProjectIdAssociationsParams,
) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filters := pointer(params.Status)
	filter, offset, filters, err := h.pageFilter("associations", filters, params.Cursor, params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filter.State = pointer(params.Status)
	values, err := h.store.ListAssociations(r.Context(), accessapi.Principal(r.Context()), id, filter)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "associations", filters, offset, filter.Limit, values)
}

func (h *Handler) PostApiV1ProjectsProjectIdAssociationsAssociationIdCorrection(
	w http.ResponseWriter,
	r *http.Request,
	rawProjectID, rawAssociationID string,
) {
	projectID, associationID, err := parsePair(rawProjectID, rawAssociationID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Action          string `json:"action"`
		TargetProjectID string `json:"target_project_id"`
		Reason          string `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	var targetID int64
	if body.TargetProjectID != "" {
		targetID, err = parseID(body.TargetProjectID)
		if err != nil {
			h.problem(w, r, err)
			return
		}
	}
	jobID, changed, err := h.store.CorrectAssociation(r.Context(), accessapi.Principal(r.Context()),
		projectID, associationID, targetID, body.Action, body.Reason, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, map[string]any{"job_id": strconv.FormatInt(jobID, 10), "changed": changed})
}

func (h *Handler) PostApiV1ProjectsProjectIdSyncs(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Scope string `json:"scope"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.QueueSync(r.Context(), accessapi.Principal(r.Context()), id,
		body.Scope, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) PostApiV1ProjectsProjectIdHistoryRequests(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	from, err := parseHistoryDate(body.From)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	to := time.Now().UTC().AddDate(0, 0, 1)
	if strings.TrimSpace(body.To) != "" {
		to, err = parseHistoryDate(body.To)
		if err != nil {
			h.problem(w, r, err)
			return
		}
	}
	value, err := h.store.QueueHistory(r.Context(), accessapi.Principal(r.Context()), id,
		from, to, accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) GetApiV1ProjectsProjectIdJobs(
	w http.ResponseWriter,
	r *http.Request,
	rawID string,
	params httpapi.GetApiV1ProjectsProjectIdJobsParams,
) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filters := pointer(params.Kind) + "|" + pointer(params.State)
	filter, offset, filters, err := h.pageFilter("jobs", filters, params.Cursor, params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filter.Kind, filter.State = pointer(params.Kind), pointer(params.State)
	values, err := h.store.ListJobs(r.Context(), accessapi.Principal(r.Context()), id, filter)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "jobs", filters, offset, filter.Limit, values)
}

func (h *Handler) GetApiV1JobsJobId(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.GetJob(r.Context(), accessapi.Principal(r.Context()), id)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", etag(value.Version))
	if value.State == job.Queued || value.State == job.Running {
		w.Header().Set("Retry-After", "2")
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) GetApiV1JobsJobIdEvents(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	after := int64(0)
	if value := strings.TrimSpace(r.Header.Get("Last-Event-ID")); value != "" {
		after, err = parseEventID(value)
		if err != nil {
			h.problem(w, r, err)
			return
		}
	}
	values, err := h.store.JobEvents(r.Context(), accessapi.Principal(r.Context()), id, after, 100)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		h.streamJobEvents(w, r, id, after, values)
		return
	}
	items := make([]job.Job, 0, len(values))
	for _, value := range values {
		items = append(items, value.Job)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": items, "polling_fallback": true})
}

func (h *Handler) streamJobEvents(
	w http.ResponseWriter,
	r *http.Request,
	jobID, after int64,
	initial []projectstore.JobEvent,
) {
	current, err := h.store.GetJob(r.Context(), accessapi.Principal(r.Context()), jobID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	emit := func(values []projectstore.JobEvent) bool {
		for _, value := range values {
			encoded, marshalErr := json.Marshal(value.Job)
			if marshalErr != nil {
				return false
			}
			if _, writeErr := fmt.Fprintf(w, "id: %d\nevent: job.updated\ndata: %s\n\n", value.ID, encoded); writeErr != nil {
				return false
			}
			after = value.ID
			current = value.Job
		}
		if flusher != nil {
			flusher.Flush()
		}
		return true
	}
	if !emit(initial) || terminalJob(current.State) {
		return
	}
	var wakeups <-chan struct{}
	if h.wakeups != nil {
		wakeups, err = h.wakeups.Subscribe(r.Context(), "opi:jobs:"+strconv.FormatInt(jobID, 10))
		if err != nil {
			h.logger.Warn("Valkey Job wake-up unavailable; using PostgreSQL polling")
		}
	}

	poll := time.NewTicker(250 * time.Millisecond)
	heartbeat := time.NewTicker(15 * time.Second)
	defer poll.Stop()
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-wakeups:
			if !ok {
				wakeups = nil
				continue
			}
			values, err := h.store.JobEvents(r.Context(), accessapi.Principal(r.Context()), jobID, after, 100)
			if err != nil || !emit(values) {
				return
			}
			if terminalJob(current.State) {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		case <-poll.C:
			values, err := h.store.JobEvents(r.Context(), accessapi.Principal(r.Context()), jobID, after, 100)
			if err != nil || !emit(values) {
				return
			}
			if terminalJob(current.State) {
				return
			}
		}
	}
}

func terminalJob(state job.State) bool {
	return state == job.Succeeded || state == job.Failed || state == job.Cancelled
}

func (h *Handler) PostApiV1JobsJobIdCancellation(w http.ResponseWriter, r *http.Request, rawID string) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.CancelJob(r.Context(), accessapi.Principal(r.Context()), id,
		accessapi.RequestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) projectFilter(params httpapi.GetApiV1ProjectsParams) (projectstore.Filter, int, error) {
	filters := projectFilterKey(params)
	filter, offset, _, err := h.pageFilter("projects", filters, params.Cursor, params.Limit)
	if err != nil {
		return projectstore.Filter{}, 0, err
	}
	filter.State, filter.Query = pointer(params.State), pointer(params.Q)
	if filter.State != "" && filter.State != string(project.StateActive) &&
		filter.State != string(project.StatePaused) && filter.State != string(project.StateArchived) &&
		filter.State != string(project.StateDeleting) {
		return projectstore.Filter{}, 0, invalid("state must be active, paused, archived, or deleting")
	}
	return filter, offset, err
}

func projectFilterKey(params httpapi.GetApiV1ProjectsParams) string {
	return pointer(params.State) + "|" + strings.TrimSpace(pointer(params.Q))
}

func (h *Handler) pageFilter(
	route, filters string,
	cursor *string,
	limit *int32,
) (projectstore.Filter, int, string, error) {
	pageLimit := 50
	if limit != nil {
		pageLimit = int(*limit)
	}
	if pageLimit <= 0 || pageLimit > 100 {
		return projectstore.Filter{}, 0, filters, invalid("limit must be between 1 and 100")
	}
	offset := 0
	if cursor != nil && *cursor != "" {
		decoded, err := h.cursors.Decode(*cursor, route, filters)
		if err != nil {
			return projectstore.Filter{}, 0, filters, invalid("cursor is invalid or does not match the filters")
		}
		offset = decoded.Offset
	}
	return projectstore.Filter{Limit: pageLimit, Offset: offset}, offset, filters, nil
}

func (h *Handler) writePage(w http.ResponseWriter, r *http.Request, route, filters string, offset, limit int, values any) {
	encoded, err := json.Marshal(values)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var items []any
	if err := json.Unmarshal(encoded, &items); err != nil {
		h.problem(w, r, err)
		return
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	response := map[string]any{"items": items, "has_more": hasMore}
	if hasMore {
		next, err := h.cursors.Encode(access.Cursor{Route: route, Filters: filters, Offset: offset + limit})
		if err != nil {
			h.problem(w, r, err)
			return
		}
		response["next_cursor"] = next
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) mutationIdentity(w http.ResponseWriter, r *http.Request, rawID string) (int64, int64, bool) {
	id, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return 0, 0, false
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return 0, 0, false
	}
	return id, version, true
}

func (h *Handler) problem(w http.ResponseWriter, r *http.Request, err error) {
	httpx.WriteProblem(w, h.logger, accessapi.RequestID(r.Context()), transportError(err))
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	httpx.WriteJSON(w, h.logger, status, value)
}

func transportError(err error) error {
	switch {
	case errors.Is(err, access.ErrAuthenticationRequired):
		return httpx.NewProblem(http.StatusUnauthorized, "authentication_required", "Authentication required",
			"Sign in to continue.", err)
	case errors.Is(err, access.ErrAccessPending), errors.Is(err, access.ErrPermissionDenied),
		errors.Is(err, project.ErrPermissionDenied):
		return httpx.NewProblem(http.StatusForbidden, "permission_denied", "Permission denied",
			"Your role cannot perform this action.", err)
	case errors.Is(err, project.ErrNotFound):
		return httpx.NewProblem(http.StatusNotFound, "not_found", "Resource not found",
			"The requested resource is unavailable.", err)
	case errors.Is(err, project.ErrVersionConflict):
		return httpx.NewProblem(http.StatusPreconditionFailed, "version_conflict", "Version conflict",
			"Refresh the resource and retry with its current ETag.", err)
	case errors.Is(err, project.ErrConflict), errors.Is(err, job.ErrConflict):
		return httpx.NewProblem(http.StatusConflict, "state_conflict", "State conflict",
			"The operation conflicts with current resource state.", err)
	case errors.Is(err, project.ErrInvalid), errors.Is(err, job.ErrInvalid):
		return httpx.NewProblem(http.StatusBadRequest, "invalid_request", "Invalid request",
			"Review the submitted fields and try again.", err)
	case errors.Is(err, collector.ErrUnsafeSource):
		return httpx.NewProblem(http.StatusBadRequest, "unsafe_source", "Unsafe source",
			"The source must resolve only to public HTTPS addresses.", err)
	default:
		return err
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalid("request body must be one JSON object with known fields")
	}
	if decoder.Decode(&struct{}{}) == nil {
		return invalid("request body must contain one JSON object")
	}
	return nil
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, invalid("resource identifier must be a positive decimal integer")
	}
	return id, nil
}

func parseEventID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 0 {
		return 0, invalid("Last-Event-ID must be a non-negative decimal integer")
	}
	return id, nil
}

func parsePair(left, right string) (int64, int64, error) {
	first, err := parseID(left)
	if err != nil {
		return 0, 0, err
	}
	second, err := parseID(right)
	return first, second, err
}

func parseETag(value string) (int64, error) {
	value = strings.TrimSpace(strings.Trim(value, `"`))
	value = strings.TrimPrefix(value, "v")
	if value == "" {
		return 0, httpx.NewProblem(http.StatusPreconditionRequired, "version_required", "Version required",
			"Provide the current ETag in If-Match.", nil)
	}
	version, err := strconv.ParseInt(value, 10, 64)
	if err != nil || version <= 0 {
		return 0, invalid("If-Match is malformed")
	}
	return version, nil
}

func parseHistoryDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.DateOnly, time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, invalid("history dates must use YYYY-MM-DD or RFC 3339")
}

func etag(version int64) string { return fmt.Sprintf(`"v%d"`, version) }

func pointer[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func invalid(detail string) error {
	return httpx.NewProblem(http.StatusBadRequest, "invalid_request", "Invalid request", detail, project.ErrInvalid)
}
