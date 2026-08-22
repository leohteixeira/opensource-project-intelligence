// Command api serves the versioned HTTP contract consumed by the web
// application.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/config"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/health"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/telemetry"
)

const serviceName = "opensource-project-intelligence-api"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(logger); err != nil {
		logger.Error("the api stopped with an error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadFromEnvironment(serviceName)
	if err != nil {
		return err
	}

	shutdownTelemetry, err := telemetry.Setup(ctx, cfg.ServiceName, cfg.Environment, cfg.OTLPEndpoint)
	if err != nil {
		return err
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	server := httpx.NewServer(cfg.HTTPAddress, otelhttp.NewHandler(routes(logger, pool, cfg), "api"))

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("the api is listening", slog.String("address", cfg.HTTPAddress))

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
		close(serverErrors)
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	return errors.Join(
		server.Shutdown(shutdownCtx),
		shutdownTelemetry(shutdownCtx),
	)
}

func routes(logger *slog.Logger, pool *database.Pool, cfg config.Config) http.Handler {
	mux := http.NewServeMux()

	// Liveness: the process is up. It never touches a dependency.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, logger, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Readiness: the process can serve traffic, which requires the database.
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dependencies := []health.Dependency{{
			Name: "postgresql", Importance: health.Required, State: dependencyState(pool.Ping(ctx)),
		}}
		if cfg.JetStreamEnabled {
			dependencies = append(dependencies, health.Dependency{
				Name: "jetstream", Importance: health.Required, State: dependencyState(health.TCP(ctx, cfg.NATSURL)),
			})
		} else {
			dependencies = append(dependencies, health.Dependency{Name: "jetstream", Importance: health.Optional, State: health.Disabled})
		}
		if cfg.ObjectStorageEnabled {
			dependencies = append(dependencies, health.Dependency{
				Name: "object_storage", Importance: health.Required, State: dependencyState(health.HTTP(ctx, s3HealthURL(cfg.S3Endpoint))),
			})
		} else {
			dependencies = append(dependencies, health.Dependency{Name: "object_storage", Importance: health.Optional, State: health.Disabled})
		}
		if cfg.ValkeyEnabled {
			dependencies = append(dependencies, health.Dependency{
				Name: "valkey", Importance: health.Optional, State: dependencyState(health.TCP(ctx, cfg.ValkeyURL)),
			})
		} else {
			dependencies = append(dependencies, health.Dependency{Name: "valkey", Importance: health.Optional, State: health.Disabled})
		}

		report := health.Evaluate(dependencies)
		status := http.StatusOK
		if !report.Ready {
			status = http.StatusServiceUnavailable
		}
		httpx.WriteJSON(w, logger, status, report)
	})

	// Versioned contract surface. Handlers are registered per capability as
	// they land.
	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteJSON(w, logger, http.StatusNotFound, map[string]string{
			"error": "this endpoint does not exist yet",
		})
	})

	return mux
}

func dependencyState(err error) health.State {
	if err != nil {
		return health.Unavailable
	}
	return health.Available
}

func s3HealthURL(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	parsed.Path = "/minio/health/ready"
	parsed.RawQuery = ""
	return parsed.String()
}
