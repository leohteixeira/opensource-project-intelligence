// Package topic builds deterministic topic candidates and applies immutable analyst constraints.
package topic

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var ErrInvalid = errors.New("topic: invalid input")

type Neighbor struct {
	IssueID    int64   `json:"issue_id,string"`
	NeighborID int64   `json:"neighbor_id,string"`
	Rank       int     `json:"rank"`
	Similarity float64 `json:"similarity"`
}

type Candidate struct {
	ID               int64      `json:"id,string"`
	ProjectID        int64      `json:"project_id,string"`
	Members          []int64    `json:"members"`
	AlgorithmVersion string     `json:"algorithm_version"`
	GeneratedLabel   string     `json:"generated_label"`
	LabelRunID       *int64     `json:"label_run_id,omitempty,string"`
	CreatedAt        time.Time  `json:"created_at"`
	RetiredAt        *time.Time `json:"retired_at,omitempty"`
}

// Candidates builds connected components from mutual top-k relationships only.
func Candidates(projectID int64, neighbors []Neighbor, k int, version string) ([]Candidate, error) {
	if projectID <= 0 || k < 1 || k > 100 || strings.TrimSpace(version) == "" {
		return nil, ErrInvalid
	}
	directed := make(map[[2]int64]struct{})
	nodes := make(map[int64]struct{})
	for _, neighbor := range neighbors {
		if neighbor.IssueID <= 0 || neighbor.NeighborID <= 0 || neighbor.IssueID == neighbor.NeighborID ||
			neighbor.Rank < 1 || neighbor.Rank > k || neighbor.Similarity < 0 || neighbor.Similarity > 1 {
			return nil, ErrInvalid
		}
		directed[[2]int64{neighbor.IssueID, neighbor.NeighborID}] = struct{}{}
		nodes[neighbor.IssueID] = struct{}{}
		nodes[neighbor.NeighborID] = struct{}{}
	}
	adjacency := make(map[int64][]int64, len(nodes))
	for edge := range directed {
		if _, mutual := directed[[2]int64{edge[1], edge[0]}]; !mutual {
			continue
		}
		adjacency[edge[0]] = append(adjacency[edge[0]], edge[1])
	}
	ordered := make([]int64, 0, len(nodes))
	for node := range nodes {
		ordered = append(ordered, node)
	}
	slices.Sort(ordered)
	visited := make(map[int64]bool, len(nodes))
	values := make([]Candidate, 0)
	for _, root := range ordered {
		if visited[root] || len(adjacency[root]) == 0 {
			continue
		}
		stack, members := []int64{root}, make([]int64, 0)
		visited[root] = true
		for len(stack) > 0 {
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			members = append(members, node)
			next := slices.Clone(adjacency[node])
			slices.Sort(next)
			for _, neighbor := range next {
				if !visited[neighbor] {
					visited[neighbor] = true
					stack = append(stack, neighbor)
				}
			}
		}
		slices.Sort(members)
		values = append(values, Candidate{ProjectID: projectID, Members: members, AlgorithmVersion: version})
	}
	slices.SortFunc(values, func(left, right Candidate) int { return compareMembers(left.Members, right.Members) })
	return values, nil
}

func compareMembers(left, right []int64) int {
	for index := range min(len(left), len(right)) {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return 0
}

type Action string

const (
	ActionRename   Action = "rename"
	ActionInclude  Action = "include"
	ActionExclude  Action = "exclude"
	ActionMerge    Action = "merge"
	ActionSplit    Action = "split"
	ActionReassign Action = "reassign"
)

type Correction struct {
	ID            int64     `json:"id,string"`
	ProjectID     int64     `json:"project_id,string"`
	TopicID       int64     `json:"topic_id,string"`
	Action        Action    `json:"action"`
	IssueIDs      []int64   `json:"issue_ids"`
	OtherTopicIDs []int64   `json:"other_topic_ids"`
	Label         string    `json:"label"`
	Reason        string    `json:"reason"`
	ActorID       int64     `json:"actor_id,string"`
	RequestID     string    `json:"request_id"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
}

func (value Correction) Validate() error {
	if value.ID <= 0 || value.ProjectID <= 0 || value.TopicID <= 0 || value.ActorID <= 0 ||
		value.Version <= 0 || strings.TrimSpace(value.RequestID) == "" ||
		strings.TrimSpace(value.Reason) == "" || value.CreatedAt.IsZero() {
		return ErrInvalid
	}
	switch value.Action {
	case ActionRename:
		if strings.TrimSpace(value.Label) == "" {
			return ErrInvalid
		}
	case ActionInclude, ActionExclude, ActionSplit, ActionReassign:
		if len(value.IssueIDs) == 0 {
			return ErrInvalid
		}
	case ActionMerge:
		if len(value.OtherTopicIDs) == 0 || slices.Contains(value.OtherTopicIDs, value.TopicID) {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

type Canonical struct {
	Candidate Candidate    `json:"candidate"`
	Label     string       `json:"label"`
	Members   []int64      `json:"members"`
	History   []Correction `json:"history"`
}

// Apply preserves the generated candidate and correction history while producing the canonical
// current projection.
func Apply(candidate Candidate, corrections []Correction) (Canonical, error) {
	current := Canonical{Candidate: candidate, Label: candidate.GeneratedLabel,
		Members: slices.Clone(candidate.Members), History: slices.Clone(corrections)}
	seenRequests := make(map[string]struct{}, len(corrections))
	lastVersion := int64(0)
	for _, correction := range corrections {
		if err := correction.Validate(); err != nil || correction.TopicID != candidate.ID ||
			correction.Version <= lastVersion {
			return Canonical{}, ErrInvalid
		}
		if _, exists := seenRequests[correction.RequestID]; exists {
			return Canonical{}, ErrInvalid
		}
		seenRequests[correction.RequestID] = struct{}{}
		lastVersion = correction.Version
		switch correction.Action {
		case ActionRename:
			current.Label = correction.Label
		case ActionInclude:
			for _, id := range correction.IssueIDs {
				if !slices.Contains(current.Members, id) {
					current.Members = append(current.Members, id)
				}
			}
		case ActionExclude, ActionSplit, ActionReassign:
			current.Members = slices.DeleteFunc(current.Members, func(id int64) bool {
				return slices.Contains(correction.IssueIDs, id)
			})
		case ActionMerge:
			// Membership from the other immutable candidate is resolved by the application service.
		default:
			return Canonical{}, fmt.Errorf("%w: unknown correction", ErrInvalid)
		}
	}
	slices.Sort(current.Members)
	current.Members = slices.Compact(current.Members)
	return current, nil
}
