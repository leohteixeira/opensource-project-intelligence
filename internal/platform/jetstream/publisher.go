// Package jetstream publishes transactional outbox messages to NATS JetStream.
package jetstream

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/platform/outbox"
)

const defaultTimeout = 10 * time.Second

// Publisher is a deliberately narrow JetStream client. It implements the
// small subset of the NATS text protocol needed by the transactional outbox,
// avoiding provider types outside this adapter.
type Publisher struct {
	mu             sync.Mutex
	target         *url.URL
	conn           net.Conn
	reader         *bufio.Reader
	timeout        time.Duration
	nextSID        uint64
	consumerName   string
	consumerFilter string
	dial           func(context.Context, string) (net.Conn, error)
}

type protocolMessage struct {
	subject string
	reply   string
	headers map[string]string
	payload []byte
}

func New(rawURL string) (*Publisher, error) {
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || target.Hostname() == "" || target.Scheme != "nats" && target.Scheme != "tls" {
		return nil, errors.New("NATS_URL must be an absolute nats:// or tls:// URL")
	}
	if target.Port() == "" {
		target.Host = net.JoinHostPort(target.Hostname(), "4222")
	}
	dialer := net.Dialer{Timeout: defaultTimeout}
	return &Publisher{
		target: target, timeout: defaultTimeout,
		consumerName: defaultConsumerName, consumerFilter: defaultConsumerFilter,
		dial: func(ctx context.Context, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp", address)
		},
	}, nil
}

func (publisher *Publisher) Close() error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if publisher.conn == nil {
		return nil
	}
	err := publisher.conn.Close()
	publisher.conn, publisher.reader = nil, nil
	return err
}

