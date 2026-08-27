package export

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

type Format string

const (
	CSV          Format = "csv"
	EvidenceJSON Format = "evidence_json"
)

var (
	ErrInvalidRequest = errors.New("invalid export request")
	ErrTooLarge       = errors.New("export exceeds size quota")
	ErrBusy           = errors.New("export capacity exhausted")
	ErrNotReady       = errors.New("export artifact is not ready")
	ErrExpired        = errors.New("export artifact expired")
)

const DefaultMaxBytes int64 = 64 << 20

type Request struct {
	ProjectIDs []int64           `json:"project_ids"`
	Resource   string            `json:"resource"`
	Format     Format            `json:"format"`
	Filters    map[string]string `json:"filters,omitempty"`
	Locale     string            `json:"locale"`
	WindowFrom time.Time         `json:"window_from"`
	WindowTo   time.Time         `json:"window_to"`
	Cutoff     time.Time         `json:"cutoff"`
}

func (request Request) Validate() error {
	if len(request.ProjectIDs) == 0 || len(request.ProjectIDs) > 100 ||
		request.Resource == "" || request.Format != CSV && request.Format != EvidenceJSON ||
		request.Locale != "en" && request.Locale != "pt-BR" || request.WindowFrom.IsZero() ||
		request.WindowTo.IsZero() || request.Cutoff.IsZero() || request.WindowFrom.After(request.WindowTo) ||
		request.WindowTo.After(request.Cutoff) {
		return ErrInvalidRequest
	}
	seen := make(map[int64]struct{}, len(request.ProjectIDs))
	for _, id := range request.ProjectIDs {
		if id <= 0 {
			return ErrInvalidRequest
		}
		if _, exists := seen[id]; exists {
			return ErrInvalidRequest
		}
		seen[id] = struct{}{}
	}
	for key, value := range request.Filters {
		if strings.TrimSpace(key) == "" || len(key) > 64 || len(value) > 256 {
			return ErrInvalidRequest
		}
	}
	return nil
}

type MissingState string

const (
	Available        MissingState = "available"
	Unknown          MissingState = "unknown"
	NotApplicable    MissingState = "not_applicable"
	InsufficientData MissingState = "insufficient_data"
)

type Record struct {
	ProjectID      int64        `json:"project_id,string"`
	RepositoryID   int64        `json:"repository_id,string,omitempty"`
	Metric         string       `json:"metric"`
	Value          *float64     `json:"value,omitempty"`
	Status         MissingState `json:"status"`
	Unit           string       `json:"unit"`
	Definition     string       `json:"definition_version"`
	Formula        string       `json:"formula"`
	Coverage       string       `json:"coverage"`
	Provenance     []string     `json:"provenance"`
	PolicyContext  string       `json:"policy_context,omitempty"`
	AnalysisRunIDs []int64      `json:"analysis_run_ids,omitempty"`
}

type Package struct {
	SchemaVersion string            `json:"schema_version"`
	Resource      string            `json:"resource"`
	ProjectIDs    []int64           `json:"project_ids"`
	Filters       map[string]string `json:"filters,omitempty"`
	Locale        string            `json:"locale"`
	WindowFrom    time.Time         `json:"window_from"`
	WindowTo      time.Time         `json:"window_to"`
	Cutoff        time.Time         `json:"cutoff"`
	Rows          []Record          `json:"rows"`
}

type Generated struct {
	Body      []byte
	MediaType string
	Rows      int
	Digest    [32]byte
	DigestHex string
}

type Generator struct{ maxBytes int64 }

func NewGenerator(maxBytes int64) Generator {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	return Generator{maxBytes: maxBytes}
}

