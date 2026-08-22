package access

import (
	"errors"
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
		return errors.New("locale must be en or pt-BR")
	}
	if strings.TrimSpace(timezone) == "" {
		return errors.New("timezone is required")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return fmt.Errorf("timezone is not supported: %w", err)
	}
	return nil
}

func ValidateDeletionConfirmation(value string) error {
	if value != DeletionConfirmation {
		return errors.New("account deletion confirmation does not match")
	}
	return nil
}

func ValidateServiceAccount(account ServiceAccount, allowedScopes map[string]struct{}) error {
	if err := (IdentityKey{Issuer: account.Issuer, Subject: account.ExternalSubject}).Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(account.Name) == "" {
		return errors.New("service account name is required")
	}
	if err := ValidateRole(account.Role, true); err != nil {
		return err
	}
	if account.Status != StatusActive && account.Status != StatusSuspended {
		return errors.New("service account status must be active or suspended")
	}
	seen := make(map[string]struct{}, len(account.Scopes))
	for _, scope := range account.Scopes {
		if _, ok := allowedScopes[scope]; !ok {
			return fmt.Errorf("service account scope %q is not allowed", scope)
		}
		if _, ok := seen[scope]; ok {
			return fmt.Errorf("service account scope %q is duplicated", scope)
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
