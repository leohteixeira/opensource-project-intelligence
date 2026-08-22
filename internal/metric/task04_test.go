package metric_test

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/comparison"
	"github.com/leohteixeira/opensource-project-intelligence/internal/contributor"
	"github.com/leohteixeira/opensource-project-intelligence/internal/issue"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/timewindow"
	"github.com/leohteixeira/opensource-project-intelligence/internal/pullrequest"
	"github.com/leohteixeira/opensource-project-intelligence/internal/release"
)

var cutoff = time.Date(2026, 8, 20, 14, 35, 0, 0, time.UTC)

func window(t *testing.T, name string) metric.Window {
	t.Helper()
	value, err := metric.PresetWindow(name, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func available(t *testing.T, projectID int64, name string, value float64, valueWindow metric.Window) metric.Snapshot {
	t.Helper()
	definition, ok := metric.DefinitionByName(name)
	if !ok {
		t.Fatalf("definition %s missing", name)
	}
	snapshot, err := metric.NewSnapshot(projectID, definition, valueWindow, metric.StatusAvailable, &value,
		metric.Coverage{Eligible: 1, Observed: 1}, []metric.Factor{{Name: "evidence", Value: &value, Unit: definition.Unit}}, []int64{10})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func approved(role access.Role) access.Principal {
	return access.Principal{ActorID: 1, Workspace: 1, Status: access.StatusActive, Role: role, Kind: access.ActorMember}
}

func TestUT085UnsupportedMetricWindows(t *testing.T) {
	if _, err := metric.PresetWindow("45d", cutoff); err == nil {
		t.Fatal("unsupported window accepted")
	}
}

func TestUT086MissingEvidenceNeverBecomesZero(t *testing.T) {
	values, err := metric.ComputeCatalog(1, window(t, "90d"), metric.Facts{})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if value.Status == metric.StatusAvailable || value.Value != nil {
			t.Fatalf("%s manufactured zero", value.Definition.Name)
		}
	}
}

func TestUT087MetricEvidenceRemainsBoundedAndLinked(t *testing.T) {
	value := available(t, 1, "release_frequency", 2, window(t, "90d"))
	if len(value.Factors) != 1 || value.Factors[0].Name != "evidence" {
		t.Fatalf("unexpected factors: %#v", value.Factors)
	}
}

func TestUT088PendingAndAnonymousCannotReadMetrics(t *testing.T) {
	for _, principal := range []access.Principal{{}, {ActorID: 1, Status: access.StatusPending, Role: access.RoleViewer}} {
		if err := access.Authorize(principal, access.ActionIntelligenceRead); err == nil {
			t.Fatal("principal unexpectedly authorized")
		}
	}
}

func TestUT089MetricRepetitionIsDeterministic(t *testing.T) {
	facts := metric.Facts{ReleaseCovered: true, Releases: []release.Release{{ID: 1, PublishedAt: cutoff.Add(-time.Hour)}}}
	first, err := metric.ComputeCatalog(1, window(t, "90d"), facts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := metric.ComputeCatalog(1, window(t, "90d"), facts)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("identical facts produced different results")
	}
}

func TestUT090OverallWaitsForAllDimensions(t *testing.T) {
	dimensions := healthDimensions(0.7)
	dimensions[3].Status, dimensions[3].Score = metric.StatusUnavailable, nil
	value, err := metric.CalculateHealth(1, window(t, "90d"), "v1", dimensions)
	if err != nil {
		t.Fatal(err)
	}
	if value.Overall != nil || value.OverallStatus != metric.StatusInsufficientData {
		t.Fatal("partial overall published")
	}
}

func TestUT091ArchivedProjectsReceiveNoSnapshots(t *testing.T) {
	if metric.MaterializationAllowed("archived") || !metric.MaterializationAllowed("active") {
		t.Fatal("lifecycle gate is wrong")
	}
}

func TestUT092InvalidContributorWindowRejected(t *testing.T) {
	invalid := metric.Window{From: cutoff, To: cutoff.Add(-time.Hour), Cutoff: cutoff}
	if invalid.Validate() == nil {
		t.Fatal("invalid contributor window accepted")
	}
}

func TestUT093NoContributorsIsInsufficient(t *testing.T) {
	value := contributor.Aggregate(nil, timewindow.Window{From: cutoff.Add(-time.Hour), To: cutoff})
	if value.Status != "insufficient_data" || value.TopThreeShare != nil {
		t.Fatalf("unexpected empty result: %#v", value)
	}
}

func TestUT094ContributorJSONHasNoPrivateEmailAndCanBePaged(t *testing.T) {
	commits := []contributor.Commit{{AccountID: "a", CommittedAt: cutoff.Add(-time.Minute), DefaultBranch: true}, {AccountID: "b", CommittedAt: cutoff.Add(-2 * time.Minute), DefaultBranch: true}}
	value := contributor.Aggregate(commits, timewindow.Window{From: cutoff.Add(-time.Hour), To: cutoff})
	encoded, err := json.Marshal(value.Contributors[:1])
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || contains(string(encoded), "email") {
		t.Fatalf("private field exposed: %s", encoded)
	}
}

func TestUT095OnlyAnalystsConfirmIdentity(t *testing.T) {
	if access.Authorize(approved(access.RoleViewer), access.ActionProjectWrite) == nil {
		t.Fatal("viewer may correct identity")
	}
	if err := access.Authorize(approved(access.RoleAnalyst), access.ActionProjectWrite); err != nil {
		t.Fatal(err)
	}
}

func TestUT096RepeatedVerifiedLinkIsIdempotent(t *testing.T) {
	commit := contributor.Commit{AccountID: "a", IdentityID: "one", LinkStatus: contributor.LinkVerified, CommittedAt: cutoff.Add(-time.Minute), DefaultBranch: true}
	first := contributor.Aggregate([]contributor.Commit{commit}, timewindow.Window{From: cutoff.Add(-time.Hour), To: cutoff})
	second := contributor.Aggregate([]contributor.Commit{commit}, timewindow.Window{From: cutoff.Add(-time.Hour), To: cutoff})
	if !reflect.DeepEqual(first, second) {
		t.Fatal("reconfirmation changed result")
	}
}

func TestUT097LinkagePrecedesConcentrationPublication(t *testing.T) {
	commits := []contributor.Commit{{AccountID: "a", IdentityID: "one", LinkStatus: contributor.LinkVerified, CommittedAt: cutoff.Add(-time.Minute), DefaultBranch: true}, {AccountID: "b", IdentityID: "one", LinkStatus: contributor.LinkVerified, CommittedAt: cutoff.Add(-time.Minute), DefaultBranch: true}}
	value := contributor.Aggregate(commits, timewindow.Window{From: cutoff.Add(-time.Hour), To: cutoff})
	if value.Active != 1 || value.ResolutionCoverage != 1 {
		t.Fatalf("identity not resolved first: %#v", value)
	}
}

func TestUT098DeletedAccountsStayHistorical(t *testing.T) {
	commit := contributor.Commit{AccountID: "deleted-source-account", CommittedAt: cutoff.Add(-time.Minute), DefaultBranch: true}
	value := contributor.Aggregate([]contributor.Commit{commit}, timewindow.Window{From: cutoff.Add(-time.Hour), To: cutoff})
	if value.Active != 1 || value.Contributors[0].Key != "account:deleted-source-account" {
		t.Fatal("historical source evidence disappeared")
	}
}

func TestUT106ComparisonValidation(t *testing.T) {
	valueWindow := window(t, "90d")
	invalid := [][]comparison.Project{{{ID: 1, Resolved: true}}, {{ID: 1, Resolved: true}, {ID: 1, Resolved: true}}}
	for _, projects := range invalid {
		if _, err := comparison.Materialize(1, projects, valueWindow); err == nil {
			t.Fatal("invalid Project set accepted")
		}
	}
}

func TestUT107MissingProjectEvidenceRemainsVisible(t *testing.T) {
	valueWindow := window(t, "90d")
	value, err := comparison.Materialize(1, []comparison.Project{{ID: 1, Resolved: true, Metrics: []metric.Snapshot{available(t, 1, "release_frequency", 0, valueWindow)}}, {ID: 2, Resolved: true}}, valueWindow)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Rows) != 1 || value.Rows[0].Cells[1].Status != metric.StatusInsufficientData {
		t.Fatal("missing Project was dropped")
	}
}

func TestUT108ComparisonEvidenceIsRetained(t *testing.T) {
	valueWindow := window(t, "90d")
	snapshot := available(t, 1, "release_frequency", 1, valueWindow)
	value, err := comparison.Materialize(1, []comparison.Project{{ID: 1, Resolved: true, Metrics: []metric.Snapshot{snapshot}}, {ID: 2, Resolved: true, Metrics: []metric.Snapshot{available(t, 2, "release_frequency", 2, valueWindow)}}}, valueWindow)
	if err != nil {
		t.Fatal(err)
	}
	if len(value.Rows[0].Cells[0].Evidence) != 1 {
		t.Fatal("comparison silently truncated evidence")
	}
}

func TestUT109AnonymousAndPendingCannotCompare(t *testing.T) {
	TestUT088PendingAndAnonymousCannotReadMetrics(t)
}

func TestUT110ComparisonRepetitionIsDeterministic(t *testing.T) {
	valueWindow := window(t, "90d")
	projects := []comparison.Project{{ID: 1, Resolved: true, Metrics: []metric.Snapshot{available(t, 1, "release_frequency", 1, valueWindow)}}, {ID: 2, Resolved: true, Metrics: []metric.Snapshot{available(t, 2, "release_frequency", 2, valueWindow)}}}
	first, _ := comparison.Materialize(8, projects, valueWindow)
	second, _ := comparison.Materialize(8, projects, valueWindow)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("comparison is not deterministic")
	}
}

