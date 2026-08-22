// Package sourceadapter translates public provider responses into canonical source records.
package sourceadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/source"
)

var ErrMalformed = errors.New("source adapter: malformed provider response")

type gitLabIssue struct {
	ID          int64     `json:"id"`
	IID         int64     `json:"iid"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	WebURL      string    `json:"web_url"`
	CreatedAt   time.Time `json:"created_at"`
}

// GitLabIssues maps GitLab issue DTOs at the adapter boundary.
func GitLabIssues(body []byte, evidenceIDs []int64) ([]source.Record, error) {
	var values []gitLabIssue
	if err := json.Unmarshal(body, &values); err != nil {
		return nil, fmt.Errorf("%w: decode gitlab issues", ErrMalformed)
	}
	if len(values) != len(evidenceIDs) {
		return nil, fmt.Errorf("%w: gitlab evidence cardinality", ErrMalformed)
	}
	records := make([]source.Record, 0, len(values))
	for index, value := range values {
		record, err := source.NewRecord(source.Record{Kind: source.KindGitLab,
			ExternalID: strconv.FormatInt(value.ID, 10), CanonicalURL: value.WebURL,
			Title: value.Title, Body: value.Description, ObservedAt: value.CreatedAt,
			EvidenceID: evidenceIDs[index], Attributes: map[string]string{"iid": strconv.FormatInt(value.IID, 10)}})
		if err != nil {
			return nil, fmt.Errorf("%w: gitlab issue %d", ErrMalformed, index)
		}
		records = append(records, record)
	}
	return records, nil
}

type giteaIssue struct {
	ID      int64     `json:"id"`
	Index   int64     `json:"number"`
	Title   string    `json:"title"`
	Body    string    `json:"body"`
	HTMLURL string    `json:"html_url"`
	Created time.Time `json:"created_at"`
}

// GiteaIssues maps Gitea issue DTOs at the adapter boundary.
func GiteaIssues(body []byte, evidenceIDs []int64) ([]source.Record, error) {
	var values []giteaIssue
	if err := json.Unmarshal(body, &values); err != nil {
		return nil, fmt.Errorf("%w: decode gitea issues", ErrMalformed)
	}
	if len(values) != len(evidenceIDs) {
		return nil, fmt.Errorf("%w: gitea evidence cardinality", ErrMalformed)
	}
	records := make([]source.Record, 0, len(values))
	for index, value := range values {
		record, err := source.NewRecord(source.Record{Kind: source.KindGitea,
			ExternalID: strconv.FormatInt(value.ID, 10), CanonicalURL: value.HTMLURL,
			Title: value.Title, Body: value.Body, ObservedAt: value.Created,
			EvidenceID: evidenceIDs[index], Attributes: map[string]string{"number": strconv.FormatInt(value.Index, 10)}})
		if err != nil {
			return nil, fmt.Errorf("%w: gitea issue %d", ErrMalformed, index)
		}
		records = append(records, record)
	}
	return records, nil
}

type RegistryValue struct {
	Registry   string
	Package    string
	URL        string
	Unit       string
	Population string
	Value      float64
	ObservedAt time.Time
	EvidenceID int64
}

// Registry converts registry-specific units only after the adapter provides a population context.
func Registry(value RegistryValue) (source.Record, error) {
	registry := strings.ToLower(strings.TrimSpace(value.Registry))
	if registry == "" || strings.TrimSpace(value.Unit) == "" ||
		strings.TrimSpace(value.Population) == "" || value.Value < 0 {
		return source.Record{}, ErrMalformed
	}
	return source.NewRecord(source.Record{Kind: source.KindRegistry, ExternalID: value.Package,
		CanonicalURL: value.URL, Title: value.Package, ObservedAt: value.ObservedAt,
		EvidenceID: value.EvidenceID, Attributes: map[string]string{
			"registry": registry, "unit": value.Unit, "population": value.Population,
			"value": strconv.FormatFloat(value.Value, 'f', -1, 64),
		}})
}

// Text creates a canonical record for an already decoded public discussion, advisory, feed,
// changelog, documentation, or website item.
func Text(kind source.Kind, externalID, rawURL, title, text, language string,
	observedAt time.Time, evidenceID int64) (source.Record, error) {
	switch kind {
	case source.KindAdvisory, source.KindDiscussion, source.KindChangelog, source.KindFeed,
		source.KindDocument, source.KindWebsite:
	default:
		return source.Record{}, ErrMalformed
	}
	return source.NewRecord(source.Record{Kind: kind, ExternalID: externalID,
		CanonicalURL: rawURL, Title: title, Body: text, Language: language,
		ObservedAt: observedAt, EvidenceID: evidenceID})
}
