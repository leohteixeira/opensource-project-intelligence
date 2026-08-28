package config_test

import (
	"reflect"
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
	if cfg.AIConcurrency != 4 || cfg.ADKMaxSteps != 12 || cfg.ADKTimeout != 2*time.Minute ||
		cfg.ADKMaxOutputBytes != 65536 || cfg.ADKMaxCostMicros != 100000 ||
		cfg.ADKToolConcurrency != 1 || cfg.ExportConcurrency != 2 {
		t.Errorf("bounded operation defaults = %#v", cfg)
	}
	if cfg.TelemetryEnabled() {
		t.Error("TelemetryEnabled() = true, want false when no OTLP endpoint is configured")
	}
}

func TestLoadReadsGitHubTokenOnce(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("worker", lookupFrom(map[string]string{
		"DATABASE_URL": "postgres://user:password@localhost:5433/db",
		"GITHUB_TOKEN": "configured-token",
	}))
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}
	if cfg.GitHubToken != "configured-token" {
		t.Errorf("GitHubToken = %q, want the configured value", cfg.GitHubToken)
	}
}

func TestTask05ModelIdentityIsOperatorConfiguredAsOnePair(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("api", lookupFrom(map[string]string{
		"DATABASE_URL": "postgres://user:password@localhost:5433/db",
		"AI_PROVIDER":  "openai-compatible",
		"AI_MODEL":     "local-analysis-v1",
	}))
	if err != nil || cfg.AIProvider != "openai-compatible" || cfg.AIModel != "local-analysis-v1" {
		t.Fatalf("model identity = %#v, error=%v", cfg, err)
	}
	_, err = config.Load("api", lookupFrom(map[string]string{
		"DATABASE_URL": "postgres://user:password@localhost:5433/db",
		"AI_PROVIDER":  "openai-compatible",
	}))
	if err == nil || !strings.Contains(err.Error(), "AI_PROVIDER and AI_MODEL") {
		t.Fatalf("partial model identity error = %v", err)
	}
}

func TestTask03ProviderCredentialBoundary(t *testing.T) {
	t.Parallel()

	t.Run("UT-078 malformed required provider configuration fails without credential echo", func(t *testing.T) {
		const secret = "provider-secret-must-not-appear"
		_, err := config.Load("worker", lookupFrom(map[string]string{
			"DATABASE_URL":      "postgres://user:password@localhost:5433/db",
			"NATS_URL":          "not-a-url-" + secret,
			"JETSTREAM_ENABLED": "true",
		}))
		if err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("provider configuration error = %q", err)
		}
	})

	t.Run("UT-079 optional GitHub credential preserves anonymous operation", func(t *testing.T) {
		cfg, err := config.Load("worker", lookupFrom(map[string]string{
			"DATABASE_URL": "postgres://user:password@localhost:5433/db",
		}))
		if err != nil || cfg.GitHubToken != "" {
			t.Fatalf("anonymous provider configuration = %#v, %v", cfg, err)
		}
	})

	t.Run("UT-081 end-user credentials are absent from process configuration", func(t *testing.T) {
		typeOf := reflect.TypeOf(config.Config{})
		for index := range typeOf.NumField() {
			name := strings.ToLower(typeOf.Field(index).Name)
			if strings.Contains(name, "usercredential") || strings.Contains(name, "membertoken") {
				t.Fatalf("process configuration exposes end-user credential field %q", name)
			}
		}
	})

	t.Run("UT-082 UT-083 provider credential is read once before collection becomes active", func(t *testing.T) {
		reads := 0
		lookup := func(key string) (string, bool) {
			if key == "GITHUB_TOKEN" {
				reads++
				return "validated-at-startup", true
			}
			if key == "DATABASE_URL" {
				return "postgres://user:password@localhost:5433/db", true
			}
			return "", false
		}
		cfg, err := config.Load("worker", lookup)
		if err != nil || cfg.GitHubToken == "" || reads != 1 {
			t.Fatalf("credential reads = %d, config=%#v, error=%v", reads, cfg, err)
		}
	})
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
		"unbounded agent timeout": {
			env: map[string]string{
				"DATABASE_URL": "postgres://user:password@localhost:5433/db",
				"ADK_TIMEOUT":  "11m",
			},
			want: "ADK_TIMEOUT must not exceed 10m",
		},
		"unbounded agent steps": {
			env: map[string]string{
				"DATABASE_URL":  "postgres://user:password@localhost:5433/db",
				"ADK_MAX_STEPS": "65",
			},
			want: "ADK_MAX_STEPS must be between 1 and 64",
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
