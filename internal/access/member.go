package access

import (
	"fmt"
	"strings"
	"time"
)

const DeletionConfirmation = "DELETE MY ACCOUNT"

type Member struct {
	ID          int64     `json:"id,string"`
	IdentityID  int64     `json:"-"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	Role        Role      `json:"role"`
	Status      Status    `json:"status"`
	Locale      string    `json:"locale"`
	Timezone    string    `json:"timezone"`
	Version     int64     `json:"version"`
	RequestedAt time.Time `json:"requested_at"`
}

type ServiceAccount struct {
	ID              int64    `json:"id,string"`
	Issuer          string   `json:"issuer"`
	ExternalSubject string   `json:"external_subject"`
	Name            string   `json:"name"`
	Role            Role     `json:"role"`
	Status          Status   `json:"status"`
	Scopes          []string `json:"scopes"`
	Version         int64    `json:"version"`
}

func ValidatePreferences(locale, timezone string) error {
	if locale != "en" && locale != "pt-BR" {
		return fmt.Errorf("%w: locale must be en or pt-BR", ErrInvalidInput)
	}
	if strings.TrimSpace(timezone) == "" {
		return fmt.Errorf("%w: timezone is required", ErrInvalidInput)
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return fmt.Errorf("%w: timezone is not supported: %v", ErrInvalidInput, err)
	}
	return nil
}

func ValidateDeletionConfirmation(value string) error {
	if value != DeletionConfirmation {
		return fmt.Errorf("%w: account deletion confirmation does not match", ErrInvalidInput)
	}
	return nil
}

func ValidateServiceAccount(account ServiceAccount, allowedScopes map[string]struct{}) error {
	if err := (IdentityKey{Issuer: account.Issuer, Subject: account.ExternalSubject}).Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(account.Name) == "" {
		return fmt.Errorf("%w: service account name is required", ErrInvalidInput)
	}
	if err := ValidateRole(account.Role, true); err != nil {
		return err
	}
	if account.Status != StatusActive && account.Status != StatusSuspended {
		return fmt.Errorf("%w: service account status must be active or suspended", ErrInvalidInput)
	}
	seen := make(map[string]struct{}, len(account.Scopes))
	for _, scope := range account.Scopes {
		if _, ok := allowedScopes[scope]; !ok {
			return fmt.Errorf("%w: service account scope %q is not allowed", ErrInvalidInput, scope)
		}
		if _, ok := seen[scope]; ok {
			return fmt.Errorf("%w: service account scope %q is duplicated", ErrInvalidInput, scope)
		}
		seen[scope] = struct{}{}
	}
	return nil
}

func ProtectLastAdmin(target Member, activeAdminCount int, nextRole Role, nextStatus Status) error {
	removesAdmin := target.Role == RoleAdmin && target.Status == StatusActive &&
		(nextRole != RoleAdmin || nextStatus != StatusActive)
	if removesAdmin && activeAdminCount <= 1 {
		return ErrLastAdminRequired
	}
	return nil
}
