package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotPublic   = errors.New("GitHub resource is not public")
	ErrRateLimited = errors.New("GitHub rate limit reached")
)

type RateLimitError struct {
	Reset time.Time
}

func (err *RateLimitError) Error() string {
	if err.Reset.IsZero() {
		return ErrRateLimited.Error()
	}
	return fmt.Sprintf("%s: reset at %s", ErrRateLimited, err.Reset.UTC())
}

func (*RateLimitError) Unwrap() error { return ErrRateLimited }

type Client struct {
	HTTP       *http.Client
	BaseURL    *url.URL
	Token      string
	MaxPayload int64
}

type Repository struct {
	ExternalID    int64
	CanonicalURL  string
	Name          string
	Description   string
	DefaultBranch string
	Owner         string
	UpdatedAt     time.Time
	Raw           json.RawMessage
}

type Issue struct {
	ExternalID int64
	Number     int64
	Title      string
	State      string
	Author     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ClosedAt   *time.Time
	Raw        json.RawMessage
}

type PullRequest struct {
	ExternalID int64
	Number     int64
	Title      string
	State      string
	CreatedAt  time.Time
	ReadyAt    *time.Time
	MergedAt   *time.Time
	Raw        json.RawMessage
}

type Release struct {
	ExternalID  int64
	Tag         string
	Draft       bool
	Prerelease  bool
	PublishedAt *time.Time
	Raw         json.RawMessage
}

type Commit struct {
	ExternalID       string
	SHA              string
	AuthorExternalID string
	CommittedAt      time.Time
	DefaultBranch    bool
	MergeCommit      bool
	Raw              json.RawMessage
}

func (client Client) Repository(ctx context.Context, owner, name string) (Repository, error) {
	if !validSegment(owner) || !validSegment(name) {
		return Repository{}, errors.New("GitHub owner and repository are required")
	}
	var dto repositoryDTO
	raw, err := client.get(ctx, "/repos/"+url.PathEscape(owner)+"/"+url.PathEscape(name), &dto)
	if err != nil {
		return Repository{}, err
	}
	if dto.Private || dto.Visibility != "" && dto.Visibility != "public" {
		return Repository{}, ErrNotPublic
	}
	return Repository{
		ExternalID: dto.ID, CanonicalURL: dto.HTMLURL, Name: dto.Name,
		Description: dto.Description, DefaultBranch: dto.DefaultBranch,
		Owner: dto.Owner.Login, UpdatedAt: dto.UpdatedAt.UTC(), Raw: raw,
	}, nil
}

func (client Client) Issues(ctx context.Context, owner, name string, page int) ([]Issue, error) {
	if !validSegment(owner) || !validSegment(name) || page <= 0 {
		return nil, errors.New("GitHub owner, repository, and positive page are required")
	}
	var dtos []issueDTO
	endpoint := fmt.Sprintf("/repos/%s/%s/issues?state=all&per_page=100&page=%d",
		url.PathEscape(owner), url.PathEscape(name), page)
	_, err := client.get(ctx, endpoint, &dtos)
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(dtos))
	for _, dto := range dtos {
		if dto.PullRequest != nil {
			continue
		}
		raw, marshalErr := json.Marshal(dto)
		if marshalErr != nil {
			return nil, fmt.Errorf("retain GitHub issue provenance: %w", marshalErr)
		}
		issues = append(issues, Issue{
			ExternalID: dto.ID, Number: dto.Number, Title: dto.Title, State: dto.State,
			Author: dto.User.Login, CreatedAt: dto.CreatedAt.UTC(), UpdatedAt: dto.UpdatedAt.UTC(),
			ClosedAt: utcPointer(dto.ClosedAt), Raw: raw,
		})
	}
	return issues, nil
}

func (client Client) PullRequests(ctx context.Context, owner, name string, page int) ([]PullRequest, error) {
	if !validSegment(owner) || !validSegment(name) || page <= 0 {
		return nil, errors.New("GitHub owner, repository, and positive page are required")
	}
	var dtos []pullRequestDTO
	endpoint := fmt.Sprintf("/repos/%s/%s/pulls?state=all&per_page=100&page=%d",
		url.PathEscape(owner), url.PathEscape(name), page)
	if _, err := client.get(ctx, endpoint, &dtos); err != nil {
		return nil, err
	}
	values := make([]PullRequest, 0, len(dtos))
	for _, dto := range dtos {
		raw, err := json.Marshal(dto)
		if err != nil {
			return nil, fmt.Errorf("retain GitHub pull-request provenance: %w", err)
		}
		state := dto.State
		if dto.MergedAt != nil {
			state = "merged"
		}
		var readyAt *time.Time
		if !dto.Draft {
			readyAt = utcPointer(&dto.CreatedAt)
		}
		values = append(values, PullRequest{
			ExternalID: dto.ID, Number: dto.Number, Title: dto.Title, State: state,
			CreatedAt: dto.CreatedAt.UTC(), ReadyAt: readyAt, MergedAt: utcPointer(dto.MergedAt), Raw: raw,
		})
	}
	return values, nil
}

