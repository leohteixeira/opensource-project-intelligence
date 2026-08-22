package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/config"
)

func lookupFrom(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("api", lookupFrom(map[string]string{
		"DATABASE_URL": "postgres://user:password@localhost:5433/db",
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	if cfg.HTTPAddress != "0.0.0.0:8100" {
		t.Errorf("HTTPAddress = %q, want the wildcard bind on the assigned port", cfg.HTTPAddress)
	}
	if cfg.WorkerConcurrency != 4 {
		t.Errorf("WorkerConcurrency = %d, want 4", cfg.WorkerConcurrency)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %s, want 15s", cfg.ShutdownTimeout)
	}
	if cfg.TelemetryEnabled() {
		t.Error("TelemetryEnabled() = true, want false when no OTLP endpoint is configured")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		env  map[string]string
		want string
	}{
		"missing database url": {
			env:  map[string]string{},
			want: "DATABASE_URL is required",
		},
		"blank database url": {
			env:  map[string]string{"DATABASE_URL": "   "},
			want: "DATABASE_URL is required",
		},
		"malformed duration": {
			env: map[string]string{
				"DATABASE_URL":    "postgres://user:password@localhost:5433/db",
				"WORKER_INTERVAL": "soon",
			},
			want: "WORKER_INTERVAL must be a duration",
		},
		"non positive duration": {
			env: map[string]string{
				"DATABASE_URL":     "postgres://user:password@localhost:5433/db",
				"SHUTDOWN_TIMEOUT": "0s",
			},
			want: "SHUTDOWN_TIMEOUT must be positive",
		},
		"malformed concurrency": {
			env: map[string]string{
				"DATABASE_URL":       "postgres://user:password@localhost:5433/db",
				"WORKER_CONCURRENCY": "many",
			},
			want: "WORKER_CONCURRENCY must be an integer",
		},
		"zero concurrency": {
			env: map[string]string{
				"DATABASE_URL":       "postgres://user:password@localhost:5433/db",
				"WORKER_CONCURRENCY": "0",
			},
			want: "WORKER_CONCURRENCY must be at least 1",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := config.Load("api", lookupFrom(tc.env))
			if err == nil {
				t.Fatal("Load() returned no error, want a validation failure")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load() error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestLoadReportsEveryProblemAtOnce(t *testing.T) {
	t.Parallel()

	_, err := config.Load("api", lookupFrom(map[string]string{
		"WORKER_CONCURRENCY": "0",
		"WORKER_INTERVAL":    "soon",
	}))
	if err == nil {
		t.Fatal("Load() returned no error, want a validation failure")
	}

	for _, want := range []string{"DATABASE_URL", "WORKER_INTERVAL", "WORKER_CONCURRENCY"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %q, want it to mention %q", err, want)
		}
	}
}

func TestUT269LoadRedactsConditionalConfiguration(t *testing.T) {
	t.Parallel()

	const secret = "do-not-report-this-secret"
	_, err := config.Load("api", lookupFrom(map[string]string{
		"DATABASE_URL":           "postgres://user:" + secret + "@localhost:5433/db",
		"JETSTREAM_ENABLED":      "true",
		"OBJECT_STORAGE_ENABLED": "true",
		"S3_ACCESS_KEY":          "access-key",
		"S3_SECRET_KEY":          secret,
		"VALKEY_ENABLED":         "true",
	}))
	if err == nil {
		t.Fatal("Load() returned no error, want conditional validation failures")
	}

	for _, field := range []string{"NATS_URL", "S3_ENDPOINT", "S3_BUCKET", "VALKEY_URL"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("Load() error = %q, want safe field %q", err, field)
		}
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "access-key") {
		t.Fatalf("Load() error exposed a configuration value: %q", err)
	}
}
