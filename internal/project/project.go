// Package project owns provider-neutral Project, Repository, Source, and association rules.
package project

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var (
	ErrInvalid          = errors.New("invalid project input")
	ErrConflict         = errors.New("project state conflict")
	ErrVersionConflict  = errors.New("project version conflict")
	ErrPermissionDenied = errors.New("project permission denied")
	ErrNotFound         = errors.New("project not found")
)

type State string

const (
	StateActive   State = "active"
	StatePaused   State = "paused"
	StateArchived State = "archived"
	StateDeleting State = "deleting"
	StateDeleted  State = "deleted"
)

type RepositoryRole string

const (
	RolePrimary       RepositoryRole = "primary"
	RoleCore          RepositoryRole = "core"
	RoleDocumentation RepositoryRole = "documentation"
	RoleExamples      RepositoryRole = "examples"
	RoleSDK           RepositoryRole = "sdk"
	RoleOther         RepositoryRole = "other"
)

type SourceKind string

const (
	SourceGitHub     SourceKind = "github"
	SourceGitLab     SourceKind = "gitlab"
	SourceGitea      SourceKind = "gitea"
	SourceGit        SourceKind = "git"
	SourceNPM        SourceKind = "npm"
	SourceNuGet      SourceKind = "nuget"
	SourcePyPI       SourceKind = "pypi"
	SourceDocs       SourceKind = "docs"
	SourceWebsite    SourceKind = "website"
	SourceChangelog  SourceKind = "changelog"
	SourceRSS        SourceKind = "rss"
	SourceDiscussion SourceKind = "discussion"
	SourceAdvisory   SourceKind = "advisory"
)

type SourceState string

const (
	SourceAvailable   SourceState = "available"
	SourceUnavailable SourceState = "unavailable"
	SourcePaused      SourceState = "paused"
	SourceRemoved     SourceState = "removed"
)

type Repository struct {
	ID           int64          `json:"id,string"`
	ProjectID    int64          `json:"project_id,string"`
	Provider     string         `json:"provider"`
	CanonicalURL string         `json:"url"`
	Role         RepositoryRole `json:"role"`
	Version      int64          `json:"version"`
}

type Source struct {
	ID            int64       `json:"id,string"`
	ProjectID     int64       `json:"project_id,string"`
	Kind          SourceKind  `json:"kind"`
	CanonicalURL  string      `json:"url"`
	State         SourceState `json:"state"`
	Public        bool        `json:"public"`
	CoverageFrom  *time.Time  `json:"coverage_from,omitempty"`
	CoverageTo    *time.Time  `json:"coverage_to,omitempty"`
	LastAttemptAt *time.Time  `json:"last_attempt_at,omitempty"`
	LastSuccessAt *time.Time  `json:"last_success_at,omitempty"`
	NextRunAt     *time.Time  `json:"next_run_at,omitempty"`
	Failure       string      `json:"failure,omitempty"`
	Version       int64       `json:"version"`
}

type Project struct {
	ID           int64        `json:"id,string"`
	WorkspaceID  int64        `json:"workspace_id,string"`
	Name         string       `json:"name"`
	Slug         string       `json:"slug"`
	Description  string       `json:"description"`
	State        State        `json:"state"`
	Version      int64        `json:"version"`
	Repositories []Repository `json:"repositories,omitempty"`
	Sources      []Source     `json:"sources,omitempty"`
	DeletedAt    *time.Time   `json:"deleted_at,omitempty"`
}

func New(id, workspaceID int64, name, slug string, primary Repository) (Project, error) {
	if id <= 0 || workspaceID <= 0 || strings.TrimSpace(name) == "" || strings.TrimSpace(slug) == "" {
		return Project{}, fmt.Errorf("%w: identity, name, and slug are required", ErrInvalid)
	}
	if primary.ID <= 0 || primary.ProjectID != id || primary.Role != RolePrimary ||
		strings.TrimSpace(primary.CanonicalURL) == "" {
		return Project{}, fmt.Errorf("%w: exactly one primary repository is required", ErrInvalid)
	}
	primary.Version = max(primary.Version, 1)
	return Project{
		ID: id, WorkspaceID: workspaceID, Name: strings.TrimSpace(name), Slug: strings.TrimSpace(slug),
		State: StateActive, Version: 1, Repositories: []Repository{primary},
	}, nil
}

