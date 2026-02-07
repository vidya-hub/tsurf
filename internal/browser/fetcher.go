package browser

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout   = 15 * time.Second
	maxBodySize      = 10 * 1024 * 1024 // 10 MB
	defaultUserAgent = "tsurf/0.1 (terminal browser; +https://github.com/vidyasagar/tsurf)"
)

// SharedTransport is a tuned HTTP transport shared across all clients.
// This enables connection pooling and reuse across the application.
var SharedTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20, // Important for HN API (many requests to same host)
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ResponseHeaderTimeout: 15 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ForceAttemptHTTP2:     true,
}

// FetchResult holds the raw response from fetching a URL.
type FetchResult struct {
	URL         string
	FinalURL    string // after redirects
	StatusCode  int
	ContentType string
	Body        []byte
	Duration    time.Duration
}

// Fetcher handles HTTP requests with proper headers and timeouts.
type Fetcher struct {
	client    *http.Client
	userAgent string
}

// NewFetcher creates a Fetcher with sensible defaults using the shared transport.
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Transport: SharedTransport,
			Timeout:   defaultTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects (>10)")
				}
				return nil
			},
		},
		userAgent: defaultUserAgent,
	}
}

// Client returns the underlying HTTP client for use by other packages.
func (f *Fetcher) Client() *http.Client {
	return f.client
}

// Fetch retrieves the content at the given URL.
func (f *Fetcher) Fetch(rawURL string) (*FetchResult, error) {
	return f.FetchWithContext(context.Background(), rawURL)
}

// FetchWithContext retrieves content with a cancellable context.
func (f *Fetcher) FetchWithContext(ctx context.Context, rawURL string) (*FetchResult, error) {
	rawURL = normalizeURL(rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	start := time.Now()
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &FetchResult{
		URL:         rawURL,
		FinalURL:    resp.Request.URL.String(),
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        body,
		Duration:    time.Since(start),
	}, nil
}

// normalizeURL adds https:// if no scheme is present and handles search queries.
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}

	// If it already has a scheme, return as-is.
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}

	// If it looks like a domain (contains a dot and no spaces), add https.
	if strings.Contains(raw, ".") && !strings.Contains(raw, " ") {
		return "https://" + raw
	}

	// Otherwise treat as a DuckDuckGo search query.
	return "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(raw)
}

// IsHTML checks if the content type indicates HTML.
func IsHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml+xml")
}
