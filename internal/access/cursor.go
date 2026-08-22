package access

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
)

type Cursor struct {
	Route   string `json:"route"`
	Filters string `json:"filters"`
	Offset  int    `json:"offset"`
}

type CursorCodec struct {
	key []byte
}

func NewCursorCodec(key []byte) (*CursorCodec, error) {
	if len(key) < 32 {
		return nil, errors.New("cursor signing key must contain at least 32 bytes")
	}
	return &CursorCodec{key: append([]byte(nil), key...)}, nil
}

func (c *CursorCodec) Encode(cursor Cursor) (string, error) {
	if cursor.Route == "" || cursor.Offset < 0 {
		return "", errors.New("cursor route and non-negative offset are required")
	}
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(payload)
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payload) + "." +
		base64.RawURLEncoding.EncodeToString(signature), nil
}

func (c *CursorCodec) Decode(value, route, filters string) (Cursor, error) {
	payloadText, signatureText, found := cutCursor(value)
	if !found {
		return Cursor{}, errors.New("invalid cursor")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadText)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	signature, err := base64.RawURLEncoding.DecodeString(signatureText)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return Cursor{}, errors.New("invalid cursor")
	}
	var cursor Cursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.Route != route ||
		cursor.Filters != filters || cursor.Offset < 0 {
		return Cursor{}, errors.New("invalid cursor")
	}
	return cursor, nil
}

func cutCursor(value string) (string, string, bool) {
	for index := range len(value) {
		if value[index] == '.' {
			return value[:index], value[index+1:], true
		}
	}
	return "", "", false
}
