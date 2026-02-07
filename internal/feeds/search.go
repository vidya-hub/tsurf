package feeds

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/vidyasagar/tsurf/internal/browser"
)

// SearchResult represents a single search result.
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// SearchDDG performs a search on DuckDuckGo HTML version and parses results.
// Uses the shared HTTP transport for connection reuse.
func SearchDDG(query string) ([]SearchResult, error) {
	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)

	// Use a fetcher with shared transport for connection pooling.
	fetcher := browser.NewFetcher()
	result, err := fetcher.Fetch(searchURL)
	if err != nil {
		return nil, fmt.Errorf("searching DuckDuckGo: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(result.Body)))
	if err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	var results []SearchResult

	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		titleEl := s.Find(".result__a")
		title := strings.TrimSpace(titleEl.Text())

		href, exists := titleEl.Attr("href")
		if !exists {
			return
		}

		// DDG wraps URLs in a redirect. Extract the real URL.
		realURL := extractDDGURL(href)

		snippetEl := s.Find(".result__snippet")
		snippet := strings.TrimSpace(snippetEl.Text())

		if title != "" && realURL != "" {
			results = append(results, SearchResult{
				Title:   title,
				URL:     realURL,
				Snippet: snippet,
			})
		}
	})

	return results, nil
}

// extractDDGURL extracts the real URL from a DDG redirect URL.
func extractDDGURL(href string) string {
	// DDG links look like: //duckduckgo.com/l/?uddg=<encoded_url>&rut=...
	if strings.Contains(href, "uddg=") {
		if parsed, err := url.Parse(href); err == nil {
			if uddg := parsed.Query().Get("uddg"); uddg != "" {
				return uddg
			}
		}
	}

	// Sometimes they're direct links.
	if strings.HasPrefix(href, "http") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	return href
}

// RenderSearchResults formats search results for the viewport.
func RenderSearchResults(results []SearchResult, query string) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link

	sb.WriteString(fmt.Sprintf("  🔍 Search: %s\n", query))
	sb.WriteString(fmt.Sprintf("  %s\n\n", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	if len(results) == 0 {
		sb.WriteString("  No results found.\n")
		return sb.String(), links
	}

	for i, r := range results {
		idx := i + 1
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", idx, r.Title))
		sb.WriteString(fmt.Sprintf("       %s\n", r.URL))
		if r.Snippet != "" {
			snippet := r.Snippet
			if len(snippet) > 200 {
				snippet = snippet[:197] + "..."
			}
			sb.WriteString(fmt.Sprintf("       %s\n", snippet))
		}
		sb.WriteString("\n")

		links = append(links, browser.Link{
			Index: idx,
			Text:  r.Title,
			URL:   r.URL,
		})
	}

	sb.WriteString(fmt.Sprintf("  %d results | Use 'f <number>' to follow a link\n", len(results)))

	return sb.String(), links
}