// EnsureStream idempotently creates the durable stream owned by this service.
func (publisher *Publisher) EnsureStream(ctx context.Context) error {
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if err := publisher.ensureConnected(ctx); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"name": "OPI_EVENTS", "subjects": []string{"opi.events.>"},
		"storage": "file", "retention": "limits", "discard": "old",
		"duplicate_window": int64(24 * time.Hour),
	})
	response, err := publisher.request(ctx, "$JS.API.STREAM.CREATE.OPI_EVENTS", payload,
		map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return fmt.Errorf("create JetStream stream: %w", err)
	}
	var result struct {
		Error *struct {
			ErrCode     int    `json:"err_code"`
			Description string `json:"description"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return fmt.Errorf("decode JetStream stream response: %w", err)
	}
	if result.Error != nil && result.Error.ErrCode != 10058 {
		return fmt.Errorf("create JetStream stream (%d): %s", result.Error.ErrCode, result.Error.Description)
	}
	return nil
}

func (publisher *Publisher) Publish(ctx context.Context, message outbox.Message) error {
	if message.ID <= 0 || message.Subject == "" || !json.Valid(message.Payload) {
		return errors.New("invalid JetStream message")
	}
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if err := publisher.ensureConnected(ctx); err != nil {
		return err
	}
	headers := make(map[string]string, len(message.Headers)+1)
	for key, value := range message.Headers {
		headers[key] = value
	}
	headers["Content-Type"] = "application/json"
	response, err := publisher.request(ctx, message.Subject, message.Payload, headers)
	if err != nil {
		publisher.closeBroken()
		return fmt.Errorf("publish JetStream message: %w", err)
	}
	var acknowledgement struct {
		Error *struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
		Stream string `json:"stream"`
		Seq    uint64 `json:"seq"`
	}
	if err := json.Unmarshal(response, &acknowledgement); err != nil {
		return fmt.Errorf("decode JetStream acknowledgement: %w", err)
	}
	if acknowledgement.Error != nil {
		return fmt.Errorf("JetStream rejected message (%d): %s",
			acknowledgement.Error.Code, acknowledgement.Error.Description)
	}
	if acknowledgement.Stream == "" || acknowledgement.Seq == 0 {
		return errors.New("JetStream returned an incomplete acknowledgement")
	}
	return nil
}

func (publisher *Publisher) ensureConnected(ctx context.Context) error {
	if publisher.conn != nil {
		return nil
	}
	conn, err := publisher.dial(ctx, publisher.target.Host)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	if publisher.target.Scheme == "tls" {
		tlsConn := tls.Client(conn, &tls.Config{
			MinVersion: tls.VersionTLS12, ServerName: publisher.target.Hostname(),
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return fmt.Errorf("negotiate NATS TLS: %w", err)
		}
		conn = tlsConn
	}
	publisher.conn, publisher.reader = conn, bufio.NewReader(conn)
	if err := publisher.deadline(ctx); err != nil {
		publisher.closeBroken()
		return err
	}
	line, err := publisher.reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "INFO ") {
		publisher.closeBroken()
		return errors.New("NATS did not send a valid INFO preface")
	}
	connect := map[string]any{
		"lang": "go", "version": "opi", "protocol": 1,
		"verbose": false, "headers": true,
	}
	if publisher.target.User != nil {
		connect["user"] = publisher.target.User.Username()
		if password, ok := publisher.target.User.Password(); ok {
			connect["pass"] = password
		}
	}
	encoded, _ := json.Marshal(connect)
	if _, err := fmt.Fprintf(conn, "CONNECT %s\r\nPING\r\n", encoded); err != nil {
		publisher.closeBroken()
		return fmt.Errorf("initialize NATS connection: %w", err)
	}
	for {
		line, err = publisher.reader.ReadString('\n')
		if err != nil {
			publisher.closeBroken()
			return fmt.Errorf("read NATS handshake: %w", err)
		}
		if strings.TrimSpace(line) == "PONG" {
			return nil
		}
		if strings.HasPrefix(line, "-ERR") {
			publisher.closeBroken()
			return errors.New("NATS rejected the connection")
		}
	}
}

func (publisher *Publisher) request(
	ctx context.Context,
	subject string,
	payload []byte,
	headers map[string]string,
) ([]byte, error) {
	message, err := publisher.requestMessage(ctx, subject, payload, headers)
	if err != nil {
		return nil, err
	}
	return message.payload, nil
}

func (publisher *Publisher) requestMessage(
	ctx context.Context,
	subject string,
	payload []byte,
	headers map[string]string,
) (protocolMessage, error) {
	if err := publisher.deadline(ctx); err != nil {
		return protocolMessage{}, err
	}
	inbox, err := randomInbox()
	if err != nil {
		return protocolMessage{}, err
	}
	var header strings.Builder
	header.WriteString("NATS/1.0\r\n")
	for key, value := range headers {
		if strings.ContainsAny(key+value, "\r\n") {
			return protocolMessage{}, errors.New("invalid NATS header")
		}
		header.WriteString(key)
		header.WriteString(": ")
		header.WriteString(value)
		header.WriteString("\r\n")
	}
	header.WriteString("\r\n")
	headerBytes := []byte(header.String())
	publisher.nextSID++
	sid := strconv.FormatUint(publisher.nextSID, 10)
	if _, err := fmt.Fprintf(publisher.conn, "SUB %s %d\r\nUNSUB %d 1\r\nHPUB %s %s %d %d\r\n",
		inbox, publisher.nextSID, publisher.nextSID, subject, inbox,
		len(headerBytes), len(headerBytes)+len(payload)); err != nil {
		return protocolMessage{}, err
	}
	if _, err := publisher.conn.Write(headerBytes); err != nil {
		return protocolMessage{}, err
	}
	if _, err := publisher.conn.Write(payload); err != nil {
		return protocolMessage{}, err
	}
	if _, err := io.WriteString(publisher.conn, "\r\n"); err != nil {
		return protocolMessage{}, err
	}
	for {
		line, err := publisher.reader.ReadString('\n')
		if err != nil {
			return protocolMessage{}, err
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 && fields[0] == "MSG" && fields[2] == sid {
			size, parseErr := strconv.Atoi(fields[len(fields)-1])
			if parseErr != nil || size < 0 || size > 1<<20 {
				return protocolMessage{}, errors.New("invalid NATS response size")
			}
			body := make([]byte, size+2)
			if _, err := io.ReadFull(publisher.reader, body); err != nil {
				return protocolMessage{}, err
			}
			reply := ""
			if len(fields) == 5 {
				reply = fields[3]
			}
			return protocolMessage{subject: fields[1], reply: reply, payload: body[:size]}, nil
		}
		if len(fields) >= 5 && fields[0] == "HMSG" && fields[2] == sid {
			headerSize, headerErr := strconv.Atoi(fields[len(fields)-2])
			totalSize, totalErr := strconv.Atoi(fields[len(fields)-1])
			if headerErr != nil || totalErr != nil || headerSize < 0 ||
				totalSize < headerSize || totalSize > 1<<20 {
				return protocolMessage{}, errors.New("invalid NATS header response size")
			}
			body := make([]byte, totalSize+2)
			if _, err := io.ReadFull(publisher.reader, body); err != nil {
				return protocolMessage{}, err
			}
			reply := ""
			if len(fields) == 6 {
				reply = fields[3]
			}
			headerValues, err := parseHeaders(body[:headerSize])
			if err != nil {
				return protocolMessage{}, err
			}
			return protocolMessage{
				subject: fields[1], reply: reply, headers: headerValues,
				payload: body[headerSize:totalSize],
			}, nil
		}
		if strings.HasPrefix(line, "PING") {
			_, _ = io.WriteString(publisher.conn, "PONG\r\n")
		}
		if strings.HasPrefix(line, "-ERR") {
			return protocolMessage{}, errors.New("NATS protocol error")
		}
	}
}

func parseHeaders(raw []byte) (map[string]string, error) {
	lines := strings.Split(string(raw), "\r\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "NATS/1.0") {
		return nil, errors.New("invalid NATS header block")
	}
	values := make(map[string]string, len(lines)-1)
	if status := strings.Fields(lines[0]); len(status) >= 2 && status[0] == "NATS/1.0" {
		values["status"] = status[1]
	}
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, errors.New("invalid NATS header line")
		}
		values[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return values, nil
}

func (publisher *Publisher) deadline(ctx context.Context) error {
	deadline := time.Now().Add(publisher.timeout)
	if value, ok := ctx.Deadline(); ok && value.Before(deadline) {
		deadline = value
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return publisher.conn.SetDeadline(deadline)
}

func (publisher *Publisher) closeBroken() {
	if publisher.conn != nil {
		_ = publisher.conn.Close()
	}
	publisher.conn, publisher.reader = nil, nil
}

func randomInbox() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("create NATS inbox: %w", err)
	}
	return "_INBOX.OPI." + hex.EncodeToString(bytes), nil
}
