package browser

import (
	"bytes"
	"fmt"
	"net/url"
	"time"

	readability "github.com/go-shiori/go-readability"
)

// Article holds the extracted readable content from a page.
type Article struct {
	Title       string
	Byline      string
	Content     string // cleaned HTML
	TextContent string // plain text
	Excerpt     string
	SiteName    string
	URL         string
	FinalURL    string
	FetchTime   time.Duration
	Links       []Link
}

// Link represents a hyperlink found in the page content.
type Link struct {
	Index int
	Text  string
	URL   string
}

// Extract takes a FetchResult and extracts the readable article content.
// Note: Links are populated by the renderer during the Render() call,
// not during extraction, to avoid duplicate parsing.
func Extract(result *FetchResult) (*Article, error) {
	if !IsHTML(result.ContentType) {
		return &Article{
			Title:       result.FinalURL,
			Content:     "<pre>" + string(result.Body) + "</pre>",
			TextContent: string(result.Body),
			URL:         result.URL,
			FinalURL:    result.FinalURL,
			FetchTime:   result.Duration,
		}, nil
	}

	parsedURL, err := url.Parse(result.FinalURL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	article, err := readability.FromReader(bytes.NewReader(result.Body), parsedURL)
	if err != nil {
		return nil, fmt.Errorf("extracting article: %w", err)
	}

	return &Article{
		Title:       article.Title,
		Byline:      article.Byline,
		Content:     article.Content,
		TextContent: article.TextContent,
		Excerpt:     article.Excerpt,
		SiteName:    article.SiteName,
		URL:         result.URL,
		FinalURL:    result.FinalURL,
		FetchTime:   result.Duration,
		Links:       nil, // Links are populated by the renderer
	}, nil
}
