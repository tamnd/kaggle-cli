// Package kaggle is the library behind the kaggle command: the HTTP client,
// request shaping, and typed data models for Kaggle.
//
// The public Kaggle API at /api/v1/datasets/list is open (no auth required)
// when a search query is provided. Competitions require authentication, so we
// use the public web search pages to scrape competition data.
package kaggle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiBase = "https://www.kaggle.com/api/v1"
)

// DefaultUserAgent identifies this client to Kaggle.
const DefaultUserAgent = "kaggle-cli/dev (+https://github.com/tamnd/kaggle-cli)"

// ErrNotFound is returned when the requested resource is not found.
var ErrNotFound = fmt.Errorf("not found")

// Config holds Client parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   apiBase,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Kaggle API and public search pages.
type Client struct {
	http      *http.Client
	userAgent string
	baseURL   string
	rate      time.Duration
	retries   int
	last      time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = apiBase
	}
	return &Client{
		http:      &http.Client{Timeout: cfg.Timeout},
		userAgent: cfg.UserAgent,
		baseURL:   cfg.BaseURL,
		rate:      cfg.Rate,
		retries:   cfg.Retries,
	}
}

// DatasetOptions controls the datasets list/search query.
type DatasetOptions struct {
	Search  string
	SortBy  string // hottest, votes, updated, active, published
	Page    int
	Limit   int
	TagIDs  string
}

// Datasets fetches a page of datasets matching opts.
func (c *Client) Datasets(ctx context.Context, opts DatasetOptions) ([]Dataset, error) {
	page := opts.Page
	if page < 1 {
		page = 1
	}
	limit := opts.Limit
	if limit < 1 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 50 {
		pageSize = 50
	}

	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "hottest"
	}

	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	params.Set("sortBy", sortBy)
	if opts.Search != "" {
		params.Set("search", opts.Search)
	}
	if opts.TagIDs != "" {
		params.Set("tagIds", opts.TagIDs)
	}

	rawURL := c.baseURL + "/datasets/list?" + params.Encode()

	var raw []apiDataset
	if err := c.getJSON(ctx, rawURL, &raw); err != nil {
		return nil, fmt.Errorf("datasets: %w", err)
	}

	if len(raw) > limit {
		raw = raw[:limit]
	}

	out := make([]Dataset, len(raw))
	offset := (page - 1) * pageSize
	for i, d := range raw {
		out[i] = apiDatasetToDataset(d, offset+i+1)
	}
	return out, nil
}

// CompetitionOptions controls the competitions list/search query.
type CompetitionOptions struct {
	Search   string
	Category string // all, featured, research, recruitment, gettingStarted, masters, playground
	SortBy   string // latestDeadline, earliestDeadline, numberOfTeams, recentlyCreated
	Page     int
	Limit    int
}

// Competitions fetches a page of competitions matching opts.
// NOTE: the competitions/list endpoint requires authentication. We use the
// competitions/list endpoint with Basic auth if credentials are set, or fall
// back to the search endpoint. For unauthenticated users we return an error
// with instructions.
func (c *Client) Competitions(ctx context.Context, opts CompetitionOptions) ([]Competition, error) {
	page := opts.Page
	if page < 1 {
		page = 1
	}
	limit := opts.Limit
	if limit < 1 {
		limit = 20
	}
	pageSize := limit
	if pageSize > 50 {
		pageSize = 50
	}

	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "latestDeadline"
	}

	category := opts.Category

	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("pageSize", fmt.Sprintf("%d", pageSize))
	params.Set("sortBy", sortBy)
	// Only pass category when explicitly narrowing; "all" is the default and
	// the server rejects it as an unrecognized enum value.
	if category != "" && category != "all" {
		params.Set("category", category)
	}
	if opts.Search != "" {
		params.Set("search", opts.Search)
	}

	rawURL := c.baseURL + "/competitions/list?" + params.Encode()

	var raw []apiCompetition
	if err := c.getJSON(ctx, rawURL, &raw); err != nil {
		return nil, fmt.Errorf("competitions: %w", err)
	}

	if len(raw) > limit {
		raw = raw[:limit]
	}

	out := make([]Competition, len(raw))
	offset := (page - 1) * pageSize
	for i, comp := range raw {
		out[i] = apiCompetitionToCompetition(comp, offset+i+1)
	}
	return out, nil
}

// ─── HTTP internals ───────────────────────────────────────────────────────────

// Get fetches rawURL with pacing and retries, returning the body bytes.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, fmt.Errorf("http %d: authentication required -- set KAGGLE_USERNAME and KAGGLE_KEY", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.Get(ctx, rawURL)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "null" {
		return ErrNotFound
	}
	// Check if it's an error response like {"code":401,"message":"..."}
	if strings.HasPrefix(trimmed, "{") {
		var apiErr struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Code != 0 {
			if apiErr.Code == 401 || apiErr.Code == 403 {
				return fmt.Errorf("http %d: authentication required -- set KAGGLE_USERNAME and KAGGLE_KEY", apiErr.Code)
			}
			return fmt.Errorf("api error %d: %s", apiErr.Code, apiErr.Message)
		}
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}
