// Package accessapi exposes the public catalog and local access-governance HTTP surface.
package accessapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/analysis"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/oidc"
)

const (
	sessionCookie  = "opi_session"
	csrfCookie     = "opi_csrf"
	stateCookie    = "opi_oidc_state"
	nonceCookie    = "opi_oidc_nonce"
	verifierCookie = "opi_oidc_verifier"
	maxBodyBytes   = 64 << 10
)

type IdentityProvider interface {
	AuthorizationURL(state, nonce, challenge string) string
	Exchange(context.Context, string, string, string) (oidc.Identity, error)
	VerifyBearer(context.Context, string) (oidc.Identity, error)
}

type Config struct {
	PublicBaseURL string
	IssuerURL     string
	SessionTTL    time.Duration
	SecureCookies bool
}

type Handler struct {
	httpapi.ServerInterface

	store      *accessstore.Store
	identity   IdentityProvider
	cursors    *access.CursorCodec
	logger     *slog.Logger
	config     Config
	idempotent *idempotencyCache
	loginLimit *rateLimiter
	models     interface {
		Status() analysis.ProviderStatus
	}
}

type Option func(*Handler)

func WithModelOperations(models interface {
	Status() analysis.ProviderStatus
}) Option {
	return func(handler *Handler) { handler.models = models }
}

func New(
	store *accessstore.Store,
	identity IdentityProvider,
	cursors *access.CursorCodec,
	logger *slog.Logger,
	config Config,
	options ...Option,
) (*Handler, error) {
	if store == nil || cursors == nil || logger == nil {
		return nil, errors.New("access store, cursor codec, and logger are required")
	}
	if config.SessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}
	base, err := url.Parse(config.PublicBaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("public base URL must be absolute")
	}
	handler := &Handler{
		store: store, identity: identity, cursors: cursors, logger: logger, config: config,
		idempotent: newIdempotencyCache(), loginLimit: newRateLimiter(10, time.Minute),
	}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	return handler, nil
}

// Routes returns only the operations owned by the access task. Other generated
// operations remain unavailable until their capability handlers land.
func Routes(handler *Handler) http.Handler {
	generatedMux := http.NewServeMux()
	generated := httpapi.HandlerFromMux(handler, generatedMux)
	mux := http.NewServeMux()
	for _, pattern := range []string{
		"GET /api/v1/catalog/projects", "GET /api/v1/catalog/projects/{project_id}",
		"GET /api/v1/session", "POST /api/v1/session/logout",
		"PATCH /api/v1/me/preferences", "POST /api/v1/me/deletion",
		"GET /api/v1/admin/members", "POST /api/v1/admin/members/{member_id}/approval",
		"PATCH /api/v1/admin/members/{member_id}", "GET /api/v1/admin/service-accounts",
		"POST /api/v1/admin/service-accounts",
		"PATCH /api/v1/admin/service-accounts/{service_account_id}",
		"GET /api/v1/admin/audit", "GET /api/v1/admin/operations",
	} {
		mux.Handle(pattern, generated)
	}
	mux.HandleFunc("GET /auth/login", handler.login)
	mux.HandleFunc("GET /auth/callback", handler.callback)
	return handler.requestMiddleware(mux)
}

// Middleware authenticates the request and attaches the locally authorized
// principal. Capability handlers use it so every versioned route shares the
// same session, bearer-token, CSRF, request-ID, and audit boundary.
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return h.requestMiddleware(next)
}

// Principal returns the local principal resolved by Middleware.
func Principal(ctx context.Context) access.Principal {
	return authFrom(ctx).principal
}

// RequestID returns the correlation identifier established by Middleware.
func RequestID(ctx context.Context) string {
	return requestID(ctx)
}

func (h *Handler) GetApiV1CatalogProjects(
	w http.ResponseWriter,
	r *http.Request,
	params httpapi.GetApiV1CatalogProjectsParams,
) {
	query := pointerValue(params.Q)
	limit, err := pageLimit(params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filters := "q=" + query
	offset, err := h.cursorOffset(pointerValue(params.Cursor), "catalog", filters)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	projects, err := h.store.ListCatalog(r.Context(), query, limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "catalog", filters, offset, limit, projects)
}

func (h *Handler) GetApiV1CatalogProjectsProjectId(w http.ResponseWriter, r *http.Request, value string) {
	id, err := parseID(value)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	project, err := h.store.GetCatalogProject(r.Context(), id)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, project)
}

