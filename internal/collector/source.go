package collector

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strings"
	"time"
)

var ErrUnsafeSource = errors.New("unsafe source URL")

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type PublicURLPolicy struct {
	Resolver     Resolver
	AllowedPorts map[string]struct{}
	// AllowedHosts limits a policy to an explicit set of canonical host names.
	// A nil or empty set is appropriate for source registration, where any
	// public host may be proposed. Crawlers populate this set from their
	// configured roots so discovered links and redirects cannot expand scope.
	AllowedHosts map[string]struct{}
}

func (policy PublicURLPolicy) Validate(ctx context.Context, raw string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return nil, fmt.Errorf("%w: an absolute credential-free HTTPS URL is required", ErrUnsafeSource)
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, fmt.Errorf("%w: local hosts are forbidden", ErrUnsafeSource)
	}
	if len(policy.AllowedHosts) > 0 {
		if _, allowed := policy.AllowedHosts[host]; !allowed {
			return nil, fmt.Errorf("%w: host is outside the configured source scope", ErrUnsafeSource)
		}
	}
	port := parsed.Port()
	if port != "" && port != "443" {
		if _, allowed := policy.AllowedPorts[port]; !allowed {
			return nil, fmt.Errorf("%w: port %s is forbidden", ErrUnsafeSource, port)
		}
	}
	resolver := policy.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addresses, err := resolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return nil, fmt.Errorf("%w: resolve public host", ErrUnsafeSource)
	}
	for _, address := range addresses {
		if !publicAddress(address.Unmap()) {
			return nil, fmt.Errorf("%w: host resolves to a protected address", ErrUnsafeSource)
		}
	}
	parsed.Scheme = "https"
	parsed.Host = host
	if port != "" {
		parsed.Host = net.JoinHostPort(host, port)
	}
	parsed.Path = path.Clean("/" + strings.TrimPrefix(parsed.EscapedPath(), "/"))
	parsed.RawPath = ""
	return parsed, nil
}

func publicAddress(address netip.Addr) bool {
	return address.IsValid() && !address.IsUnspecified() && !address.IsLoopback() &&
		!address.IsPrivate() && !address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast()
}

// PublicHTTPClient resolves and dials only the validated public addresses. It
// closes the DNS-rebinding gap between URL validation and socket creation.
func PublicHTTPClient(policy PublicURLPolicy, timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	resolver := policy.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid request address", ErrUnsafeSource)
		}
		addresses, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil || len(addresses) == 0 {
			return nil, fmt.Errorf("%w: resolve request host", ErrUnsafeSource)
		}
		for _, candidate := range addresses {
			candidate = candidate.Unmap()
			if !publicAddress(candidate) {
				return nil, fmt.Errorf("%w: request host resolved to a protected address", ErrUnsafeSource)
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
	}
	redirects := RedirectValidator{Policy: policy, Limit: 5}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			previous := make([]*url.URL, 0, len(via))
			for _, prior := range via {
				previous = append(previous, prior.URL)
			}
			return redirects.Check(request.Context(), request.URL, previous)
		},
	}
}

type RedirectValidator struct {
	Policy PublicURLPolicy
	Limit  int
}

func (validator RedirectValidator) Check(ctx context.Context, next *url.URL, via []*url.URL) error {
	limit := validator.Limit
	if limit <= 0 {
		limit = 5
	}
	if len(via) >= limit {
		return fmt.Errorf("%w: redirect limit exceeded", ErrUnsafeSource)
	}
	_, err := validator.Policy.Validate(ctx, next.String())
	return err
}
