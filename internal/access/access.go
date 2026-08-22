// Package access owns local membership, roles, service scopes, sessions, and authorization.
// External identity claims establish identity only; they never grant product access.
package access

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

type Role string

const (
	RoleViewer  Role = "viewer"
	RoleAnalyst Role = "analyst"
	RoleAdmin   Role = "admin"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusActive    Status = "active"
	StatusRejected  Status = "rejected"
	StatusSuspended Status = "suspended"
	StatusDeleted   Status = "deleted"
)

type ActorKind string

const (
	ActorMember         ActorKind = "member"
	ActorServiceAccount ActorKind = "service_account"
	ActorSystem         ActorKind = "system"
	ActorDeleted        ActorKind = "deleted_actor"
)

type Action string

const (
	ActionCatalogRead      Action = "catalog:read"
	ActionIntelligenceRead Action = "intelligence:read"
	ActionProjectWrite     Action = "projects:write"
	ActionExportWrite      Action = "exports:write"
	ActionMembershipGovern Action = "members:govern"
	ActionServiceGovern    Action = "service_accounts:govern"
	ActionAuditRead        Action = "audit:read"
	ActionOperationsRead   Action = "operations:read"
	ActionProjectLifecycle Action = "projects:lifecycle"
	ActionPolicyGovern     Action = "policies:govern"
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrAccessPending          = errors.New("access pending")
	ErrPermissionDenied       = errors.New("permission denied")
	ErrLastAdminRequired      = errors.New("last active admin is required")
	ErrVersionConflict        = errors.New("membership version conflict")
	ErrNotFound               = errors.New("resource not found")
)

type IdentityKey struct {
	Issuer  string
	Subject string
}

func (k IdentityKey) Validate() error {
	issuer, err := url.ParseRequestURI(k.Issuer)
	if err != nil || issuer.Scheme != "https" || issuer.Host == "" {
		return errors.New("identity issuer must be an absolute HTTPS URL")
	}
	if strings.TrimSpace(k.Subject) == "" || len(k.Subject) > 255 {
		return errors.New("identity subject is required and must not exceed 255 characters")
	}
	return nil
}

type Principal struct {
	ActorID   int64
	Kind      ActorKind
	Role      Role
	Status    Status
	Scopes    []string
	Version   int64
	Workspace int64
}

func (p Principal) IsApproved() bool {
	return p.ActorID > 0 && p.Status == StatusActive && validRole(p.Role)
}

func Authorize(p Principal, action Action) error {
	if action == ActionCatalogRead {
		return nil
	}
	if p.ActorID == 0 {
		return ErrAuthenticationRequired
	}
	if p.Status == StatusPending {
		return ErrAccessPending
	}
	if !p.IsApproved() {
		return ErrPermissionDenied
	}

	minimum, known := minimumRole(action)
	if !known || roleRank(p.Role) < roleRank(minimum) {
		return ErrPermissionDenied
	}
	if p.Kind == ActorServiceAccount {
		if p.Role == RoleAdmin || action == ActionMembershipGovern || action == ActionServiceGovern ||
			action == ActionAuditRead || action == ActionOperationsRead ||
			action == ActionProjectLifecycle || action == ActionPolicyGovern {
			return ErrPermissionDenied
		}
		if scope := requiredScope(action); scope != "" && !slices.Contains(p.Scopes, scope) {
			return ErrPermissionDenied
		}
	}
	return nil
}

func ValidateRole(role Role, service bool) error {
	if !validRole(role) || service && role == RoleAdmin {
		return fmt.Errorf("invalid role %q", role)
	}
	return nil
}

func validRole(role Role) bool {
	return role == RoleViewer || role == RoleAnalyst || role == RoleAdmin
}

func roleRank(role Role) int {
	switch role {
	case RoleViewer:
		return 1
	case RoleAnalyst:
		return 2
	case RoleAdmin:
		return 3
	default:
		return 0
	}
}

func minimumRole(action Action) (Role, bool) {
	switch action {
	case ActionIntelligenceRead:
		return RoleViewer, true
	case ActionProjectWrite, ActionExportWrite:
		return RoleAnalyst, true
	case ActionMembershipGovern, ActionServiceGovern, ActionAuditRead, ActionOperationsRead,
		ActionProjectLifecycle, ActionPolicyGovern:
		return RoleAdmin, true
	default:
		return "", false
	}
}

func requiredScope(action Action) string {
	switch action {
	case ActionIntelligenceRead:
		return "projects:read"
	case ActionProjectWrite:
		return "projects:write"
	case ActionExportWrite:
		return "exports:write"
	default:
		return ""
	}
}
