//go:build integration

package health_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/health"
)

func TestIT135ReadinessMatrix(t *testing.T) {
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for healthy dependency: %v", err)
	}
	defer tcpListener.Close()
	go func() {
		for {
			connection, acceptErr := tcpListener.Accept()
			if acceptErr != nil {
				return
			}
			connection.Close()
		}
	}()
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer httpServer.Close()

	ctx := context.Background()
	availableTCP := "tcp://" + tcpListener.Addr().String()
	unavailableTCP := "tcp://127.0.0.1:1"
	state := func(err error) health.State {
		if err != nil {
			return health.Unavailable
		}
		return health.Available
	}
	available := func() []health.Dependency {
		return []health.Dependency{
			{Name: "database", Importance: health.Required, State: state(health.TCP(ctx, availableTCP))},
			{Name: "jetstream", Importance: health.Required, State: state(health.TCP(ctx, availableTCP))},
			{Name: "object_storage", Importance: health.Required, State: state(health.HTTP(ctx, httpServer.URL))},
			{Name: "valkey", Importance: health.Optional, State: state(health.TCP(ctx, availableTCP))},
		}
	}

	tests := []struct {
		name       string
		dependency int
		wantReady  bool
		wantState  health.State
	}{
		{name: "database required", dependency: 0, wantReady: false, wantState: health.Unavailable},
		{name: "JetStream required", dependency: 1, wantReady: false, wantState: health.Unavailable},
		{name: "object storage required", dependency: 2, wantReady: false, wantState: health.Unavailable},
		{name: "Valkey optional", dependency: 3, wantReady: true, wantState: health.Degraded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dependencies := available()
			if tc.dependency == 2 {
				dependencies[tc.dependency].State = state(health.HTTP(ctx, "http://127.0.0.1:1"))
			} else {
				dependencies[tc.dependency].State = state(health.TCP(ctx, unavailableTCP))
			}
			report := health.Evaluate(dependencies)
			if report.Ready != tc.wantReady {
				t.Errorf("Ready = %t, want %t", report.Ready, tc.wantReady)
			}
			name := dependencies[tc.dependency].Name
			if report.Dependencies[name] != tc.wantState {
				t.Errorf("%s state = %q, want %q", name, report.Dependencies[name], tc.wantState)
			}
		})
	}
}