func (p Project) Validate() error {
	if p.ID <= 0 || p.WorkspaceID <= 0 || strings.TrimSpace(p.Name) == "" ||
		strings.TrimSpace(p.Slug) == "" || p.Version <= 0 || !validState(p.State) {
		return fmt.Errorf("%w: malformed project", ErrInvalid)
	}
	primary := 0
	seen := make(map[string]struct{}, len(p.Repositories))
	for _, repository := range p.Repositories {
		if repository.ID <= 0 || repository.ProjectID != p.ID || !validRole(repository.Role) ||
			strings.TrimSpace(repository.CanonicalURL) == "" {
			return fmt.Errorf("%w: malformed repository", ErrInvalid)
		}
		canonical := strings.ToLower(repository.CanonicalURL)
		if _, exists := seen[canonical]; exists {
			return fmt.Errorf("%w: duplicate canonical repository URL", ErrConflict)
		}
		seen[canonical] = struct{}{}
		if repository.Role == RolePrimary {
			primary++
		}
	}
	if p.State != StateDeleted && primary != 1 {
		return fmt.Errorf("%w: project must have exactly one primary repository", ErrConflict)
	}
	return nil
}

func (p Project) CanSynchronize() error {
	if p.State != StateActive {
		return fmt.Errorf("%w: %s projects cannot synchronize", ErrConflict, p.State)
	}
	return nil
}

func (p Project) Transition(to State, expectedVersion int64, admin bool) (Project, error) {
	if !admin {
		return Project{}, ErrPermissionDenied
	}
	if expectedVersion != p.Version {
		return Project{}, ErrVersionConflict
	}
	if p.State == to && (to == StatePaused || to == StateArchived) {
		return p, nil
	}
	allowed := map[State][]State{
		StateActive:   {StatePaused, StateArchived, StateDeleting},
		StatePaused:   {StateActive, StateArchived, StateDeleting},
		StateArchived: {StateActive, StatePaused, StateDeleting},
		StateDeleting: {StateDeleted},
	}
	if !slices.Contains(allowed[p.State], to) {
		return Project{}, fmt.Errorf("%w: transition %s to %s", ErrConflict, p.State, to)
	}
	p.State = to
	p.Version++
	if to == StateDeleted {
		now := time.Now().UTC()
		p.DeletedAt = &now
		p.Repositories = nil
		p.Sources = nil
	}
	return p, nil
}

func (p Project) AddRepository(repository Repository, limit int) (Project, error) {
	if p.State == StateArchived || p.State == StateDeleting || p.State == StateDeleted {
		return Project{}, fmt.Errorf("%w: project is read-only", ErrConflict)
	}
	if limit > 0 && len(p.Repositories) >= limit {
		return Project{}, fmt.Errorf("%w: repository limit reached", ErrConflict)
	}
	if repository.ID <= 0 || repository.ProjectID != p.ID || !validRole(repository.Role) ||
		strings.TrimSpace(repository.CanonicalURL) == "" {
		return Project{}, fmt.Errorf("%w: malformed repository", ErrInvalid)
	}
	for _, current := range p.Repositories {
		if strings.EqualFold(current.CanonicalURL, repository.CanonicalURL) {
			return p, nil
		}
	}
	if repository.Role == RolePrimary {
		for index := range p.Repositories {
			if p.Repositories[index].Role == RolePrimary {
				p.Repositories[index].Role = RoleCore
				p.Repositories[index].Version++
			}
		}
	}
	repository.Version = max(repository.Version, 1)
	p.Repositories = append(p.Repositories, repository)
	p.Version++
	return p, p.Validate()
}

