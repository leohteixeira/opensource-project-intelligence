// Package health classifies dependencies without exposing their configuration.
package health

// Importance determines whether a failed dependency blocks readiness.
type Importance string

const (
	Required Importance = "required"
	Optional Importance = "optional"
)

// State is the safe operational state of one dependency.
type State string

const (
	Available   State = "available"
	Unavailable State = "unavailable"
	Degraded    State = "degraded"
	Disabled    State = "disabled"
)

// Dependency is a safe named probe outcome.
type Dependency struct {
	Name       string
	Importance Importance
	State      State
}

// Report separates traffic readiness from optional degradation.
type Report struct {
	Ready        bool             `json:"-"`
	Status       string           `json:"status"`
	Dependencies map[string]State `json:"dependencies"`
}

// Evaluate computes readiness without carrying errors, endpoints, or credentials into the report.
func Evaluate(dependencies []Dependency) Report {
	report := Report{Ready: true, Status: "ready", Dependencies: make(map[string]State, len(dependencies))}
	for _, dependency := range dependencies {
		state := dependency.State
		if dependency.Importance == Optional && state == Unavailable {
			state = Degraded
		}
		report.Dependencies[dependency.Name] = state
		if dependency.Importance == Required && state != Available {
			report.Ready = false
			report.Status = "not_ready"
		}
	}
	return report
}