func TestUT111ComparisonWaitsForIdentityResolution(t *testing.T) {
	if _, err := comparison.Materialize(1, []comparison.Project{{ID: 1, Resolved: true}, {ID: 2}}, window(t, "90d")); err == nil {
		t.Fatal("unresolved Project accepted")
	}
}

func TestUT112ArchivedRemainsAndDeletedBecomesUnavailable(t *testing.T) {
	valueWindow := window(t, "90d")
	if _, err := comparison.Materialize(1, []comparison.Project{{ID: 1, Resolved: true, Archived: true}, {ID: 2, Resolved: true}}, valueWindow); err != nil {
		t.Fatal(err)
	}
	if _, err := comparison.Materialize(1, []comparison.Project{{ID: 1, Resolved: true, Deleted: true}, {ID: 2, Resolved: true}}, valueWindow); err == nil {
		t.Fatal("deleted Project accepted")
	}
}

func TestUT229StableReleaseCohort(t *testing.T) {
	values := []release.Release{{PublishedAt: cutoff.Add(-time.Hour)}, {PublishedAt: cutoff.Add(-time.Hour), Draft: true}, {PublishedAt: cutoff.Add(-time.Hour), Prerelease: true}}
	stable := release.StableInWindow(values, timewindow.Window{From: cutoff.Add(-24 * time.Hour), To: cutoff})
	if len(stable) != 1 {
		t.Fatalf("got %d stable releases", len(stable))
	}
}

