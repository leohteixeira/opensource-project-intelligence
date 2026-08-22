//go:build integration

package accessstore_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/oidc"
)

const testIssuer = "https://keycloak.integration.test/realms/opi"

type sequenceIDs struct{ value atomic.Int64 }

func (s *sequenceIDs) Next(context.Context) (int64, error) { return s.value.Add(1), nil }

type controlledIdentityProvider struct {
	identity oidc.Identity
	exchange error
	bearers  map[string]oidc.Identity
}

func (p *controlledIdentityProvider) AuthorizationURL(state, nonce, challenge string) string {
	values := url.Values{"state": {state}, "nonce": {nonce}, "code_challenge": {challenge}}
	return "https://identity.integration.test/authorize?" + values.Encode()
}

func (p *controlledIdentityProvider) Exchange(context.Context, string, string, string) (oidc.Identity, error) {
	if p.exchange != nil {
		return oidc.Identity{}, p.exchange
	}
	return p.identity, nil
}

func (p *controlledIdentityProvider) VerifyBearer(_ context.Context, raw string) (oidc.Identity, error) {
	identity, ok := p.bearers[raw]
	if !ok {
		return oidc.Identity{}, errors.New("invalid controlled token")
	}
	return identity, nil
}

type integrationHarness struct {
	t        *testing.T
	ctx      context.Context
	dbURL    string
	pool     *database.Pool
	store    *accessstore.Store
	ids      *sequenceIDs
	provider *controlledIdentityProvider
	handler  http.Handler
}

type seededMember struct {
	id       int64
	identity access.IdentityKey
	token    string
	csrf     string
}

