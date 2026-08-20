// Command worker runs the scheduler and the collection jobs.
//
// It is a separate binary from the API but shares the same internal packages.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/config"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/telemetry"
)

const serviceName = "opensource-project-intelligence-worker"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(logger); err != nil {
		logger.Error("the worker stopped with an error", slog.Any("error", err))
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

	logger.Info(
		"the worker started",
		slog.Duration("interval", cfg.WorkerInterval),
		slog.Int("concurrency", cfg.WorkerConcurrency),
	)

	loopErr := loop(ctx, logger, cfg, pool)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	logger.Info("the worker stopped")

	return errors.Join(loopErr, shutdownTelemetry(shutdownCtx))
}

func loop(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	pool *database.Pool,
) error {
	ticker := time.NewTicker(cfg.WorkerInterval)
	defer ticker.Stop()

	for {
		if err := tick(ctx, logger, cfg, pool); err != nil {
			// A failed cycle must not stop the worker: the scheduler retries on
			// the next tick, and every task is idempotent.
			logger.Warn("the collection cycle failed", slog.Any("error", err))
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// tick runs one collection cycle. Today it only reports database availability;
// the real collection jobs are registered here as each capability lands.
func tick(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	pool *database.Pool,
) error {
	tasks := []collector.Task{
		func(taskCtx context.Context) error {
			pingCtx, cancel := context.WithTimeout(taskCtx, 2*time.Second)
			defer cancel()

			if err := pool.Ping(pingCtx); err != nil {
				return err
			}

			logger.Info("heartbeat", slog.String("database", "available"))

			return nil
		},
	}

	return collector.RunBounded(ctx, cfg.WorkerConcurrency, tasks)
}
