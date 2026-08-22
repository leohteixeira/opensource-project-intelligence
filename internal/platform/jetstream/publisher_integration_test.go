//go:build integration

package jetstream

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/outbox"
)

func TestRealJetStreamDeduplicatesStableOutboxIdentity(t *testing.T) {
	rawURL := os.Getenv("OPI_INTEGRATION_NATS_URL")
	if rawURL == "" {
		t.Skip("OPI_INTEGRATION_NATS_URL is required for the real JetStream contract")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	publisher, err := New(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	if err := publisher.EnsureStream(ctx); err != nil {
		t.Fatalf("IT-104 ensure stream: %v", err)
	}
	before := streamMessages(t, ctx, publisher)
	id := time.Now().UnixNano() & (1<<62 - 1)
	message := outbox.Message{
		ID: id, Subject: "opi.events.task03.integration.v1",
		Payload: []byte(`{"schema_version":1,"public":true}`),
		Headers: map[string]string{
			"Nats-Msg-Id": strconv.FormatInt(id, 10), "Correlation-Id": "IT-104",
		},
	}
	if err := publisher.Publish(ctx, message); err != nil {
		t.Fatalf("IT-104 publish committed outbox shape: %v", err)
	}
	if err := publisher.Publish(ctx, message); err != nil {
		t.Fatalf("IT-105 retry after lost acknowledgement: %v", err)
	}
	after := streamMessages(t, ctx, publisher)
	if after != before+1 {
		t.Fatalf("IT-105 stream messages = %d, want %d after duplicate publish", after, before+1)
	}
}

func TestRealJetStreamDurableConsumerAcknowledgesBrokerSelectedWork(t *testing.T) {
	rawURL := os.Getenv("OPI_INTEGRATION_NATS_URL")
	if rawURL == "" {
		t.Skip("OPI_INTEGRATION_NATS_URL is required for the real JetStream contract")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	publisher, err := New(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer publisher.Close()
	if err := publisher.EnsureStream(ctx); err != nil {
		t.Fatalf("ensure stream: %v", err)
	}
	identity := strconv.FormatInt(time.Now().UnixNano()&(1<<62-1), 10)
	publisher.consumerName = "OPI_TASK03_" + identity
	publisher.consumerFilter = "opi.events.task03.integration.work." + identity
	if err := publisher.EnsureConsumer(ctx); err != nil {
		t.Fatalf("ensure durable consumer: %v", err)
	}

	jobID := time.Now().UnixNano() & (1<<62 - 1)
	messageID := jobID + 1
	message := outbox.Message{
		ID: messageID, Subject: publisher.consumerFilter,
		Payload: []byte(`{"schema_version":1,"state":"queued"}`),
		Headers: map[string]string{
			"Nats-Msg-Id":    strconv.FormatInt(messageID, 10),
			"Correlation-Id": "durable-consumer", "Job-Id": strconv.FormatInt(jobID, 10),
		},
	}
	if err := publisher.Publish(ctx, message); err != nil {
		t.Fatalf("publish durable work notification: %v", err)
	}

	var delivery *Delivery
	for delivery == nil {
		delivery, err = publisher.Pull(ctx, 2*time.Second)
		if err != nil {
			t.Fatalf("pull durable work notification: %v", err)
		}
		if err := ctx.Err(); err != nil {
			t.Fatal(err)
		}
	}
	if delivery.JobID != jobID {
		t.Fatalf("pulled Job ID = %d, want %d", delivery.JobID, jobID)
	}
	if err := delivery.InProgress(ctx); err != nil {
		t.Fatalf("extend acknowledgement deadline: %v", err)
	}
	if err := delivery.Ack(ctx); err != nil {
		t.Fatalf("acknowledge durable work: %v", err)
	}
	if err := delivery.Ack(ctx); err != nil {
		t.Fatalf("repeat acknowledgement must be idempotent: %v", err)
	}
}

func streamMessages(t *testing.T, ctx context.Context, publisher *Publisher) uint64 {
	t.Helper()
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if err := publisher.ensureConnected(ctx); err != nil {
		t.Fatal(err)
	}
	response, err := publisher.request(ctx, "$JS.API.STREAM.INFO.OPI_EVENTS", nil,
		map[string]string{"Content-Type": "application/json"})
	if err != nil {
		t.Fatal(err)
	}
	var info struct {
		State struct {
			Messages uint64 `json:"messages"`
		} `json:"state"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(response, &info); err != nil || len(info.Error) != 0 && string(info.Error) != "null" {
		t.Fatalf("decode stream info: %v: %s", err, response)
	}
	return info.State.Messages
}
