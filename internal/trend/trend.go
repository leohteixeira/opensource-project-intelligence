// Package trend calculates reproducible observed trends and predictive early warnings.
// Observations and forecasts intentionally use distinct result types and method metadata.
package trend

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
)

var ErrInvalid = errors.New("invalid trend request")

type Kind string

const (
	KindObserved Kind = "observed"
	KindForecast Kind = "forecast"
)

type Direction string

const (
	DirectionIncrease Direction = "increase"
	DirectionDecrease Direction = "decrease"
	DirectionStable   Direction = "stable"
	DirectionNone     Direction = "insufficient_data"
)

type Point struct {
	At         time.Time `json:"at"`
	Value      float64   `json:"value"`
	EvidenceID int64     `json:"evidence_id,string,omitempty"`
}

type Definition struct {
	Version       string  `json:"version"`
	MinimumPoints int     `json:"minimum_points"`
	MaximumPoints int     `json:"maximum_points"`
	Consistency   float64 `json:"consistency_threshold"`
}

var ObservedV1 = Definition{Version: "theil-sen-mann-kendall-v1", MinimumPoints: 6, MaximumPoints: 730, Consistency: 0.6}

type Observed struct {
	ID                int64           `json:"id,string,omitempty"`
	ProjectID         int64           `json:"project_id,string"`
	MetricName        string          `json:"metric_name"`
	MetricVersion     string          `json:"metric_version"`
	Kind              Kind            `json:"kind"`
	Direction         Direction       `json:"direction"`
	Magnitude         *float64        `json:"magnitude,omitempty"`
	ObservationWindow metric.Window   `json:"observation_window"`
	BaselineWindow    metric.Window   `json:"baseline_window"`
	MethodVersion     string          `json:"method_version"`
	Coverage          metric.Coverage `json:"coverage"`
	MinimumPoints     int             `json:"minimum_points"`
	EvidenceIDs       []int64         `json:"evidence_ids"`
	InputDigest       string          `json:"input_digest,omitempty"`
	SupersededBy      int64           `json:"superseded_by,string,omitempty"`
}

func CalculateObserved(projectID int64, definition Definition, metricName, metricVersion string,
	observation, baseline metric.Window, points []Point) (Observed, error) {
	if projectID <= 0 || strings.TrimSpace(metricName) == "" || strings.TrimSpace(metricVersion) == "" ||
		strings.TrimSpace(definition.Version) == "" || definition.MinimumPoints < 3 ||
		definition.MaximumPoints < definition.MinimumPoints || definition.Consistency <= 0 ||
		definition.Consistency > 1 || observation.Validate() != nil || baseline.Validate() != nil ||
		observation.Cutoff != baseline.Cutoff || baseline.To.After(observation.From) {
		return Observed{}, fmt.Errorf("%w: invalid definition or windows", ErrInvalid)
	}
	if len(points) > definition.MaximumPoints {
		return Observed{}, fmt.Errorf("%w: at most %d points are supported", ErrInvalid, definition.MaximumPoints)
	}
	ordered := slices.Clone(points)
	slices.SortFunc(ordered, func(a, b Point) int { return a.At.Compare(b.At) })
	result := Observed{ProjectID: projectID, MetricName: metricName, MetricVersion: metricVersion,
		Kind: KindObserved, Direction: DirectionNone, ObservationWindow: observation,
		BaselineWindow: baseline, MethodVersion: definition.Version, MinimumPoints: definition.MinimumPoints,
		Coverage:    metric.Coverage{Eligible: max(definition.MinimumPoints, len(ordered)), Observed: len(ordered)},
		EvidenceIDs: []int64{}}
	for index, point := range ordered {
		if point.At.Location() != time.UTC || !point.At.Before(observation.To) || point.At.Before(baseline.From) ||
			math.IsNaN(point.Value) || math.IsInf(point.Value, 0) || index > 0 && !ordered[index-1].At.Before(point.At) {
			return Observed{}, fmt.Errorf("%w: points must be unique, finite, ordered UTC samples in the signal windows", ErrInvalid)
		}
		if point.EvidenceID > 0 {
			result.EvidenceIDs = append(result.EvidenceIDs, point.EvidenceID)
		}
	}
	if len(ordered) < definition.MinimumPoints {
		result.Coverage.Note = fmt.Sprintf("at least %d points are required", definition.MinimumPoints)
		return result, nil
	}
	slope := TheilSen(ordered)
	consistency := MannKendallConsistency(ordered)
	result.Magnitude = &slope
	if consistency < definition.Consistency {
		result.Direction = DirectionStable
		result.Coverage.Note = "directional consistency is below the published threshold"
	} else if slope > 0 {
		result.Direction = DirectionIncrease
	} else if slope < 0 {
		result.Direction = DirectionDecrease
	} else {
		result.Direction = DirectionStable
	}
	return result, nil
}

