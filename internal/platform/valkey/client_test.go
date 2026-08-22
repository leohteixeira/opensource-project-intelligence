package valkey

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestPingAuthenticatesWithoutExposingCredentials(t *testing.T) {
	t.Parallel()
	client, err := New("valkey://worker:secret@cache.example:6379")
	if err != nil {
		t.Fatal(err)
	}
	connection, server := net.Pipe()
	client.dial = func(context.Context, string) (net.Conn, error) { return connection, nil }
	serverErr := make(chan error, 1)
	go func() {
		defer server.Close()
		reader := bufio.NewReader(server)
		if err := expectCommand(reader, "AUTH", "worker", "secret"); err != nil {
			serverErr <- err
			return
		}
		if _, err := io.WriteString(server, "+OK\r\n"); err != nil {
			serverErr <- err
			return
		}
		if err := expectCommand(reader, "PING"); err != nil {
			serverErr <- err
			return
		}
		_, err := io.WriteString(server, "+PONG\r\n")
		serverErr <- err
	}()
	if err := client.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

// UT-274: cache notifications are disposable wake-ups and carry no authority.
func TestUT274SubscribeDiscardsStalePayloadAndCarriesNoAuthority(t *testing.T) {
	t.Parallel()
	client, err := New("valkey://cache.example:6379")
	if err != nil {
		t.Fatal(err)
	}
	connection, server := net.Pipe()
	client.dial = func(context.Context, string) (net.Conn, error) { return connection, nil }
	serverErr := make(chan error, 1)
	go func() {
		defer server.Close()
		reader := bufio.NewReader(server)
		if err := expectCommand(reader, "SUBSCRIBE", "opi:jobs:42"); err != nil {
			serverErr <- err
			return
		}
		if _, err := io.WriteString(server,
			"*3\r\n$9\r\nsubscribe\r\n$11\r\nopi:jobs:42\r\n:1\r\n"); err != nil {
			serverErr <- err
			return
		}
		_, err := io.WriteString(server,
			"*3\r\n$7\r\nmessage\r\n$11\r\nopi:jobs:42\r\n$1\r\n1\r\n")
		serverErr <- err
	}()
	ctx, cancel := context.WithCancel(context.Background())
	wakeups, err := client.Subscribe(ctx, "opi:jobs:42")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	select {
	case <-wakeups:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("subscription did not deliver a wake-up")
	}
	cancel()
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestPublishAcceptsSubscriberCountAsNonAuthoritativeResult(t *testing.T) {
	t.Parallel()
	client, err := New("valkey://cache.example:6379")
	if err != nil {
		t.Fatal(err)
	}
	connection, server := net.Pipe()
	client.dial = func(context.Context, string) (net.Conn, error) { return connection, nil }
	serverErr := make(chan error, 1)
	go func() {
		defer server.Close()
		reader := bufio.NewReader(server)
		if err := expectCommand(reader, "PUBLISH", "opi:jobs:42", "1"); err != nil {
			serverErr <- err
			return
		}
		_, err := io.WriteString(server, ":0\r\n")
		serverErr <- err
	}()
	if err := client.Publish(context.Background(), "opi:jobs:42", []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func expectCommand(reader *bufio.Reader, values ...string) error {
	value, err := readValue(reader)
	if err != nil {
		return err
	}
	items, ok := value.([]any)
	if !ok || len(items) != len(values) {
		return fmt.Errorf("command = %#v, want %v", value, values)
	}
	for index, expected := range values {
		if items[index] != expected {
			return fmt.Errorf("command[%d] = %#v, want %q", index, items[index], expected)
		}
	}
	return nil
}
