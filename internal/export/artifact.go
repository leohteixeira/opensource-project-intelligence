// Package export owns immutable export identity and download expiry rules.
package export

import (
	"crypto/sha256"
	"errors"
	"time"
)

var ErrGone = errors.New("export expired")

const Lifetime = 24 * time.Hour

type Artifact struct {
	ProjectID   int64
	Key         string
	MediaType   string
	Digest      [32]byte
	Size        int64
	CompletedAt time.Time
}

func NewArtifact(projectID int64, key, mediaType string, body []byte, completedAt time.Time) (Artifact, error) {
	if projectID <= 0 || key == "" || mediaType == "" || completedAt.IsZero() {
		return Artifact{}, errors.New("invalid export artifact")
	}
	return Artifact{ProjectID: projectID, Key: key, MediaType: mediaType,
		Digest: sha256.Sum256(body), Size: int64(len(body)), CompletedAt: completedAt.UTC()}, nil
}

func (artifact Artifact) Authorize(projectID int64, at time.Time) error {
	if artifact.ProjectID != projectID || projectID <= 0 {
		return errors.New("export authorization failed")
	}
	if !at.Before(artifact.CompletedAt.Add(Lifetime)) {
		return ErrGone
	}
	return nil
}

func (artifact Artifact) Verify(body []byte) bool {
	return int64(len(body)) == artifact.Size && sha256.Sum256(body) == artifact.Digest
}
