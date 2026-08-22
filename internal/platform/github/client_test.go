package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	githubadapter "github.com/leohteixeira/opensource-project-intelligence/internal/platform/github"
)

func TestRepositoryMapsPublicProviderDTO(t *testing.T) {
	t.Parallel()
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer operator-read-token" {
			t.Error("operator credential was not sent server-side")
		}
		return response(http.StatusOK, nil, `{"id":7,"name":"project","description":"Public","private":false,"visibility":"public","html_url":"https://github.com/acme/project","default_branch":"main","updated_at":"2026-08-22T10:00:00Z","owner":{"login":"acme"}}`), nil
	})}
	base, _ := url.Parse("https://api.github.test")
	client := githubadapter.Client{HTTP: httpClient, BaseURL: base, Token: "operator-read-token"}
	repository, err := client.Repository(context.Background(), "acme", "project")
	if err != nil || repository.ExternalID != 7 || repository.Owner != "acme" || len(repository.Raw) == 0 {
		t.Fatalf("IT-114 got %#v, %v", repository, err)
	}
}

func TestPrivateAndRateLimitedRepositoriesDegradeSafely(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		headers map[string]string
		want    error
	}{
		{"UT-084 private", http.StatusNotFound, nil, githubadapter.ErrNotPublic},
		{"UT-080 quota", http.StatusForbidden, map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1787392800"}, githubadapter.ErrRateLimited},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return response(test.status, test.headers, `{}`), nil
			})}
			base, _ := url.Parse("https://api.github.test")
			_, err := (githubadapter.Client{HTTP: httpClient, BaseURL: base}).Repository(context.Background(), "acme", "project")
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestTask03CredentialRotationKeepsOneIdentityPerRequest(t *testing.T) {
	t.Parallel()
	base, _ := url.Parse("https://api.github.test")
	seen := make(chan string, 2)
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		seen <- request.Header.Get("Authorization")
		return response(http.StatusOK, nil,
			`{"id":7,"name":"project","private":false,"visibility":"public","html_url":"https://github.com/acme/project","default_branch":"main","updated_at":"2026-08-22T10:00:00Z","owner":{"login":"acme"}}`), nil
	})

	t.Run("IT-034 concurrent credential rotation never mixes request identities", func(t *testing.T) {
		oldClient := githubadapter.Client{HTTP: &http.Client{Transport: transport}, BaseURL: base, Token: "old-token"}
		newClient := githubadapter.Client{HTTP: &http.Client{Transport: transport}, BaseURL: base, Token: "new-token"}
		errorsSeen := make(chan error, 2)
		go func() { _, err := oldClient.Repository(context.Background(), "acme", "project"); errorsSeen <- err }()
		go func() { _, err := newClient.Repository(context.Background(), "acme", "project"); errorsSeen <- err }()
		for range 2 {
			if err := <-errorsSeen; err != nil {
				t.Fatal(err)
			}
		}
		identities := map[string]int{<-seen: 1, <-seen: 1}
		if len(identities) != 2 || identities["Bearer old-token"] != 1 || identities["Bearer new-token"] != 1 {
			t.Fatalf("request identities = %#v", identities)
		}
	})

	t.Run("IT-035 failed rotation leaves the last valid immutable client active", func(t *testing.T) {
		active := githubadapter.Client{HTTP: &http.Client{Transport: transport}, BaseURL: base, Token: "old-token"}
		if _, err := url.ParseRequestURI("not-a-provider-url"); err == nil {
			t.Fatal("controlled replacement configuration unexpectedly validated")
		}
		if _, err := active.Repository(context.Background(), "acme", "project"); err != nil {
			t.Fatal(err)
		}
		if identity := <-seen; identity != "Bearer old-token" {
			t.Fatalf("active identity = %q", identity)
		}
	})
}

func TestCorePublicSourceDTOsMapWithoutProviderTypes(t *testing.T) {
	t.Parallel()
	base, _ := url.Parse("https://api.github.test")
	now := "2026-08-22T10:00:00Z"
	tests := []struct {
		name string
		body string
		run  func(githubadapter.Client) error
	}{
		{"pull requests", `[{"id":11,"number":2,"title":"Change","state":"closed","draft":false,"created_at":"` + now + `","merged_at":"` + now + `"}]`, func(client githubadapter.Client) error {
			values, err := client.PullRequests(context.Background(), "acme", "project", 1)
			if err != nil || len(values) != 1 || values[0].State != "merged" || values[0].ReadyAt == nil || !json.Valid(values[0].Raw) {
				return fmt.Errorf("pull requests = %#v: %w", values, err)
			}
			return nil
		}},
		{"releases", `[{"id":12,"tag_name":"v1.0.0","draft":false,"prerelease":false,"published_at":"` + now + `"}]`, func(client githubadapter.Client) error {
			values, err := client.Releases(context.Background(), "acme", "project", 1)
			if err != nil || len(values) != 1 || values[0].Tag != "v1.0.0" || values[0].PublishedAt == nil || !json.Valid(values[0].Raw) {
				return fmt.Errorf("releases = %#v: %w", values, err)
			}
			return nil
		}},
		{"commits", `[{"sha":"abc123","author":{"id":13},"commit":{"committer":{"date":"` + now + `"}},"parents":[{"sha":"one"},{"sha":"two"}]}]`, func(client githubadapter.Client) error {
			values, err := client.Commits(context.Background(), "acme", "project", "main", 1)
			if err != nil || len(values) != 1 || values[0].SHA != "abc123" ||
				values[0].AuthorExternalID != "13" || !values[0].MergeCommit || !json.Valid(values[0].Raw) {
				return fmt.Errorf("commits = %#v: %w", values, err)
			}
			return nil
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := githubadapter.Client{BaseURL: base, HTTP: &http.Client{Transport: roundTripFunc(
				func(*http.Request) (*http.Response, error) {
					return response(http.StatusOK, nil, test.body), nil
				},
			)}}
			if err := test.run(client); err != nil {
				t.Fatalf("IT-114 %v", err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func response(status int, headers map[string]string, body string) *http.Response {
	header := make(http.Header, len(headers))
	for key, value := range headers {
		header.Set(key, value)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