func TestTask02IntegrationContracts(t *testing.T) {
	h := newIntegrationHarness(t)

	t.Run("IT-001 catalog archive concurrency", func(t *testing.T) {
		h.reset(t)
		h.seedProject(t, 11, "Visible", "active")
		before, err := h.store.ListCatalog(h.ctx, "", 20, 0)
		if err != nil || len(before) != 1 {
			t.Fatalf("initial catalog = %v, %v", before, err)
		}
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE public_catalog_projects SET state = 'archived' WHERE id = 11`); err != nil {
			t.Fatalf("archive project: %v", err)
		}
		after, err := h.store.ListCatalog(h.ctx, "", 20, 0)
		if err != nil || len(after) != 0 || before[0].Name != "Visible" {
			t.Fatalf("catalog after archive = %v, %v", after, err)
		}
	})

	t.Run("IT-002 interrupted catalog page retains safe representation", func(t *testing.T) {
		h.reset(t)
		h.seedProject(t, 12, "Retained", "active")
		last, err := h.store.ListCatalog(h.ctx, "", 20, 0)
		if err != nil {
			t.Fatalf("load catalog: %v", err)
		}
		cancelled, cancel := context.WithCancel(h.ctx)
		cancel()
		if _, err := h.store.ListCatalog(cancelled, "", 20, 0); err == nil {
			t.Fatal("cancelled request unexpectedly succeeded")
		}
		if len(last) != 1 || last[0].Name != "Retained" {
			t.Fatalf("last safe page was corrupted: %#v", last)
		}
	})

	t.Run("IT-003 catalog scale remains bounded", func(t *testing.T) {
		h.reset(t)
		batch := &pgx.Batch{}
		for index := 0; index < 1_000; index++ {
			batch.Queue(`INSERT INTO public_catalog_projects (id,name,slug,description) VALUES ($1,$2,$3,$4)`,
				int64(index+1), fmt.Sprintf("Project %04d", index), fmt.Sprintf("p-%04d", index), "public")
		}
		results := h.pool.Unwrap().SendBatch(h.ctx, batch)
		if err := results.Close(); err != nil {
			t.Fatalf("seed scaled catalog: %v", err)
		}
		page, err := h.store.ListCatalog(h.ctx, "Project", 25, 0)
		if err != nil || len(page) != 25 {
			t.Fatalf("scaled page size = %d, error = %v", len(page), err)
		}
	})

	t.Run("IT-004 simultaneous first login creates one applicant", func(t *testing.T) {
		h.reset(t)
		key := access.IdentityKey{Issuer: testIssuer, Subject: "concurrent-applicant"}
		var wait sync.WaitGroup
		ids := make(chan int64, 12)
		errs := make(chan error, 12)
		for range 12 {
			wait.Add(1)
			go func() {
				defer wait.Done()
				member, err := h.store.UpsertApplicant(h.ctx, key, "Applicant", "applicant@example.test")
				if err != nil {
					errs <- err
					return
				}
				ids <- member.ID
			}()
		}
		wait.Wait()
		close(ids)
		close(errs)
		for err := range errs {
			t.Errorf("UpsertApplicant() error = %v", err)
		}
		unique := map[int64]struct{}{}
		for id := range ids {
			unique[id] = struct{}{}
		}
		if len(unique) != 1 {
			t.Fatalf("membership IDs = %v, want one", unique)
		}
	})

	t.Run("IT-005 interrupted authentication can restart cleanly", func(t *testing.T) {
		h.reset(t)
		state, stateHash, _ := access.NewSecret()
		nonce, nonceHash, _ := access.NewSecret()
		verifier, verifierHash, _ := access.NewSecret()
		if err := h.store.CreateLoginFlow(h.ctx, stateHash, nonceHash, verifierHash, "/en/catalog", time.Now().Add(time.Minute)); err != nil {
			t.Fatalf("CreateLoginFlow() error = %v", err)
		}
		if _, err := h.store.ConsumeLoginFlow(h.ctx, state, nonce, "wrong"); !errors.Is(err, access.ErrAuthenticationRequired) {
			t.Fatalf("wrong flow error = %v", err)
		}
		if _, err := h.store.ConsumeLoginFlow(h.ctx, state, nonce, verifier); err != nil {
			t.Fatalf("clean retry failed: %v", err)
		}
		var sessions int
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM browser_sessions`).Scan(&sessions)
		if sessions != 0 {
			t.Fatalf("partial sessions = %d", sessions)
		}
	})

	t.Run("IT-006 applicant burst does not disrupt approved session", func(t *testing.T) {
		h.reset(t)
		approved := h.seedMember(t, "approved", access.RoleViewer, access.StatusActive)
		var wait sync.WaitGroup
		for index := 0; index < 60; index++ {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				_, _ = h.store.UpsertApplicant(h.ctx, access.IdentityKey{Issuer: testIssuer, Subject: fmt.Sprintf("burst-%d", index)}, "Burst", "")
			}(index)
		}
		wait.Wait()
		id, verifier, _ := access.DecodeSessionToken(approved.token)
		if _, err := h.store.ResolveSession(h.ctx, id, verifier); err != nil {
			t.Fatalf("approved session failed: %v", err)
		}
	})

	t.Run("IT-007 conflicting membership updates report stale action", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		target := h.seedMember(t, "target", access.RoleViewer, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		var wait sync.WaitGroup
		errs := make(chan error, 2)
		for _, role := range []access.Role{access.RoleAnalyst, access.RoleAdmin} {
			wait.Add(1)
			go func(role access.Role) {
				defer wait.Done()
				_, err := h.store.UpdateMember(h.ctx, principal, target.id, 1, role, access.StatusActive, "conflict")
				errs <- err
			}(role)
		}
		wait.Wait()
		close(errs)
		succeeded := 0
		for err := range errs {
			if err == nil {
				succeeded++
			}
		}
		if succeeded != 1 {
			t.Fatalf("successful conflicting updates = %d, want 1", succeeded)
		}
	})

	t.Run("IT-008 failed role change preserves prior authority", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		target := h.seedMember(t, "target", access.RoleViewer, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		if _, err := h.store.UpdateMember(h.ctx, principal, target.id, 1, "owner", access.StatusActive, "invalid"); !errors.Is(err, access.ErrInvalidInput) {
			t.Fatalf("invalid role error = %v", err)
		}
		resolved, _ := h.store.ResolveIdentity(h.ctx, target.identity)
		if resolved.Role != access.RoleViewer {
			t.Fatalf("role after failure = %s", resolved.Role)
		}
	})

	t.Run("IT-009 applicant volume remains explicitly pending", func(t *testing.T) {
		h.reset(t)
		for index := 0; index < 300; index++ {
			_, err := h.store.UpsertApplicant(h.ctx, access.IdentityKey{Issuer: testIssuer, Subject: fmt.Sprintf("pending-%d", index)}, "Pending", "")
			if err != nil {
				t.Fatalf("seed applicant %d: %v", index, err)
			}
		}
		page, err := h.store.ListMembers(h.ctx, accessstore.MemberFilter{State: access.StatusPending, Limit: 51})
		if err != nil || len(page) != 51 {
			t.Fatalf("pending page = %d, %v", len(page), err)
		}
		for _, member := range page {
			if member.Role != "" || member.Status != access.StatusPending {
				t.Fatalf("implicit approval: %#v", member)
			}
		}
	})

	t.Run("IT-010 IT-132 deletion wins over concurrent profile edits and erases personal data", func(t *testing.T) {
		h.reset(t)
		h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		target := h.seedMember(t, "delete-race", access.RoleViewer, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, target.identity)
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			_, _ = h.store.UpdatePreferences(h.ctx, principal, 1, "pt-BR", "UTC", "prefs")
		}()
		go func() {
			defer wait.Done()
			_, _ = h.store.DeleteAccount(h.ctx, principal, access.DeletionConfirmation, "delete")
		}()
		wait.Wait()
		var status, name, email string
		if err := h.pool.Unwrap().QueryRow(h.ctx, `SELECT m.status,i.display_name,i.email FROM memberships m JOIN external_identities i ON i.id=m.identity_id WHERE m.id=$1`, target.id).Scan(&status, &name, &email); err != nil {
			t.Fatalf("read deletion result: %v", err)
		}
		if status != "deleted" || name != "" || email != "" {
			t.Fatalf("deletion result = %q %q %q", status, name, email)
		}
	})

	t.Run("IT-011 interrupted confirmation creates no deletion", func(t *testing.T) {
		h.reset(t)
		target := h.seedMember(t, "not-deleted", access.RoleViewer, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, target.identity)
		if _, err := h.store.DeleteAccount(h.ctx, principal, "", "interrupted"); !errors.Is(err, access.ErrInvalidInput) {
			t.Fatalf("error = %v", err)
		}
		resolved, err := h.store.ResolveIdentity(h.ctx, target.identity)
		if err != nil || resolved.Status != access.StatusActive {
			t.Fatalf("member after interruption = %#v, %v", resolved, err)
		}
	})

	t.Run("IT-012 preference update touches one membership", func(t *testing.T) {
		h.reset(t)
		target := h.seedMember(t, "preferences", access.RoleViewer, access.StatusActive)
		for index := 0; index < 100; index++ {
			h.seedMember(t, fmt.Sprintf("other-%d", index), access.RoleViewer, access.StatusActive)
		}
		principal, _ := h.store.ResolveIdentity(h.ctx, target.identity)
		member, err := h.store.UpdatePreferences(h.ctx, principal, 1, "pt-BR", "America/Sao_Paulo", "preferences")
		if err != nil || member.Version != 2 {
			t.Fatalf("UpdatePreferences() = %#v, %v", member, err)
		}
		var changed int
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM memberships WHERE version > 1`).Scan(&changed)
		if changed != 1 {
			t.Fatalf("updated memberships = %d", changed)
		}
	})

	t.Run("IT-094 scope update authorizes one committed version", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		account, err := h.store.CreateServiceAccount(h.ctx, principal, access.ServiceAccount{Issuer: testIssuer, ExternalSubject: "scope-race", Name: "Scope race", Role: access.RoleAnalyst, Status: access.StatusActive, Scopes: []string{"projects:read"}}, "create")
		if err != nil {
			t.Fatalf("create service: %v", err)
		}
		updated, err := h.store.UpdateServiceAccount(h.ctx, principal, account.ID, account.Version, access.RoleAnalyst, access.StatusActive, []string{"projects:write"}, "scope")
		if err != nil {
			t.Fatalf("update service: %v", err)
		}
		resolved, err := h.store.ResolveIdentity(h.ctx, access.IdentityKey{Issuer: testIssuer, Subject: "scope-race"})
		if err != nil || resolved.Version != updated.Version || len(resolved.Scopes) != 1 || resolved.Scopes[0] != "projects:write" {
			t.Fatalf("resolved service = %#v, %v", resolved, err)
		}
	})

	t.Run("IT-095 interrupted idempotent action retries once", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		body := `{"external_subject":"retry-service","name":"Retry","role":"viewer","scopes":["projects:read"]}`
		first := h.request(t, http.MethodPost, "/api/v1/admin/service-accounts", body, &admin, "retry-key", "")
		second := h.request(t, http.MethodPost, "/api/v1/admin/service-accounts", body, &admin, "retry-key", "")
		if first.Code != http.StatusCreated || second.Code != first.Code || second.Body.String() != first.Body.String() {
			t.Fatalf("retry responses = %d/%d %q/%q", first.Code, second.Code, first.Body.String(), second.Body.String())
		}
		var count int
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM service_accounts`).Scan(&count)
		if count != 1 {
			t.Fatalf("service outcomes = %d", count)
		}
	})

	t.Run("IT-096 service identity load remains isolated at scale", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		for index := 0; index < 80; index++ {
			_, err := h.store.CreateServiceAccount(h.ctx, principal, access.ServiceAccount{Issuer: testIssuer, ExternalSubject: fmt.Sprintf("svc-%d", index), Name: "Service", Role: access.RoleViewer, Status: access.StatusActive, Scopes: []string{"projects:read"}}, "scale")
			if err != nil {
				t.Fatalf("create service %d: %v", index, err)
			}
		}
		var wait sync.WaitGroup
		errs := make(chan error, 80)
		for index := 0; index < 80; index++ {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				p, err := h.store.ResolveIdentity(h.ctx, access.IdentityKey{Issuer: testIssuer, Subject: fmt.Sprintf("svc-%d", index)})
				if err == nil && (p.ActorID == 0 || len(p.Scopes) != 1) {
					err = errors.New("cross-account resolution")
				}
				errs <- err
			}(index)
		}
		wait.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Errorf("service resolution: %v", err)
			}
		}
	})

	t.Run("IT-100 controlled OIDC callback creates one opaque session", func(t *testing.T) {
		h.reset(t)
		h.provider.identity = oidc.Identity{Key: access.IdentityKey{Issuer: testIssuer, Subject: "oidc-user"}, DisplayName: "OIDC User", Email: "oidc@example.test"}
		login := h.request(t, http.MethodGet, "/auth/login?return_to=%2Fen%2Fcatalog", "", nil, "", "")
		if login.Code != http.StatusFound {
			t.Fatalf("login status = %d: %s", login.Code, login.Body.String())
		}
		location, _ := url.Parse(login.Header().Get("Location"))
		callback := httptest.NewRequest(http.MethodGet, "/auth/callback?code=valid&state="+url.QueryEscape(location.Query().Get("state")), nil)
		for _, cookie := range login.Result().Cookies() {
			callback.AddCookie(cookie)
		}
		result := httptest.NewRecorder()
		h.handler.ServeHTTP(result, callback)
		if result.Code != http.StatusSeeOther {
			t.Fatalf("callback status = %d: %s", result.Code, result.Body.String())
		}
		var memberships, sessions int
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM memberships`).Scan(&memberships)
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM browser_sessions WHERE revoked_at IS NULL`).Scan(&sessions)
		if memberships != 1 || sessions != 1 {
			t.Fatalf("OIDC results: memberships=%d sessions=%d", memberships, sessions)
		}
	})

	t.Run("IT-101 invalid OIDC exchange creates no local state", func(t *testing.T) {
		h.reset(t)
		h.provider.exchange = errors.New("wrong issuer or signature")
		defer func() { h.provider.exchange = nil }()
		login := h.request(t, http.MethodGet, "/auth/login?return_to=%2Fen", "", nil, "", "")
		location, _ := url.Parse(login.Header().Get("Location"))
		callback := httptest.NewRequest(http.MethodGet, "/auth/callback?code=invalid&state="+url.QueryEscape(location.Query().Get("state")), nil)
		for _, cookie := range login.Result().Cookies() {
			callback.AddCookie(cookie)
		}
		result := httptest.NewRecorder()
		h.handler.ServeHTTP(result, callback)
		if result.Code != http.StatusUnauthorized {
			t.Fatalf("invalid callback status = %d", result.Code)
		}
		var memberships, sessions int
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM memberships`).Scan(&memberships)
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM browser_sessions`).Scan(&sessions)
		if memberships != 0 || sessions != 0 {
			t.Fatalf("invalid OIDC wrote memberships=%d sessions=%d", memberships, sessions)
		}
	})

	t.Run("IT-102 suspension immediately revokes active sessions", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		target := h.seedMember(t, "suspend", access.RoleViewer, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		if _, err := h.store.UpdateMember(h.ctx, principal, target.id, 1, access.RoleViewer, access.StatusSuspended, "suspend"); err != nil {
			t.Fatalf("suspend: %v", err)
		}
		id, verifier, _ := access.DecodeSessionToken(target.token)
		if _, err := h.store.ResolveSession(h.ctx, id, verifier); !errors.Is(err, access.ErrAuthenticationRequired) {
			t.Fatalf("revoked session error = %v", err)
		}
	})

	runHTTPOperationContracts(t, h)
}