func TestUT230ContributorCohort(t *testing.T) {
	values := []contributor.Commit{{AccountID: "human", DefaultBranch: true, CommittedAt: cutoff.Add(-time.Minute)}, {AccountID: "bot", Bot: true, DefaultBranch: true, CommittedAt: cutoff.Add(-time.Minute)}, {AccountID: "merge", MergeCommit: true, DefaultBranch: true, CommittedAt: cutoff.Add(-time.Minute)}, {AccountID: "other", CommittedAt: cutoff.Add(-time.Minute)}}
	result := contributor.Aggregate(values, timewindow.Window{From: cutoff.Add(-time.Hour), To: cutoff})
	if result.Active != 1 || result.Contributors[0].Key != "account:human" {
		t.Fatalf("wrong cohort: %#v", result)
	}
}

func TestUT231IssueFirstResponseCohort(t *testing.T) {
	opened := cutoff.Add(-4 * time.Hour)
	value := issue.Issue{OpenerID: "opener", CreatedAt: opened, Responses: []issue.Response{{At: opened.Add(time.Hour), ActorID: "opener", Public: true, Member: true}, {At: opened.Add(2 * time.Hour), ActorID: "bot", Public: true, Member: true, Bot: true}, {At: opened.Add(3 * time.Hour), ActorID: "maintainer", Public: true, Member: true}}}
	duration, ok := issue.FirstResponse(value, cutoff)
	if !ok || duration != 3*time.Hour {
		t.Fatalf("got %v %t", duration, ok)
	}
}

