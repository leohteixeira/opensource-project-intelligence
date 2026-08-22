package access

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const tokenBytes = 32

func NewSecret() (string, [32]byte, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", [32]byte{}, fmt.Errorf("generate secret: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	return encoded, sha256.Sum256([]byte(encoded)), nil
}

func EncodeSessionToken(id int64, verifier string) string {
	return strconv.FormatInt(id, 10) + "." + verifier
}

func DecodeSessionToken(token string) (int64, string, error) {
	idText, verifier, found := strings.Cut(token, ".")
	if !found || verifier == "" {
		return 0, "", errors.New("invalid session token")
	}
	id, err := strconv.ParseInt(idText, 10, 64)
	if err != nil || id <= 0 {
		return 0, "", errors.New("invalid session token")
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(verifier); err != nil || len(decoded) != tokenBytes {
		return 0, "", errors.New("invalid session token")
	}
	return id, verifier, nil
}

func VerifySecret(verifier string, expected [32]byte) bool {
	actual := sha256.Sum256([]byte(verifier))
	return subtle.ConstantTimeCompare(actual[:], expected[:]) == 1
}
