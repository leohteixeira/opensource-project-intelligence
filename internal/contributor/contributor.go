// Package contributor owns contributors, verified identity resolution, and sustainability metrics.
package contributor

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/timewindow"
)

var ErrInvalid = errors.New("invalid contributor input")

type LinkStatus string

const (
	LinkUnresolved       LinkStatus = "unresolved"
	LinkVerified         LinkStatus = "verified"
	LinkAnalystConfirmed LinkStatus = "analyst_confirmed"
)

type Commit struct {
	ID            int64
	AccountID     string
	IdentityID    string
	LinkStatus    LinkStatus
	CommittedAt   time.Time
	DefaultBranch bool
	MergeCommit   bool
	Bot           bool
}

type Contributor struct {
	Key     string     `json:"key"`
	Commits int        `json:"commits"`
	Status  LinkStatus `json:"identity_status"`
}

type Summary struct {
	Status             string        `json:"status"`
	Active             int           `json:"active"`
	TopOneShare        *float64      `json:"top_one_share,omitempty"`
	TopThreeShare      *float64      `json:"top_three_share,omitempty"`
	ResolutionCoverage float64       `json:"resolution_coverage"`
	Contributors       []Contributor `json:"contributors"`
}

func (c Commit) eligible(window timewindow.Window) bool {
	return c.DefaultBranch && !c.MergeCommit && !c.Bot && window.Contains(c.CommittedAt.UTC())
}

func identityKey(value Commit) (string, LinkStatus) {
	if (value.LinkStatus == LinkVerified || value.LinkStatus == LinkAnalystConfirmed) &&
		strings.TrimSpace(value.IdentityID) != "" {
		return "identity:" + value.IdentityID, value.LinkStatus
	}
	return "account:" + value.AccountID, LinkUnresolved
}

// Aggregate computes concentration without merging unverified accounts.
func Aggregate(commits []Commit, window timewindow.Window) Summary {
	counts := make(map[string]int)
	statuses := make(map[string]LinkStatus)
	eligible := 0
	resolved := 0
	for _, commit := range commits {
		if !commit.eligible(window) || strings.TrimSpace(commit.AccountID) == "" {
			continue
		}
		key, status := identityKey(commit)
		counts[key]++
		statuses[key] = status
		eligible++
		if status != LinkUnresolved {
			resolved++
		}
	}
	result := Summary{Status: "insufficient_data", Contributors: make([]Contributor, 0, len(counts))}
	if eligible == 0 {
		return result
	}
	for key, count := range counts {
		result.Contributors = append(result.Contributors, Contributor{Key: key, Commits: count, Status: statuses[key]})
	}
	slices.SortFunc(result.Contributors, func(left, right Contributor) int {
		if left.Commits != right.Commits {
			return right.Commits - left.Commits
		}
		return strings.Compare(left.Key, right.Key)
	})
	result.Status = "available"
	result.Active = len(result.Contributors)
	result.ResolutionCoverage = float64(resolved) / float64(eligible)
	topOne := float64(result.Contributors[0].Commits) / float64(eligible)
	topThreeCount := 0
	for index := 0; index < min(3, len(result.Contributors)); index++ {
		topThreeCount += result.Contributors[index].Commits
	}
	topThree := float64(topThreeCount) / float64(eligible)
	result.TopOneShare = &topOne
	result.TopThreeShare = &topThree
	return result
}

func ValidateStatus(status LinkStatus) error {
	if status != LinkVerified && status != LinkAnalystConfirmed && status != LinkUnresolved {
		return fmt.Errorf("%w: unsupported link status", ErrInvalid)
	}
	return nil
}
