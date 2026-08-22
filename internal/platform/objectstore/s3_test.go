package objectstore

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestS3StagesVerifiesPromotesAndDeletes(t *testing.T) {
	t.Parallel()
	objects := map[string][]byte{}
	var mutex sync.Mutex
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header),
			Body: io.NopCloser(strings.NewReader("")), Request: request}
		if !strings.HasPrefix(request.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
			response.StatusCode = http.StatusUnauthorized
			return response, nil
		}
		mutex.Lock()
		defer mutex.Unlock()
		switch request.Method {
		case http.MethodPut:
			if source := request.Header.Get("X-Amz-Copy-Source"); source != "" {
				body, exists := objects[source]
				if !exists {
					response.StatusCode = http.StatusNotFound
					return response, nil
				}
				objects[request.URL.Path] = append([]byte(nil), body...)
				return response, nil
			}
			body, _ := io.ReadAll(request.Body)
			objects[request.URL.Path] = body
		case http.MethodGet:
			body, exists := objects[request.URL.Path]
			if !exists {
				response.StatusCode = http.StatusNotFound
				return response, nil
			}
			response.Body = io.NopCloser(strings.NewReader(string(body)))
		case http.MethodDelete:
			delete(objects, request.URL.Path)
			response.StatusCode = http.StatusNoContent
		}
		return response, nil
	})}

	store, err := NewS3(Config{Endpoint: "https://s3.example", Bucket: "evidence", AccessKey: "key",
		SecretKey: "secret", HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Stage(ctx, "staged/value", []byte("proof"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	if err := store.Promote(ctx, "staged/value", "projects/1/value"); err != nil {
		t.Fatal(err)
	}
	body, err := store.Read(ctx, "projects/1/value")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "proof" {
		t.Fatalf("unexpected promoted body %q", body)
	}
	if _, exists := objects["/evidence/staged/value"]; exists {
		t.Fatal("staged object remained visible after promotion")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestS3RejectsTraversalKeys(t *testing.T) {
	t.Parallel()
	store, err := NewS3(Config{Endpoint: "https://s3.example", Bucket: "evidence",
		AccessKey: "key", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Stage(context.Background(), "../secret", nil, "text/plain"); err == nil {
		t.Fatal("expected traversal key to be rejected")
	}
}
