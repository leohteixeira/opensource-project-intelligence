package accessapi

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
)

func TestUT010LoginCallbackRateLimitIsBounded(t *testing.T) {
	t.Parallel()
	limiter := newRateLimiter(2, time.Minute)
	if !limiter.Allow("client") || !limiter.Allow("client") || limiter.Allow("client") {
		t.Fatal("rate limiter did not enforce its configured callback bound")
	}
	if !limiter.Allow("another-client") {
		t.Fatal("one client exhausted a different identity bucket")
	}
}

func TestUT012RepeatedRegistrationKeepsOneOutcome(t *testing.T) {
	t.Parallel()
	cache := newIdempotencyCache()
	digest := sha256.Sum256([]byte(`{"subject":"same"}`))
	if _, _, result := cache.Begin("registration", digest); result != idempotencyNew {
		t.Fatalf("first result = %v, want new", result)
	}
	cache.Complete("registration", digest, cachedResponse{status: http.StatusOK})
	if _, _, result := cache.Begin("registration", digest); result != idempotencyReplay {
		t.Fatalf("repeat result = %v, want replay", result)
	}
}

func TestUT013CallbackWithoutLoginFlowFailsSafely(t *testing.T) {
	t.Parallel()
	if _, _, err := access.DecodeSessionToken("missing-flow"); err == nil {
		t.Fatal("unbound callback-shaped token was accepted")
	}
}

func TestUT019RepeatedApprovalReplaysOriginalRepresentation(t *testing.T) {
	t.Parallel()
	cache := newIdempotencyCache()
	digest := sha256.Sum256([]byte(`{"decision":"approve","role":"viewer"}`))
	_, _, _ = cache.Begin("approval", digest)
	want := cachedResponse{status: http.StatusOK, header: http.Header{"Etag": {`"v2"`}}, body: []byte(`{"status":"active"}`)}
	cache.Complete("approval", digest, want)
	got, _, result := cache.Begin("approval", digest)
	if result != idempotencyReplay || string(got.body) != string(want.body) || got.header.Get("ETag") != `"v2"` {
		t.Fatalf("repeated approval did not replay original response: %#v", got)
	}
}

func TestUT024ConditionalPreferenceVersionsRejectStaleInput(t *testing.T) {
	t.Parallel()
	if version, err := parseETag(`"v7"`); err != nil || version != 7 {
		t.Fatalf("parseETag() = %d, %v, want 7", version, err)
	}
	err := transportError(access.ErrVersionConflict)
	var problem *httpx.ProblemError
	if !errors.As(err, &problem) || problem.Status != http.StatusPreconditionFailed {
		t.Fatalf("stale version maps to %#v, want 412 problem", err)
	}
}

func TestUT026RepeatedDeletionReturnsCompletedOutcome(t *testing.T) {
	t.Parallel()
	cache := newIdempotencyCache()
	digest := sha256.Sum256([]byte(`{"confirmation":"DELETE MY ACCOUNT"}`))
	_, _, _ = cache.Begin("deletion", digest)
	want := cachedResponse{status: http.StatusAccepted, body: []byte(`{"state":"succeeded"}`)}
	cache.Complete("deletion", digest, want)
	got, _, result := cache.Begin("deletion", digest)
	if result != idempotencyReplay || got.status != http.StatusAccepted || string(got.body) != string(want.body) {
		t.Fatal("completed deletion was not replayed safely")
	}
}

func TestUT222ConcurrentBearerRetryProducesOneBusinessOutcome(t *testing.T) {
	t.Parallel()
	cache := newIdempotencyCache()
	digest := sha256.Sum256([]byte("same-command"))
	_, _, _ = cache.Begin("service:44:command", digest)
	_, ready, result := cache.Begin("service:44:command", digest)
	if result != idempotencyWait || ready == nil {
		t.Fatalf("concurrent retry result = %v, want wait", result)
	}
	cache.Complete("service:44:command", digest, cachedResponse{status: http.StatusCreated})
	select {
	case <-ready:
	case <-time.After(time.Second):
		t.Fatal("concurrent retry was not released")
	}
	if got, _, replay := cache.Begin("service:44:command", digest); replay != idempotencyReplay || got.status != http.StatusCreated {
		t.Fatalf("completed service outcome = %v/%d", replay, got.status)
	}
}

func TestResponseRecorderPreservesReplayHeaders(t *testing.T) {
	t.Parallel()
	recorder := newResponseRecorder()
	recorder.Header().Set("ETag", `"v3"`)
	recorder.WriteHeader(http.StatusCreated)
	_, _ = recorder.Write([]byte("created"))
	response := recorder.response()
	w := httptest.NewRecorder()
	response.replay(w)
	if w.Code != http.StatusCreated || w.Header().Get("ETag") != `"v3"` || w.Body.String() != "created" {
		t.Fatalf("replay response = %d %q %q", w.Code, w.Header().Get("ETag"), w.Body.String())
	}
}