func TestUT232UnansweredIssueCoverage(t *testing.T) {
	facts := metric.Facts{IssueCovered: true, Issues: []issue.Issue{{ID: 1, CreatedAt: cutoff.Add(-time.Hour)}}}
	values, err := metric.ComputeCatalog(1, window(t, "30d"), facts)
	if err != nil {
		t.Fatal(err)
	}
	value := findMetric(t, values, "median_issue_first_response")
	if value.Status != metric.StatusInsufficientData || value.Coverage.Eligible != 1 || value.Coverage.Observed != 0 {
		t.Fatalf("unexpected censoring: %#v", value)
	}
}

func TestUT233PRMergeCohort(t *testing.T) {
	ready := cutoff.Add(-3 * time.Hour)
	duration, fallback := pullrequest.MergeDuration(pullrequest.PullRequest{CreatedAt: cutoff.Add(-8 * time.Hour), ReadyAt: &ready, MergedAt: cutoff.Add(-time.Hour)})
	if fallback || duration != 2*time.Hour {
		t.Fatalf("got %v fallback=%t", duration, fallback)
	}
}

func TestUT234PRReadinessFallback(t *testing.T) {
	duration, fallback := pullrequest.MergeDuration(pullrequest.PullRequest{CreatedAt: cutoff.Add(-4 * time.Hour), MergedAt: cutoff.Add(-time.Hour)})
	if !fallback || duration != 3*time.Hour {
		t.Fatalf("got %v fallback=%t", duration, fallback)
	}
}

func TestUT235BacklogReconstruction(t *testing.T) {
	w := timewindow.Window{From: cutoff.Add(-2 * time.Hour), To: cutoff}
	events := []issue.StateEvent{{IssueID: 1, At: cutoff.Add(-3 * time.Hour), State: "open"}, {IssueID: 1, At: cutoff.Add(-90 * time.Minute), State: "closed"}, {IssueID: 1, At: cutoff.Add(-30 * time.Minute), State: "open"}, {IssueID: 2, At: cutoff.Add(-time.Hour), State: "open"}}
	if got := issue.BacklogChange(events, w); got != 1 {
		t.Fatalf("got backlog change %d", got)
	}
}

func TestUT236HealthEqualWeights(t *testing.T) {
	value, err := metric.CalculateHealth(1, window(t, "90d"), "v1", healthDimensions(0.7))
	if err != nil {
		t.Fatal(err)
	}
	if value.Overall == nil || math.Abs(*value.Overall-0.7) > 1e-12 {
		t.Fatalf("unexpected overall %#v", value.Overall)
	}
	for _, dimension := range value.Dimensions {
		if math.Abs(dimension.Weight-1.0/7.0) > 1e-12 {
			t.Fatal("weight was redistributed")
		}
	}
}

func TestUT237HealthMissingWeightNotRedistributed(t *testing.T) {
	TestUT090OverallWaitsForAllDimensions(t)
}

func TestUT238CommonComparisonCutoff(t *testing.T) {
	valueWindow := window(t, "90d")
	otherWindow := valueWindow
	otherWindow.Cutoff, otherWindow.To, otherWindow.From = cutoff.Add(-time.Hour), cutoff.Add(-time.Hour), valueWindow.From.Add(-time.Hour)
	value, err := comparison.Materialize(1, []comparison.Project{{ID: 1, Resolved: true, Metrics: []metric.Snapshot{available(t, 1, "release_frequency", 1, valueWindow)}}, {ID: 2, Resolved: true, Metrics: []metric.Snapshot{available(t, 2, "release_frequency", 2, otherWindow)}}}, valueWindow)
	if err != nil {
		t.Fatal(err)
	}
	if value.Rows[0].Cells[1].Status != metric.StatusIncomparable || value.Rows[0].Cells[1].Value != nil {
		t.Fatal("mixed cutoff remained numeric")
	}
}

func healthDimensions(score float64) []metric.Dimension {
	values := make([]metric.Dimension, 0, 7)
	for _, name := range metric.HealthDimensionNames {
		value := score
		values = append(values, metric.Dimension{Name: name, Status: metric.StatusAvailable, Score: &value, Version: "v1"})
	}
	return values
}

func findMetric(t *testing.T, values []metric.Snapshot, name string) metric.Snapshot {
	t.Helper()
	for _, value := range values {
		if value.Definition.Name == name {
			return value
		}
	}
	t.Fatalf("metric %s not found", name)
	return metric.Snapshot{}
}

func contains(value, needle string) bool {
	for index := 0; index+len(needle) <= len(value); index++ {
		if value[index:index+len(needle)] == needle {
			return true
		}
	}
	return false
}
