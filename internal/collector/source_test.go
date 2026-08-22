package collector_test

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
)

type resolver map[string][]netip.Addr

func (values resolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	addresses, found := values[host]
	if !found {
		return nil, errors.New("not found")
	}
	return addresses, nil
}

func TestPublicURLPolicy(t *testing.T) {
	t.Parallel()
	policy := collector.PublicURLPolicy{Resolver: resolver{
		"github.com":      {netip.MustParseAddr("140.82.112.4")},
		"private.example": {netip.MustParseAddr("10.0.0.5")},
		"link.example":    {netip.MustParseAddr("169.254.169.254")},
	}}
	tests := []struct {
		id      string
		url     string
		wantErr bool
	}{
		{"UT-036 valid public repository URL", "https://github.com/acme/project", false},
		{"UT-254 private DNS is rejected", "https://private.example/project", true},
		{"UT-254 link-local metadata is rejected", "https://link.example/latest/meta-data", true},
		{"UT-254 credentials are rejected", "https://token@github.com/acme/project", true},
		{"UT-254 unsafe scheme is rejected", "file:///etc/passwd", true},
	}
	for _, test := range tests {
		t.Run(test.id, func(t *testing.T) {
			_, err := policy.Validate(context.Background(), test.url)
			if (err != nil) != test.wantErr {
				t.Fatalf("got %v", err)
			}
		})
	}
}
