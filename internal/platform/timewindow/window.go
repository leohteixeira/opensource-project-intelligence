// Package timewindow defines the shared half-open UTC window contract.
package timewindow

import "time"

// Window includes From and excludes To.
type Window struct {
	From time.Time
	To   time.Time
}

// Contains reports whether instant belongs to [From, To).
func (w Window) Contains(instant time.Time) bool {
	return !instant.Before(w.From) && instant.Before(w.To)
}
