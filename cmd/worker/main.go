// Command worker runs the scheduler and the collection jobs.
//
// It is a separate binary from the API but shares the same internal packages.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/collector"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/config"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/exportstore"
	gh "github.com/leohteixeira/opensource-project-intelligence/internal/platform/github"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/id"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/ingestion"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/intelligencestore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/jetstream"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/jobstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/objectstore"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/outbox"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/telemetry"
	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/valkey"
	"github.com/leohteixeira/opensource-project-intelligence/internal/worker"
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
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("read worker hostname: %w", err)
	}
	holder := serviceName + ":" + hostname + ":" + strconv.Itoa(os.Getpid())
	ids, err := id.New(ctx, database.NewSnowflakeLeaser(pool), id.Config{
		Holder: holder, LeaseTTL: 30 * time.Second, RenewalInterval: 10 * time.Second,
		RegressionTolerance: 5 * time.Millisecond,
	})
	if err != nil {
		return fmt.Errorf("start worker identifier generator: %w", err)
	}
	jobs := jobstore.New(pool, ids)
	if cfg.ObjectStorageEnabled {
		blobs, err := objectstore.NewS3(objectstore.Config{Endpoint: cfg.S3Endpoint,
			Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey})
		if err != nil {
			return err
		}
		if err := jobs.UseBlobStore(blobs); err != nil {
			return err
		}
	}
	if _, err := jobs.RecoverExpired(ctx); err != nil {
		return err
	}
	githubClient := gh.Client{
		HTTP:  collector.PublicHTTPClient(collector.PublicURLPolicy{}, 30*time.Second),
		Token: cfg.GitHubToken,
	}
	runner, err := worker.NewWithGitHub(jobs, holder, logger, githubClient, jobs,
		ingestion.New(pool, ids))
	if err != nil {
		return err
	}
	if err := runner.UseIntelligence(intelligencestore.New(pool, ids)); err != nil {
		return err
	}
	if cfg.ObjectStorageEnabled {
		blobs, err := objectstore.NewS3(objectstore.Config{Endpoint: cfg.S3Endpoint,
			Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey})
		if err != nil {
			return err
		}
		exports, err := exportstore.New(pool, ids, blobs, cfg.ExportConcurrency)
		if err != nil {
			return err
		}
		if err := runner.UseExports(exports); err != nil {
			return err
		}
	}
	var relay *outbox.Relay
	var publisher *jetstream.Publisher
	var acceleration *valkey.Client
	if cfg.ValkeyEnabled {
		acceleration, err = valkey.New(cfg.ValkeyURL)
		if err != nil {
			return err
		}
	}
	if cfg.JetStreamEnabled {
		publisher, err = jetstream.New(cfg.NATSURL)
		if err != nil {
			return err
		}
		defer publisher.Close()
		if err := publisher.EnsureStream(ctx); err != nil {
			return err
		}
		if err := publisher.EnsureConsumer(ctx); err != nil {
			return err
		}
		relay, err = outbox.New(pool, publisher)
		if err != nil {
			return err
		}
	}

	logger.Info(
		"the worker started",
		slog.Duration("interval", cfg.WorkerInterval),
		slog.Int("concurrency", cfg.WorkerConcurrency),
	)

	loopErr := loop(ctx, logger, cfg, pool, runner, relay, publisher, acceleration)

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
	runner *worker.Runner,
	relay *outbox.Relay,
	consumer *jetstream.Publisher,
	acceleration *valkey.Client,
) error {
	ticker := time.NewTicker(cfg.WorkerInterval)
	defer ticker.Stop()

	for {
		if err := tick(ctx, logger, cfg, pool, runner, relay, consumer, acceleration); err != nil {
			// A failed cycle must not stop the worker: the scheduler retries on
			// the next tick, and every task is idempotent. Cancellation is the
			// expected graceful-shutdown signal, not an operational failure.
			if ctx.Err() == nil {
				logger.Warn("the collection cycle failed", slog.Any("error", err))
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// tick verifies database availability, relays committed outbox events, and
// executes a bounded batch of durable collection jobs.
func tick(
	ctx context.Context,
	logger *slog.Logger,
	cfg config.Config,
	pool *database.Pool,
	runner *worker.Runner,
	relay *outbox.Relay,
	consumer *jetstream.Publisher,
	acceleration *valkey.Client,
) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		return err
	}
	if relay != nil {
		if _, err := relay.PublishBatch(ctx, 100); err != nil {
			return err
		}
	}
	tasks := make([]collector.Task, cfg.WorkerConcurrency)
	for index := range tasks {
		tasks[index] = func(taskCtx context.Context) error {
			if consumer != nil {
				return consumeOne(taskCtx, logger, runner, consumer, acceleration)
			}
			worked, err := runner.RunOne(taskCtx)
			if worked {
				logger.Info("durable job cycle completed")
			}
			return err
		}
	}
	return collector.RunBounded(ctx, cfg.WorkerConcurrency, tasks)
}

func consumeOne(
	ctx context.Context,
	logger *slog.Logger,
	runner *worker.Runner,
	consumer *jetstream.Publisher,
	acceleration *valkey.Client,
) error {
	delivery, err := consumer.Pull(ctx, 5*time.Second)
	if err != nil || delivery == nil {
		return err
	}
	type result struct {
		worked bool
		err    error
	}
	finished := make(chan result, 1)
	go func() {
		worked, runErr := runner.RunJob(ctx, delivery.JobID)
		finished <- result{worked: worked, err: runErr}
	}()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// Do not acknowledge on shutdown. The durable consumer redelivers
			// after the PostgreSQL lease expires and recovery makes it claimable.
			return nil
		case <-ticker.C:
			if err := delivery.InProgress(ctx); err != nil {
				return fmt.Errorf("extend JetStream delivery acknowledgement: %w", err)
			}
		case outcome := <-finished:
			if outcome.err != nil && !outcome.worked {
				if retryErr := delivery.Retry(ctx, time.Second); retryErr != nil {
					return errors.Join(outcome.err, retryErr)
				}
				return outcome.err
			}
			if acceleration != nil && ctx.Err() == nil {
				channel := "opi:jobs:" + strconv.FormatInt(delivery.JobID, 10)
				if err := acceleration.Publish(ctx, channel, []byte("1")); err != nil {
					// PostgreSQL already contains the authoritative result. A
					// disposable notification failure must not cause redelivery.
					logger.Warn("Valkey Job wake-up failed; clients will poll PostgreSQL",
						slog.Int64("job_id", delivery.JobID))
				}
			}
			if err := delivery.Ack(ctx); err != nil {
				return fmt.Errorf("acknowledge durable Job delivery: %w", err)
			}
			if outcome.worked {
				logger.Info("broker-driven durable job cycle completed", slog.Int64("job_id", delivery.JobID))
			}
			return outcome.err
		}
	}
}
