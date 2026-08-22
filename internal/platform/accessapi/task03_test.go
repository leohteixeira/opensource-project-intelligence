package accessapi

import (
	"net/http"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
)

// IT-139: unsafe cookie mutations require the full same-origin defense set.
func TestIT139CookieMutationsRequireSameOriginFetchMetadataAndCSRF(t *testing.T) {
	t.Parallel()
	csrf, hash, err := access.NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	handler := Handler{config: Config{PublicBaseURL: "https://opi.example"}}
	session := accessstore.Session{CSRFHash: hash}
	request, err := http.NewRequest(http.MethodPost, "https://opi.example/api/v1/projects", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*http.Request){
		func(*http.Request) {},
		func(request *http.Request) { request.Header.Set("Origin", "https://attacker.example") },
		func(request *http.Request) {
			request.Header.Set("Origin", "https://opi.example")
			request.Header.Set("Sec-Fetch-Site", "cross-site")
		},
		func(request *http.Request) {
			request.Header.Set("Origin", "https://opi.example")
			request.Header.Set("Sec-Fetch-Site", "same-origin")
			request.Header.Set("X-CSRF-Token", "wrong")
		},
	} {
		candidate := request.Clone(request.Context())
		mutate(candidate)
		if err := handler.validateBrowserMutation(candidate, session); err == nil {
			t.Fatal("unsafe browser mutation passed a missing or mismatched origin/CSRF check")
		}
	}
	request.Header.Set("Origin", "https://opi.example")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("X-CSRF-Token", csrf)
	if err := handler.validateBrowserMutation(request, session); err != nil {
		t.Fatalf("valid same-origin browser mutation failed: %v", err)
	}
}
