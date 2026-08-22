// Package config loads process configuration from environment variables.
//
// Configuration is read once at start up and passed explicitly through
// constructors. There is no mutable global state.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every setting shared by the API and the worker.
type Config struct {
	// Environment is the deployment environment name, for example
	// "development" or "production".
	Environment string

	// HTTPAddress is the listen address of the API. It binds to all
	// interfaces so that Dev Container port forwarding works.
	HTTPAddress string

	// DatabaseURL is the PostgreSQL connection URI. It is never logged.
	DatabaseURL string

	// ShutdownTimeout bounds how long a graceful shutdown may take.
	ShutdownTimeout time.Duration

	// WorkerInterval is how often the worker wakes up.
	WorkerInterval time.Duration

	// WorkerConcurrency bounds how many collection tasks run at once.
	WorkerConcurrency int

	// OTLPEndpoint enables OpenTelemetry export when it is not empty.
	OTLPEndpoint string

	// ServiceName identifies the process in traces and metrics.
	ServiceName string

	// JetStreamEnabled requires durable asynchronous delivery.
	JetStreamEnabled bool

	// NATSURL is the JetStream endpoint. It is never reported by readiness.
	NATSURL string

	// ObjectStorageEnabled requires S3-compatible evidence storage.
	ObjectStorageEnabled bool
	S3Endpoint           string
	S3Bucket             string
	S3AccessKey          string
	S3SecretKey          string

	// ValkeyEnabled enables disposable acceleration state.
	ValkeyEnabled bool
	ValkeyURL     string
}

// Load reads the configuration from lookup, which is normally os.LookupEnv.
//
// It returns an error describing every missing or malformed value at once, so
// that a misconfigured deployment fails fast with a complete report.
func Load(serviceName string, lookup func(string) (string, bool)) (Config, error) {
	var problems []string

	databaseURL, ok := lookup("DATABASE_URL")
	if !ok || strings.TrimSpace(databaseURL) == "" {
		problems = append(problems, "DATABASE_URL is required")
	}

	cfg := Config{
		Environment: stringOrDefault(lookup, "ENVIRONMENT", "development"),
		HTTPAddress: stringOrDefault(lookup, "HTTP_ADDRESS", "0.0.0.0:8100"),
		DatabaseURL: databaseURL,
		ServiceName: serviceName,
		OTLPEndpoint: strings.TrimSpace(
			stringOrDefault(lookup, "OTEL_EXPORTER_OTLP_ENDPOINT", "")),
		NATSURL:     strings.TrimSpace(stringOrDefault(lookup, "NATS_URL", "")),
		S3Endpoint:  strings.TrimSpace(stringOrDefault(lookup, "S3_ENDPOINT", "")),
		S3Bucket:    strings.TrimSpace(stringOrDefault(lookup, "S3_BUCKET", "")),
		S3AccessKey: stringOrDefault(lookup, "S3_ACCESS_KEY", ""),
		S3SecretKey: stringOrDefault(lookup, "S3_SECRET_KEY", ""),
		ValkeyURL:   strings.TrimSpace(stringOrDefault(lookup, "VALKEY_URL", "")),
	}

	var err error

	if cfg.ShutdownTimeout, err = durationOrDefault(lookup, "SHUTDOWN_TIMEOUT", 15*time.Second); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.WorkerInterval, err = durationOrDefault(lookup, "WORKER_INTERVAL", 30*time.Second); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.WorkerConcurrency, err = intOrDefault(lookup, "WORKER_CONCURRENCY", 4); err != nil {
		problems = append(problems, err.Error())
	} else if cfg.WorkerConcurrency < 1 {
		problems = append(problems, "WORKER_CONCURRENCY must be at least 1")
	}
	if cfg.JetStreamEnabled, err = boolOrDefault(lookup, "JETSTREAM_ENABLED", false); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.ObjectStorageEnabled, err = boolOrDefault(lookup, "OBJECT_STORAGE_ENABLED", false); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.ValkeyEnabled, err = boolOrDefault(lookup, "VALKEY_ENABLED", false); err != nil {
		problems = append(problems, err.Error())
	}

	if cfg.JetStreamEnabled && cfg.NATSURL == "" {
		problems = append(problems, "NATS_URL is required when JETSTREAM_ENABLED is true")
	}
	if cfg.ObjectStorageEnabled {
		fields := []struct {
			name  string
			value string
		}{
			{"S3_ENDPOINT", cfg.S3Endpoint},
			{"S3_BUCKET", cfg.S3Bucket},
			{"S3_ACCESS_KEY", cfg.S3AccessKey},
			{"S3_SECRET_KEY", cfg.S3SecretKey},
		}
		for _, field := range fields {
			if strings.TrimSpace(field.value) == "" {
				problems = append(problems, field.name+" is required when OBJECT_STORAGE_ENABLED is true")
			}
		}
	}
	if cfg.ValkeyEnabled && cfg.ValkeyURL == "" {
		problems = append(problems, "VALKEY_URL is required when VALKEY_ENABLED is true")
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(problems, "; "))
	}

	return cfg, nil
}

// LoadFromEnvironment is the production entry point of Load.
func LoadFromEnvironment(serviceName string) (Config, error) {
	return Load(serviceName, os.LookupEnv)
}

// TelemetryEnabled reports whether traces and metrics should be exported.
func (c Config) TelemetryEnabled() bool {
	return c.OTLPEndpoint != ""
}

func stringOrDefault(lookup func(string) (string, bool), key, fallback string) string {
	if value, ok := lookup(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func durationOrDefault(
	lookup func(string) (string, bool),
	key string,
	fallback time.Duration,
) (time.Duration, error) {
	raw, ok := lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 30s: %w", key, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}

	return value, nil
}

func intOrDefault(lookup func(string) (string, bool), key string, fallback int) (int, error) {
	raw, ok := lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}

	return value, nil
}

func boolOrDefault(
	lookup func(string) (string, bool),
	key string,
	fallback bool,
) (bool, error) {
	raw, ok := lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return value, nil
}
