// Package valkey provides disposable cache and notification acceleration.
// PostgreSQL remains authoritative for every business decision.
package valkey

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout = 2 * time.Second
	maxBulkBytes   = 1 << 20
)

type Client struct {
	target  *url.URL
	timeout time.Duration
	dial    func(context.Context, string) (net.Conn, error)
}

func New(rawURL string) (*Client, error) {
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || target.Hostname() == "" ||
		target.Scheme != "valkey" && target.Scheme != "redis" && target.Scheme != "rediss" {
		return nil, errors.New("VALKEY_URL must be an absolute valkey://, redis://, or rediss:// URL")
	}
	if target.Port() == "" {
		target.Host = net.JoinHostPort(target.Hostname(), "6379")
	}
	dialer := net.Dialer{Timeout: defaultTimeout}
	return &Client{target: target, timeout: defaultTimeout,
		dial: func(ctx context.Context, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp", address)
		}}, nil
}

func (client *Client) Ping(ctx context.Context) error {
	value, err := client.command(ctx, "PING")
	if err != nil {
		return err
	}
	if value != "PONG" {
		return errors.New("Valkey returned an invalid PING response")
	}
	return nil
}

// Publish sends a best-effort wake-up. The payload must not contain canonical
// state because subscribers always re-read PostgreSQL after receiving it.
func (client *Client) Publish(ctx context.Context, channel string, payload []byte) error {
	if err := validChannel(channel); err != nil {
		return err
	}
	if len(payload) > 4096 {
		return errors.New("Valkey notification payload is too large")
	}
	_, err := client.command(ctx, "PUBLISH", channel, string(payload))
	return err
}

// Subscribe returns coalesced wake-ups until ctx is cancelled. Losing Valkey
// closes the channel; callers must retain a PostgreSQL polling fallback.
func (client *Client) Subscribe(ctx context.Context, channel string) (<-chan struct{}, error) {
	if err := validChannel(channel); err != nil {
		return nil, err
	}
	connection, reader, err := client.connect(ctx)
	if err != nil {
		return nil, err
	}
	if err := writeCommand(connection, "SUBSCRIBE", channel); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("subscribe to Valkey wake-ups: %w", err)
	}
	if err := connection.SetReadDeadline(time.Now().Add(client.timeout)); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("set Valkey subscription deadline: %w", err)
	}
	acknowledgement, err := readValue(reader)
	if err != nil || !subscriptionAcknowledged(acknowledgement, channel) {
		_ = connection.Close()
		return nil, errors.New("Valkey did not acknowledge the subscription")
	}
	if err := connection.SetReadDeadline(time.Time{}); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("clear Valkey subscription deadline: %w", err)
	}
	wakeups := make(chan struct{}, 1)
	go func() {
		defer close(wakeups)
		defer connection.Close()
		go func() {
			<-ctx.Done()
			_ = connection.Close()
		}()
		for {
			value, readErr := readValue(reader)
			if readErr != nil {
				return
			}
			items, ok := value.([]any)
			if !ok || len(items) != 3 || items[0] != "message" || items[1] != channel {
				continue
			}
			select {
			case wakeups <- struct{}{}:
			default:
			}
		}
	}()
	return wakeups, nil
}

func (client *Client) command(ctx context.Context, values ...string) (any, error) {
	connection, reader, err := client.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	if err := writeCommand(connection, values...); err != nil {
		return nil, fmt.Errorf("write Valkey command: %w", err)
	}
	value, err := readValue(reader)
	if err != nil {
		return nil, fmt.Errorf("read Valkey response: %w", err)
	}
	return value, nil
}

func (client *Client) connect(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	connection, err := client.dial(ctx, client.target.Host)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to Valkey: %w", err)
	}
	if client.target.Scheme == "rediss" {
		secured := tls.Client(connection, &tls.Config{
			MinVersion: tls.VersionTLS12, ServerName: client.target.Hostname(),
		})
		if err := secured.HandshakeContext(ctx); err != nil {
			_ = connection.Close()
			return nil, nil, fmt.Errorf("negotiate Valkey TLS: %w", err)
		}
		connection = secured
	}
	deadline := time.Now().Add(client.timeout)
	if value, ok := ctx.Deadline(); ok && value.Before(deadline) {
		deadline = value
	}
	if err := connection.SetDeadline(deadline); err != nil {
		_ = connection.Close()
		return nil, nil, fmt.Errorf("set Valkey deadline: %w", err)
	}
	reader := bufio.NewReader(connection)
	if client.target.User != nil {
		password, _ := client.target.User.Password()
		arguments := []string{"AUTH"}
		if username := client.target.User.Username(); username != "" {
			arguments = append(arguments, username)
		}
		arguments = append(arguments, password)
		if err := writeCommand(connection, arguments...); err != nil {
			_ = connection.Close()
			return nil, nil, fmt.Errorf("authenticate to Valkey: %w", err)
		}
		if value, err := readValue(reader); err != nil || value != "OK" {
			_ = connection.Close()
			return nil, nil, errors.New("Valkey authentication failed")
		}
	}
	return connection, reader, nil
}

func validChannel(channel string) error {
	if channel == "" || len(channel) > 200 || strings.ContainsAny(channel, " \r\n\t") {
		return errors.New("invalid Valkey channel")
	}
	return nil
}

func writeCommand(writer io.Writer, values ...string) error {
	if len(values) == 0 {
		return errors.New("Valkey command is required")
	}
	if _, err := fmt.Fprintf(writer, "*%d\r\n", len(values)); err != nil {
		return err
	}
	for _, value := range values {
		if _, err := fmt.Fprintf(writer, "$%d\r\n%s\r\n", len(value), value); err != nil {
			return err
		}
	}
	return nil
}

func readValue(reader *bufio.Reader) (any, error) {
	prefix, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	line, err := reader.ReadString('\n')
	if err != nil || !strings.HasSuffix(line, "\r\n") {
		return nil, errors.New("invalid Valkey response")
	}
	line = strings.TrimSuffix(line, "\r\n")
	switch prefix {
	case '+':
		return line, nil
	case '-':
		return nil, fmt.Errorf("Valkey command failed: %s", line)
	case ':':
		return strconv.ParseInt(line, 10, 64)
	case '$':
		size, parseErr := strconv.Atoi(line)
		if parseErr != nil || size < -1 || size > maxBulkBytes {
			return nil, errors.New("invalid Valkey bulk response size")
		}
		if size == -1 {
			return nil, nil
		}
		body := make([]byte, size+2)
		if _, err := io.ReadFull(reader, body); err != nil || string(body[size:]) != "\r\n" {
			return nil, errors.New("invalid Valkey bulk response")
		}
		return string(body[:size]), nil
	case '*':
		count, parseErr := strconv.Atoi(line)
		if parseErr != nil || count < 0 || count > 1000 {
			return nil, errors.New("invalid Valkey array response size")
		}
		values := make([]any, count)
		for index := range values {
			values[index], err = readValue(reader)
			if err != nil {
				return nil, err
			}
		}
		return values, nil
	default:
		return nil, errors.New("unsupported Valkey response")
	}
}

func subscriptionAcknowledged(value any, channel string) bool {
	items, ok := value.([]any)
	if !ok || len(items) != 3 {
		return false
	}
	count, ok := items[2].(int64)
	return ok && items[0] == "subscribe" && items[1] == channel && count == 1
}