func runHTTPOperationContracts(t *testing.T, h *integrationHarness) {
	t.Helper()

	t.Run("IT-147 catalog collection success", func(t *testing.T) {
		h.reset(t)
		h.seedProject(t, 147, "Public", "active")
		response := h.request(t, http.MethodGet, "/api/v1/catalog/projects?limit=1", "", nil, "", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, "source_links")
		requireNotContains(t, response, "score")
	})
	t.Run("IT-148 catalog collection rejects malformed cursor", func(t *testing.T) {
		h.reset(t)
		response := h.request(t, http.MethodGet, "/api/v1/catalog/projects?cursor=unsigned&limit=999", "", nil, "", "")
		requireStatus(t, response, http.StatusBadRequest)
		requireNotContains(t, response, "SELECT")
	})
	t.Run("IT-149 catalog resource success", func(t *testing.T) {
		h.reset(t)
		h.seedProject(t, 149, "One public project", "paused")
		response := h.request(t, http.MethodGet, "/api/v1/catalog/projects/149", "", nil, "", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, "One public project")
	})
	t.Run("IT-150 catalog resource rejects malformed identity", func(t *testing.T) {
		h.reset(t)
		response := h.request(t, http.MethodGet, "/api/v1/catalog/projects/01", "", nil, "", "")
		requireStatus(t, response, http.StatusBadRequest)
		requireNotContains(t, response, "internal_error")
	})
	t.Run("IT-151 anonymous session success", func(t *testing.T) {
		h.reset(t)
		response := h.request(t, http.MethodGet, "/api/v1/session", "", nil, "", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, `"authenticated":false`)
	})
	t.Run("IT-152 session rejects malformed bearer", func(t *testing.T) {
		h.reset(t)
		request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
		request.Header.Set("Authorization", "Bearer invalid")
		response := httptest.NewRecorder()
		h.handler.ServeHTTP(response, request)
		requireStatus(t, response, http.StatusUnauthorized)
		requireNotContains(t, response, "subject")
	})
	t.Run("IT-153 logout success", func(t *testing.T) {
		h.reset(t)
		member := h.seedMember(t, "logout", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodPost, "/api/v1/session/logout", "", &member, "", "")
		requireStatus(t, response, http.StatusNoContent)
		id, verifier, _ := access.DecodeSessionToken(member.token)
		if _, err := h.store.ResolveSession(h.ctx, id, verifier); !errors.Is(err, access.ErrAuthenticationRequired) {
			t.Fatalf("logout session error = %v", err)
		}
	})
	t.Run("IT-154 logout rejects missing browser defenses", func(t *testing.T) {
		h.reset(t)
		member := h.seedMember(t, "logout-denied", access.RoleViewer, access.StatusActive)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/session/logout", nil)
		request.AddCookie(&http.Cookie{Name: "opi_session", Value: member.token})
		response := httptest.NewRecorder()
		h.handler.ServeHTTP(response, request)
		requireStatus(t, response, http.StatusForbidden)
		id, verifier, _ := access.DecodeSessionToken(member.token)
		if _, err := h.store.ResolveSession(h.ctx, id, verifier); err != nil {
			t.Fatalf("denied logout revoked session: %v", err)
		}
	})
	t.Run("IT-155 preference update success", func(t *testing.T) {
		h.reset(t)
		member := h.seedMember(t, "preferences", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodPatch, "/api/v1/me/preferences", `{"locale":"pt-BR","timezone":"America/Sao_Paulo"}`, &member, "", `"v1"`)
		requireStatus(t, response, http.StatusOK)
		if response.Header().Get("ETag") != `"v2"` {
			t.Fatalf("ETag = %q", response.Header().Get("ETag"))
		}
		requireContains(t, response, "pt-BR")
	})
	t.Run("IT-156 preference update rejects invalid input", func(t *testing.T) {
		h.reset(t)
		member := h.seedMember(t, "preferences-invalid", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodPatch, "/api/v1/me/preferences", `{"locale":"fr","timezone":"UTC"}`, &member, "", `"v1"`)
		requireStatus(t, response, http.StatusBadRequest)
		resolved, _ := h.store.ResolveIdentity(h.ctx, member.identity)
		if resolved.Version != 1 {
			t.Fatalf("invalid preference version = %d", resolved.Version)
		}
	})
	t.Run("IT-157 account deletion success", func(t *testing.T) {
		h.reset(t)
		h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		member := h.seedMember(t, "delete", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodPost, "/api/v1/me/deletion", `{"confirmation":"DELETE MY ACCOUNT"}`, &member, "delete-157", "")
		requireStatus(t, response, http.StatusAccepted)
		requireContains(t, response, "account_deletion")
	})
	t.Run("IT-158 account deletion rejects inexact confirmation", func(t *testing.T) {
		h.reset(t)
		member := h.seedMember(t, "keep", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodPost, "/api/v1/me/deletion", `{"confirmation":"DELETE"}`, &member, "delete-158", "")
		requireStatus(t, response, http.StatusBadRequest)
		resolved, _ := h.store.ResolveIdentity(h.ctx, member.identity)
		if resolved.Status != access.StatusActive {
			t.Fatalf("invalid deletion status = %s", resolved.Status)
		}
	})
	t.Run("IT-159 administrator member list success", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		h.seedMember(t, "applicant", "", access.StatusPending)
		response := h.request(t, http.MethodGet, "/api/v1/admin/members?state=pending&limit=20", "", &admin, "", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, "applicant")
	})
	t.Run("IT-160 member list denies insufficient principal", func(t *testing.T) {
		h.reset(t)
		viewer := h.seedMember(t, "viewer", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodGet, "/api/v1/admin/members", "", &viewer, "", "")
		requireStatus(t, response, http.StatusForbidden)
		requireNotContains(t, response, "display_name")
	})
	t.Run("IT-161 applicant approval success", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		applicant := h.seedMember(t, "applicant", "", access.StatusPending)
		response := h.request(t, http.MethodPost, "/api/v1/admin/members/"+strconv.FormatInt(applicant.id, 10)+"/approval", `{"decision":"approve","role":"viewer"}`, &admin, "approval-161", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, `"status":"active"`)
	})
	t.Run("IT-162 applicant approval rejects invalid role", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		applicant := h.seedMember(t, "applicant", "", access.StatusPending)
		response := h.request(t, http.MethodPost, "/api/v1/admin/members/"+strconv.FormatInt(applicant.id, 10)+"/approval", `{"decision":"approve","role":"owner"}`, &admin, "approval-162", "")
		requireStatus(t, response, http.StatusBadRequest)
		resolved, _ := h.store.ResolveIdentity(h.ctx, applicant.identity)
		if resolved.Status != access.StatusPending {
			t.Fatalf("invalid approval status = %s", resolved.Status)
		}
	})
	t.Run("IT-163 membership conditional update success", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		target := h.seedMember(t, "member", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodPatch, "/api/v1/admin/members/"+strconv.FormatInt(target.id, 10), `{"role":"analyst","state":"active"}`, &admin, "", `"v1"`)
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, `"role":"analyst"`)
	})
	t.Run("IT-164 membership update rejects stale ETag", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		target := h.seedMember(t, "member", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodPatch, "/api/v1/admin/members/"+strconv.FormatInt(target.id, 10), `{"role":"analyst","state":"active"}`, &admin, "", `"v9"`)
		requireStatus(t, response, http.StatusPreconditionFailed)
		resolved, _ := h.store.ResolveIdentity(h.ctx, target.identity)
		if resolved.Role != access.RoleViewer {
			t.Fatalf("stale update role = %s", resolved.Role)
		}
	})
	t.Run("IT-165 service account list success without secrets", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		_, _ = h.store.CreateServiceAccount(h.ctx, principal, access.ServiceAccount{Issuer: testIssuer, ExternalSubject: "listed", Name: "Listed", Role: access.RoleViewer, Status: access.StatusActive, Scopes: []string{"projects:read"}}, "list")
		response := h.request(t, http.MethodGet, "/api/v1/admin/service-accounts", "", &admin, "", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, "listed")
		requireNotContains(t, response, "token")
		requireNotContains(t, response, "secret")
	})
	t.Run("IT-166 service account list denies insufficient principal", func(t *testing.T) {
		h.reset(t)
		viewer := h.seedMember(t, "viewer", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodGet, "/api/v1/admin/service-accounts", "", &viewer, "", "")
		requireStatus(t, response, http.StatusForbidden)
		requireNotContains(t, response, "external_subject")
	})
	t.Run("IT-167 service account creation success", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		response := h.request(t, http.MethodPost, "/api/v1/admin/service-accounts", `{"external_subject":"exporter","name":"Exporter","role":"analyst","scopes":["projects:read","exports:write"]}`, &admin, "service-167", "")
		requireStatus(t, response, http.StatusCreated)
		requireContains(t, response, "exporter")
		if response.Header().Get("ETag") != `"v1"` {
			t.Fatalf("ETag = %q", response.Header().Get("ETag"))
		}
	})
	t.Run("IT-168 service account creation rejects Admin", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		response := h.request(t, http.MethodPost, "/api/v1/admin/service-accounts", `{"external_subject":"bad","name":"Bad","role":"admin","scopes":[]}`, &admin, "service-168", "")
		requireStatus(t, response, http.StatusBadRequest)
		var count int
		_ = h.pool.Unwrap().QueryRow(h.ctx, `SELECT count(*) FROM service_accounts`).Scan(&count)
		if count != 0 {
			t.Fatalf("invalid service accounts = %d", count)
		}
	})
	t.Run("IT-169 service account conditional update success", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		account, _ := h.store.CreateServiceAccount(h.ctx, principal, access.ServiceAccount{Issuer: testIssuer, ExternalSubject: "patch", Name: "Patch", Role: access.RoleViewer, Status: access.StatusActive, Scopes: []string{"projects:read"}}, "create")
		response := h.request(t, http.MethodPatch, "/api/v1/admin/service-accounts/"+strconv.FormatInt(account.ID, 10), `{"role":"analyst","state":"active","scopes":["projects:write"]}`, &admin, "", `"v1"`)
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, "projects:write")
	})
	t.Run("IT-170 service account update rejects stale ETag", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		account, _ := h.store.CreateServiceAccount(h.ctx, principal, access.ServiceAccount{Issuer: testIssuer, ExternalSubject: "stale", Name: "Stale", Role: access.RoleViewer, Status: access.StatusActive, Scopes: []string{"projects:read"}}, "create")
		response := h.request(t, http.MethodPatch, "/api/v1/admin/service-accounts/"+strconv.FormatInt(account.ID, 10), `{"role":"analyst","state":"active"}`, &admin, "", `"v5"`)
		requireStatus(t, response, http.StatusPreconditionFailed)
	})
	t.Run("IT-171 immutable audit collection success", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		applicant := h.seedMember(t, "audit-target", "", access.StatusPending)
		principal, _ := h.store.ResolveIdentity(h.ctx, admin.identity)
		_, _ = h.store.ApproveMember(h.ctx, principal, applicant.id, "approve", access.RoleViewer, "audit-request")
		response := h.request(t, http.MethodGet, "/api/v1/admin/audit", "", &admin, "", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, "membership.approve")
		requireNotContains(t, response, "password")
		if _, err := h.pool.Unwrap().Exec(h.ctx, `UPDATE audit_events SET outcome='failed'`); err == nil {
			t.Fatal("immutable audit event was updated")
		}
	})
	t.Run("IT-172 audit collection denies insufficient principal", func(t *testing.T) {
		h.reset(t)
		viewer := h.seedMember(t, "viewer", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodGet, "/api/v1/admin/audit", "", &viewer, "", "")
		requireStatus(t, response, http.StatusForbidden)
		requireNotContains(t, response, "changes")
	})
	t.Run("IT-173 operations status is redacted", func(t *testing.T) {
		h.reset(t)
		admin := h.seedMember(t, "admin", access.RoleAdmin, access.StatusActive)
		response := h.request(t, http.MethodGet, "/api/v1/admin/operations", "", &admin, "", "")
		requireStatus(t, response, http.StatusOK)
		requireContains(t, response, `"redacted":true`)
		requireNotContains(t, response, "client_secret")
		requireNotContains(t, response, "https://")
	})
	t.Run("IT-174 operations status denies insufficient principal", func(t *testing.T) {
		h.reset(t)
		viewer := h.seedMember(t, "viewer", access.RoleViewer, access.StatusActive)
		response := h.request(t, http.MethodGet, "/api/v1/admin/operations", "", &viewer, "", "")
		requireStatus(t, response, http.StatusForbidden)
		requireNotContains(t, response, "capabilities")
	})
}

