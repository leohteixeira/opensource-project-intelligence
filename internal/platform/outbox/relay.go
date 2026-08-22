// Package outbox relays committed PostgreSQL events to a durable broker.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/database"
)

type Message struct {
	ID            int64
	Subject       string
	SchemaVersion int
	Payload       json.RawMessage
	Headers       map[string]string
}

// Publisher is implemented by the JetStream adapter. Message.ID is the stable
// broker de-duplication key, so relay retries remain at-least-once without
// duplicating consumer effects.
type Publisher interface {
	Publish(context.Context, Message) error
}

type Relay struct {
	pool      *pgxpool.Pool
	publisher Publisher
	now       func() time.Time
}

func New(pool *database.Pool, publisher Publisher) (*Relay, error) {
	if pool == nil || publisher == nil {
		return nil, fmt.Errorf("outbox pool and publisher are required")
	}
	return &Relay{pool: pool.Unwrap(), publisher: publisher, now: time.Now}, nil
}

func (r *Relay) PublishBatch(ctx context.Context, limit int) (int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `SELECT id,event_type,schema_version,payload,COALESCE(job_id,0),correlation_id,causation_id
		FROM outbox_events WHERE published_at IS NULL AND available_at<=now()
		ORDER BY occurred_at,id LIMIT $1`, limit)
	if err != nil {
		return 0, fmt.Errorf("read outbox batch: %w", err)
	}
	type event struct {
		id            int64
		eventType     string
		schemaVersion int
		payload       []byte
		jobID         int64
		correlationID string
		causationID   string
	}
	events := make([]event, 0, limit)
	for rows.Next() {
		var value event
		if err := rows.Scan(&value.id, &value.eventType, &value.schemaVersion, &value.payload, &value.jobID,
			&value.correlationID, &value.causationID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, value)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read outbox events: %w", err)
	}

	published := 0
	for _, event := range events {
		subject := "opi.events." + strings.ReplaceAll(event.eventType, "_", ".") +
			fmt.Sprintf(".v%d", event.schemaVersion)
		message := Message{
			ID: event.id, Subject: subject, SchemaVersion: event.schemaVersion, Payload: event.payload,
			Headers: map[string]string{
				"Nats-Msg-Id": fmt.Sprint(event.id), "Correlation-Id": event.correlationID,
				"Causation-Id": event.causationID, "Job-Id": fmt.Sprint(event.jobID),
			},
		}
		if err := r.publisher.Publish(ctx, message); err != nil {
			_, _ = r.pool.Exec(ctx, `UPDATE outbox_events SET attempts=attempts+1,last_error=$1,
				available_at=$2 WHERE id=$3 AND published_at IS NULL`, "broker publish failed",
				r.now().UTC().Add(backoff(event.id)), event.id)
			return published, fmt.Errorf("publish outbox event %d: %w", event.id, err)
		}
		command, err := r.pool.Exec(ctx, `UPDATE outbox_events SET published_at=$1,attempts=attempts+1,
			last_error='' WHERE id=$2 AND published_at IS NULL`, r.now().UTC(), event.id)
		if err != nil {
			return published, fmt.Errorf("mark outbox event published: %w", err)
		}
		if command.RowsAffected() == 1 {
			published++
		}
	}
	return published, nil
}

func backoff(seed int64) time.Duration {
	seconds := 1 + seed%30
	return time.Duration(seconds) * time.Second
}