func (generator Generator) Generate(ctx context.Context, request Request, records []Record) (Generated, error) {
	if err := request.Validate(); err != nil {
		return Generated{}, err
	}
	if err := ctx.Err(); err != nil {
		return Generated{}, err
	}
	rows := slices.Clone(records)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ProjectID != rows[j].ProjectID {
			return rows[i].ProjectID < rows[j].ProjectID
		}
		if rows[i].RepositoryID != rows[j].RepositoryID {
			return rows[i].RepositoryID < rows[j].RepositoryID
		}
		return rows[i].Metric < rows[j].Metric
	})
	var body []byte
	var mediaType string
	var err error
	switch request.Format {
	case CSV:
		body, err = generateCSV(ctx, request.Locale, rows)
		mediaType = "text/csv; charset=utf-8"
	case EvidenceJSON:
		body, err = json.Marshal(Package{SchemaVersion: "opi.evidence-export/v1", Resource: request.Resource,
			ProjectIDs: slices.Clone(request.ProjectIDs), Filters: request.Filters, Locale: request.Locale,
			WindowFrom: request.WindowFrom.UTC(), WindowTo: request.WindowTo.UTC(), Cutoff: request.Cutoff.UTC(), Rows: rows})
		mediaType = "application/vnd.opi.evidence+json"
	default:
		err = ErrInvalidRequest
	}
	if err != nil {
		return Generated{}, err
	}
	if int64(len(body)) > generator.maxBytes {
		return Generated{}, ErrTooLarge
	}
	digest := sha256.Sum256(body)
	return Generated{Body: body, MediaType: mediaType, Rows: len(rows), Digest: digest,
		DigestHex: hex.EncodeToString(digest[:])}, nil
}

var machineColumns = []string{"project_id", "repository_id", "metric", "value", "status", "unit",
	"definition_version", "formula", "coverage", "provenance", "policy_context", "analysis_run_ids"}

var localizedColumns = map[string][]string{
	"en":    {"Project", "Repository", "Metric", "Value", "Status", "Unit", "Definition version", "Formula", "Coverage", "Provenance", "Policy context", "Analysis run IDs"},
	"pt-BR": {"Projeto", "Repositório", "Métrica", "Valor", "Estado", "Unidade", "Versão da definição", "Fórmula", "Cobertura", "Proveniência", "Contexto da política", "IDs das execuções de IA"},
}

func generateCSV(ctx context.Context, locale string, rows []Record) ([]byte, error) {
	var body bytes.Buffer
	writer := csv.NewWriter(&body)
	if err := writer.Write(machineColumns); err != nil {
		return nil, err
	}
	if err := writer.Write(localizedColumns[locale]); err != nil {
		return nil, err
	}
	for index, row := range rows {
		if index%128 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		value := ""
		if row.Value != nil {
			value = fmt.Sprintf("%.17g", *row.Value)
		}
		analysisIDs := make([]string, len(row.AnalysisRunIDs))
		for i, id := range row.AnalysisRunIDs {
			analysisIDs[i] = fmt.Sprint(id)
		}
		if err := writer.Write([]string{fmt.Sprint(row.ProjectID), nullableID(row.RepositoryID), row.Metric,
			value, string(row.Status), row.Unit, row.Definition, row.Formula, row.Coverage,
			strings.Join(row.Provenance, "|"), row.PolicyContext, strings.Join(analysisIDs, "|")}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func nullableID(id int64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprint(id)
}

// Coordinator provides bounded in-process generation. Durable Jobs remain the
// owner of retries and lifecycle; this gate protects memory and CPU per process.
type Coordinator struct {
	gate chan struct{}
	gen  Generator
}

func NewCoordinator(concurrency int, maxBytes int64) (*Coordinator, error) {
	if concurrency <= 0 || concurrency > 64 {
		return nil, ErrInvalidRequest
	}
	return &Coordinator{gate: make(chan struct{}, concurrency), gen: NewGenerator(maxBytes)}, nil
}

func (coordinator *Coordinator) Generate(ctx context.Context, request Request, records []Record) (Generated, error) {
	select {
	case coordinator.gate <- struct{}{}:
		defer func() { <-coordinator.gate }()
	case <-ctx.Done():
		return Generated{}, ctx.Err()
	default:
		return Generated{}, ErrBusy
	}
	return coordinator.gen.Generate(ctx, request, records)
}
