package release

import (
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/timewindow"
)

// Release is the provider-neutral evidence used by release metrics.
type Release struct {
	ID          int64
	PublishedAt time.Time
	Draft       bool
	Prerelease  bool
}

// StableInWindow returns stable releases in deterministic evidence order.
func StableInWindow(values []Release, window timewindow.Window) []Release {
	stable := make([]Release, 0, len(values))
	for _, value := range values {
		if !value.Draft && !value.Prerelease && window.Contains(value.PublishedAt.UTC()) {
			stable = append(stable, value)
		}
	}
	return stable
}
