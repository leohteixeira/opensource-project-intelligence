// Package knowledge owns bounded crawl contracts, deterministic chunks, and hybrid retrieval.
package knowledge

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalid       = errors.New("knowledge: invalid input")
	ErrLimitExceeded = errors.New("knowledge: crawl limit exceeded")
	ErrNoEvidence    = errors.New("knowledge: no accessible evidence")
)

type Limits struct {
	MaxDomains        int      `json:"max_domains"`
	MaxDepth          int      `json:"max_depth"`
	MaxPages          int      `json:"max_pages"`
	MaxPageBytes      int64    `json:"max_page_bytes"`
	MaxTotalBytes     int64    `json:"max_total_bytes"`
	RequestsPerMinute int      `json:"requests_per_minute"`
	MediaTypes        []string `json:"media_types"`
}

func (limits Limits) Validate() error {
	if limits.MaxDomains < 1 || limits.MaxDomains > 16 || limits.MaxDepth < 0 || limits.MaxDepth > 8 ||
		limits.MaxPages < 1 || limits.MaxPages > 10_000 || limits.MaxPageBytes < 1 ||
		limits.MaxTotalBytes < limits.MaxPageBytes || limits.RequestsPerMinute < 1 ||
		limits.RequestsPerMinute > 6_000 || len(limits.MediaTypes) == 0 {
		return ErrInvalid
	}
	return nil
}

type Budget struct {
	Limits  Limits
	Domains map[string]struct{}
	Pages   int
	Bytes   int64
}

// Accept accounts for one page before parsing it so a rejected or malformed page still consumes
// the externally controlled resource budget.
func (budget *Budget) Accept(host string, depth int, size int64, mediaType string) error {
	if budget == nil || budget.Limits.Validate() != nil || strings.TrimSpace(host) == "" ||
		depth < 0 || size < 0 || strings.TrimSpace(mediaType) == "" {
		return ErrInvalid
	}
	if budget.Domains == nil {
		budget.Domains = make(map[string]struct{})
	}
	host = strings.ToLower(strings.TrimSpace(host))
	budget.Domains[host] = struct{}{}
	budget.Pages++
	budget.Bytes += size
	if len(budget.Domains) > budget.Limits.MaxDomains || depth > budget.Limits.MaxDepth ||
		budget.Pages > budget.Limits.MaxPages || size > budget.Limits.MaxPageBytes ||
		budget.Bytes > budget.Limits.MaxTotalBytes || !slices.Contains(budget.Limits.MediaTypes, mediaType) {
		return ErrLimitExceeded
	}
	return nil
}

type Snapshot struct {
	ID         int64     `json:"id,string"`
	ProjectID  int64     `json:"project_id,string"`
	SourceID   int64     `json:"source_id,string"`
	URL        string    `json:"url"`
	ObservedAt time.Time `json:"observed_at"`
	Digest     [32]byte  `json:"digest"`
	MediaType  string    `json:"media_type"`
	Language   string    `json:"language"`
	Current    bool      `json:"current"`
}

func NewSnapshot(value Snapshot, body []byte) (Snapshot, error) {
	if value.ID <= 0 || value.ProjectID <= 0 || value.SourceID <= 0 ||
		!strings.HasPrefix(value.URL, "https://") || value.ObservedAt.IsZero() ||
		strings.TrimSpace(value.MediaType) == "" || len(body) == 0 {
		return Snapshot{}, ErrInvalid
	}
	value.Digest = sha256.Sum256(body)
	return value, nil
}

type Chunk struct {
	ID            int64     `json:"id,string"`
	ProjectID     int64     `json:"project_id,string"`
	SourceID      int64     `json:"source_id,string"`
	SnapshotID    int64     `json:"snapshot_id,string"`
	Heading       string    `json:"heading"`
	Text          string    `json:"text"`
	Language      string    `json:"language"`
	StartOffset   int       `json:"start_offset"`
	EndOffset     int       `json:"end_offset"`
	ParserVersion string    `json:"parser_version"`
	ObservedAt    time.Time `json:"observed_at"`
	Current       bool      `json:"current"`
}

