package jetstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultConsumerName   = "OPI_WORKER"
	defaultConsumerFilter = "opi.events.>"
	streamName            = "OPI_EVENTS"
)

// Delivery is one durable broker notification that points to authoritative
// PostgreSQL work. The payload is deliberately not exposed to business code.
type Delivery struct {
	JobID int64

	mu        sync.Mutex
	publisher *Publisher
	reply     string
	settled   bool
}

// EnsureConsumer idempotently creates the durable pull consumer used by workers.
func (publisher *Publisher) EnsureConsumer(ctx context.Context) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if err := publisher.ensureConnected(ctx); err != nil {
		return err
	}
	consumerName := publisher.durableConsumerName()
	payload, err := json.Marshal(map[string]any{
		"stream_name": streamName,
		"config": map[string]any{
			"durable_name":   consumerName,
			"ack_policy":     "explicit",
			"ack_wait":       int64(2 * time.Minute),
			"max_deliver":    5,
			"filter_subject": publisher.durableConsumerFilter(),
			"replay_policy":  "instant",
		},
	})
	if err != nil {
		return fmt.Errorf("encode JetStream consumer configuration: %w", err)
	}
	response, err := publisher.request(ctx,
		"$JS.API.CONSUMER.DURABLE.CREATE."+streamName+"."+consumerName,
		payload, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return fmt.Errorf("create JetStream consumer: %w", err)
	}
	var result struct {
		Name  string `json:"name"`
		Error *struct {
			Code        int    `json:"code"`
			ErrCode     int    `json:"err_code"`
			Description string `json:"description"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("decode JetStream consumer response: %w", err)
	}
	if result.Error != nil {
		return fmt.Errorf("create JetStream consumer (%d/%d): %s",
			result.Error.Code, result.Error.ErrCode, result.Error.Description)
	}
	if result.Name != consumerName {
		return errors.New("JetStream returned an incomplete consumer acknowledgement")
	}
	return nil
}

// Pull requests one durable notification. A nil delivery means the bounded
// pull expired without work.
func (publisher *Publisher) Pull(ctx context.Context, wait time.Duration) (*Delivery, error) {
	if wait <= 0 || wait > 30*time.Second {
		wait = 5 * time.Second
	}
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if err := publisher.ensureConnected(ctx); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"batch": 1, "expires": int64(wait), "no_wait": false,
	})
	if err != nil {
		return nil, fmt.Errorf("encode JetStream pull request: %w", err)
	}
	message, err := publisher.requestMessage(ctx,
		"$JS.API.CONSUMER.MSG.NEXT."+streamName+"."+publisher.durableConsumerName(),
		payload, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("pull JetStream delivery: %w", err)
	}
	if status := message.headers["status"]; status == "404" || status == "408" {
		return nil, nil
	}
	if message.reply == "" {
		return nil, errors.New("JetStream delivery omitted its acknowledgement subject")
	}
	jobID, err := strconv.ParseInt(message.headers["job-id"], 10, 64)
	if err != nil || jobID <= 0 {
		// Events without durable work are valid on the shared stream. Ack them
		// here so the worker consumer does not redeliver projection-only events.
		if err := publisher.publishControl(ctx, message.reply, "+ACK"); err != nil {
			return nil, fmt.Errorf("acknowledge non-job event: %w", err)
		}
		return nil, nil
	}
	return &Delivery{JobID: jobID, publisher: publisher, reply: message.reply}, nil
}

func (publisher *Publisher) durableConsumerName() string {
	if publisher.consumerName == "" {
		return defaultConsumerName
	}
	return publisher.consumerName
}

func (publisher *Publisher) durableConsumerFilter() string {
	if publisher.consumerFilter == "" {
		return defaultConsumerFilter
	}
	return publisher.consumerFilter
}

func (delivery *Delivery) Ack(ctx context.Context) error {
	return delivery.settle(ctx, "+ACK")
}

func (delivery *Delivery) Retry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return delivery.settle(ctx, "-NAK")
	}
	return delivery.settle(ctx, fmt.Sprintf("-NAK {\"delay\":%d}", int64(delay)))
}

func (delivery *Delivery) Terminate(ctx context.Context) error {
	return delivery.settle(ctx, "+TERM")
}

// InProgress extends the server acknowledgement deadline while the
// PostgreSQL lease holder is still processing bounded work.
func (delivery *Delivery) InProgress(ctx context.Context) error {
	if delivery == nil || delivery.publisher == nil || strings.TrimSpace(delivery.reply) == "" {
		return errors.New("invalid JetStream delivery")
	}
	delivery.mu.Lock()
	defer delivery.mu.Unlock()
	if delivery.settled {
		return nil
	}
	delivery.publisher.mu.Lock()
	defer delivery.publisher.mu.Unlock()
	if err := delivery.publisher.ensureConnected(ctx); err != nil {
		return err
	}
	return delivery.publisher.publishControl(ctx, delivery.reply, "+WPI")
}

func (delivery *Delivery) settle(ctx context.Context, command string) error {
	if delivery == nil || delivery.publisher == nil || strings.TrimSpace(delivery.reply) == "" {
		return errors.New("invalid JetStream delivery")
	}
	delivery.mu.Lock()
	defer delivery.mu.Unlock()
	if delivery.settled {
		return nil
	}
	delivery.publisher.mu.Lock()
	defer delivery.publisher.mu.Unlock()
	if err := delivery.publisher.ensureConnected(ctx); err != nil {
		return err
	}
	if err := delivery.publisher.publishControl(ctx, delivery.reply, command); err != nil {
		delivery.publisher.closeBroken()
		return err
	}
	delivery.settled = true
	return nil
}

func (publisher *Publisher) publishControl(ctx context.Context, subject, payload string) error {
	if err := publisher.deadline(ctx); err != nil {
		return err
	}
	if strings.ContainsAny(subject, " \r\n\t") || subject == "" {
		return errors.New("invalid NATS control subject")
	}
	if _, err := fmt.Fprintf(publisher.conn, "PUB %s %d\r\n%s\r\nPING\r\n",
		subject, len(payload), payload); err != nil {
		return err
	}
	for {
		line, err := publisher.reader.ReadString('\n')
		if err != nil {
			return err
		}
		switch {
		case strings.TrimSpace(line) == "PONG":
			return nil
		case strings.HasPrefix(line, "PING"):
			if _, err := io.WriteString(publisher.conn, "PONG\r\n"); err != nil {
				return err
			}
		case strings.HasPrefix(line, "-ERR"):
			return errors.New("NATS rejected the delivery acknowledgement")
		}
	}
}
