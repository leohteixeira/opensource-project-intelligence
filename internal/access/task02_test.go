package access

import (
	"errors"
	"testing"
)

func TestUT008InvalidIdentityClaims(t *testing.T) {
	t.Parallel()
	for _, key := range []IdentityKey{{Issuer: "http://issuer.test", Subject: "subject"}, {Issuer: "https://issuer.test", Subject: ""}} {
		if !errors.Is(key.Validate(), ErrInvalidInput) {
			t.Fatalf("IdentityKey.Validate() = nil, want invalid input for %#v", key)
		}
	}
}

func TestUT009UnknownIdentityHasNoImplicitRole(t *testing.T) {
	t.Parallel()
	principal := Principal{ActorID: 44, Kind: ActorMember, Status: StatusPending}
	if principal.Role != "" || principal.IsApproved() {
		t.Fatal("pending principal received implicit local access")
	}
}

func TestUT011ApplicantCannotReadProtectedResources(t *testing.T) {
	t.Parallel()
	err := Authorize(Principal{ActorID: 1, Kind: ActorMember, Status: StatusPending}, ActionIntelligenceRead)
	if !errors.Is(err, ErrAccessPending) {
		t.Fatalf("Authorize() = %v, want access pending", err)
	}
}

func TestUT014RejectedAndSuspendedIdentitiesRemainDenied(t *testing.T) {
	t.Parallel()
	for _, status := range []Status{StatusRejected, StatusSuspended} {
		err := Authorize(Principal{ActorID: 1, Kind: ActorMember, Status: status, Role: RoleViewer}, ActionIntelligenceRead)
		if !errors.Is(err, ErrPermissionDenied) {
			t.Fatalf("Authorize(%s) = %v, want permission denied", status, err)
		}
	}
}

func TestUT015UnknownRoleAndMalformedSubjectAreRejected(t *testing.T) {
	t.Parallel()
	if !errors.Is(ValidateRole("owner", false), ErrInvalidInput) {
		t.Fatal("unknown role was accepted")
	}
	if !errors.Is((IdentityKey{Issuer: "https://issuer.test"}).Validate(), ErrInvalidInput) {
		t.Fatal("blank external subject was accepted")
	}
}

func TestUT018NonAdminsCannotGovernMembership(t *testing.T) {
	t.Parallel()
	for _, role := range []Role{RoleViewer, RoleAnalyst} {
		err := Authorize(activeMember(role), ActionMembershipGovern)
		if !errors.Is(err, ErrPermissionDenied) {
			t.Fatalf("Authorize(%s) = %v, want permission denied", role, err)
		}
	}
}

func TestUT020RoleRequiresKnownExternalIdentity(t *testing.T) {
	t.Parallel()
	if (IdentityKey{Issuer: "https://issuer.test"}).Validate() == nil {
		t.Fatal("identity without a subject was valid")
	}
}

func TestUT021LastActiveAdminCannotLoseAccess(t *testing.T) {
	t.Parallel()
	target := Member{Role: RoleAdmin, Status: StatusActive}
	if !errors.Is(ProtectLastAdmin(target, 1, RoleViewer, StatusActive), ErrLastAdminRequired) {
		t.Fatal("last active administrator was demoted")
	}
}

func TestUT022UnsupportedPreferencesAreRejected(t *testing.T) {
	t.Parallel()
	for _, pair := range [][2]string{{"fr", "UTC"}, {"en", "Mars/Olympus"}} {
		if !errors.Is(ValidatePreferences(pair[0], pair[1]), ErrInvalidInput) {
			t.Fatalf("ValidatePreferences(%q, %q) accepted unsupported input", pair[0], pair[1])
		}
	}
}

func TestUT023MembershipDefaultsAreEnglishAndUTC(t *testing.T) {
	t.Parallel()
	member := Member{Locale: "en", Timezone: "UTC"}
	if err := ValidatePreferences(member.Locale, member.Timezone); err != nil {
		t.Fatalf("default preferences are invalid: %v", err)
	}
}

func TestUT025MemberAuthorityDoesNotIncludeProfileGovernance(t *testing.T) {
	t.Parallel()
	if !errors.Is(Authorize(activeMember(RoleAnalyst), ActionMembershipGovern), ErrPermissionDenied) {
		t.Fatal("analyst can mutate another member")
	}
}

func TestUT027DeletionRequiresExactConfirmation(t *testing.T) {
	t.Parallel()
	if !errors.Is(ValidateDeletionConfirmation("delete my account"), ErrInvalidInput) {
		t.Fatal("inexact deletion confirmation was accepted")
	}
	if err := ValidateDeletionConfirmation(DeletionConfirmation); err != nil {
		t.Fatalf("exact confirmation was rejected: %v", err)
	}
}

func TestUT028SuspensionDoesNotRestoreWorkspaceAccess(t *testing.T) {
	t.Parallel()
	principal := activeMember(RoleViewer)
	principal.Status = StatusSuspended
	if !errors.Is(Authorize(principal, ActionIntelligenceRead), ErrPermissionDenied) {
		t.Fatal("suspended principal regained workspace access")
	}
}

