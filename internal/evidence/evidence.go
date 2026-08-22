// Package evidence owns immutable byte identity and purge-manifest rules.
package evidence

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrInvalid = errors.New("invalid evidence")
	ErrCorrupt = errors.New("evidence checksum mismatch")
)

type Object struct {
	ID        int64    `json:"id,string"`
	ProjectID int64    `json:"project_id,string"`
	Key       string   `json:"key"`
	Digest    [32]byte `json:"digest"`
	Size      int64    `json:"size"`
	MediaType string   `json:"media_type"`
}

func NewObject(id, projectID int64, mediaType string, body []byte) (Object, error) {
	if id <= 0 || projectID <= 0 || strings.TrimSpace(mediaType) == "" {
		return Object{}, ErrInvalid
	}
	digest := sha256.Sum256(body)
	return Object{
		ID: id, ProjectID: projectID,
		Key:    fmt.Sprintf("projects/%d/sha256/%x", projectID, digest),
		Digest: digest, Size: int64(len(body)), MediaType: mediaType,
	}, nil
}

func (object Object) Verify(body []byte) error {
	if int64(len(body)) != object.Size || sha256.Sum256(body) != object.Digest {
		return ErrCorrupt
	}
	return nil
}

type PurgeManifest struct {
	ProjectID int64    `json:"project_id,string"`
	Keys      []string `json:"keys"`
	Deleted   []string `json:"deleted"`
}

func (manifest PurgeManifest) Remaining() []string {
	remaining := make([]string, 0, len(manifest.Keys))
	for _, key := range manifest.Keys {
		if !slices.Contains(manifest.Deleted, key) {
			remaining = append(remaining, key)
		}
	}
	return remaining
}

func (manifest PurgeManifest) MarkDeleted(key string) (PurgeManifest, error) {
	if !slices.Contains(manifest.Keys, key) {
		return PurgeManifest{}, ErrInvalid
	}
	if !slices.Contains(manifest.Deleted, key) {
		manifest.Deleted = append(manifest.Deleted, key)
	}
	return manifest, nil
}
