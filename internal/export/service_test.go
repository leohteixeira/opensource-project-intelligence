package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func validRequest(format Format) Request {
	return Request{ProjectIDs: []int64{2, 1}, Resource: "metrics", Format: format,
		Filters: map[string]string{"state": "active"}, Locale: "en",
		WindowFrom: time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC),
		WindowTo:   time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
		Cutoff:     time.Date(2026, 8, 20, 14, 35, 0, 0, time.UTC)}
}

func records() []Record {
	value := 31.2
	return []Record{{ProjectID: 2, Metric: "median_pr_merge_time", Value: &value, Status: Available,
		Unit: "hours", Definition: "v3", Formula: "median(merged-created)", Coverage: "90d of 90d",
		Provenance: []string{"github:pull_request:100"}, PolicyContext: "policy:v2", AnalysisRunIDs: []int64{99}},
		{ProjectID: 1, Metric: "maintainer_count", Status: InsufficientData, Unit: "count",
			Definition: "v2", Formula: "distinct(maintainer)", Coverage: "30d of 90d", Provenance: []string{}}}
}

func TestUT197RequestValidationFreezesOneScopeWindowAndCutoff(t *testing.T) {
	request := validRequest(CSV)
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	request.ProjectIDs = append(request.ProjectIDs, request.ProjectIDs[0])
	if !errors.Is(request.Validate(), ErrInvalidRequest) {
		t.Fatal("duplicate scope was accepted")
	}
	request = validRequest(CSV)
	request.WindowTo = request.Cutoff.Add(time.Second)
	if !errors.Is(request.Validate(), ErrInvalidRequest) {
		t.Fatal("future window was accepted")
	}
}

func TestUT198CSVUsesStableMachineFieldsAndLocalizedLabels(t *testing.T) {
	generator := NewGenerator(0)
	english, err := generator.Generate(context.Background(), validRequest(CSV), records())
	if err != nil {
		t.Fatal(err)
	}
	portugueseRequest := validRequest(CSV)
	portugueseRequest.Locale = "pt-BR"
	portuguese, err := generator.Generate(context.Background(), portugueseRequest, records())
	if err != nil {
		t.Fatal(err)
	}
	read := func(body []byte) [][]string {
		rows, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	enRows, ptRows := read(english.Body), read(portuguese.Body)
	if strings.Join(enRows[0], "|") != strings.Join(ptRows[0], "|") || enRows[1][0] != "Project" || ptRows[1][0] != "Projeto" {
		t.Fatalf("headers en=%v pt=%v", enRows[:2], ptRows[:2])
	}
}

func TestUT199MissingStatesNeverBecomeZeroOrEmptyStatus(t *testing.T) {
	generated, err := NewGenerator(0).Generate(context.Background(), validRequest(CSV), records())
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(generated.Body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[2][0] != "1" || rows[2][3] != "" || rows[2][4] != string(InsufficientData) {
		t.Fatalf("missing row = %v", rows[2])
	}
}

func TestUT200EvidenceJSONRetainsInterpretationAndProvenance(t *testing.T) {
	generated, err := NewGenerator(0).Generate(context.Background(), validRequest(EvidenceJSON), records())
	if err != nil {
		t.Fatal(err)
	}
	var value Package
	if err := json.Unmarshal(generated.Body, &value); err != nil {
		t.Fatal(err)
	}
	if value.SchemaVersion != "opi.evidence-export/v1" || value.Cutoff != validRequest(EvidenceJSON).Cutoff ||
		value.Rows[1].Formula == "" || len(value.Rows[1].Provenance) == 0 || len(value.Rows[1].AnalysisRunIDs) == 0 {
		t.Fatalf("package = %#v", value)
	}
}

func TestUT201EquivalentScopeCutoffProducesEquivalentChecksummedBytes(t *testing.T) {
	generator := NewGenerator(0)
	first, _ := generator.Generate(context.Background(), validRequest(EvidenceJSON), records())
	second, _ := generator.Generate(context.Background(), validRequest(EvidenceJSON), records())
	if first.Digest != second.Digest || string(first.Body) != string(second.Body) || first.DigestHex == "" {
		t.Fatal("equivalent exports diverged")
	}
}

func TestUT202ZeroRowExportIsValidAndOversizedExportFails(t *testing.T) {
	empty, err := NewGenerator(1024).Generate(context.Background(), validRequest(CSV), nil)
	if err != nil || empty.Rows != 0 || len(empty.Body) == 0 {
		t.Fatalf("empty = %#v, %v", empty, err)
	}
	if _, err := NewGenerator(32).Generate(context.Background(), validRequest(CSV), records()); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("size error = %v", err)
	}
}

func TestUT203CancellationAndCapacityAreBounded(t *testing.T) {
	coordinator, err := NewCoordinator(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	coordinator.gate <- struct{}{}
	if _, err := coordinator.Generate(context.Background(), validRequest(CSV), records()); !errors.Is(err, ErrBusy) {
		t.Fatalf("capacity error = %v", err)
	}
	<-coordinator.gate
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := coordinator.Generate(ctx, validRequest(CSV), records()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestUT204ArtifactAuthorizationChecksumAndExactExpiry(t *testing.T) {
	completed := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	body := []byte("project_id\n1\n")
	artifact, err := NewArtifact(1, "projects/1/exports/2.csv", "text/csv", body, completed)
	if err != nil {
		t.Fatal(err)
	}
	if !artifact.Verify(body) || artifact.Verify([]byte("changed")) {
		t.Fatal("checksum verification failed")
	}
	if err := artifact.Authorize(1, completed.Add(Lifetime-time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	if err := artifact.Authorize(1, completed.Add(Lifetime)); !errors.Is(err, ErrGone) {
		t.Fatalf("expiry error = %v", err)
	}
	if err := artifact.Authorize(2, completed); err == nil {
		t.Fatal("cross-project download was allowed")
	}
}