func newIntegrationHarness(t *testing.T) *integrationHarness {
	t.Helper()
	ctx := context.Background()
	baseURL := os.Getenv("OPI_INTEGRATION_DATABASE_URL")
	if baseURL == "" {
		t.Fatal("OPI_INTEGRATION_DATABASE_URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	databaseName := fmt.Sprintf("opi_task02_%d", time.Now().UnixNano())
	connection, err := pgx.Connect(ctx, baseURL)
	if err != nil {
		t.Fatalf("connect integration server: %v", err)
	}
	if _, err = connection.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{databaseName}.Sanitize()); err != nil {
		connection.Close(ctx)
		t.Fatalf("create integration database: %v", err)
	}
	connection.Close(ctx)
	parsed.Path = "/" + databaseName
	databaseURL := parsed.String()
	root := integrationRepositoryRoot(t)
	command := exec.Command(filepath.Join(root, "scripts", "migrate.sh"), "up")
	command.Dir = root
	command.Env = append(os.Environ(), "DATABASE_URL="+databaseURL)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("migrate task database: %v\n%s", err, output)
	}
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open task database: %v", err)
	}
	ids := &sequenceIDs{}
	ids.value.Store(8_000_000_000_000_000_000)
	store := accessstore.New(pool, ids)
	provider := &controlledIdentityProvider{bearers: map[string]oidc.Identity{}}
	cursors, _ := access.NewCursorCodec(bytes.Repeat([]byte{0x42}, 32))
	handler, err := accessapi.New(store, provider, cursors, slog.New(slog.NewTextHandler(io.Discard, nil)), accessapi.Config{PublicBaseURL: "http://opi.integration.test", IssuerURL: testIssuer, SessionTTL: time.Hour})
	if err != nil {
		t.Fatalf("construct access handler: %v", err)
	}
	h := &integrationHarness{t: t, ctx: ctx, dbURL: databaseURL, pool: pool, store: store, ids: ids, provider: provider, handler: accessapi.Routes(handler)}
	t.Cleanup(func() {
		pool.Close()
		cleanup, cleanupErr := pgx.Connect(context.Background(), baseURL)
		if cleanupErr != nil {
			t.Errorf("connect database cleanup: %v", cleanupErr)
			return
		}
		defer cleanup.Close(context.Background())
		if _, cleanupErr = cleanup.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{databaseName}.Sanitize()+" WITH (FORCE)"); cleanupErr != nil {
			t.Errorf("drop task database: %v", cleanupErr)
		}
	})
	return h
}

