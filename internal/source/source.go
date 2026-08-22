// Package source defines provider-neutral public-source records accepted by the application.
package source

import (
	"errors"
	"strings"
	"time"
)

var ErrInvalid = errors.New("source: invalid canonical record")

type Kind string

const (
	KindGitLab     Kind = "gitlab"
	KindGitea      Kind = "gitea"
	KindRegistry   Kind = "registry"
	KindAdvisory   Kind = "advisory"
	KindDiscussion Kind = "discussion"
	KindChangelog  Kind = "changelog"
	KindFeed       Kind = "feed"
	KindDocument   Kind = "documentation"
	KindWebsite    Kind = "website"
)

// Record is the canonical adapter output. Payload is immutable raw evidence identity, never a
// provider DTO.
type Record struct {
	Kind         Kind              `json:"kind"`
	ExternalID   string            `json:"external_id"`
	CanonicalURL string            `json:"canonical_url"`
	Title        string            `json:"title"`
	Body         string            `json:"body"`
	Language     string            `json:"language"`
	ObservedAt   time.Time         `json:"observed_at"`
	EvidenceID   int64             `json:"evidence_id,string"`
	Attributes   map[string]string `json:"attributes"`
}

func NewRecord(value Record) (Record, error) {
	value.ExternalID = strings.TrimSpace(value.ExternalID)
	value.CanonicalURL = strings.TrimSpace(value.CanonicalURL)
	value.Language = strings.ToLower(strings.TrimSpace(value.Language))
	if value.Kind == "" || value.ExternalID == "" || !strings.HasPrefix(value.CanonicalURL, "https://") ||
		value.ObservedAt.IsZero() || value.EvidenceID <= 0 {
		return Record{}, ErrInvalid
	}
	if value.Attributes == nil {
		value.Attributes = map[string]string{}
	} else {
		copy := make(map[string]string, len(value.Attributes))
		for key, item := range value.Attributes {
			copy[key] = item
		}
		value.Attributes = copy
	}
	return value, nil
}
