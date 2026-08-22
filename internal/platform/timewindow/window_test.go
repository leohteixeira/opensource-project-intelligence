package timewindow_test

import (
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/timewindow"
)

func TestUT228WindowIsHalfOpen(t *testing.T) {
	t.Parallel()

	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	window := timewindow.Window{From: from, To: to}

	tests := map[string]struct {
		instant time.Time
		want    bool
	}{
		"at from":     {instant: from, want: true},
		"inside":      {instant: from.Add(time.Hour), want: true},
		"at to":       {instant: to, want: false},
		"before from": {instant: from.Add(-time.Nanosecond), want: false},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := window.Contains(tc.instant); got != tc.want {
				t.Errorf("Contains(%s) = %t, want %t", tc.instant, got, tc.want)
			}
		})
	}
}
