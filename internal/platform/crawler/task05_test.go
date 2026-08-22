package crawler

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
)

type task05Resolver map[string][]netip.Addr

func (resolver task05Resolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	return resolver[host], nil
}

func TestUT155UnsupportedUnsafeAndOutOfScopeURLsAreRejected(t *testing.T) {
	policy := collector.PublicURLPolicy{Resolver: task05Resolver{
		"public.example":  {netip.MustParseAddr("203.0.113.10")},
		"outside.example": {netip.MustParseAddr("203.0.113.11")},
		"private.example": {netip.MustParseAddr("127.0.0.1")},
	}, AllowedHosts: map[string]struct{}{"public.example": {}}}
	for _, raw := range []string{"http://public.example", "https://private.example", "https://localhost",
		"https://outside.example"} {
		if _, err := policy.Validate(context.Background(), raw); !errors.Is(err, collector.ErrUnsafeSource) {
			t.Fatalf("%q error = %v", raw, err)
		}
	}
}

func TestCrawlPolicyFreezesConfiguredRootDomains(t *testing.T) {
	base := collector.PublicURLPolicy{Resolver: task05Resolver{
		"docs.example": {netip.MustParseAddr("203.0.113.10")},
		"help.example": {netip.MustParseAddr("203.0.113.11")},
		"cdn.example":  {netip.MustParseAddr("203.0.113.12")},
	}}
	policy, roots, err := crawlPolicy(context.Background(), base,
		[]string{"https://docs.example/start", "https://help.example/guide"})
	if err != nil || len(roots) != 2 {
		t.Fatalf("crawl scope = %v, roots=%v", err, roots)
	}
	if _, err := policy.Validate(context.Background(), "https://docs.example/next"); err != nil {
		t.Fatalf("configured domain rejected: %v", err)
	}
	if _, err := policy.Validate(context.Background(), "https://cdn.example/asset"); !errors.Is(err, collector.ErrUnsafeSource) {
		t.Fatalf("discovered domain expanded scope: %v", err)
	}
}

func TestCrawlerReadPageEnforcesBytesTypesAndLinks(t *testing.T) {
	base, _ := url.Parse("https://docs.example/start")
	response := &http.Response{StatusCode: http.StatusOK, Header: http.Header{
		"Content-Type": []string{"text/html; charset=utf-8"}},
		Body: ioCloser{Reader: strings.NewReader(`<a href="/next#part">next</a><a href="http://unsafe.example">unsafe</a>`)}}
	page, err := readPage(response, base, 0, 1_024, time.Now())
	if err != nil || len(page.Links) != 1 || page.Links[0] != "https://docs.example/next" {
		t.Fatalf("page = %#v, err=%v", page, err)
	}
	oversized := &http.Response{StatusCode: http.StatusOK, Header: http.Header{
		"Content-Type": []string{"text/plain"}}, Body: ioCloser{Reader: strings.NewReader("12345")}}
	if _, err := readPage(oversized, base, 0, 4, time.Now()); !errors.Is(err, knowledge.ErrLimitExceeded) {
		t.Fatalf("oversized error = %v", err)
	}
}

type ioCloser struct{ *strings.Reader }

func (ioCloser) Close() error { return nil }