func (p Project) ChangeRepositoryRole(repositoryID int64, role RepositoryRole) (Project, error) {
	if p.State == StateArchived || p.State == StateDeleting || p.State == StateDeleted {
		return Project{}, fmt.Errorf("%w: project is read-only", ErrConflict)
	}
	if !validRole(role) {
		return Project{}, fmt.Errorf("%w: unsupported repository role", ErrInvalid)
	}
	index := slices.IndexFunc(p.Repositories, func(repository Repository) bool { return repository.ID == repositoryID })
	if index < 0 {
		return Project{}, ErrNotFound
	}
	if p.Repositories[index].Role == role {
		return p, nil
	}
	if role == RolePrimary {
		for current := range p.Repositories {
			if p.Repositories[current].Role == RolePrimary {
				p.Repositories[current].Role = RoleCore
				p.Repositories[current].Version++
			}
		}
	} else if p.Repositories[index].Role == RolePrimary {
		return Project{}, fmt.Errorf("%w: replacement primary is required", ErrConflict)
	}
	p.Repositories[index].Role = role
	p.Repositories[index].Version++
	p.Version++
	return p, p.Validate()
}

func (p Project) RemoveRepository(repositoryID int64) (Project, error) {
	if p.State == StateArchived || p.State == StateDeleting || p.State == StateDeleted {
		return Project{}, fmt.Errorf("%w: project is read-only", ErrConflict)
	}
	index := slices.IndexFunc(p.Repositories, func(repository Repository) bool { return repository.ID == repositoryID })
	if index < 0 {
		return p, nil
	}
	if p.Repositories[index].Role == RolePrimary {
		return Project{}, fmt.Errorf("%w: the primary repository cannot be removed", ErrConflict)
	}
	p.Repositories = slices.Delete(p.Repositories, index, index+1)
	p.Version++
	return p, p.Validate()
}

type AssociationStatus string

const (
	AssociationLinked     AssociationStatus = "linked"
	AssociationUnresolved AssociationStatus = "unresolved"
	AssociationCorrected  AssociationStatus = "corrected"
)

type Association struct {
	ID              int64             `json:"id,string"`
	SourceID        int64             `json:"source_id,string"`
	ProjectID       int64             `json:"project_id,string"`
	Method          string            `json:"method"`
	Confidence      float64           `json:"confidence"`
	Evidence        []int64           `json:"evidence_ids"`
	DecisionVersion string            `json:"decision_version"`
	Status          AssociationStatus `json:"status"`
	Constraint      *Correction       `json:"correction,omitempty"`
	Version         int64             `json:"version"`
}

type Correction struct {
	Action    string    `json:"action"`
	ProjectID int64     `json:"project_id,string,omitempty"`
	Reason    string    `json:"reason"`
	ActorID   int64     `json:"actor_id,string"`
	At        time.Time `json:"at"`
}

func (association Association) Correct(correction Correction) (Association, bool, error) {
	if association.ID <= 0 || correction.ActorID <= 0 || strings.TrimSpace(correction.Reason) == "" ||
		!slices.Contains([]string{"split", "reassign", "confirm"}, correction.Action) {
		return Association{}, false, fmt.Errorf("%w: invalid association correction", ErrInvalid)
	}
	if correction.Action == "reassign" && correction.ProjectID <= 0 {
		return Association{}, false, fmt.Errorf("%w: reassignment target is required", ErrInvalid)
	}
	if association.Constraint != nil && association.Constraint.Action == correction.Action &&
		association.Constraint.ProjectID == correction.ProjectID {
		return association, false, nil
	}
	correction.At = correction.At.UTC()
	association.Constraint = &correction
	association.Status = AssociationCorrected
	association.Version++
	if correction.Action == "reassign" {
		association.ProjectID = correction.ProjectID
	}
	return association, true, nil
}

func validState(state State) bool {
	return slices.Contains([]State{StateActive, StatePaused, StateArchived, StateDeleting, StateDeleted}, state)
}

func validRole(role RepositoryRole) bool {
	return slices.Contains([]RepositoryRole{
		RolePrimary, RoleCore, RoleDocumentation, RoleExamples, RoleSDK, RoleOther,
	}, role)
}

func ValidSourceKind(kind SourceKind) bool {
	return slices.Contains([]SourceKind{
		SourceGitHub, SourceGitLab, SourceGitea, SourceGit, SourceNPM, SourceNuGet, SourcePyPI,
		SourceDocs, SourceWebsite, SourceChangelog, SourceRSS, SourceDiscussion, SourceAdvisory,
	}, kind)
}