func TheilSen(points []Point) float64 {
	if len(points) < 2 {
		return 0
	}
	slopes := make([]float64, 0, len(points)*(len(points)-1)/2)
	for left := 0; left < len(points)-1; left++ {
		for right := left + 1; right < len(points); right++ {
			days := points[right].At.Sub(points[left].At).Hours() / 24
			if days > 0 {
				slopes = append(slopes, (points[right].Value-points[left].Value)/days)
			}
		}
	}
	value, _ := metric.Median(slopes)
	return value
}

func MannKendallConsistency(points []Point) float64 {
	if len(points) < 2 {
		return 0
	}
	pairs, score := 0, 0
	for left := 0; left < len(points)-1; left++ {
		for right := left + 1; right < len(points); right++ {
			pairs++
			switch {
			case points[right].Value > points[left].Value:
				score++
			case points[right].Value < points[left].Value:
				score--
			}
		}
	}
	return math.Abs(float64(score)) / float64(pairs)
}

type ForecastDefinition struct {
	Version        string  `json:"version"`
	MinimumPoints  int     `json:"minimum_points"`
	MaximumPoints  int     `json:"maximum_points"`
	MaximumHorizon int     `json:"maximum_horizon_days"`
	SeasonLength   int     `json:"season_length"`
	Alpha          float64 `json:"alpha"`
}

var ForecastV1 = ForecastDefinition{Version: "rolling-baseline-v1", MinimumPoints: 12, MaximumPoints: 730, MaximumHorizon: 90, SeasonLength: 7, Alpha: 0.35}

type Forecast struct {
	ID            int64           `json:"id,string,omitempty"`
	ProjectID     int64           `json:"project_id,string"`
	MetricName    string          `json:"metric_name"`
	MetricVersion string          `json:"metric_version"`
	Kind          Kind            `json:"kind"`
	Status        Direction       `json:"status"`
	HorizonDays   int             `json:"horizon_days"`
	Predicted     *float64        `json:"predicted,omitempty"`
	IntervalLow   *float64        `json:"interval_low,omitempty"`
	IntervalHigh  *float64        `json:"interval_high,omitempty"`
	Confidence    *float64        `json:"confidence,omitempty"`
	BacktestError *float64        `json:"backtest_error,omitempty"`
	ModelVersion  string          `json:"model_version"`
	SelectedModel string          `json:"selected_model,omitempty"`
	Coverage      metric.Coverage `json:"coverage"`
	EvidenceIDs   []int64         `json:"evidence_ids"`
	OutcomeStatus string          `json:"outcome_status"`
	Explanation   string          `json:"explanation,omitempty"`
	SupersededBy  int64           `json:"superseded_by,string,omitempty"`
}

// ExplainForecast attaches optional prose only after the deterministic signal is published.
// It never changes the predictive fields or the outcome-evaluation status.
func ExplainForecast(value Forecast, explanation string) (Forecast, error) {
	if value.ID <= 0 || value.Kind != KindForecast || strings.TrimSpace(explanation) == "" {
		return Forecast{}, fmt.Errorf("%w: a published forecast and explanation are required", ErrInvalid)
	}
	value.Explanation = strings.TrimSpace(explanation)
	return value, nil
}

