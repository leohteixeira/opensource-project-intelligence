// Command api serves the versioned HTTP contract consumed by the web
// application.
package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/leohteixeira/opensource-project-intelligence/internal/access"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessapi"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/accessstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/config"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/health"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/httpx"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/id"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/oidc"
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

	handler, err := routes(ctx, logger, pool, cfg)
	if err != nil {
		return err
	}
	server := httpx.NewServer(cfg.HTTPAddress, otelhttp.NewHandler(handler, "api"))

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

func routes(ctx context.Context, logger *slog.Logger, pool *database.Pool, cfg config.Config) (http.Handler, error) {
	mux := http.NewServeMux()

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("read API hostname: %w", err)
	}
	ids, err := id.New(ctx, database.NewSnowflakeLeaser(pool), id.Config{
		Holder:   serviceName + ":" + hostname + ":" + strconv.Itoa(os.Getpid()),
		LeaseTTL: 30 * time.Second, RenewalInterval: 10 * time.Second,
		RegressionTolerance: 5 * time.Millisecond,
	})
	if err != nil {
		return nil, fmt.Errorf("start identifier generator: %w", err)
	}
	store := accessstore.New(pool, ids)
	if err := store.EnsureWorkspace(ctx); err != nil {
		return nil, err
	}
	signingKey, err := loadSigningKey(cfg.SessionKeyFile)
	if err != nil {
		return nil, err
	}
	cursors, err := access.NewCursorCodec(signingKey)
	if err != nil {
		return nil, fmt.Errorf("configure signed cursors: %w", err)
	}
	var identity *oidc.Client
	if cfg.KeycloakIssuerURL != "" || cfg.KeycloakClientID != "" {
		identity, err = oidc.New(ctx, oidc.Config{
			IssuerURL: cfg.KeycloakIssuerURL, ClientID: cfg.KeycloakClientID,
			ClientSecretFile: cfg.KeycloakClientSecretFile, PublicBaseURL: cfg.PublicBaseURL,
		})
		if err != nil {
			return nil, err
		}
	}
	publicURL, _ := url.Parse(cfg.PublicBaseURL)
	accessHandler, err := accessapi.New(store, identity, cursors, logger, accessapi.Config{
		PublicBaseURL: cfg.PublicBaseURL, IssuerURL: cfg.KeycloakIssuerURL,
		SessionTTL: cfg.SessionDuration, SecureCookies: publicURL.Scheme == "https",
	})
	if err != nil {
		return nil, err
	}
	accessRoutes := accessapi.Routes(accessHandler)

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
	for _, prefix := range []string{"/api/v1/catalog/", "/api/v1/session", "/api/v1/session/", "/api/v1/me/", "/api/v1/admin/", "/auth/"} {
		mux.Handle(prefix, accessRoutes)
	}

	return mux, nil
}

func loadSigningKey(path string) ([]byte, error) {
	if path == "" {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate development signing key: %w", err)
		}
		return key, nil
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session key: %w", err)
	}
	if len(key) < 32 {
		return nil, errors.New("session key must contain at least 32 bytes")
	}
	return key, nil
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
