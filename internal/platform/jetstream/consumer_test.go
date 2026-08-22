package jetstream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestPullReturnsBrokerSelectedJobAndAcknowledgesOnce(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	defer server.Close()
	publisher := &Publisher{conn: client, reader: bufio.NewReader(client), timeout: time.Second}
	defer publisher.Close()
	serverErr := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(server)
		_, err := readRequest(reader)
		if err != nil {
			serverErr <- err
			return
		}
		headers := "NATS/1.0\r\nJob-Id: 734003200000000123\r\n\r\n"
		if _, err := fmt.Fprintf(server, "HMSG opi.events.job.queued.v1 1 $JS.ACK.OPI %d %d\r\n%s{}\r\n",
			len(headers), len(headers)+2, headers); err != nil {
			serverErr <- err
			return
		}
		serverErr <- expectControl(reader, server, "$JS.ACK.OPI", "+ACK")
	}()

	delivery, err := publisher.Pull(context.Background(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if delivery == nil || delivery.JobID != 734003200000000123 {
		t.Fatalf("delivery = %#v", delivery)
	}
	if err := delivery.Ack(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := delivery.Ack(context.Background()); err != nil {
		t.Fatalf("repeated acknowledgement must be idempotent: %v", err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestPullAcknowledgesProjectionOnlyEvents(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	defer server.Close()
	publisher := &Publisher{conn: client, reader: bufio.NewReader(client), timeout: time.Second}
	defer publisher.Close()
	serverErr := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(server)
		_, err := readRequest(reader)
		if err != nil {
			serverErr <- err
			return
		}
		headers := "NATS/1.0\r\nContent-Type: application/json\r\n\r\n"
		if _, err := fmt.Fprintf(server, "HMSG opi.events.project.updated.v1 1 $JS.ACK.PROJECTION %d %d\r\n%s{}\r\n",
			len(headers), len(headers)+2, headers); err != nil {
			serverErr <- err
			return
		}
		serverErr <- expectControl(reader, server, "$JS.ACK.PROJECTION", "+ACK")
	}()

	delivery, err := publisher.Pull(context.Background(), time.Second)
	if err != nil || delivery != nil {
		t.Fatalf("Pull() = %#v, %v; want acknowledged non-Job event", delivery, err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestPullTreatsJetStreamTimeoutStatusAsNoWork(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	defer server.Close()
	publisher := &Publisher{conn: client, reader: bufio.NewReader(client), timeout: time.Second}
	defer publisher.Close()
	serverErr := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(server)
		inbox, err := readRequest(reader)
		if err != nil {
			serverErr <- err
			return
		}
		headers := "NATS/1.0 408 Request Timeout\r\n\r\n"
		_, err = fmt.Fprintf(server, "HMSG %s 1 %d %d\r\n%s\r\n",
			inbox, len(headers), len(headers), headers)
		serverErr <- err
	}()

	delivery, err := publisher.Pull(context.Background(), time.Second)
	if err != nil || delivery != nil {
		t.Fatalf("Pull() = %#v, %v; want empty bounded pull", delivery, err)
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func readRequest(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[0] != "SUB" {
		return "", fmt.Errorf("unexpected subscription %q", line)
	}
	inbox := fields[1]
	if line, err = reader.ReadString('\n'); err != nil || !strings.HasPrefix(line, "UNSUB ") {
		return "", fmt.Errorf("unexpected unsubscribe %q: %w", line, err)
	}
	line, err = reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	fields = strings.Fields(line)
	if len(fields) != 5 || fields[0] != "HPUB" ||
		fields[1] != "$JS.API.CONSUMER.MSG.NEXT.OPI_EVENTS.OPI_WORKER" {
		return "", fmt.Errorf("unexpected pull request %q", line)
	}
	var total int
	if _, err := fmt.Sscan(fields[4], &total); err != nil {
		return "", err
	}
	body := make([]byte, total+2)
	if _, err := io.ReadFull(reader, body); err != nil {
		return "", err
	}
	return inbox, nil
}

func expectControl(reader *bufio.Reader, server net.Conn, subject, payload string) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[0] != "PUB" || fields[1] != subject {
		return fmt.Errorf("unexpected acknowledgement %q", line)
	}
	var size int
	if _, err := fmt.Sscan(fields[2], &size); err != nil {
		return err
	}
	body := make([]byte, size+2)
	if _, err := io.ReadFull(reader, body); err != nil {
		return err
	}
	if string(body[:size]) != payload {
		return fmt.Errorf("acknowledgement = %q, want %q", body[:size], payload)
	}
	if line, err = reader.ReadString('\n'); err != nil || strings.TrimSpace(line) != "PING" {
		return fmt.Errorf("unexpected acknowledgement flush %q: %w", line, err)
	}
	_, err = io.WriteString(server, "PONG\r\n")
	return err
}
