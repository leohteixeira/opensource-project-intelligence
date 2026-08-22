package analysis_test

import (
	"errors"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
)

func TestTask03PublicationReadiness(t *testing.T) {
	t.Parallel()

	t.Run("UT-069 incomplete dependencies cannot publish as current", func(t *testing.T) {
		for _, value := range []analysis.Readiness{
			{MetricsComplete: true, ObservationCount: 1},
			{CollectionComplete: true, ObservationCount: 1},
		} {
			if !errors.Is(value.AuthorizePublication(), analysis.ErrDependenciesIncomplete) {
				t.Fatalf("readiness %#v was publishable", value)
			}
		}
	})

	t.Run("UT-072 empty history is insufficient data rather than zero activity", func(t *testing.T) {
		value := analysis.Readiness{CollectionComplete: true, MetricsComplete: true}
		if !errors.Is(value.AuthorizePublication(), analysis.ErrInsufficientData) {
			t.Fatalf("empty history returned %v", value.AuthorizePublication())
		}
		value.ObservationCount = 1
		if err := value.AuthorizePublication(); err != nil {
			t.Fatalf("complete non-empty history returned %v", err)
		}
	})
}