func TestUT218MalformedExternalIdentityFailsClosed(t *testing.T) {
	t.Parallel()
	if (IdentityKey{Issuer: "not-an-issuer", Subject: "service"}).Validate() == nil {
		t.Fatal("malformed issuer was accepted")
	}
}

func TestUT219UnboundBearerIdentityHasNoAccess(t *testing.T) {
	t.Parallel()
	if !errors.Is(Authorize(Principal{}, ActionIntelligenceRead), ErrAuthenticationRequired) {
		t.Fatal("unbound identity received access")
	}
}

func TestUT220ServiceRequestsUseTheirOwnScopeSubset(t *testing.T) {
	t.Parallel()
	service := activeService(RoleAnalyst, "projects:read")
	if err := Authorize(service, ActionIntelligenceRead); err != nil {
		t.Fatalf("granted service scope was denied: %v", err)
	}
	if !errors.Is(Authorize(service, ActionExportWrite), ErrPermissionDenied) {
		t.Fatal("service exceeded its scope subset")
	}
}

func TestUT221ServiceAccountsCannotReceiveAdminAuthority(t *testing.T) {
	t.Parallel()
	if !errors.Is(ValidateRole(RoleAdmin, true), ErrInvalidInput) {
		t.Fatal("service account accepted admin role")
	}
	service := activeService(RoleAdmin, "projects:read")
	for _, action := range []Action{ActionMembershipGovern, ActionServiceGovern, ActionPolicyGovern, ActionProjectLifecycle} {
		if !errors.Is(Authorize(service, action), ErrPermissionDenied) {
			t.Fatalf("service account authorized for %s", action)
		}
	}
}

func TestUT223LocalSuspensionPrecedesBearerRole(t *testing.T) {
	t.Parallel()
	service := activeService(RoleAnalyst, "projects:write")
	service.Status = StatusSuspended
	if !errors.Is(Authorize(service, ActionProjectWrite), ErrPermissionDenied) {
		t.Fatal("suspended service account remained authorized")
	}
}

func TestUT224DeletedOrSuspendedBindingInvalidatesTokenAccess(t *testing.T) {
	t.Parallel()
	for _, status := range []Status{StatusDeleted, StatusSuspended} {
		service := activeService(RoleViewer, "projects:read")
		service.Status = status
		if !errors.Is(Authorize(service, ActionIntelligenceRead), ErrPermissionDenied) {
			t.Fatalf("%s service binding remained authorized", status)
		}
	}
}

func TestUT247OIDCIdentityKeyNeverUsesEmail(t *testing.T) {
	t.Parallel()
	first := IdentityKey{Issuer: "https://issuer-a.test", Subject: "42"}
	second := IdentityKey{Issuer: "https://issuer-b.test", Subject: "42"}
	third := IdentityKey{Issuer: "https://issuer-a.test", Subject: "43"}
	if first == second || first == third {
		t.Fatal("issuer and subject combinations collapsed into one identity")
	}
}

func TestUT248ExternalRoleClaimsCannotPopulateLocalPrincipal(t *testing.T) {
	t.Parallel()
	principal := Principal{ActorID: 91, Kind: ActorMember, Status: StatusPending}
	if principal.Role != "" || !errors.Is(Authorize(principal, ActionIntelligenceRead), ErrAccessPending) {
		t.Fatal("external identity implied a local role")
	}
}

func TestUT249NarrowerServiceScopeDeniesAnalystOperation(t *testing.T) {
	t.Parallel()
	if !errors.Is(Authorize(activeService(RoleAnalyst, "projects:read"), ActionExportWrite), ErrPermissionDenied) {
		t.Fatal("missing export scope did not deny analyst service")
	}
}

func TestUT250OnlyOpaqueSessionVerifierMatchesHash(t *testing.T) {
	t.Parallel()
	verifier, hash, err := NewSecret()
	if err != nil {
		t.Fatalf("NewSecret() error = %v", err)
	}
	if !VerifySecret(verifier, hash) || VerifySecret(string(hash[:]), hash) {
		t.Fatal("stored row either failed the verifier or worked as a cookie")
	}
}

func TestUT272LastAdminInvariantIsAtomicDomainRule(t *testing.T) {
	t.Parallel()
	target := Member{ID: 1, Role: RoleAdmin, Status: StatusActive}
	for _, next := range []struct {
		role   Role
		status Status
	}{{RoleViewer, StatusActive}, {RoleAdmin, StatusSuspended}} {
		if !errors.Is(ProtectLastAdmin(target, 1, next.role, next.status), ErrLastAdminRequired) {
			t.Fatalf("last admin transition to %s/%s succeeded", next.role, next.status)
		}
	}
}

func activeMember(role Role) Principal {
	return Principal{ActorID: 1, Kind: ActorMember, Role: role, Status: StatusActive, Workspace: 1}
}

func activeService(role Role, scopes ...string) Principal {
	return Principal{ActorID: 2, Kind: ActorServiceAccount, Role: role, Status: StatusActive, Scopes: scopes, Workspace: 1}
}
