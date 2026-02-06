package browser

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
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

	links := extractLinks(article.Content, result.FinalURL)

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
		Links:       links,
	}, nil
}

// extractLinks parses links from HTML content and assigns numbered indices.
func extractLinks(htmlContent string, baseURL string) []Link {
	var links []Link
	base, _ := url.Parse(baseURL)

	idx := 0
	remaining := htmlContent
	for {
		hrefStart := strings.Index(remaining, "href=\"")
		if hrefStart == -1 {
			break
		}
		remaining = remaining[hrefStart+6:]

		hrefEnd := strings.Index(remaining, "\"")
		if hrefEnd == -1 {
			break
		}
		href := remaining[:hrefEnd]
		remaining = remaining[hrefEnd:]

		// Skip anchors and javascript.
		if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			continue
		}

		// Resolve relative URLs.
		resolved := href
		if base != nil {
			if parsed, err := url.Parse(href); err == nil {
				resolved = base.ResolveReference(parsed).String()
			}
		}

		// Extract link text (simplified: look for > and <).
		text := ""
		closeTag := strings.Index(remaining, ">")
		if closeTag != -1 {
			endTag := strings.Index(remaining[closeTag:], "<")
			if endTag != -1 {
				text = strings.TrimSpace(remaining[closeTag+1 : closeTag+endTag])
			}
		}
		if text == "" {
			text = resolved
		}

		idx++
		links = append(links, Link{
			Index: idx,
			Text:  text,
			URL:   resolved,
		})
	}

	return links
}
