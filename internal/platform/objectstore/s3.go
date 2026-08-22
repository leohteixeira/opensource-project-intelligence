// Package objectstore implements the S3-compatible immutable byte boundary.
package objectstore

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"
)

const defaultMaxObjectBytes = 64 << 20

type Config struct {
	Endpoint   string
	Bucket     string
	AccessKey  string
	SecretKey  string
	Region     string
	HTTPClient *http.Client
	MaxBytes   int64
}

type S3 struct {
	endpoint  *url.URL
	bucket    string
	accessKey string
	secretKey string
	region    string
	client    *http.Client
	maxBytes  int64
	now       func() time.Time
}

func NewS3(config Config) (*S3, error) {
	endpoint, err := url.Parse(strings.TrimSpace(config.Endpoint))
	if err != nil || endpoint.Host == "" || endpoint.Scheme != "http" && endpoint.Scheme != "https" ||
		strings.Trim(config.Bucket, " /\t") == "" || strings.Contains(config.Bucket, "/") ||
		config.AccessKey == "" || config.SecretKey == "" {
		return nil, errors.New("valid S3 endpoint, bucket, access key, and secret key are required")
	}
	region := config.Region
	if region == "" {
		region = "us-east-1"
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	maxBytes := config.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxObjectBytes
	}
	return &S3{endpoint: endpoint, bucket: config.Bucket, accessKey: config.AccessKey,
		secretKey: config.SecretKey, region: region, client: client, maxBytes: maxBytes, now: time.Now}, nil
}

func (store *S3) Stage(ctx context.Context, key string, body []byte, mediaType string) error {
	if !validKey(key) || int64(len(body)) > store.maxBytes || strings.TrimSpace(mediaType) == "" {
		return errors.New("invalid staged object")
	}
	response, err := store.do(ctx, http.MethodPut, key, body,
		map[string]string{"content-type": mediaType})
	return finish(response, err, "stage S3 object")
}

func (store *S3) Read(ctx context.Context, key string) ([]byte, error) {
	if !validKey(key) {
		return nil, errors.New("invalid object key")
	}
	response, err := store.do(ctx, http.MethodGet, key, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("read S3 object: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("read S3 object: status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, store.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read S3 object body: %w", err)
	}
	if int64(len(body)) > store.maxBytes {
		return nil, errors.New("S3 object exceeds the configured byte limit")
	}
	return body, nil
}

func (store *S3) Promote(ctx context.Context, stagedKey, finalKey string) error {
	if !validKey(stagedKey) || !validKey(finalKey) || stagedKey == finalKey {
		return errors.New("invalid object promotion")
	}
	copySource := "/" + escapeSegment(store.bucket) + "/" + escapeKey(stagedKey)
	response, err := store.do(ctx, http.MethodPut, finalKey, nil,
		map[string]string{"x-amz-copy-source": copySource})
	if err := finish(response, err, "promote S3 object"); err != nil {
		return err
	}
	if err := store.Delete(ctx, stagedKey); err != nil {
		return fmt.Errorf("remove staged S3 object: %w", err)
	}
	return nil
}

func (store *S3) Delete(ctx context.Context, key string) error {
	if !validKey(key) {
		return errors.New("invalid object key")
	}
	response, err := store.do(ctx, http.MethodDelete, key, nil, nil)
	return finish(response, err, "delete S3 object")
}

func (store *S3) do(
	ctx context.Context,
	method, key string,
	body []byte,
	extraHeaders map[string]string,
) (*http.Response, error) {
	target := *store.endpoint
	target.Path = path.Join(store.endpoint.Path, store.bucket, key)
	target.RawPath = path.Join(store.endpoint.EscapedPath(), escapeSegment(store.bucket), escapeKey(key))
	payloadHash := sha256.Sum256(body)
	request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	now := store.now().UTC()
	request.Header.Set("X-Amz-Date", now.Format("20060102T150405Z"))
	request.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(payloadHash[:]))
	for key, value := range extraHeaders {
		request.Header.Set(key, value)
	}
	store.sign(request, now, payloadHash)
	return store.client.Do(request)
}

func (store *S3) sign(request *http.Request, now time.Time, payloadHash [32]byte) {
	headers := map[string]string{
		"host":                 request.URL.Host,
		"x-amz-content-sha256": hex.EncodeToString(payloadHash[:]),
		"x-amz-date":           request.Header.Get("X-Amz-Date"),
	}
	for _, name := range []string{"content-type", "x-amz-copy-source"} {
		if value := request.Header.Get(name); value != "" {
			headers[name] = strings.Join(strings.Fields(value), " ")
		}
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	var canonicalHeaders strings.Builder
	for _, name := range names {
		fmt.Fprintf(&canonicalHeaders, "%s:%s\n", name, headers[name])
	}
	signedHeaders := strings.Join(names, ";")
	canonical := strings.Join([]string{request.Method, request.URL.EscapedPath(),
		request.URL.Query().Encode(), canonicalHeaders.String(), signedHeaders,
		hex.EncodeToString(payloadHash[:])}, "\n")
	canonicalHash := sha256.Sum256([]byte(canonical))
	date := now.Format("20060102")
	scope := date + "/" + store.region + "/s3/aws4_request"
	toSign := "AWS4-HMAC-SHA256\n" + now.Format("20060102T150405Z") + "\n" + scope + "\n" +
		hex.EncodeToString(canonicalHash[:])
	dateKey := hmacSHA256([]byte("AWS4"+store.secretKey), date)
	regionKey := hmacSHA256(dateKey, store.region)
	serviceKey := hmacSHA256(regionKey, "s3")
	signingKey := hmacSHA256(serviceKey, "aws4_request")
	signature := hex.EncodeToString(hmacSHA256(signingKey, toSign))
	request.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+store.accessKey+"/"+scope+
		", SignedHeaders="+signedHeaders+", Signature="+signature)
}

func finish(response *http.Response, err error, action string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("%s: status %d", action, response.StatusCode)
	}
	return nil
}

func validKey(key string) bool {
	return key != "" && !strings.HasPrefix(key, "/") && !strings.Contains(key, "\\") &&
		!strings.ContainsAny(key, "\r\n") && path.Clean(key) == key && key != "." &&
		!strings.HasPrefix(key, "../") && !strings.Contains(key, "/../")
}

func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for index := range parts {
		parts[index] = escapeSegment(parts[index])
	}
	return strings.Join(parts, "/")
}

func escapeSegment(value string) string {
	return strings.ReplaceAll(url.PathEscape(value), "+", "%20")
}

func hmacSHA256(key []byte, value string) []byte {
	digest := hmac.New(sha256.New, key)
	_, _ = digest.Write([]byte(value))
	return digest.Sum(nil)
}
