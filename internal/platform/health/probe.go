package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
)

// TCP probes an endpoint without returning its potentially sensitive address in errors.
func TCP(ctx context.Context, endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("dependency endpoint is invalid")
	}
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		return fmt.Errorf("dependency is unavailable")
	}
	return connection.Close()
}

// HTTP probes a safe health URL and reports no response content or endpoint.
func HTTP(ctx context.Context, endpoint string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("dependency endpoint is invalid")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("dependency is unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("dependency health check failed")
	}
	return nil
}