func (h *Handler) GetApiV1Session(w http.ResponseWriter, r *http.Request) {
	state := authFrom(r.Context())
	if state.bearer && state.err != nil {
		h.problem(w, r, state.err)
		return
	}
	if state.principal.ActorID == 0 {
		h.writeJSON(w, http.StatusOK, map[string]any{"state": "anonymous", "authenticated": false})
		return
	}
	response := map[string]any{
		"authenticated": true,
		"state":         state.principal.Status,
		"role":          state.principal.Role,
		"actor_kind":    state.principal.Kind,
		"workspace_id":  strconv.FormatInt(state.principal.Workspace, 10),
	}
	if state.principal.Kind == access.ActorMember {
		response["member"] = state.session.Member
		if csrf, err := r.Cookie(csrfCookie); err == nil && access.VerifySecret(csrf.Value, state.session.CSRFHash) {
			response["csrf_token"] = csrf.Value
		}
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) PostApiV1SessionLogout(w http.ResponseWriter, r *http.Request) {
	state, ok := h.requireMember(w, r, false)
	if !ok {
		return
	}
	if err := h.store.RevokeSession(r.Context(), state.session.ID); err != nil {
		h.problem(w, r, err)
		return
	}
	h.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) PatchApiV1MePreferences(w http.ResponseWriter, r *http.Request) {
	state, ok := h.requireMember(w, r, true)
	if !ok {
		return
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Locale   string `json:"locale"`
		Timezone string `json:"timezone"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	member, err := h.store.UpdatePreferences(r.Context(), state.principal, version,
		body.Locale, body.Timezone, requestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", formatETag(member.Version))
	h.writeJSON(w, http.StatusOK, member)
}

func (h *Handler) PostApiV1MeDeletion(w http.ResponseWriter, r *http.Request) {
	state, ok := h.requireMember(w, r, false)
	if !ok {
		return
	}
	var body struct {
		Confirmation string `json:"confirmation"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	jobID, err := h.store.DeleteAccount(r.Context(), state.principal, body.Confirmation,
		requestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.clearSessionCookies(w)
	h.writeJSON(w, http.StatusAccepted, map[string]any{
		"id": strconv.FormatInt(jobID, 10), "kind": "account_deletion", "state": "succeeded",
	})
}

func (h *Handler) GetApiV1AdminMembers(
	w http.ResponseWriter,
	r *http.Request,
	params httpapi.GetApiV1AdminMembersParams,
) {
	if _, ok := h.requireAdmin(w, r, access.ActionMembershipGovern); !ok {
		return
	}
	limit, err := pageLimit(params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filters := strings.Join([]string{pointerValue(params.State), pointerValue(params.Role), pointerValue(params.Q)}, "|")
	offset, err := h.cursorOffset(pointerValue(params.Cursor), "members", filters)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	members, err := h.store.ListMembers(r.Context(), accessstore.MemberFilter{
		State: access.Status(pointerValue(params.State)), Role: access.Role(pointerValue(params.Role)),
		Query: pointerValue(params.Q), Limit: limit + 1, Offset: offset,
	})
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "members", filters, offset, limit, members)
}

func (h *Handler) PostApiV1AdminMembersMemberIdApproval(w http.ResponseWriter, r *http.Request, value string) {
	state, ok := h.requireAdmin(w, r, access.ActionMembershipGovern)
	if !ok {
		return
	}
	id, err := parseID(value)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Decision string      `json:"decision"`
		Role     access.Role `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	member, err := h.store.ApproveMember(r.Context(), state.principal, id, body.Decision,
		body.Role, requestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", formatETag(member.Version))
	h.writeJSON(w, http.StatusOK, member)
}

func (h *Handler) PatchApiV1AdminMembersMemberId(w http.ResponseWriter, r *http.Request, value string) {
	state, ok := h.requireAdmin(w, r, access.ActionMembershipGovern)
	if !ok {
		return
	}
	id, err := parseID(value)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Role  access.Role   `json:"role"`
		State access.Status `json:"state"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	member, err := h.store.UpdateMember(r.Context(), state.principal, id, version,
		body.Role, body.State, requestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", formatETag(member.Version))
	h.writeJSON(w, http.StatusOK, member)
}

func (h *Handler) GetApiV1AdminServiceAccounts(
	w http.ResponseWriter,
	r *http.Request,
	params httpapi.GetApiV1AdminServiceAccountsParams,
) {
	if _, ok := h.requireAdmin(w, r, access.ActionServiceGovern); !ok {
		return
	}
	limit, err := pageLimit(params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filters := pointerValue(params.State) + "|" + pointerValue(params.Q)
	offset, err := h.cursorOffset(pointerValue(params.Cursor), "service-accounts", filters)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	accounts, err := h.store.ListServiceAccounts(r.Context(), pointerValue(params.Q), limit+1, offset)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	if state := pointerValue(params.State); state != "" {
		accounts = filterAccounts(accounts, access.Status(state))
	}
	h.writePage(w, r, "service-accounts", filters, offset, limit, accounts)
}

func (h *Handler) PostApiV1AdminServiceAccounts(w http.ResponseWriter, r *http.Request) {
	state, ok := h.requireAdmin(w, r, access.ActionServiceGovern)
	if !ok {
		return
	}
	var body struct {
		ExternalSubject string        `json:"external_subject"`
		Name            string        `json:"name"`
		Role            access.Role   `json:"role"`
		Scopes          []string      `json:"scopes"`
		State           access.Status `json:"state"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	if body.State == "" {
		body.State = access.StatusActive
	}
	account, err := h.store.CreateServiceAccount(r.Context(), state.principal, access.ServiceAccount{
		Issuer: h.config.IssuerURL, ExternalSubject: body.ExternalSubject, Name: body.Name,
		Role: body.Role, Status: body.State, Scopes: body.Scopes,
	}, requestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", formatETag(account.Version))
	h.writeJSON(w, http.StatusCreated, account)
}

func (h *Handler) PatchApiV1AdminServiceAccountsServiceAccountId(
	w http.ResponseWriter,
	r *http.Request,
	value string,
) {
	state, ok := h.requireAdmin(w, r, access.ActionServiceGovern)
	if !ok {
		return
	}
	id, err := parseID(value)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	version, err := parseETag(r.Header.Get("If-Match"))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	var body struct {
		Role   access.Role   `json:"role"`
		State  access.Status `json:"state"`
		Scopes *[]string     `json:"scopes"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.problem(w, r, err)
		return
	}
	var scopes []string
	if body.Scopes != nil {
		scopes = *body.Scopes
	}
	account, err := h.store.UpdateServiceAccount(r.Context(), state.principal, id, version,
		body.Role, body.State, scopes, requestID(r.Context()))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	w.Header().Set("ETag", formatETag(account.Version))
	h.writeJSON(w, http.StatusOK, account)
}

func (h *Handler) GetApiV1AdminAudit(
	w http.ResponseWriter,
	r *http.Request,
	params httpapi.GetApiV1AdminAuditParams,
) {
	state, ok := h.requireAdmin(w, r, access.ActionAuditRead)
	if !ok {
		return
	}
	limit, err := pageLimit(params.Limit)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filters := strings.Join([]string{pointerValue(params.Actor), pointerValue(params.Action),
		pointerValue(params.Resource), pointerValue(params.Outcome), pointerValue(params.From),
		pointerValue(params.To)}, "|")
	offset, err := h.cursorOffset(pointerValue(params.Cursor), "audit", filters)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filter, err := auditFilter(params)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	filter.Limit, filter.Offset = limit+1, offset
	events, err := h.store.ListAudit(r.Context(), state.principal, filter)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.writePage(w, r, "audit", filters, offset, limit, events)
}

func auditFilter(params httpapi.GetApiV1AdminAuditParams) (accessstore.AuditFilter, error) {
	filter := accessstore.AuditFilter{Action: strings.TrimSpace(pointerValue(params.Action)),
		Resource: strings.TrimSpace(pointerValue(params.Resource)),
		Outcome:  strings.TrimSpace(pointerValue(params.Outcome))}
	if filter.Outcome != "" && filter.Outcome != "succeeded" && filter.Outcome != "failed" &&
		filter.Outcome != "denied" && filter.Outcome != "stale" {
		return accessstore.AuditFilter{}, invalidRequest(errors.New("unsupported audit outcome"))
	}
	if raw := strings.TrimSpace(pointerValue(params.Actor)); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return accessstore.AuditFilter{}, invalidRequest(errors.New("actor must be a positive identifier"))
		}
		filter.ActorID = &id
	}
	for raw, target := range map[string]**time.Time{
		pointerValue(params.From): &filter.OccurredFrom,
		pointerValue(params.To):   &filter.OccurredTo,
	} {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return accessstore.AuditFilter{}, invalidRequest(errors.New("audit time bounds must use RFC3339"))
		}
		value = value.UTC()
		*target = &value
	}
	if filter.OccurredFrom != nil && filter.OccurredTo != nil && filter.OccurredFrom.After(*filter.OccurredTo) {
		return accessstore.AuditFilter{}, invalidRequest(errors.New("audit from must not be after to"))
	}
	return filter, nil
}

func (h *Handler) GetApiV1AdminOperations(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireAdmin(w, r, access.ActionOperationsRead); !ok {
		return
	}
	status := "unavailable"
	if h.identity != nil {
		status = "healthy"
	}
	capabilities := []map[string]any{{
		"name": "external_identity", "configured": h.identity != nil, "status": status,
	}}
	response := map[string]any{
		"status":       status,
		"capabilities": capabilities,
		"redacted":     true,
	}
	if h.models != nil {
		model := h.models.Status()
		response["model_provider"] = model
		capabilities = append(capabilities, map[string]any{
			"name": "model_provider", "configured": model.Configured, "status": model.Health,
		})
		response["capabilities"] = capabilities
		if model.Health == "degraded" {
			response["status"] = "degraded"
		}
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if h.identity == nil {
		h.problem(w, r, errors.New("external identity is not configured"))
		return
	}
	if !h.loginLimit.Allow(clientAddress(r)) {
		err := httpx.NewProblem(http.StatusTooManyRequests, "rate_limited", "Too many requests",
			"Try signing in again later.", nil)
		h.problem(w, r, err)
		return
	}
	returnTo, err := oidc.ValidateReturnTo(r.URL.Query().Get("return_to"))
	if err != nil {
		h.problem(w, r, invalidRequest(err))
		return
	}
	state, stateHash, err := access.NewSecret()
	if err != nil {
		h.problem(w, r, err)
		return
	}
	nonce, nonceHash, err := access.NewSecret()
	if err != nil {
		h.problem(w, r, err)
		return
	}
	verifier, verifierHash, err := access.NewSecret()
	if err != nil {
		h.problem(w, r, err)
		return
	}
	expires := time.Now().UTC().Add(10 * time.Minute)
	if err := h.store.CreateLoginFlow(r.Context(), stateHash, nonceHash, verifierHash, returnTo, expires); err != nil {
		h.problem(w, r, err)
		return
	}
	for _, cookie := range []struct{ name, value string }{
		{stateCookie, state}, {nonceCookie, nonce}, {verifierCookie, verifier},
	} {
		h.setCookie(w, cookie.name, cookie.value, 10*time.Minute, true)
	}
	http.Redirect(w, r, h.identity.AuthorizationURL(state, nonce, oidc.PKCEChallenge(verifier)),
		http.StatusFound)
}

func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	if h.identity == nil || !h.loginLimit.Allow(clientAddress(r)) {
		h.problem(w, r, access.ErrAuthenticationRequired)
		return
	}
	state, stateErr := r.Cookie(stateCookie)
	nonce, nonceErr := r.Cookie(nonceCookie)
	verifier, verifierErr := r.Cookie(verifierCookie)
	if stateErr != nil || nonceErr != nil || verifierErr != nil ||
		state.Value != r.URL.Query().Get("state") || r.URL.Query().Get("code") == "" {
		h.problem(w, r, access.ErrAuthenticationRequired)
		return
	}
	flow, err := h.store.ConsumeLoginFlow(r.Context(), state.Value, nonce.Value, verifier.Value)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	identity, err := h.identity.Exchange(r.Context(), r.URL.Query().Get("code"), flow.Verifier, flow.Nonce)
	if err != nil {
		h.problem(w, r, access.ErrAuthenticationRequired)
		return
	}
	member, err := h.store.UpsertApplicant(r.Context(), identity.Key, identity.DisplayName, identity.Email)
	if err != nil {
		h.problem(w, r, err)
		return
	}
	verifierValue, verifierHash, err := access.NewSecret()
	if err != nil {
		h.problem(w, r, err)
		return
	}
	csrfValue, csrfHash, err := access.NewSecret()
	if err != nil {
		h.problem(w, r, err)
		return
	}
	sessionID, err := h.store.CreateSession(r.Context(), member.ID, verifierHash, csrfHash,
		time.Now().UTC().Add(h.config.SessionTTL))
	if err != nil {
		h.problem(w, r, err)
		return
	}
	h.setCookie(w, sessionCookie, access.EncodeSessionToken(sessionID, verifierValue), h.config.SessionTTL, true)
	h.setCookie(w, csrfCookie, csrfValue, h.config.SessionTTL, false)
	for _, name := range []string{stateCookie, nonceCookie, verifierCookie} {
		h.deleteCookie(w, name)
	}
	http.Redirect(w, r, flow.ReturnTo, http.StatusSeeOther)
}

func (h *Handler) requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := r.WithContext(context.WithValue(r.Context(), requestIDKey{}, newRequestID()))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		request = request.WithContext(context.WithValue(request.Context(), authKey{}, h.authenticate(request)))
		state := authFrom(request.Context())
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		if isUnsafe(request.Method) && strings.HasPrefix(request.URL.Path, "/api/") {
			defer func() {
				auditCtx, cancel := context.WithTimeout(context.WithoutCancel(request.Context()), 2*time.Second)
				defer cancel()
				if err := h.store.RecordHTTPMutation(auditCtx, state.principal, request.Method,
					request.URL.Path, writer.status, requestID(request.Context())); err != nil {
					h.logger.Warn("record HTTP mutation audit", slog.Any("error", err))
				}
			}()
		}
		if isUnsafe(request.Method) && state.cookie && strings.HasPrefix(request.URL.Path, "/api/") {
			if err := h.validateBrowserMutation(request, state.session); err != nil {
				h.problem(writer, request, err)
				return
			}
		}
		h.serveIdempotent(writer, request, next)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (writer *statusWriter) WriteHeader(status int) {
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}

func (writer *statusWriter) Write(body []byte) (int, error) {
	return writer.ResponseWriter.Write(body)
}

func (writer *statusWriter) Flush() {
	if flusher, ok := writer.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (h *Handler) authenticate(r *http.Request) authState {
	if authorization := r.Header.Get("Authorization"); authorization != "" {
		if h.identity == nil || !strings.HasPrefix(authorization, "Bearer ") ||
			strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")) == "" {
			return authState{bearer: true, err: access.ErrAuthenticationRequired}
		}
		identity, err := h.identity.VerifyBearer(r.Context(), strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer ")))
		if err != nil {
			return authState{bearer: true, err: access.ErrAuthenticationRequired}
		}
		principal, err := h.store.ResolveIdentity(r.Context(), identity.Key)
		if err != nil {
			return authState{bearer: true, err: access.ErrPermissionDenied}
		}
		return authState{principal: principal, bearer: true}
	}
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return authState{}
	}
	id, verifier, err := access.DecodeSessionToken(cookie.Value)
	if err != nil {
		return authState{cookie: true, err: access.ErrAuthenticationRequired}
	}
	session, err := h.store.ResolveSession(r.Context(), id, verifier)
	if err != nil {
		return authState{cookie: true, err: err}
	}
	return authState{principal: session.Principal, session: session, cookie: true}
}

func (h *Handler) validateBrowserMutation(r *http.Request, session accessstore.Session) error {
	if r.Header.Get("Origin") != strings.TrimSuffix(h.config.PublicBaseURL, "/") ||
		r.Header.Get("Sec-Fetch-Site") != "same-origin" {
		return httpx.NewProblem(http.StatusForbidden, "origin_denied", "Request origin denied",
			"This action must come from the application origin.", nil)
	}
	csrf := r.Header.Get("X-CSRF-Token")
	if csrf == "" || !access.VerifySecret(csrf, session.CSRFHash) {
		return httpx.NewProblem(http.StatusForbidden, "csrf_denied", "CSRF validation failed",
			"Refresh the session and try again.", nil)
	}
	return nil
}

func (h *Handler) requireMember(w http.ResponseWriter, r *http.Request, approved bool) (authState, bool) {
	state := authFrom(r.Context())
	if state.err != nil || state.principal.ActorID == 0 {
		h.problem(w, r, access.ErrAuthenticationRequired)
		return authState{}, false
	}
	if state.principal.Kind != access.ActorMember {
		h.problem(w, r, access.ErrPermissionDenied)
		return authState{}, false
	}
	if approved && !state.principal.IsApproved() {
		h.problem(w, r, access.ErrPermissionDenied)
		return authState{}, false
	}
	return state, true
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request, action access.Action) (authState, bool) {
	state := authFrom(r.Context())
	if state.err != nil {
		h.problem(w, r, state.err)
		return authState{}, false
	}
	if err := access.Authorize(state.principal, action); err != nil {
		h.problem(w, r, err)
		return authState{}, false
	}
	return state, true
}

func (h *Handler) serveIdempotent(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if !requiresIdempotency(r.Method, r.URL.Path) {
		next.ServeHTTP(w, r)
		return
	}
	state := authFrom(r.Context())
	if state.principal.ActorID == 0 {
		next.ServeHTTP(w, r)
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 200 {
		h.problem(w, r, httpx.NewProblem(http.StatusBadRequest, "idempotency_key_required",
			"Idempotency key required", "Provide a valid Idempotency-Key header.", nil))
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		h.problem(w, r, invalidRequest(err))
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	cacheKey := fmt.Sprintf("%s:%d:%s:%s:%s", state.principal.Kind, state.principal.ActorID,
		r.Method, r.URL.Path, key)
	digest := sha256.Sum256(body)
	if response, ready, result := h.idempotent.Begin(cacheKey, digest); result != idempotencyNew {
		if result == idempotencyConflict {
			h.problem(w, r, httpx.NewProblem(http.StatusConflict, "idempotency_conflict",
				"Idempotency conflict", "The key was already used with different input.", nil))
			return
		}
		if result == idempotencyWait {
			select {
			case <-ready:
				response, _, result = h.idempotent.Begin(cacheKey, digest)
				if result != idempotencyReplay {
					h.problem(w, r, httpx.NewProblem(http.StatusConflict, "idempotency_conflict",
						"Idempotency conflict", "The original request did not complete.", nil))
					return
				}
			case <-r.Context().Done():
				return
			}
		}
		response.replay(w)
		return
	}
	recorder := newResponseRecorder()
	next.ServeHTTP(recorder, r)
	response := recorder.response()
	h.idempotent.Complete(cacheKey, digest, response)
	response.replay(w)
}

func (h *Handler) cursorOffset(value, route, filters string) (int, error) {
	if value == "" {
		return 0, nil
	}
	cursor, err := h.cursors.Decode(value, route, filters)
	if err != nil {
		return 0, invalidRequest(err)
	}
	return cursor.Offset, nil
}

func (h *Handler) writePage(w http.ResponseWriter, r *http.Request, route, filters string, offset, limit int, values any) {
	items, hasMore := trimPage(values, limit)
	response := map[string]any{"items": items, "has_more": hasMore}
	if hasMore {
		next, err := h.cursors.Encode(access.Cursor{Route: route, Filters: filters, Offset: offset + limit})
		if err != nil {
			h.problem(w, r, err)
			return
		}
		response["next_cursor"] = next
	}
	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) problem(w http.ResponseWriter, r *http.Request, err error) {
	httpx.WriteProblem(w, h.logger, requestID(r.Context()), transportError(err))
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	httpx.WriteJSON(w, h.logger, status, value)
}

func (h *Handler) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: int(ttl.Seconds()),
		Expires: time.Now().UTC().Add(ttl), HttpOnly: httpOnly, Secure: h.config.SecureCookies,
		SameSite: http.SameSiteLaxMode})
}

func (h *Handler) deleteCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Path: "/", MaxAge: -1, Expires: time.Unix(1, 0),
		HttpOnly: name != csrfCookie, Secure: h.config.SecureCookies, SameSite: http.SameSiteLaxMode})
}

