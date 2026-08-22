package intelligenceapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/alert"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/policy"
	"github.com/leohteixeira/opensource-project-intelligence/internal/radar"
	"github.com/leohteixeira/opensource-project-intelligence/internal/trend"
)

func (h *Handler) GetApiV1ProjectsProjectIdTrends(w http.ResponseWriter, r *http.Request,
	rawProjectID string, params httpapi.GetApiV1ProjectsProjectIdTrendsParams) {
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	kind := trend.KindObserved
	if params.Kind != nil && strings.TrimSpace(*params.Kind) != "" {
		kind = trend.Kind(strings.ToLower(strings.TrimSpace(*params.Kind)))
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limit := pageLimit(params.Limit)
	values, err := h.store.Signals(r.Context(), accessapi.Principal(r.Context()), projectID, kind, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	response := page(values, limit, offset)
	response["kind"] = kind
	if params.Window != nil {
		response["window"] = strings.TrimSpace(*params.Window)
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetApiV1ProjectsProjectIdRecommendation(w http.ResponseWriter, r *http.Request,
	rawProjectID string, params httpapi.GetApiV1ProjectsProjectIdRecommendationParams) {
	projectID, window, ok := h.requestWindow(w, r, rawProjectID, params.Window, params.Cutoff)
	if !ok {
		return
	}
	selector := "default"
	if params.Policy != nil {
		selector = strings.TrimSpace(*params.Policy)
	}
	principal := accessapi.Principal(r.Context())
	selected, err := h.store.ActivePolicy(r.Context(), principal, selector)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	metrics, err := h.store.Metrics(r.Context(), principal, projectID, window)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	facts := make([]policy.Fact, 0, len(metrics))
	for _, snapshot := range metrics {
		evidence := make([]int64, 0, len(snapshot.Factors))
		for _, factor := range snapshot.Factors {
			if factor.EvidenceID > 0 {
				evidence = append(evidence, factor.EvidenceID)
			}
		}
		facts = append(facts, policy.Fact{MetricName: snapshot.Definition.Name,
			MetricVersion: snapshot.Definition.Version, Status: snapshot.Status, Value: snapshot.Value,
			Coverage: snapshot.Coverage, EvidenceIDs: evidence, SnapshotID: snapshot.ID})
	}
	evaluation, err := policy.Evaluate(projectID, selected, window, facts)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	evaluation.CreatedAt = h.now().UTC()
	evaluation.InputDigest = recommendationDigest(selected, window, facts)
	evaluation, err = h.store.SaveEvaluation(r.Context(), principal, evaluation)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, evaluation)
}

func recommendationDigest(selected policy.Version, window metric.Window, facts []policy.Fact) string {
	payload, _ := json.Marshal(struct {
		PolicyID int64         `json:"policy_id"`
		Version  int           `json:"version"`
		Window   metric.Window `json:"window"`
		Facts    []policy.Fact `json:"facts"`
	}{selected.FamilyID, selected.Version, window, facts})
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func (h *Handler) GetApiV1Policies(w http.ResponseWriter, r *http.Request,
	params httpapi.GetApiV1PoliciesParams) {
	state := ""
	if params.State != nil {
		state = strings.TrimSpace(*params.State)
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limit := pageLimit(params.Limit)
	values, err := h.store.Policies(r.Context(), accessapi.Principal(r.Context()), state, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, page(values, limit, offset))
}

type policyBody struct {
	Name         string                    `json:"name"`
	Description  string                    `json:"description"`
	Owner        string                    `json:"owner"`
	Rules        []policy.Rule             `json:"rules"`
	RadarMapping map[policy.Outcome]string `json:"radar_mapping"`
}

func (h *Handler) PostApiV1Policies(w http.ResponseWriter, r *http.Request) {
	requestID, ok := requireIdempotencyKey(w, r, h)
	if !ok {
		return
	}
	var body policyBody
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.CreatePolicy(r.Context(), accessapi.Principal(r.Context()), policy.Version{
		Version: 1, Name: body.Name, Description: body.Description, Owner: body.Owner,
		State: policy.StateDraft, Rules: body.Rules, RadarMap: body.RadarMapping, CreatedAt: h.now().UTC(),
	}, requestID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusCreated, value.Revision, value)
}

func (h *Handler) PostApiV1PoliciesPolicyIdVersions(w http.ResponseWriter, r *http.Request,
	rawPolicyID string) {
	policyID, err := parseID(rawPolicyID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	requestID, ok := requireIdempotencyKey(w, r, h)
	if !ok {
		return
	}
	latest, err := parseVersionHeader(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Rules        []policy.Rule             `json:"rules"`
		RadarMapping map[policy.Outcome]string `json:"radar_mapping"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.CreatePolicyVersion(r.Context(), accessapi.Principal(r.Context()), policyID,
		policy.Version{Rules: body.Rules, RadarMap: body.RadarMapping, CreatedAt: h.now().UTC()}, requestID, int(latest))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusCreated, value.Revision, value)
}

func (h *Handler) GetApiV1PoliciesPolicyIdVersionsVersion(w http.ResponseWriter, r *http.Request,
	rawPolicyID, rawVersion string) {
	policyID, err := parseID(rawPolicyID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := strconv.Atoi(rawVersion)
	if err != nil || version <= 0 {
		h.problem(w, r, policy.ErrInvalid)
		return
	}
	value, err := h.store.PolicyVersion(r.Context(), accessapi.Principal(r.Context()), policyID, version)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusOK, value.Revision, value)
}

func (h *Handler) PostApiV1PoliciesPolicyIdVersionsVersionActivation(w http.ResponseWriter,
	r *http.Request, rawPolicyID, rawVersion string) {
	policyID, err := parseID(rawPolicyID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := strconv.Atoi(rawVersion)
	if err != nil || version <= 0 {
		h.problem(w, r, policy.ErrInvalid)
		return
	}
	revision, err := parseVersionHeader(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil || strings.TrimSpace(body.Reason) == "" {
		if err == nil {
			err = policy.ErrInvalid
		}
		h.problem(w, r, err)
		return
	}
	value, err := h.store.ActivatePolicy(r.Context(), accessapi.Principal(r.Context()), policyID,
		version, revision, h.now().UTC())
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusOK, value.Revision, value)
}

func (h *Handler) GetApiV1Radar(w http.ResponseWriter, r *http.Request, params httpapi.GetApiV1RadarParams) {
	values, err := h.store.Radar(r.Context(), accessapi.Principal(r.Context()), h.now().UTC(), 200, 0)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filtered := values[:0]
	for _, value := range values {
		if params.Policy != nil && strings.TrimSpace(*params.Policy) != "" &&
			strings.TrimSpace(*params.Policy) != "default" && strings.TrimSpace(*params.Policy) != strconv.FormatInt(value.Evaluation.PolicyID, 10) {
			continue
		}
		filtered = append(filtered, value)
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"items": filtered, "count": len(filtered)})
}

func (h *Handler) PostApiV1RadarProjectIdOverride(w http.ResponseWriter, r *http.Request,
	rawProjectID string) {
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
		Ring     radar.Ring `json:"ring"`
		Reason   string     `json:"reason"`
		Owner    string     `json:"owner"`
		ReviewOn string     `json:"review_on"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	reviewOn, err := time.Parse("2006-01-02", body.ReviewOn)
	if err != nil {
		h.problem(w, r, radar.ErrInvalid)
		return
	}
	principal := accessapi.Principal(r.Context())
	owner := strings.TrimSpace(body.Owner)
	if owner == "" {
		owner = strconv.FormatInt(principal.ActorID, 10)
	}
	value, err := radar.NewOverride(principal, body.Ring, body.Reason, owner, reviewOn.UTC(), h.now().UTC())
	if err == nil {
		value, err = h.store.SaveRadarOverride(r.Context(), principal, projectID, value, requestID)
	}
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusCreated, value.Revision, value)
}

func (h *Handler) DeleteApiV1RadarProjectIdOverride(w http.ResponseWriter, r *http.Request,
	rawProjectID string) {
	projectID, err := parseID(rawProjectID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	revision, err := parseVersionHeader(r.Header.Get("If-Match"))
	if err == nil {
		err = h.store.RemoveRadarOverride(r.Context(), accessapi.Principal(r.Context()), projectID,
			revision, h.now().UTC())
	}
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type alertRuleBody struct {
	Name                 string         `json:"name"`
	Signal               string         `json:"signal"`
	Operator             alert.Operator `json:"operator"`
	Threshold            float64        `json:"threshold"`
	Scope                string         `json:"scope"`
	ProjectID            int64          `json:"project_id,string"`
	Severity             alert.Severity `json:"severity"`
	CooldownSeconds      int64          `json:"cooldown_seconds"`
	DeduplicationSeconds int64          `json:"deduplication_seconds"`
	Enabled              bool           `json:"enabled"`
}

func (b alertRuleBody) rule(id int64, at time.Time) alert.Rule {
	return alert.Rule{ID: id, Name: b.Name, Signal: b.Signal, Operator: b.Operator,
		Threshold: b.Threshold, Scope: b.Scope, ProjectID: b.ProjectID, Severity: b.Severity,
		Cooldown:      time.Duration(b.CooldownSeconds) * time.Second,
		Deduplication: time.Duration(b.DeduplicationSeconds) * time.Second, Enabled: b.Enabled, UpdatedAt: at}
}

func (h *Handler) PostApiV1AlertRules(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireIdempotencyKey(w, r, h); !ok {
		return
	}
	var body alertRuleBody
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.CreateAlertRule(r.Context(), accessapi.Principal(r.Context()), body.rule(0, h.now().UTC()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusCreated, value.Version, value)
}

func (h *Handler) PatchApiV1AlertRulesRuleId(w http.ResponseWriter, r *http.Request, rawRuleID string) {
	ruleID, err := parseID(rawRuleID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := parseVersionHeader(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body alertRuleBody
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.UpdateAlertRule(r.Context(), accessapi.Principal(r.Context()),
		body.rule(ruleID, h.now().UTC()), version)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusOK, value.Version, value)
}

func (h *Handler) GetApiV1Alerts(w http.ResponseWriter, r *http.Request, params httpapi.GetApiV1AlertsParams) {
	state := ""
	if params.State != nil {
		state = strings.TrimSpace(*params.State)
	}
	projectID := int64(0)
	var err error
	if params.Project != nil && strings.TrimSpace(*params.Project) != "" {
		projectID, err = parseID(*params.Project)
		if err != nil {
			h.problem(w, r, err)
			return
		}
	}
	offset, err := decodeCursor(params.Cursor)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	limit := pageLimit(params.Limit)
	values, err := h.store.Alerts(r.Context(), accessapi.Principal(r.Context()), state,
		projectID, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, page(values, limit, offset))
}

func (h *Handler) PostApiV1AlertsAlertIdRead(w http.ResponseWriter, r *http.Request, rawAlertID string) {
	alertID, err := parseID(rawAlertID)
	if err == nil {
		err = h.store.MarkAlertRead(r.Context(), accessapi.Principal(r.Context()), alertID, h.now().UTC())
	}
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PostApiV1AlertsAlertIdTransition(w http.ResponseWriter, r *http.Request,
	rawAlertID string) {
	alertID, err := parseID(rawAlertID)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	revision, err := parseVersionHeader(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		To     alert.State `json:"to"`
		Reason string      `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	value, err := h.store.TransitionAlert(r.Context(), accessapi.Principal(r.Context()), alertID,
		body.To, body.Reason, revision, h.now().UTC())
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeVersionedJSON(w, http.StatusOK, value.Revision, value)
}
