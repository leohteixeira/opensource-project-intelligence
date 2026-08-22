// Package pullrequest owns provider-neutral pull-request metric cohorts.
package pullrequest

import "time"

type PullRequest struct {
	CreatedAt time.Time
	ReadyAt   *time.Time
	MergedAt  time.Time
}

// MergeDuration measures readiness-to-merge and exposes the created-time fallback.
func MergeDuration(value PullRequest) (duration time.Duration, usedFallback bool) {
	start := value.CreatedAt
	usedFallback = true
	if value.ReadyAt != nil {
		start = *value.ReadyAt
		usedFallback = false
	}
	return value.MergedAt.Sub(start), usedFallback
}