func (h *integrationHarness) reset(t *testing.T) {
	t.Helper()
	_, err := h.pool.Unwrap().Exec(h.ctx, `TRUNCATE TABLE oidc_login_flows,browser_sessions,service_account_scopes,service_accounts,memberships,external_identities,public_catalog_projects,audit_events,outbox_events,object_references,evidence_vectors,jobs,workspaces RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("reset integration data: %v", err)
	}
	if err := h.store.EnsureWorkspace(h.ctx); err != nil {
		t.Fatalf("ensure workspace: %v", err)
	}
	h.provider.exchange = nil
	h.provider.identity = oidc.Identity{}
	h.provider.bearers = map[string]oidc.Identity{}
}

func (h *integrationHarness) seedProject(t *testing.T, id int64, name, state string) {
	t.Helper()
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-" + strconv.FormatInt(id, 10)
	_, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO public_catalog_projects(id,name,slug,description,source_links,state) VALUES($1,$2,$3,'Public description','["https://github.com/example/project"]',$4)`, id, name, slug, state)
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
}

func (h *integrationHarness) seedMember(t *testing.T, subject string, role access.Role, status access.Status) seededMember {
	t.Helper()
	identityID, _ := h.ids.Next(h.ctx)
	memberID, _ := h.ids.Next(h.ctx)
	key := access.IdentityKey{Issuer: testIssuer, Subject: subject}
	_, err := h.pool.Unwrap().Exec(h.ctx, `INSERT INTO external_identities(id,issuer,subject,display_name,email) VALUES($1,$2,$3,$4,$5)`, identityID, key.Issuer, key.Subject, subject, subject+"@example.test")
	if err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	var roleValue any = role
	if role == "" {
		roleValue = nil
	}
	_, err = h.pool.Unwrap().Exec(h.ctx, `INSERT INTO memberships(id,workspace_id,identity_id,role,status) VALUES($1,$2,$3,$4,$5)`, memberID, accessstore.WorkspaceID, identityID, roleValue, status)
	if err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	verifier, verifierHash, _ := access.NewSecret()
	csrf, csrfHash, _ := access.NewSecret()
	sessionID, err := h.store.CreateSession(h.ctx, memberID, verifierHash, csrfHash, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return seededMember{id: memberID, identity: key, token: access.EncodeSessionToken(sessionID, verifier), csrf: csrf}
}

func (h *integrationHarness) request(t *testing.T, method, path, body string, member *seededMember, idempotency, etag string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if member != nil {
		request.AddCookie(&http.Cookie{Name: "opi_session", Value: member.token})
		if method != http.MethodGet && method != http.MethodHead {
			request.Header.Set("Origin", "http://opi.integration.test")
			request.Header.Set("Sec-Fetch-Site", "same-origin")
			request.Header.Set("X-CSRF-Token", member.csrf)
		}
	}
	if idempotency != "" {
		request.Header.Set("Idempotency-Key", idempotency)
	}
	if etag != "" {
		request.Header.Set("If-Match", etag)
	}
	response := httptest.NewRecorder()
	h.handler.ServeHTTP(response, request)
	return response
}

func requireStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status = %d, want %d: %s", response.Code, want, response.Body.String())
	}
}
func requireContains(t *testing.T, response *httptest.ResponseRecorder, want string) {
	t.Helper()
	if !strings.Contains(response.Body.String(), want) {
		t.Fatalf("response %q does not contain %q", response.Body.String(), want)
	}
}
func requireNotContains(t *testing.T, response *httptest.ResponseRecorder, value string) {
	t.Helper()
	if strings.Contains(strings.ToLower(response.Body.String()), strings.ToLower(value)) {
		t.Fatalf("response exposed %q: %s", value, response.Body.String())
	}
}

func integrationRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func sortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
func decodeObject(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return value
}