func (h *Handler) clearSessionCookies(w http.ResponseWriter) {
	h.deleteCookie(w, sessionCookie)
	h.deleteCookie(w, csrfCookie)
}

type authState struct {
	principal access.Principal
	session   accessstore.Session
	cookie    bool
	bearer    bool
	err       error
}

type authKey struct{}
type requestIDKey struct{}

func authFrom(ctx context.Context) authState {
	state, _ := ctx.Value(authKey{}).(authState)
	return state
}

func requestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

func newRequestID() string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func pointerValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func pageLimit(value *int32) (int, error) {
	if value == nil {
		return 50, nil
	}
	if *value < 1 || *value > 200 {
		return 0, invalidRequest(errors.New("limit must be between 1 and 200"))
	}
	return int(*value), nil
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 || strconv.FormatInt(id, 10) != value {
		return 0, invalidRequest(errors.New("identifier must be a positive canonical decimal string"))
	}
	return id, nil
}

func parseETag(value string) (int64, error) {
	if len(value) < 4 || value[0] != '"' || value[len(value)-1] != '"' || value[1] != 'v' {
		return 0, httpx.NewProblem(http.StatusPreconditionRequired, "if_match_required",
			"Version required", "Provide the current ETag in If-Match.", nil)
	}
	version, err := strconv.ParseInt(value[2:len(value)-1], 10, 64)
	if err != nil || version <= 0 {
		return 0, invalidRequest(errors.New("If-Match is malformed"))
	}
	return version, nil
}