// ChunkText splits UTF-8 text deterministically at headings and rune-safe byte bounds.
func ChunkText(snapshot Snapshot, text, parserVersion string, maxBytes int) ([]Chunk, error) {
	if snapshot.ID <= 0 || !utf8.ValidString(text) || strings.TrimSpace(parserVersion) == "" || maxBytes < 64 {
		return nil, ErrInvalid
	}
	type section struct {
		heading, body string
		offset        int
	}
	sections := make([]section, 0)
	current := section{offset: 0}
	offset := 0
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			if current.body != "" {
				sections = append(sections, current)
			}
			current = section{heading: strings.TrimSpace(strings.TrimLeft(trimmed, "#")), offset: offset}
		} else {
			if current.body != "" {
				current.body += "\n"
			}
			current.body += line
		}
		offset += len(line) + 1
	}
	if current.body != "" || current.heading != "" {
		sections = append(sections, current)
	}
	chunks := make([]Chunk, 0, len(sections))
	for _, value := range sections {
		body := strings.TrimSpace(value.body)
		for consumed := 0; consumed < len(body); {
			end := min(len(body), consumed+maxBytes)
			for end < len(body) && end > consumed && !utf8.RuneStart(body[end]) {
				end--
			}
			if end <= consumed {
				return nil, ErrInvalid
			}
			part := strings.TrimSpace(body[consumed:end])
			if part != "" {
				chunks = append(chunks, Chunk{ProjectID: snapshot.ProjectID, SourceID: snapshot.SourceID,
					SnapshotID: snapshot.ID, Heading: value.heading, Text: part,
					Language: snapshot.Language, StartOffset: value.offset + consumed,
					EndOffset: value.offset + end, ParserVersion: parserVersion,
					ObservedAt: snapshot.ObservedAt, Current: snapshot.Current})
			}
			consumed = end
		}
	}
	return chunks, nil
}

type Candidate struct {
	Chunk       Chunk
	LexicalRank int
	VectorRank  int
}

type Filter struct {
	ProjectID int64
	SourceIDs []int64
	Cutoff    time.Time
}

type Result struct {
	Chunk Chunk    `json:"chunk"`
	Score float64  `json:"score"`
	Modes []string `json:"modes"`
}

// Fuse applies authorization/current/cutoff filters before deterministic versioned RRF ranking.
func Fuse(candidates []Candidate, filter Filter, limit, rankConstant int) ([]Result, error) {
	if filter.ProjectID <= 0 || filter.Cutoff.IsZero() || limit < 1 || limit > 100 || rankConstant < 1 {
		return nil, ErrInvalid
	}
	byChunk := make(map[int64]Result)
	for _, candidate := range candidates {
		chunk := candidate.Chunk
		if chunk.ID <= 0 || chunk.ProjectID != filter.ProjectID || !chunk.Current ||
			chunk.ObservedAt.After(filter.Cutoff) || len(filter.SourceIDs) > 0 && !slices.Contains(filter.SourceIDs, chunk.SourceID) {
			continue
		}
		result := byChunk[chunk.ID]
		result.Chunk = chunk
		if candidate.LexicalRank > 0 {
			result.Score += 1 / float64(rankConstant+candidate.LexicalRank)
			result.Modes = appendMode(result.Modes, "lexical")
		}
		if candidate.VectorRank > 0 {
			result.Score += 1 / float64(rankConstant+candidate.VectorRank)
			result.Modes = appendMode(result.Modes, "vector")
		}
		if result.Score > 0 && !math.IsNaN(result.Score) {
			byChunk[chunk.ID] = result
		}
	}
	results := make([]Result, 0, len(byChunk))
	for _, result := range byChunk {
		results = append(results, result)
	}
	slices.SortFunc(results, func(left, right Result) int {
		if left.Score > right.Score {
			return -1
		}
		if left.Score < right.Score {
			return 1
		}
		if left.Chunk.SourceID < right.Chunk.SourceID {
			return -1
		}
		if left.Chunk.SourceID > right.Chunk.SourceID {
			return 1
		}
		if left.Chunk.ID < right.Chunk.ID {
			return -1
		}
		if left.Chunk.ID > right.Chunk.ID {
			return 1
		}
		return 0
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func appendMode(values []string, value string) []string {
	if slices.Contains(values, value) {
		return values
	}
	return append(values, value)
}

type Citation struct {
	SnapshotID  int64 `json:"snapshot_id,string"`
	ChunkID     int64 `json:"chunk_id,string"`
	StartOffset int   `json:"start_offset"`
	EndOffset   int   `json:"end_offset"`
}

func Cite(results []Result) ([]Citation, error) {
	if len(results) == 0 {
		return nil, ErrNoEvidence
	}
	values := make([]Citation, 0, len(results))
	for _, result := range results {
		chunk := result.Chunk
		if chunk.SnapshotID <= 0 || chunk.ID <= 0 || chunk.StartOffset < 0 || chunk.EndOffset <= chunk.StartOffset {
			return nil, fmt.Errorf("%w: invalid citation", ErrInvalid)
		}
		values = append(values, Citation{SnapshotID: chunk.SnapshotID, ChunkID: chunk.ID,
			StartOffset: chunk.StartOffset, EndOffset: chunk.EndOffset})
	}
	return values, nil
}
