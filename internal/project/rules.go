package project

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const DefaultHistoryDays = 180

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type HistoryRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func ValidateHistoryRange(from, to, now time.Time, maximumDays int) (HistoryRange, error) {
	from = day(from)
	to = day(to)
	now = day(now)
	if from.IsZero() || to.IsZero() || !from.Before(to) || from.After(now) {
		return HistoryRange{}, fmt.Errorf("%w: history range must be ordered and not future-only", ErrInvalid)
	}
	if maximumDays > 0 && int(to.Sub(from).Hours()/24) > maximumDays {
		return HistoryRange{}, fmt.Errorf("%w: history range exceeds %d days", ErrInvalid, maximumDays)
	}
	if to.After(now.AddDate(0, 0, 1)) {
		to = now.AddDate(0, 0, 1)
	}
	return HistoryRange{From: from, To: to}, nil
}

func InitialHistoryRange(now time.Time, days int) HistoryRange {
	if days <= 0 {
		days = DefaultHistoryDays
	}
	to := day(now).AddDate(0, 0, 1)
	return HistoryRange{From: to.AddDate(0, 0, -days), To: to}
}

func CanonicalRepositoryURL(raw string) (provider, owner, name, canonical string, err error) {
	parsed, parseErr := url.ParseRequestURI(strings.TrimSpace(raw))
	if parseErr != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" ||
		parsed.Fragment != "" {
		return "", "", "", "", fmt.Errorf("%w: repository URL must be public HTTPS", ErrInvalid)
	}
	host := strings.ToLower(parsed.Hostname())
	if parsed.Port() != "" && parsed.Port() != "443" {
		return "", "", "", "", fmt.Errorf("%w: repository port is unsupported", ErrInvalid)
	}
	switch host {
	case "github.com":
		provider = "github"
	case "gitlab.com":
		provider = "gitlab"
	default:
		return "", "", "", "", fmt.Errorf("%w: repository provider is unsupported", ErrInvalid)
	}
	parts := strings.Split(strings.Trim(strings.TrimSuffix(parsed.Path, ".git"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", "", fmt.Errorf("%w: repository owner and name are required", ErrInvalid)
	}
	owner, name = strings.ToLower(parts[0]), strings.ToLower(parts[1])
	canonical = "https://" + host + "/" + owner + "/" + name
	return provider, owner, name, canonical, nil
}

func Slug(value string) (string, error) {
	slug := strings.Trim(strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", "-")), "-")
	if len(slug) == 0 || len(slug) > 100 || !slugPattern.MatchString(slug) {
		return "", fmt.Errorf("%w: invalid project slug", ErrInvalid)
	}
	return slug, nil
}

func (source Source) MarkUnavailable(reason string) (Source, error) {
	if strings.TrimSpace(reason) == "" {
		return Source{}, fmt.Errorf("%w: source failure reason is required", ErrInvalid)
	}
	source.State = SourceUnavailable
	source.Public = false
	source.Failure = strings.TrimSpace(reason)
	source.NextRunAt = nil
	source.Version++
	return source, nil
}

func day(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