func formatETag(version int64) string { return fmt.Sprintf("\"v%d\"", version) }

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return invalidRequest(err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return invalidRequest(errors.New("request body must contain one JSON document"))
	}
	return nil
}

func invalidRequest(cause error) error {
	return httpx.NewProblem(http.StatusBadRequest, "invalid_request", "Invalid request",
		"Correct the request and try again.", cause)
}

func transportError(err error) error {
	var problem *httpx.ProblemError
	if errors.As(err, &problem) {
		return problem
	}
	switch {
	case errors.Is(err, access.ErrAuthenticationRequired):
		return httpx.NewProblem(http.StatusUnauthorized, "authentication_required", "Authentication required",
			"Sign in and try again.", err)
	case errors.Is(err, access.ErrAccessPending):
		return httpx.NewProblem(http.StatusForbidden, "access_pending", "Access pending",
			"An administrator must approve this account.", err)
	case errors.Is(err, access.ErrPermissionDenied):
		return httpx.NewProblem(http.StatusForbidden, "permission_denied", "Permission denied",
			"Your current local access does not allow this action.", err)
	case errors.Is(err, access.ErrLastAdminRequired):
		return httpx.NewProblem(http.StatusConflict, "last_admin_required", "Last administrator required",
			"Assign another active administrator before changing this account.", err)
	case errors.Is(err, access.ErrVersionConflict):
		return httpx.NewProblem(http.StatusPreconditionFailed, "version_conflict", "Version conflict",
			"Refresh the resource and review the latest version.", err)
	case errors.Is(err, access.ErrNotFound):
		return httpx.NewProblem(http.StatusNotFound, "resource_not_found", "Resource not found",
			"The requested resource does not exist.", err)
	case errors.Is(err, access.ErrInvalidInput):
		return invalidRequest(err)
	default:
		return httpx.NewProblem(http.StatusInternalServerError, "internal_error", "Internal server error",
			"The request could not be completed.", err)
	}
}

