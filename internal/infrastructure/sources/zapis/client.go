// Package zapis is a read-only adapter for the public zapis.kz catalogue.
package zapis

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
	Code          = "zapis_kz"
	maxBodyBytes  = 10 << 20
	requestTimout = 20 * time.Second
)

type Config struct {
	BaseURL      string // https://zapis.kz/rest/clients-app/v1
	AssetBaseURL string // https://zapis.kz, prefix for relative image paths
	UserAgent    string
	// Delay between requests: keep load on the source low.
	Delay time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) (*Client, error) {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	cfg.AssetBaseURL = strings.TrimRight(cfg.AssetBaseURL, "/")

	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("zapis base url: %w", err)
	}

	return &Client{cfg: cfg, http: &http.Client{Timeout: requestTimout}}, nil
}

// get fetches path with the city context header and decodes into target.
// Returns the raw body for snapshots.
func (c *Client) get(ctx context.Context, path string, cityID string, target any) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	if c.cfg.UserAgent != "" {
		req.Header.Set("User-Agent", c.cfg.UserAgent)
	}
	// City context is a header, not a query parameter.
	if cityID != "" {
		req.Header.Set("city_id", cityID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("GET %s: read body: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: status %d: %.300s", path, resp.StatusCode, body)
	}
	if err = json.Unmarshal(body, target); err != nil {
		return nil, fmt.Errorf("GET %s: decode: %w", path, err)
	}

	if err = c.pause(ctx); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Client) pause(ctx context.Context) error {
	if c.cfg.Delay <= 0 {
		return nil
	}
	timer := time.NewTimer(c.cfg.Delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) assetURL(path string) string {
	if path == "" || strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.cfg.AssetBaseURL + "/" + strings.TrimLeft(path, "/")
}
