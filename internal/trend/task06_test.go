package trend_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
	"github.com/leohteixeira/opensource-project-intelligence/internal/trend"
)

var cutoff = time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

func windows(t *testing.T) (metric.Window, metric.Window) {
	t.Helper()
	observation := metric.Window{From: cutoff.AddDate(0, 0, -30), To: cutoff, Cutoff: cutoff}
	baseline := metric.Window{From: cutoff.AddDate(0, 0, -60), To: observation.From, Cutoff: cutoff}
	return observation, baseline
}

func TestUT116ProtectedSignalsRequireApprovedMember(t *testing.T) {
	for _, principal := range []access.Principal{{}, {ActorID: 1, Status: access.StatusPending, Role: access.RoleViewer}} {
		if err := access.Authorize(principal, access.ActionIntelligenceRead); err == nil ||
			(!errors.Is(err, access.ErrAuthenticationRequired) && !errors.Is(err, access.ErrAccessPending)) {
			t.Fatalf("principal unexpectedly authorized: %#v err=%v", principal, err)
		}
	}
}

func TestUT118ExplanationRequiresPublishedSignal(t *testing.T) {
	value := trend.Forecast{Kind: trend.KindForecast}
	if _, err := trend.ExplainForecast(value, "warning"); err == nil {
		t.Fatal("explanation published before signal")
	}
	value.ID = 10
	if explained, err := trend.ExplainForecast(value, " warning "); err != nil || explained.Explanation != "warning" {
		t.Fatalf("explained=%#v err=%v", explained, err)
	}
}

func TestUT119SupersededWarningRetainsHistory(t *testing.T) {
	value := trend.Forecast{ID: 10, Kind: trend.KindForecast, OutcomeStatus: "warning_confirmed"}
	superseded, err := trend.SupersedeForecast(value, 11)
	if err != nil || superseded.ID != 10 || superseded.SupersededBy != 11 || superseded.OutcomeStatus != "warning_confirmed" {
		t.Fatalf("superseded=%#v err=%v", superseded, err)
	}
}

func points(count int, value func(int) float64) []trend.Point {
	result := make([]trend.Point, count)
	for index := range count {
		result[index] = trend.Point{At: cutoff.AddDate(0, 0, index-count), Value: value(index), EvidenceID: int64(index + 1)}
	}
	return result
}

func TestUT113InvalidWindowsAndHorizon(t *testing.T) {
	observation, baseline := windows(t)
	bad := baseline
	bad.To = observation.To
	if _, err := trend.CalculateObserved(1, trend.ObservedV1, "release_frequency", "v1", observation, bad, points(8, func(i int) float64 { return float64(i) })); err == nil {
		t.Fatal("overlapping baseline accepted")
	}
	if _, err := trend.CalculateForecast(1, trend.ForecastV1, "release_frequency", "v1", points(20, func(i int) float64 { return float64(i) }), 91); err == nil {
		t.Fatal("unbounded forecast accepted")
	}
}

func TestUT114SparseHistoryIsExplicit(t *testing.T) {
	observation, baseline := windows(t)
	result, err := trend.CalculateObserved(1, trend.ObservedV1, "release_frequency", "v1", observation, baseline, points(3, func(i int) float64 { return float64(i) }))
	if err != nil || result.Direction != trend.DirectionNone || result.MinimumPoints != 6 || result.Coverage.Note == "" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT115DetectorBounds(t *testing.T) {
	definition := trend.ObservedV1
	definition.MaximumPoints = 6
	observation, baseline := windows(t)
	if _, err := trend.CalculateObserved(1, definition, "release_frequency", "v1", observation, baseline, points(7, func(i int) float64 { return float64(i) })); err == nil {
		t.Fatal("unbounded detector accepted")
	}
}

func TestUT117TrendRerunDeterminism(t *testing.T) {
	observation, baseline := windows(t)
	input := points(8, func(i int) float64 { return float64(i) })
	first, _ := trend.CalculateObserved(1, trend.ObservedV1, "release_frequency", "v1", observation, baseline, input)
	second, _ := trend.CalculateObserved(1, trend.ObservedV1, "release_frequency", "v1", observation, baseline, input)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same inputs produced different observations")
	}
}

func TestUT239TheilSenResistsOutlier(t *testing.T) {
	input := points(7, func(i int) float64 { return float64(i) })
	input[3].Value = 1_000
	if slope := trend.TheilSen(input); slope != 1 {
		t.Fatalf("slope=%v", slope)
	}
}

func TestUT240MannKendallNoisySeries(t *testing.T) {
	observation, baseline := windows(t)
	input := points(8, func(i int) float64 { return []float64{1, 2, 1, 2, 1, 2, 1, 2}[i] })
	result, err := trend.CalculateObserved(1, trend.ObservedV1, "release_frequency", "v1", observation, baseline, input)
	if err != nil || result.Direction != trend.DirectionStable {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUT241ForecastBacktestPublishesInterval(t *testing.T) {
	input := points(28, func(i int) float64 { return []float64{1, 2, 3, 4, 5, 6, 7}[i%7] })
	result, err := trend.CalculateForecast(1, trend.ForecastV1, "release_frequency", "v1", input, 14)
	if err != nil || result.SelectedModel != "seasonal_baseline" || result.IntervalLow == nil || result.IntervalHigh == nil || result.BacktestError == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
