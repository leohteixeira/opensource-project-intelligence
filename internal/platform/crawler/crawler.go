// Package crawler fetches public documentation through per-hop safety and hard crawl limits.
package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
	"github.com/leohteixeira/opensource-project-intelligence/internal/knowledge"
)

var ErrRobotsDenied = errors.New("crawler: robots policy denied the URL")

type Robots interface {
	Allowed(context.Context, *url.URL) (bool, error)
}

type AllowRobots struct{}

func (AllowRobots) Allowed(context.Context, *url.URL) (bool, error) { return true, nil }

type Page struct {
	URL        string
	Depth      int
	MediaType  string
	Body       []byte
	ObservedAt time.Time
	Links      []string
}

type Crawler struct {
	Policy collector.PublicURLPolicy
	HTTP   *http.Client
	Robots Robots
	Limits knowledge.Limits
	Now    func() time.Time
}

// Crawl visits a bounded public graph. Each queued URL is revalidated immediately before I/O and
// the configured HTTP client validates DNS and every redirect hop again while dialing.
func (crawler Crawler) Crawl(ctx context.Context, roots []string) ([]Page, error) {
	if err := crawler.Limits.Validate(); err != nil || len(roots) == 0 {
		return nil, knowledge.ErrInvalid
	}
	policy, canonicalRoots, err := crawlPolicy(ctx, crawler.Policy, roots)
	if err != nil {
		return nil, err
	}
	client := crawler.HTTP
	if client == nil {
		client = collector.PublicHTTPClient(policy, 30*time.Second)
	} else {
		copy := *client
		previousCheck := copy.CheckRedirect
		redirects := collector.RedirectValidator{Policy: policy, Limit: 5}
		copy.CheckRedirect = func(request *http.Request, via []*http.Request) error {
			previous := make([]*url.URL, 0, len(via))
			for _, prior := range via {
				previous = append(previous, prior.URL)
			}
			if err := redirects.Check(request.Context(), request.URL, previous); err != nil {
				return err
			}
			if previousCheck != nil {
				return previousCheck(request, via)
			}
			return nil
		}
		client = &copy
	}
	robots := crawler.Robots
	if robots == nil {
		robots = AllowRobots{}
	}
	now := crawler.Now
	if now == nil {
		now = time.Now
	}
	type queued struct {
		raw   string
		depth int
	}
	queue := make([]queued, 0, len(canonicalRoots))
	for _, root := range canonicalRoots {
		queue = append(queue, queued{raw: root})
	}
	seen := make(map[string]struct{})
	budget := knowledge.Budget{Limits: crawler.Limits}
	pages := make([]Page, 0, min(len(roots), crawler.Limits.MaxPages))
	requestInterval := time.Minute / time.Duration(crawler.Limits.RequestsPerMinute)
	firstRequest := true
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return pages, err
		}
		next := queue[0]
		queue = queue[1:]
		validated, err := policy.Validate(ctx, next.raw)
		if err != nil {
			return pages, fmt.Errorf("validate crawl hop: %w", err)
		}
		key := validated.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		allowed, err := robots.Allowed(ctx, validated)
		if err != nil {
			return pages, fmt.Errorf("read robots policy: %w", err)
		}
		if !allowed {
			return pages, ErrRobotsDenied
		}
		if !firstRequest {
			timer := time.NewTimer(requestInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return pages, ctx.Err()
			case <-timer.C:
			}
		}
		firstRequest = false
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, key, nil)
		if err != nil {
			return pages, fmt.Errorf("build crawl request: %w", err)
		}
		request.Header.Set("Accept", strings.Join(crawler.Limits.MediaTypes, ", "))
		response, err := client.Do(request)
		if err != nil {
			return pages, fmt.Errorf("fetch crawl page: %w", err)
		}
		page, readErr := readPage(response, validated, next.depth, crawler.Limits.MaxPageBytes, now())
		if readErr != nil {
			return pages, readErr
		}
		if err := budget.Accept(validated.Hostname(), next.depth, int64(len(page.Body)), page.MediaType); err != nil {
			return pages, err
		}
		pages = append(pages, page)
		if next.depth >= crawler.Limits.MaxDepth {
			continue
		}
		for _, link := range page.Links {
			queue = append(queue, queued{raw: link, depth: next.depth + 1})
		}
	}
	return pages, nil
}

// crawlPolicy freezes the configured roots as the only eligible domain set.
// A page may link deeply within those domains, but discovery never authorizes
// another host merely because a fetched document linked to it.
func crawlPolicy(ctx context.Context, base collector.PublicURLPolicy,
	roots []string) (collector.PublicURLPolicy, []string, error) {
	allowedHosts := make(map[string]struct{}, len(roots))
	canonicalRoots := make([]string, 0, len(roots))
	for _, root := range roots {
		validated, err := base.Validate(ctx, root)
		if err != nil {
			return collector.PublicURLPolicy{}, nil, fmt.Errorf("validate crawl root: %w", err)
		}
		allowedHosts[validated.Hostname()] = struct{}{}
		canonicalRoots = append(canonicalRoots, validated.String())
	}
	base.AllowedHosts = allowedHosts
	return base, canonicalRoots, nil
}

func readPage(response *http.Response, base *url.URL, depth int, maxBytes int64, observedAt time.Time) (Page, error) {
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Page{}, fmt.Errorf("crawler: upstream status %d", response.StatusCode)
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	reader := io.LimitReader(response.Body, maxBytes+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return Page{}, fmt.Errorf("read crawl response: %w", err)
	}
	if int64(len(body)) > maxBytes {
		return Page{}, knowledge.ErrLimitExceeded
	}
	page := Page{URL: base.String(), Depth: depth, MediaType: mediaType, Body: body, ObservedAt: observedAt.UTC()}
	if mediaType == "text/html" {
		page.Links = links(base, body)
	}
	return page, nil
}

func links(base *url.URL, body []byte) []string {
	document, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil
	}
	values, seen := make([]string, 0), make(map[string]struct{})
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "a" {
			for _, attribute := range node.Attr {
				if attribute.Key != "href" {
					continue
				}
				parsed, err := url.Parse(strings.TrimSpace(attribute.Val))
				if err != nil {
					continue
				}
				resolved := base.ResolveReference(parsed)
				resolved.Fragment = ""
				if resolved.Scheme != "https" {
					continue
				}
				value := resolved.String()
				if _, exists := seen[value]; !exists {
					seen[value] = struct{}{}
					values = append(values, value)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	return values
}
