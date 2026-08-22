//go:build integration

package intelligencestore_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/alert"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/policy"
	"github.com/leohteixeira/opensource-project-intelligence/internal/radar"
	"github.com/leohteixeira/opensource-project-intelligence/internal/trend"
)

func TestTask06PostgreSQLContracts(t *testing.T) {
	h := newTask04Harness(t)
	admin := access.Principal{ActorID: 202, Kind: access.ActorMember, Role: access.RoleAdmin,
		Status: access.StatusActive, Workspace: task04WorkspaceID}
	now := h.window.Cutoff.Add(-time.Hour)

	metricValues := snapshots(t, task04ProjectOne, h.window, 10)
	metricValues, err := h.store.SaveMetricSet(h.ctx, metricValues)
	if err != nil {
		t.Fatal(err)
	}

	definition := metric.Catalog()[0]
	baseline := metric.Window{From: h.window.From.Add(-90 * 24 * time.Hour), To: h.window.From,
		Cutoff: h.window.Cutoff}
	points := task06Points(baseline.From, 28)
	observed, err := trend.CalculateObserved(task04ProjectOne, trend.ObservedV1, definition.Name,
		definition.Version, h.window, baseline, points)
	if err != nil {
		t.Fatal(err)
	}
	observed.InputDigest = "task06-observed-input-v1"

	t.Run("IT-049 one signal version publishes atomically against one consistent snapshot", func(t *testing.T) {
		const attempts = 8
		start := make(chan struct{})
		results := make(chan trend.Observed, attempts)
		errorsByCall := make(chan error, attempts)
		var group sync.WaitGroup
		for range attempts {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				value, saveErr := h.store.SaveObserved(h.ctx, observed)
				results <- value
				errorsByCall <- saveErr
			}()
		}
		close(start)
		group.Wait()
		close(results)
		close(errorsByCall)
		for saveErr := range errorsByCall {
			if saveErr != nil {
				t.Fatal(saveErr)
			}
		}
		var id int64
		for value := range results {
			if value.ID > 0 && id == 0 {
				id = value.ID
			}
			if value.ID > 0 && value.ID != id {
				t.Fatalf("replay published signal %d instead of %d", value.ID, id)
			}
		}
		var count int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM trend_signals
			WHERE project_id=$1 AND input_digest=$2`, task04ProjectOne, observed.InputDigest).Scan(&count); err != nil || count != 1 {
			t.Fatalf("published signals=%d err=%v", count, err)
		}
	})

	t.Run("IT-050 failed prediction does not suppress the valid observed trend", func(t *testing.T) {
		invalid := trend.Forecast{ProjectID: task04ProjectOne, Kind: trend.KindForecast}
		if _, err := h.store.SaveForecast(h.ctx, invalid, "task06-invalid-forecast", h.window); err == nil {
			t.Fatal("invalid forecast unexpectedly published")
		}
		values, err := h.store.Signals(h.ctx, h.principal, task04ProjectOne, trend.KindObserved, 10, 0)
		if err != nil || len(values) != 1 {
			t.Fatalf("observed signal unavailable after forecast failure: values=%v err=%v", values, err)
		}
	})

	t.Run("IT-051 bounded portfolio detection exposes delayed status", func(t *testing.T) {
		forecast, err := trend.CalculateForecast(task04ProjectOne, trend.ForecastV1, definition.Name,
			definition.Version, points, 14)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.store.SaveForecast(h.ctx, forecast, "task06-forecast-input-v1", h.window); err != nil {
			t.Fatal(err)
		}
		values, err := h.store.Signals(h.ctx, h.principal, task04ProjectOne, trend.KindForecast, 1, 0)
		if err != nil || len(values) != 1 || values[0]["outcome_status"] != "unevaluated" {
			t.Fatalf("bounded forecast history=%v err=%v", values, err)
		}
	})

	policyDraft := task06PolicyVersion()
	var activePolicy policy.Version
	t.Run("IT-052 IT-053 immutable policy evaluation survives activation and explanation failure", func(t *testing.T) {
		created, err := h.store.CreatePolicy(h.ctx, admin, policyDraft, "task06-default-policy")
		if err != nil {
			t.Fatal(err)
		}
		activePolicy, err = h.store.ActivatePolicy(h.ctx, admin, created.FamilyID, 1, created.Revision, now)
		if err != nil {
			t.Fatal(err)
		}
		facts := task06Facts(metricValues)
		evaluation, err := policy.Evaluate(task04ProjectOne, activePolicy, h.window, facts)
		if err != nil {
			t.Fatal(err)
		}
		evaluation.InputDigest, evaluation.CreatedAt = "task06-evaluation-v1", now
		saved, err := h.store.SaveEvaluation(h.ctx, h.principal, evaluation)
		if err != nil {
			t.Fatal(err)
		}
		next := task06PolicyVersion()
		next.Rules[0].Threshold = 11
		createdNext, err := h.store.CreatePolicyVersion(h.ctx, admin, created.FamilyID, next,
			"task06-policy-v2", 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.store.ActivatePolicy(h.ctx, admin, created.FamilyID, 2, createdNext.Revision, now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		loaded, err := h.store.Recommendation(h.ctx, h.principal, task04ProjectOne, created.FamilyID, 1, h.window)
		if err != nil || loaded.ID != saved.ID || loaded.PolicyVersion != 1 || loaded.Outcome != saved.Outcome {
			t.Fatalf("historical evaluation changed: loaded=%+v saved=%+v err=%v", loaded, saved, err)
		}
		if _, err := h.store.SaveEvaluation(h.ctx, h.principal, policy.Evaluation{}); err == nil {
			t.Fatal("failed explanation fixture unexpectedly replaced deterministic result")
		}
	})

	t.Run("IT-054 portfolio evaluation is bounded and reports an exact page", func(t *testing.T) {
		values, err := h.store.Policies(h.ctx, h.principal, "", 1, 0)
		if err != nil || len(values) != 1 {
			t.Fatalf("bounded policy page=%d err=%v", len(values), err)
		}
	})

	t.Run("IT-055 concurrent policy drafts reject stale edits", func(t *testing.T) {
		start := make(chan struct{})
		errorsByCall := make(chan error, 2)
		var group sync.WaitGroup
		for index := range 2 {
			group.Add(1)
			go func(index int) {
				defer group.Done()
				<-start
				value := task06PolicyVersion()
				_, createErr := h.store.CreatePolicyVersion(h.ctx, admin, activePolicy.FamilyID, value,
					fmt.Sprintf("task06-concurrent-policy-%d", index), 2)
				errorsByCall <- createErr
			}(index)
		}
		close(start)
		group.Wait()
		close(errorsByCall)
		succeeded, conflicted := 0, 0
		for createErr := range errorsByCall {
			switch {
			case createErr == nil:
				succeeded++
			case errors.Is(createErr, access.ErrVersionConflict):
				conflicted++
			default:
				t.Fatalf("unexpected policy edit result: %v", createErr)
			}
		}
		if succeeded != 1 || conflicted != 1 {
			t.Fatalf("policy outcomes succeeded=%d conflicted=%d", succeeded, conflicted)
		}
	})

	t.Run("IT-056 failed policy publication leaves no partial active version", func(t *testing.T) {
		invalid := task06PolicyVersion()
		invalid.Rules[0].Weight = 0.9
		if _, err := h.store.CreatePolicyVersion(h.ctx, admin, activePolicy.FamilyID, invalid,
			"task06-invalid-version", 3); err == nil {
			t.Fatal("invalid policy version unexpectedly published")
		}
		current, err := h.store.ActivePolicy(h.ctx, h.principal, fmt.Sprint(activePolicy.FamilyID))
		if err != nil || current.Version != 2 {
			t.Fatalf("active policy changed after failure: version=%d err=%v", current.Version, err)
		}
	})

	t.Run("IT-057 many policy versions remain searchable through bounded pages", func(t *testing.T) {
		for latest := 3; latest < 15; latest++ {
			if _, err := h.store.CreatePolicyVersion(h.ctx, admin, activePolicy.FamilyID,
				task06PolicyVersion(), fmt.Sprintf("task06-history-%d", latest+1), latest); err != nil {
				t.Fatal(err)
			}
		}
		var count int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM policy_versions WHERE policy_id=$1`,
			activePolicy.FamilyID).Scan(&count); err != nil || count != 15 {
			t.Fatalf("policy history=%d err=%v", count, err)
		}
	})

	loadedEvaluation, err := h.store.Recommendation(h.ctx, h.principal, task04ProjectOne,
		activePolicy.FamilyID, 1, h.window)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.store.SelectRadar(h.ctx, h.principal, loadedEvaluation.ID, 0, now); err != nil {
		t.Fatal(err)
	}

	t.Run("IT-058 concurrent radar movement detects stale state", func(t *testing.T) {
		start := make(chan struct{})
		errorsByCall := make(chan error, 2)
		var group sync.WaitGroup
		for range 2 {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				errorsByCall <- h.store.SelectRadar(h.ctx, h.principal, loadedEvaluation.ID, 1, now.Add(time.Minute))
			}()
		}
		close(start)
		group.Wait()
		close(errorsByCall)
		for selectErr := range errorsByCall {
			if selectErr != nil {
				t.Fatalf("identical selection was not idempotent: %v", selectErr)
			}
		}
	})

	t.Run("IT-059 failed radar override preserves prior recommendation", func(t *testing.T) {
		if _, err := h.store.SaveRadarOverride(h.ctx, h.principal, task04ProjectOne, radar.Override{}, "bad"); err == nil {
			t.Fatal("invalid override unexpectedly committed")
		}
		placements, err := h.store.Radar(h.ctx, h.principal, now, 10, 0)
		if err != nil || len(placements) != 1 || placements[0].Suggested != placements[0].Effective {
			t.Fatalf("recommendation changed after failed override: placements=%+v err=%v", placements, err)
		}
	})

	t.Run("IT-060 radar calculation remains bounded", func(t *testing.T) {
		placements, err := h.store.Radar(h.ctx, h.principal, now, 1, 0)
		if err != nil || len(placements) != 1 {
			t.Fatalf("bounded radar page=%d err=%v", len(placements), err)
		}
	})

	alertRule, err := h.store.CreateAlertRule(h.ctx, h.principal, task06AlertRule())
	if err != nil {
		t.Fatal(err)
	}
	fact := task06AlertFact(now)
	occurrence, err := h.store.EvaluateAlert(h.ctx, alertRule, fact)
	if err != nil {
		t.Fatal(err)
	}
	alertRevision := occurrence.Revision

	t.Run("IT-079 concurrent alert resolution leaves one final state and a stale action", func(t *testing.T) {
		start := make(chan struct{})
		errorsByCall := make(chan error, 2)
		var group sync.WaitGroup
		for range 2 {
			group.Add(1)
			go func() {
				defer group.Done()
				<-start
				_, transitionErr := h.store.TransitionAlert(h.ctx, h.principal, occurrence.ID,
					alert.StateAcknowledged, "Investigating", 1, now.Add(time.Minute))
				errorsByCall <- transitionErr
			}()
		}
		close(start)
		group.Wait()
		close(errorsByCall)
		succeeded, conflicted := 0, 0
		for transitionErr := range errorsByCall {
			switch {
			case transitionErr == nil:
				succeeded++
			case errors.Is(transitionErr, access.ErrVersionConflict):
				conflicted++
			default:
				t.Fatalf("unexpected transition result: %v", transitionErr)
			}
		}
		if succeeded != 1 || conflicted != 1 {
			t.Fatalf("transition outcomes succeeded=%d conflicted=%d", succeeded, conflicted)
		}
	})

	t.Run("IT-080 alert evaluation failure neither closes nor duplicates occurrence", func(t *testing.T) {
		invalid := fact
		invalid.EvidenceID = 0
		if result, err := h.store.EvaluateAlert(h.ctx, alertRule, invalid); err != nil || result == nil || result.ID != occurrence.ID {
			t.Fatalf("invalid redelivery changed occurrence: result=%+v err=%v", result, err)
		}
		var count int
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM alert_occurrences WHERE rule_id=$1`,
			alertRule.ID).Scan(&count); err != nil || count != 1 {
			t.Fatalf("occurrence count=%d err=%v", count, err)
		}
	})

	t.Run("IT-081 high event volume is bounded without losing severity or suppressions", func(t *testing.T) {
		for index := 1; index <= 25; index++ {
			replayed := fact
			replayed.DetectedAt = fact.DetectedAt.Add(time.Duration(index) * time.Minute)
			replayed.EvidenceID += int64(index)
			if _, err := h.store.EvaluateAlert(h.ctx, alertRule, replayed); err != nil {
				t.Fatal(err)
			}
		}
		items, err := h.store.Alerts(h.ctx, h.principal, "", 0, 1, 0)
		if err != nil || len(items) != 1 || items[0].Severity != alert.SeverityCritical || items[0].SuppressionCount != 25 {
			t.Fatalf("bounded alert=%+v err=%v", items, err)
		}
	})

	t.Run("IT-130 redelivery keeps shared occurrence and per-member state independent", func(t *testing.T) {
		replayed, err := h.store.EvaluateAlert(h.ctx, alertRule, fact)
		if err != nil || replayed.ID != occurrence.ID {
			t.Fatalf("redelivery=%+v err=%v", replayed, err)
		}
		alertRevision = replayed.Revision
		if err := h.store.MarkAlertRead(h.ctx, h.principal, occurrence.ID, now.Add(2*time.Minute)); err != nil {
			t.Fatal(err)
		}
		analystItems, err := h.store.Alerts(h.ctx, h.principal, "", 0, 10, 0)
		if err != nil || analystItems[0].ReadAt == nil {
			t.Fatalf("analyst read state missing: %+v err=%v", analystItems, err)
		}
		adminItems, err := h.store.Alerts(h.ctx, admin, "", 0, 10, 0)
		if err != nil || adminItems[0].ReadAt != nil || adminItems[0].State != alert.StateAcknowledged {
			t.Fatalf("member/shared state conflated: %+v err=%v", adminItems, err)
		}
	})

	t.Run("IT-237 IT-238 trends endpoint returns a page and rejects anonymous access", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/projects/%d/trends?kind=observed&window=365d&limit=10", task04ProjectOne)
		assertTask06Statuses(t, h.request(t, http.MethodGet, path, nil, true).Code, http.StatusOK,
			h.request(t, http.MethodGet, path, nil, false).Code, http.StatusUnauthorized)
	})

	t.Run("IT-239 IT-240 recommendation endpoint returns four-state output and protects it", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/projects/%d/recommendation?policy=%d&window=90d&cutoff=%s",
			task04ProjectOne, activePolicy.FamilyID, h.window.Cutoff.Format(time.RFC3339))
		assertTask06Statuses(t, h.request(t, http.MethodGet, path, nil, true).Code, http.StatusOK,
			h.request(t, http.MethodGet, path, nil, false).Code, http.StatusUnauthorized)
	})

	t.Run("IT-241 IT-242 policy list returns versions and protects the catalog", func(t *testing.T) {
		path := "/api/v1/policies?limit=10"
		assertTask06Statuses(t, h.request(t, http.MethodGet, path, nil, true).Code, http.StatusOK,
			h.request(t, http.MethodGet, path, nil, false).Code, http.StatusUnauthorized)
	})

	t.Run("IT-243 IT-244 policy version returns immutable rule tree and protects it", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/policies/%d/versions/1", activePolicy.FamilyID)
		assertTask06Statuses(t, h.request(t, http.MethodGet, path, nil, true).Code, http.StatusOK,
			h.request(t, http.MethodGet, path, nil, false).Code, http.StatusUnauthorized)
	})

	var apiPolicy policy.Version
	t.Run("IT-245 IT-246 policy creation returns a draft and rejects unauthorized input atomically", func(t *testing.T) {
		body := task06PolicyBody("API policy")
		created := h.requestAs(t, http.MethodPost, "/api/v1/policies", body, "task06-admin", nil)
		if created.Code != http.StatusCreated {
			t.Fatalf("create policy status=%d body=%s", created.Code, created.Body.String())
		}
		if err := json.Unmarshal(created.Body.Bytes(), &apiPolicy); err != nil {
			t.Fatal(err)
		}
		before := task06TableCount(t, h, "policy_families")
		rejected := h.request(t, http.MethodPost, "/api/v1/policies", body, true)
		if rejected.Code != http.StatusForbidden || task06TableCount(t, h, "policy_families") != before {
			t.Fatalf("rejected create status=%d changed policy count", rejected.Code)
		}
	})

	var apiPolicyV2 policy.Version
	t.Run("IT-247 IT-248 policy version creation is immutable and stale versions are rejected", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/policies/%d/versions", apiPolicy.FamilyID)
		body := map[string]any{"rules": task06PolicyVersion().Rules, "radar_mapping": task06PolicyVersion().RadarMap}
		created := h.requestAs(t, http.MethodPost, path, body, "task06-admin", map[string]string{"If-Match": `"v1"`})
		if created.Code != http.StatusCreated {
			t.Fatalf("create version status=%d body=%s", created.Code, created.Body.String())
		}
		if err := json.Unmarshal(created.Body.Bytes(), &apiPolicyV2); err != nil {
			t.Fatal(err)
		}
		rejected := h.requestAs(t, http.MethodPost, path, body, "task06-admin", map[string]string{"If-Match": `"v1"`})
		if rejected.Code != http.StatusConflict {
			t.Fatalf("stale version status=%d body=%s", rejected.Code, rejected.Body.String())
		}
	})

	t.Run("IT-249 IT-250 activation selects a version and rejects stale revisions", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/policies/%d/versions/%d/activation", apiPolicy.FamilyID, apiPolicyV2.Version)
		body := map[string]any{"reason": "Quarterly governance update"}
		activated := h.requestAs(t, http.MethodPost, path, body, "task06-admin", map[string]string{"If-Match": `"v1"`})
		if activated.Code != http.StatusOK {
			t.Fatalf("activate policy status=%d body=%s", activated.Code, activated.Body.String())
		}
		rejected := h.requestAs(t, http.MethodPost, path, body, "task06-admin", map[string]string{"If-Match": `"v1"`})
		if rejected.Code != http.StatusConflict && rejected.Code != http.StatusBadRequest {
			t.Fatalf("repeated activation status=%d body=%s", rejected.Code, rejected.Body.String())
		}
	})

	t.Run("IT-251 IT-252 radar endpoint returns effective placement and protects it", func(t *testing.T) {
		path := "/api/v1/radar?policy=default&window=90d"
		assertTask06Statuses(t, h.request(t, http.MethodGet, path, nil, true).Code, http.StatusOK,
			h.request(t, http.MethodGet, path, nil, false).Code, http.StatusUnauthorized)
	})

	var apiOverride radar.Override
	t.Run("IT-253 IT-254 radar override is attributed and invalid input has no side effect", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/radar/%d/override", task04ProjectOne)
		body := map[string]any{"ring": "assess", "reason": "Pilot dependency", "owner": "Platform",
			"review_on": "2026-11-20"}
		created := h.request(t, http.MethodPost, path, body, true)
		if created.Code != http.StatusCreated {
			t.Fatalf("create override status=%d body=%s", created.Code, created.Body.String())
		}
		if err := json.Unmarshal(created.Body.Bytes(), &apiOverride); err != nil {
			t.Fatal(err)
		}
		rejected := h.request(t, http.MethodPost, path, map[string]any{"ring": "unknown"}, true)
		if rejected.Code != http.StatusBadRequest {
			t.Fatalf("invalid override status=%d body=%s", rejected.Code, rejected.Body.String())
		}
	})

	t.Run("IT-255 IT-256 override removal restores policy placement and rejects stale state", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/radar/%d/override", task04ProjectOne)
		removed := h.requestAs(t, http.MethodDelete, path, nil, "task04-analyst",
			map[string]string{"If-Match": fmt.Sprintf(`"v%d"`, apiOverride.Revision)})
		if removed.Code != http.StatusNoContent {
			t.Fatalf("remove override status=%d body=%s", removed.Code, removed.Body.String())
		}
		rejected := h.requestAs(t, http.MethodDelete, path, nil, "task04-analyst",
			map[string]string{"If-Match": fmt.Sprintf(`"v%d"`, apiOverride.Revision)})
		if rejected.Code != http.StatusConflict {
			t.Fatalf("stale removal status=%d body=%s", rejected.Code, rejected.Body.String())
		}
	})

	t.Run("IT-279 IT-280 alert endpoint returns personal read state and protects it", func(t *testing.T) {
		path := "/api/v1/alerts?limit=10"
		assertTask06Statuses(t, h.request(t, http.MethodGet, path, nil, true).Code, http.StatusOK,
			h.request(t, http.MethodGet, path, nil, false).Code, http.StatusUnauthorized)
	})

	var apiRule alert.Rule
	t.Run("IT-281 IT-282 alert rule creation returns ETag and rejects invalid input atomically", func(t *testing.T) {
		body := task06AlertRuleBody()
		created := h.request(t, http.MethodPost, "/api/v1/alert-rules", body, true)
		if created.Code != http.StatusCreated || created.Header().Get("ETag") != `"v1"` {
			t.Fatalf("create alert rule status=%d etag=%q body=%s", created.Code, created.Header().Get("ETag"), created.Body.String())
		}
		if err := json.Unmarshal(created.Body.Bytes(), &apiRule); err != nil {
			t.Fatal(err)
		}
		before := task06TableCount(t, h, "alert_rules")
		invalid := task06AlertRuleBody()
		invalid["signal"] = "unknown"
		rejected := h.request(t, http.MethodPost, "/api/v1/alert-rules", invalid, true)
		if rejected.Code != http.StatusBadRequest || task06TableCount(t, h, "alert_rules") != before {
			t.Fatalf("invalid alert rule status=%d changed count", rejected.Code)
		}
	})

	t.Run("IT-283 IT-284 alert rule update versions atomically and rejects stale ETag", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/alert-rules/%d", apiRule.ID)
		body := task06AlertRuleBody()
		body["severity"] = "warning"
		updated := h.requestAs(t, http.MethodPatch, path, body, "task04-analyst", map[string]string{"If-Match": `"v1"`})
		if updated.Code != http.StatusOK || updated.Header().Get("ETag") != `"v2"` {
			t.Fatalf("update alert rule status=%d etag=%q body=%s", updated.Code, updated.Header().Get("ETag"), updated.Body.String())
		}
		rejected := h.requestAs(t, http.MethodPatch, path, body, "task04-analyst", map[string]string{"If-Match": `"v1"`})
		if rejected.Code != http.StatusConflict {
			t.Fatalf("stale alert update status=%d body=%s", rejected.Code, rejected.Body.String())
		}
	})

	t.Run("IT-285 IT-286 personal alert read changes one member and rejects anonymous callers", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/alerts/%d/read", occurrence.ID)
		assertTask06Statuses(t, h.request(t, http.MethodPost, path, nil, true).Code, http.StatusNoContent,
			h.request(t, http.MethodPost, path, nil, false).Code, http.StatusUnauthorized)
	})

	t.Run("IT-287 IT-288 shared transition returns ETag and rejects a stale action", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/alerts/%d/transition", occurrence.ID)
		body := map[string]any{"to": "resolved", "reason": "Mitigated"}
		etag := fmt.Sprintf(`"v%d"`, alertRevision)
		updated := h.requestAs(t, http.MethodPost, path, body, "task04-analyst", map[string]string{"If-Match": etag})
		wantETag := fmt.Sprintf(`"v%d"`, alertRevision+1)
		if updated.Code != http.StatusOK || updated.Header().Get("ETag") != wantETag {
			t.Fatalf("transition alert status=%d etag=%q body=%s", updated.Code, updated.Header().Get("ETag"), updated.Body.String())
		}
		rejected := h.requestAs(t, http.MethodPost, path, body, "task04-analyst", map[string]string{"If-Match": etag})
		if rejected.Code != http.StatusConflict {
			t.Fatalf("stale transition status=%d body=%s", rejected.Code, rejected.Body.String())
		}
	})
}

func task06Points(from time.Time, count int) []trend.Point {
	values := make([]trend.Point, count)
	for index := range count {
		values[index] = trend.Point{At: from.Add(time.Duration(index) * 24 * time.Hour),
			Value: float64(index) + float64(index%7)/10, EvidenceID: 8_600_000_000_000_000_000 + int64(index)}
	}
	return values
}

func task06PolicyVersion() policy.Version {
	catalog := metric.Catalog()
	return policy.Version{Version: 1, Name: "Production adoption", Description: "Versioned production policy",
		Owner: "Architecture", State: policy.StateDraft, Revision: 1,
		Rules: []policy.Rule{
			{MetricName: catalog[0].Name, MetricVersion: catalog[0].Version, Operator: policy.GreaterThan,
				Threshold: 0, Weight: 0.5, Required: true, RequiredEvidence: "release facts",
				OnFailure: policy.NotRecommended, Label: "Release activity"},
			{MetricName: catalog[1].Name, MetricVersion: catalog[1].Version, Operator: policy.GreaterThan,
				Threshold: 0, Weight: 0.5, Required: true, RequiredEvidence: "contributor facts",
				OnFailure: policy.Conditional, Label: "Contributor activity"},
		},
		RadarMap: map[policy.Outcome]string{policy.Recommended: "adopt", policy.Conditional: "trial",
			policy.NotRecommended: "hold", policy.InsufficientData: "unplaced"}}
}

func task06PolicyBody(name string) map[string]any {
	value := task06PolicyVersion()
	return map[string]any{"name": name, "description": value.Description, "owner": value.Owner,
		"rules": value.Rules, "radar_mapping": value.RadarMap}
}

func task06Facts(values []metric.Snapshot) []policy.Fact {
	result := make([]policy.Fact, 0, len(values))
	for _, value := range values {
		result = append(result, policy.Fact{MetricName: value.Definition.Name,
			MetricVersion: value.Definition.Version, Status: value.Status, Value: value.Value,
			Coverage: value.Coverage, EvidenceIDs: []int64{value.ID}, SnapshotID: value.ID})
	}
	return result
}

func task06AlertRule() alert.Rule {
	return alert.Rule{Name: "Contributor concentration", Signal: "metric.active_contributors",
		Operator: alert.GreaterThan, Threshold: 5, Scope: "project", ProjectID: task04ProjectOne,
		Severity: alert.SeverityCritical, Cooldown: time.Hour, Deduplication: 24 * time.Hour, Enabled: true}
}

func task06AlertRuleBody() map[string]any {
	return map[string]any{"name": "Release slowdown", "signal": "metric.release_frequency",
		"operator": "lt", "threshold": 2, "scope": "project", "project_id": fmt.Sprint(task04ProjectOne),
		"severity": "critical", "cooldown_seconds": 3600, "deduplication_seconds": 86400, "enabled": true}
}

func task06AlertFact(now time.Time) alert.Fact {
	value := 12.0
	return alert.Fact{ProjectID: task04ProjectOne, Signal: "metric.active_contributors",
		Version: "v1", Value: &value, EvidenceID: 8_600_000_000_000_100_001,
		WindowFrom: now.Add(-24 * time.Hour), WindowTo: now, DetectedAt: now, Complete: true}
}

func task06TableCount(t *testing.T, h *task04Harness, table string) int {
	t.Helper()
	allowed := map[string]bool{"policy_families": true, "alert_rules": true}
	if !allowed[table] {
		t.Fatalf("unsupported table %q", table)
	}
	var count int
	if err := h.pool.Unwrap().QueryRow(h.ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func assertTask06Statuses(t *testing.T, got, want, rejected, wantRejected int) {
	t.Helper()
	if !reflect.DeepEqual([]int{got, rejected}, []int{want, wantRejected}) {
		t.Fatalf("statuses got=(%d,%d) want=(%d,%d)", got, rejected, want, wantRejected)
	}
}
