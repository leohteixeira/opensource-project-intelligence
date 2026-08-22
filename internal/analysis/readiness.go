package analysis

import "errors"

var (
	ErrDependenciesIncomplete = errors.New("required collection or metrics are incomplete")
	ErrInsufficientData       = errors.New("insufficient collected history")
)

// Readiness is the deterministic publication gate consumed by downstream
// intelligence. It prevents an incomplete or empty observation set from being
// represented as a current zero-valued result.
type Readiness struct {
	CollectionComplete bool
	MetricsComplete    bool
	ObservationCount   int
}

func (value Readiness) AuthorizePublication() error {
	if !value.CollectionComplete || !value.MetricsComplete {
		return ErrDependenciesIncomplete
	}
	if value.ObservationCount <= 0 {
		return ErrInsufficientData
	}
	return nil
}