func trimPage(values any, limit int) (any, bool) {
	switch items := values.(type) {
	case []accessstore.CatalogProject:
		return trim(items, limit)
	case []access.Member:
		return trim(items, limit)
	case []access.ServiceAccount:
		return trim(items, limit)
	case []accessstore.AuditEvent:
		return trim(items, limit)
	default:
		return values, false
	}
}

func trim[T any](items []T, limit int) ([]T, bool) {
	if len(items) <= limit {
		return items, false
	}
	return items[:limit], true
}

func filterAccounts(accounts []access.ServiceAccount, state access.Status) []access.ServiceAccount {
	filtered := make([]access.ServiceAccount, 0, len(accounts))
	for _, account := range accounts {
		if account.Status == state {
			filtered = append(filtered, account)
		}
	}
	return filtered
}

func isUnsafe(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func requiresIdempotency(method, path string) bool {
	if method != http.MethodPost || path == "/api/v1/session/logout" {
		return false
	}
	return path == "/api/v1/me/deletion" || path == "/api/v1/admin/service-accounts" ||
		path == "/api/v1/projects" || strings.HasPrefix(path, "/api/v1/projects/") ||
		strings.HasPrefix(path, "/api/v1/jobs/") || strings.HasSuffix(path, "/approval")
}

func clientAddress(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, entries: make(map[string][]time.Time)}
}

func (l *rateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-l.window)
	recent := l.entries[key][:0]
	for _, seen := range l.entries[key] {
		if seen.After(cutoff) {
			recent = append(recent, seen)
		}
	}
	if len(recent) >= l.limit {
		l.entries[key] = recent
		return false
	}
	l.entries[key] = append(recent, now)
	return true
}
