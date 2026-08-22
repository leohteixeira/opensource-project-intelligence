package accessapi

import (
	"crypto/sha256"
	"net/http"
	"sync"
)

type idempotencyResult int

const (
	idempotencyNew idempotencyResult = iota
	idempotencyReplay
	idempotencyConflict
)

type cachedResponse struct {
	status int
	header http.Header
	body   []byte
}

func (r cachedResponse) replay(w http.ResponseWriter) {
	for key, values := range r.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(r.status)
	_, _ = w.Write(r.body)
}

type idempotencyEntry struct {
	digest   [sha256.Size]byte
	response cachedResponse
	complete bool
}

type idempotencyCache struct {
	mu      sync.Mutex
	entries map[string]idempotencyEntry
}

func newIdempotencyCache() *idempotencyCache {
	return &idempotencyCache{entries: make(map[string]idempotencyEntry)}
}

func (c *idempotencyCache) Begin(key string, digest [sha256.Size]byte) (cachedResponse, idempotencyResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, exists := c.entries[key]
	if !exists {
		c.entries[key] = idempotencyEntry{digest: digest}
		return cachedResponse{}, idempotencyNew
	}
	if entry.digest != digest || !entry.complete {
		return cachedResponse{}, idempotencyConflict
	}
	return entry.response, idempotencyReplay
}

func (c *idempotencyCache) Complete(key string, digest [sha256.Size]byte, response cachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = idempotencyEntry{digest: digest, response: response, complete: true}
}

type responseRecorder struct {
	header http.Header
	status int
	body   []byte
}

func newResponseRecorder() *responseRecorder {
	return &responseRecorder{header: make(http.Header), status: http.StatusOK}
}

func (r *responseRecorder) Header() http.Header { return r.header }

func (r *responseRecorder) WriteHeader(status int) { r.status = status }

func (r *responseRecorder) Write(body []byte) (int, error) {
	r.body = append(r.body, body...)
	return len(body), nil
}

func (r *responseRecorder) response() cachedResponse {
	return cachedResponse{status: r.status, header: r.header.Clone(), body: append([]byte(nil), r.body...)}
}