func (client Client) Releases(ctx context.Context, owner, name string, page int) ([]Release, error) {
	if !validSegment(owner) || !validSegment(name) || page <= 0 {
		return nil, errors.New("GitHub owner, repository, and positive page are required")
	}
	var dtos []releaseDTO
	endpoint := fmt.Sprintf("/repos/%s/%s/releases?per_page=100&page=%d",
		url.PathEscape(owner), url.PathEscape(name), page)
	if _, err := client.get(ctx, endpoint, &dtos); err != nil {
		return nil, err
	}
	values := make([]Release, 0, len(dtos))
	for _, dto := range dtos {
		raw, err := json.Marshal(dto)
		if err != nil {
			return nil, fmt.Errorf("retain GitHub release provenance: %w", err)
		}
		values = append(values, Release{
			ExternalID: dto.ID, Tag: dto.TagName, Draft: dto.Draft, Prerelease: dto.Prerelease,
			PublishedAt: utcPointer(dto.PublishedAt), Raw: raw,
		})
	}
	return values, nil
}

func (client Client) Commits(
	ctx context.Context,
	owner, name, defaultBranch string,
	page int,
) ([]Commit, error) {
	if !validSegment(owner) || !validSegment(name) || !validSegment(defaultBranch) || page <= 0 {
		return nil, errors.New("GitHub owner, repository, branch, and positive page are required")
	}
	var dtos []commitDTO
	endpoint := fmt.Sprintf("/repos/%s/%s/commits?sha=%s&per_page=100&page=%d",
		url.PathEscape(owner), url.PathEscape(name), url.QueryEscape(defaultBranch), page)
	if _, err := client.get(ctx, endpoint, &dtos); err != nil {
		return nil, err
	}
	values := make([]Commit, 0, len(dtos))
	for _, dto := range dtos {
		raw, err := json.Marshal(dto)
		if err != nil {
			return nil, fmt.Errorf("retain GitHub commit provenance: %w", err)
		}
		authorID := ""
		if dto.Author != nil && dto.Author.ID > 0 {
			authorID = strconv.FormatInt(dto.Author.ID, 10)
		}
		values = append(values, Commit{
			ExternalID: dto.SHA, SHA: dto.SHA, AuthorExternalID: authorID,
			CommittedAt: dto.Commit.Committer.Date.UTC(), DefaultBranch: true,
			MergeCommit: len(dto.Parents) > 1, Raw: raw,
		})
	}
	return values, nil
}

func (client Client) get(ctx context.Context, endpoint string, destination any) (json.RawMessage, error) {
	base := client.BaseURL
	if base == nil {
		base, _ = url.Parse("https://api.github.com")
	}
	target, err := base.Parse(endpoint)
	if err != nil || target.Scheme != "https" && target.Hostname() != "127.0.0.1" && target.Hostname() != "localhost" {
		return nil, errors.New("invalid GitHub API endpoint")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build GitHub request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if client.Token != "" {
		request.Header.Set("Authorization", "Bearer "+client.Token)
	}
	httpClient := client.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request GitHub public data: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusForbidden &&
		strings.EqualFold(response.Header.Get("X-Resource-Visibility"), "private") {
		return nil, ErrNotPublic
	}
	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusForbidden &&
		response.Header.Get("X-RateLimit-Remaining") == "0" {
		reset, _ := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64)
		return nil, &RateLimitError{Reset: time.Unix(reset, 0).UTC()}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub returned status %d", response.StatusCode)
	}
	limit := client.MaxPayload
	if limit <= 0 {
		limit = 8 << 20
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("read GitHub response: %w", err)
	}
	if int64(len(raw)) > limit {
		return nil, errors.New("GitHub response exceeds the configured byte limit")
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		return nil, fmt.Errorf("decode GitHub response: %w", err)
	}
	return raw, nil
}

func validSegment(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, "/\\\r\n")
}

func utcPointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

type repositoryDTO struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Private       bool      `json:"private"`
	Visibility    string    `json:"visibility"`
	HTMLURL       string    `json:"html_url"`
	DefaultBranch string    `json:"default_branch"`
	UpdatedAt     time.Time `json:"updated_at"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type issueDTO struct {
	ID          int64      `json:"id"`
	Number      int64      `json:"number"`
	Title       string     `json:"title"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	PullRequest any        `json:"pull_request"`
	User        struct {
		Login string `json:"login"`
	} `json:"user"`
}

type pullRequestDTO struct {
	ID        int64      `json:"id"`
	Number    int64      `json:"number"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Draft     bool       `json:"draft"`
	CreatedAt time.Time  `json:"created_at"`
	MergedAt  *time.Time `json:"merged_at"`
}

type releaseDTO struct {
	ID          int64      `json:"id"`
	TagName     string     `json:"tag_name"`
	Draft       bool       `json:"draft"`
	Prerelease  bool       `json:"prerelease"`
	PublishedAt *time.Time `json:"published_at"`
}

type commitDTO struct {
	SHA    string `json:"sha"`
	Author *struct {
		ID int64 `json:"id"`
	} `json:"author"`
	Commit struct {
		Committer struct {
			Date time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
}
