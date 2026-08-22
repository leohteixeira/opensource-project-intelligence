package intelligenceapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/adoption"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/intelligencestore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/topic"
)

func (h *Handler) GetApiV1ProjectsProjectIdAdoption(w http.ResponseWriter, r *http.Request,
	rawID string, params httpapi.GetApiV1ProjectsProjectIdAdoptionParams) {
	if !h.authorizeTask05(w, r, access.ActionIntelligenceRead) {
		return
	}
	projectID, window, ok := h.requestWindow(w, r, rawID, params.Window, nil)
	if !ok {
		return
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limit := pageLimit(params.Limit)
	values, err := h.store.Adoption(r.Context(), accessapi.Principal(r.Context()), projectID,
		window.From, window.To, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, page(values, limit, offset))
}

func (h *Handler) GetApiV1ProjectsProjectIdSecurity(w http.ResponseWriter, r *http.Request,
	rawID string, params httpapi.GetApiV1ProjectsProjectIdSecurityParams) {
	if !h.authorizeTask05(w, r, access.ActionIntelligenceRead) {
		return
	}
	projectID, window, ok := h.requestWindow(w, r, rawID, params.Window, nil)
	if !ok {
		return
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limit := pageLimit(params.Limit)
	loaded, err := h.store.Security(r.Context(), accessapi.Principal(r.Context()), projectID,
		window.From, window.To, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	items, hasMore := loaded.Items, len(loaded.Items) > limit
	if hasMore {
		items = items[:limit]
	}
	result := adoption.SummarizeSecurity(loaded.Observed, loaded.Complete, items)
	response := map[string]any{"security": result, "has_more": hasMore}
	if hasMore {
		response["next_cursor"] = encodeCursor(offset + limit)
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetApiV1ProjectsProjectIdTopics(w http.ResponseWriter, r *http.Request,
	rawID string, params httpapi.GetApiV1ProjectsProjectIdTopicsParams) {
	if !h.authorizeTask05(w, r, access.ActionIntelligenceRead) {
		return
	}
	projectID, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limit := pageLimit(params.Limit)
	values, err := h.store.Topics(r.Context(), accessapi.Principal(r.Context()), projectID, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, page(values, limit, offset))
}

func (h *Handler) PostApiV1ProjectsProjectIdTopicsTopicIdCorrections(w http.ResponseWriter,
	r *http.Request, rawProjectID, rawTopicID string) {
	if !h.authorizeTask05(w, r, access.ActionProjectWrite) {
		return
	}
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	topicID, err := parseID(rawTopicID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	requestID, ok := requireIdempotencyKey(w, r, h)
	if !ok {
		return
	}
	version, err := parseVersionHeader(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Action        topic.Action `json:"action"`
		IssueIDs      []string     `json:"issue_ids"`
		OtherTopicIDs []string     `json:"other_topic_ids"`
		Label         string       `json:"label"`
		Reason        string       `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	issueIDs, err := parseIDs(body.IssueIDs)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	otherTopicIDs, err := parseIDs(body.OtherTopicIDs)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.CorrectTopic(r.Context(), accessapi.Principal(r.Context()), topic.Correction{
		ProjectID: projectID, TopicID: topicID, Action: body.Action, IssueIDs: issueIDs,
		OtherTopicIDs: otherTopicIDs, Label: body.Label, Reason: body.Reason, RequestID: requestID,
		Version: version + 1,
	})
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusAccepted, value.Version, value)
}

func (h *Handler) GetApiV1ProjectsProjectIdReleases(w http.ResponseWriter, r *http.Request,
	rawID string, params httpapi.GetApiV1ProjectsProjectIdReleasesParams) {
	if !h.authorizeTask05(w, r, access.ActionIntelligenceRead) {
		return
	}
	projectID, err := parseID(rawID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limit := pageLimit(params.Limit)
	values, err := h.store.Releases(r.Context(), accessapi.Principal(r.Context()), projectID, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, page(values, limit, offset))
}

func (h *Handler) GetApiV1ProjectsProjectIdReleasesReleaseId(w http.ResponseWriter, r *http.Request,
	rawProjectID, rawReleaseID string) {
	if !h.authorizeTask05(w, r, access.ActionIntelligenceRead) {
		return
	}
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	releaseID, err := parseID(rawReleaseID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.Release(r.Context(), accessapi.Principal(r.Context()), projectID, releaseID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) PostApiV1ProjectsProjectIdCrawls(w http.ResponseWriter, r *http.Request,
	rawProjectID string) {
	if !h.authorizeTask05(w, r, access.ActionProjectWrite) {
		return
	}
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	requestID, ok := requireIdempotencyKey(w, r, h)
	if !ok {
		return
	}
	var body struct {
		SourceIDs []string `json:"source_ids"`
		MaxDepth  int      `json:"max_depth"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	sourceIDs, err := parseIDs(body.SourceIDs)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limits := knowledge.Limits{MaxDomains: 4, MaxDepth: body.MaxDepth, MaxPages: 500,
		MaxPageBytes: 2 << 20, MaxTotalBytes: 100 << 20, RequestsPerMinute: 60,
		MediaTypes: []string{"text/html", "text/plain", "text/markdown", "application/json"}}
	value, err := h.store.QueueCrawl(r.Context(), accessapi.Principal(r.Context()), projectID,
		sourceIDs, limits, requestID, h.now().UTC())
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) PostApiV1ProjectsProjectIdKnowledgeSearch(w http.ResponseWriter, r *http.Request,
	rawProjectID string) {
	if !h.authorizeTask05(w, r, access.ActionIntelligenceRead) {
		return
	}
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Query    string `json:"query"`
		Limit    int    `json:"limit"`
		Language string `json:"language"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	_, err = normalizeAnalysisLanguage(body.Language)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if body.Limit == 0 {
		body.Limit = 20
	}
	cutoff := h.now().UTC()
	values, err := h.store.Search(r.Context(), accessapi.Principal(r.Context()), projectID,
		intelligencestore.SearchRequest{Query: body.Query, Cutoff: cutoff, Limit: body.Limit})
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": values, "retrieval_version": "rrf-v1",
		"modes": []string{"lexical"}, "cutoff": cutoff})
}

func (h *Handler) PostApiV1ProjectsProjectIdQueries(w http.ResponseWriter, r *http.Request,
	rawProjectID string) {
	if !h.authorizeTask05(w, r, access.ActionProjectWrite) {
		return
	}
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if _, ok := requireIdempotencyKey(w, r, h); !ok {
		return
	}
	var body struct {
		Question string `json:"question"`
		Window   string `json:"window"`
		Language string `json:"language"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	language, err := normalizeAnalysisLanguage(body.Language)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	cutoff := h.now().UTC()
	if _, err := metric.ResolveWindow(body.Window, cutoff); err != nil {
		h.problem(w, r, err)
		return
	}
	query := analysis.Query{ProjectID: projectID, Question: body.Question, Cutoff: cutoff,
		MaxResults: 20, MaxOutputBytes: 64 << 10}
	if err := query.Validate(); err != nil {
		h.problem(w, r, err)
		return
	}
	if h.model.Provider == "" || h.model.Model == "" {
		h.problem(w, r, analysis.ErrProviderUnavailable)
		return
	}
	runID, seriesID, err := h.nextTwoIDs(r)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	principal := accessapi.Principal(r.Context())
	value, err := analysis.NewRun(analysis.Run{ID: runID, SeriesID: seriesID, ProjectID: projectID,
		Kind: "natural_language_query", PromptVersion: "query-v1", SchemaVersion: "analysis-v1",
		RetrievalVersion: "rrf-v1", Provider: h.model.Provider, Model: h.model.Model,
		Language: language, RequestedBy: principal.ActorID, Cutoff: cutoff, CreatedAt: h.now().UTC()})
	if err == nil {
		value, err = h.store.QueueRun(r.Context(), principal, value)
	}
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) GetApiV1AnalysisRunsRunId(w http.ResponseWriter, r *http.Request, rawRunID string) {
	if !h.authorizeTask05(w, r, access.ActionIntelligenceRead) {
		return
	}
	runID, err := parseID(rawRunID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.Run(r.Context(), accessapi.Principal(r.Context()), runID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, value)
}

func (h *Handler) PostApiV1AnalysisRunsRunIdReruns(w http.ResponseWriter, r *http.Request,
	rawRunID string) {
	if !h.authorizeTask05(w, r, access.ActionProjectWrite) {
		return
	}
	runID, err := parseID(rawRunID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if _, ok := requireIdempotencyKey(w, r, h); !ok {
		return
	}
	parent, err := h.store.Run(r.Context(), accessapi.Principal(r.Context()), runID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Language string `json:"language"`
		Reason   string `json:"reason"`
	}
	if err := decodeOptionalJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	language := parent.Language
	if strings.TrimSpace(body.Language) != "" {
		language, err = normalizeAnalysisLanguage(body.Language)
		if err != nil {
			h.problem(w, r, err)
			return
		}
	}
	if strings.TrimSpace(body.Reason) == "" {
		h.problem(w, r, analysis.ErrInvalidRun)
		return
	}
	if h.model.Provider == "" || h.model.Model == "" {
		h.problem(w, r, analysis.ErrProviderUnavailable)
		return
	}
	newID, err := h.ids.Next(r.Context())
	if err != nil {
		h.problem(w, r, err)
		return
	}
	principal := accessapi.Principal(r.Context())
	value, err := analysis.NewRun(analysis.Run{ID: newID, SeriesID: parent.SeriesID,
		ProjectID: parent.ProjectID, ParentRunID: &parent.ID, Kind: parent.Kind,
		PromptVersion: parent.PromptVersion, SchemaVersion: parent.SchemaVersion,
		RetrievalVersion: parent.RetrievalVersion, Provider: h.model.Provider, Model: h.model.Model,
		Language: language, RequestedBy: principal.ActorID, Reason: body.Reason,
		Cutoff: parent.Cutoff, CreatedAt: h.now().UTC()})
	if err == nil {
		value, err = h.store.QueueRun(r.Context(), principal, value)
	}
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusAccepted, value)
}

func (h *Handler) PostApiV1AnalysisRunsRunIdFeedback(w http.ResponseWriter, r *http.Request,
	rawRunID string) {
	if !h.authorizeTask05(w, r, access.ActionProjectWrite) {
		return
	}
	runID, err := parseID(rawRunID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	requestID, ok := requireIdempotencyKey(w, r, h)
	if !ok {
		return
	}
	var body struct {
		Rating  string `json:"rating"`
		Note    string `json:"note"`
		Comment string `json:"comment"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	if body.Note == "" {
		body.Note = body.Comment
	}
	value, err := h.store.SaveFeedback(r.Context(), accessapi.Principal(r.Context()),
		analysis.Feedback{RunID: runID, Rating: body.Rating, Note: body.Note, RequestID: requestID})
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, value)
}

func (h *Handler) PostApiV1AnalysisSeriesSeriesIdSelection(w http.ResponseWriter, r *http.Request,
	rawSeriesID string) {
	if !h.authorizeTask05(w, r, access.ActionProjectWrite) {
		return
	}
	seriesID, err := parseID(rawSeriesID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	requestID, ok := requireIdempotencyKey(w, r, h)
	if !ok {
		return
	}
	version, err := parseVersionHeader(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		RunID string `json:"run_id"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	runID, err := parseID(body.RunID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.SelectRun(r.Context(), accessapi.Principal(r.Context()), analysis.Selection{
		SeriesID: seriesID, RunID: runID, RequestID: requestID, Version: version + 1,
	})
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusOK, value.Version, value)
}

func page[T any](values []T, limit, offset int) map[string]any {
	hasMore := len(values) > limit
	if hasMore {
		values = values[:limit]
	}
	response := map[string]any{"items": values, "has_more": hasMore}
	if hasMore {
		response["next_cursor"] = encodeCursor(offset + limit)
	}
	return response
}

func (h *Handler) authorizeTask05(w http.ResponseWriter, r *http.Request, action access.Action) bool {
	if err := access.Authorize(accessapi.Principal(r.Context()), action); err != nil {
		h.problem(w, r, err)
		return false
	}
	return true
}

func parseIDs(raw []string) ([]int64, error) {
	values := make([]int64, 0, len(raw))
	for _, item := range raw {
		value, err := parseID(item)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func requireIdempotencyKey(w http.ResponseWriter, r *http.Request, h *Handler) (string, bool) {
	value := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if value == "" {
		h.problem(w, r, analysis.ErrInvalidRun)
		return "", false
	}
	return value, true
}

func decodeOptionalJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return analysis.ErrInvalidRun
	}
	if decoder.Decode(&struct{}{}) == nil {
		return analysis.ErrInvalidRun
	}
	return nil
}

func normalizeAnalysisLanguage(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "en":
		return "en", nil
	case "pt-br":
		return "pt-BR", nil
	default:
		return "", analysis.ErrInvalidRun
	}
}

func (h *Handler) nextTwoIDs(r *http.Request) (int64, int64, error) {
	left, err := h.ids.Next(r.Context())
	if err != nil {
		return 0, 0, err
	}
	right, err := h.ids.Next(r.Context())
	return left, right, err
}
