//go:build integration

package valkey

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestRealValkeyWakeupsAreDisposableAcceleration(t *testing.T) {
	rawURL := os.Getenv("OPI_INTEGRATION_VALKEY_URL")
	if rawURL == "" {
		t.Skip("OPI_INTEGRATION_VALKEY_URL is required for the real Valkey contract")
	}
	client, err := New(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("ping real Valkey: %v", err)
	}
	channel := "opi:jobs:integration:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	wakeups, err := client.Subscribe(ctx, channel)
	if err != nil {
		t.Fatalf("subscribe to disposable wake-ups: %v", err)
	}
	if err := client.Publish(ctx, channel, []byte("1")); err != nil {
		t.Fatalf("publish disposable wake-up: %v", err)
	}
	select {
	case _, ok := <-wakeups:
		if !ok {
			t.Fatal("Valkey subscription closed before delivering the wake-up")
		}
	case <-ctx.Done():
		t.Fatalf("real Valkey wake-up was not delivered: %v", ctx.Err())
	}
}
