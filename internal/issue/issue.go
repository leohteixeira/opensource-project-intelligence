// Package issue owns provider-neutral issue metric cohorts.
package issue

import (
	"slices"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/timewindow"
)

type Response struct {
	At      time.Time
	ActorID string
	Public  bool
	Bot     bool
	Member  bool
}

type Issue struct {
	ID        int64
	OpenerID  string
	CreatedAt time.Time
	Responses []Response
}

// FirstResponse selects the first qualifying public member/collaborator response.
func FirstResponse(value Issue, cutoff time.Time) (time.Duration, bool) {
	responses := slices.Clone(value.Responses)
	slices.SortFunc(responses, func(left, right Response) int { return left.At.Compare(right.At) })
	for _, response := range responses {
		if response.At.After(cutoff) || !response.Public || response.Bot || !response.Member ||
			response.ActorID == value.OpenerID {
			continue
		}
		return response.At.Sub(value.CreatedAt), true
	}
	return 0, false
}

type StateEvent struct {
	IssueID int64
	At      time.Time
	State   string
}

// BacklogChange reconstructs open-at-end minus open-at-start from ordered state events.
func BacklogChange(events []StateEvent, window timewindow.Window) int {
	ordered := slices.Clone(events)
	slices.SortFunc(ordered, func(left, right StateEvent) int {
		if compared := left.At.Compare(right.At); compared != 0 {
			return compared
		}
		if left.IssueID < right.IssueID {
			return -1
		}
		if left.IssueID > right.IssueID {
			return 1
		}
		return 0
	})
	states := make(map[int64]string)
	start := 0
	for _, event := range ordered {
		if !event.At.Before(window.From) {
			break
		}
		states[event.IssueID] = event.State
	}
	for _, state := range states {
		if state == "open" {
			start++
		}
	}
	for _, event := range ordered {
		if event.At.Before(window.From) || !event.At.Before(window.To) {
			continue
		}
		states[event.IssueID] = event.State
	}
	end := 0
	for _, state := range states {
		if state == "open" {
			end++
		}
	}
	return end - start
}