// SupersedeForecast retains the historical warning and its evaluation status while linking its replacement.
func SupersedeForecast(value Forecast, replacementID int64) (Forecast, error) {
	if value.ID <= 0 || replacementID <= 0 || value.ID == replacementID || value.SupersededBy != 0 {
		return Forecast{}, fmt.Errorf("%w: a distinct replacement is required", ErrInvalid)
	}
	value.SupersededBy = replacementID
	return value, nil
}

func CalculateForecast(projectID int64, definition ForecastDefinition, metricName, metricVersion string,
	points []Point, horizonDays int) (Forecast, error) {
	if projectID <= 0 || strings.TrimSpace(metricName) == "" || strings.TrimSpace(metricVersion) == "" ||
		strings.TrimSpace(definition.Version) == "" || definition.MinimumPoints < 3 ||
		definition.MaximumPoints < definition.MinimumPoints || definition.SeasonLength < 2 ||
		definition.Alpha <= 0 || definition.Alpha > 1 || horizonDays <= 0 || horizonDays > definition.MaximumHorizon ||
		len(points) > definition.MaximumPoints {
		return Forecast{}, fmt.Errorf("%w: invalid model, history, or forecast horizon", ErrInvalid)
	}
	result := Forecast{ProjectID: projectID, MetricName: metricName, MetricVersion: metricVersion,
		Kind: KindForecast, Status: DirectionNone, HorizonDays: horizonDays, ModelVersion: definition.Version,
		Coverage:    metric.Coverage{Eligible: max(definition.MinimumPoints, len(points)), Observed: len(points)},
		EvidenceIDs: []int64{}, OutcomeStatus: "pending"}
	for _, point := range points {
		if math.IsNaN(point.Value) || math.IsInf(point.Value, 0) {
			return Forecast{}, fmt.Errorf("%w: forecast values must be finite", ErrInvalid)
		}
		if point.EvidenceID > 0 {
			result.EvidenceIDs = append(result.EvidenceIDs, point.EvidenceID)
		}
	}
	if len(points) < definition.MinimumPoints {
		result.Coverage.Note = fmt.Sprintf("at least %d points are required", definition.MinimumPoints)
		return result, nil
	}
	exponentialError := rollingError(points, definition.SeasonLength, func(history []Point) float64 {
		return exponential(history, definition.Alpha)
	})
	seasonalError := rollingError(points, definition.SeasonLength, func(history []Point) float64 {
		return seasonal(history, definition.SeasonLength)
	})
	predicted, selected, backtest := exponential(points, definition.Alpha), "exponential_smoothing", exponentialError
	if seasonalError < exponentialError {
		predicted, selected, backtest = seasonal(points, definition.SeasonLength), "seasonal_baseline", seasonalError
	}
	width := 1.96 * backtest * math.Sqrt(1+float64(horizonDays)/float64(definition.SeasonLength))
	low, high := predicted-width, predicted+width
	confidence := 1 / (1 + backtest)
	result.Status, result.Predicted, result.IntervalLow, result.IntervalHigh = DirectionStable, &predicted, &low, &high
	result.Confidence, result.BacktestError, result.SelectedModel, result.OutcomeStatus = &confidence, &backtest, selected, "unevaluated"
	return result, nil
}

func rollingError(points []Point, start int, predictor func([]Point) float64) float64 {
	start = max(2, start)
	total, count := 0.0, 0
	for index := start; index < len(points); index++ {
		total += math.Abs(predictor(points[:index]) - points[index].Value)
		count++
	}
	if count == 0 {
		return math.Inf(1)
	}
	return total / float64(count)
}

func exponential(points []Point, alpha float64) float64 {
	level := points[0].Value
	for _, point := range points[1:] {
		level = alpha*point.Value + (1-alpha)*level
	}
	return level
}

func seasonal(points []Point, season int) float64 {
	start := max(0, len(points)-season)
	values := make([]float64, 0, len(points)-start)
	for _, point := range points[start:] {
		values = append(values, point.Value)
	}
	value, _ := metric.Median(values)
	return value
}
