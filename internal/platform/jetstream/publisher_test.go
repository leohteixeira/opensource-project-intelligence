package jetstream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/outbox"
)

func TestPublisherWaitsForJetStreamAcknowledgement(t *testing.T) {
	t.Parallel()
	clientConnection, serverConnection := net.Pipe()
	defer serverConnection.Close()
	serverErr := make(chan error, 1)
	go func() {
		connection := serverConnection
		defer connection.Close()
		reader := bufio.NewReader(connection)
		_, _ = io.WriteString(connection, "INFO {}\r\n")
		for {
			line, readErr := reader.ReadString('\n')
			if readErr != nil {
				serverErr <- readErr
				return
			}
			if strings.HasPrefix(line, "PING") {
				_, _ = io.WriteString(connection, "PONG\r\n")
				break
			}
		}
		line, readErr := reader.ReadString('\n')
		if readErr != nil || !strings.HasPrefix(line, "SUB _INBOX.OPI.") {
			serverErr <- fmt.Errorf("unexpected subscription %q: %w", line, readErr)
			return
		}
		inbox := strings.Fields(line)[1]
		line, readErr = reader.ReadString('\n')
		if readErr != nil || !strings.HasPrefix(line, "UNSUB 1 1") {
			serverErr <- fmt.Errorf("unexpected unsubscribe %q: %w", line, readErr)
			return
		}
		line, readErr = reader.ReadString('\n')
		fields := strings.Fields(line)
		if readErr != nil || len(fields) != 5 || fields[0] != "HPUB" || fields[1] != "opi.events.project.registered.v1" {
			serverErr <- fmt.Errorf("unexpected publish %q: %w", line, readErr)
			return
		}
		var total int
		_, _ = fmt.Sscan(fields[4], &total)
		payload := make([]byte, total+2)
		if _, readErr = io.ReadFull(reader, payload); readErr != nil {
			serverErr <- readErr
			return
		}
		ack := `{"stream":"OPI_EVENTS","seq":1}`
		headers := "NATS/1.0\r\nContent-Type: application/json\r\n\r\n"
		_, _ = fmt.Fprintf(connection, "HMSG %s 1 %d %d\r\n%s%s\r\n",
			inbox, len(headers), len(headers)+len(ack), headers, ack)
		serverErr <- nil
	}()

	publisher, err := New("nats://broker.example:4222")
	if err != nil {
		t.Fatal(err)
	}
	publisher.dial = func(context.Context, string) (net.Conn, error) { return clientConnection, nil }
	defer publisher.Close()
	err = publisher.Publish(context.Background(), outbox.Message{ID: 1,
		Subject: "opi.events.project.registered.v1", Payload: []byte(`{"project_id":"1"}`),
		Headers: map[string]string{"Nats-Msg-Id": "1"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}
