package feeds

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

const (
	rssTimeout  = 10 * time.Second
	maxRSSBytes = 2 * 1024 * 1024 // 2MB limit for RSS feeds
)

// Feed represents a parsed RSS/Atom feed.
type Feed struct {
	Title       string
	Description string
	Link        string
	Items       []FeedItem
}

// FeedItem represents a single item from a feed.
type FeedItem struct {
	Title       string
	Link        string
	Description string
	Published   time.Time
	Author      string
	GUID        string
}

// RSSClient fetches and parses RSS/Atom feeds.
type RSSClient struct {
	client *http.Client
}

// NewRSSClient creates a new RSS feed client.
func NewRSSClient() *RSSClient {
	return &RSSClient{
		client: &http.Client{
			Timeout:   rssTimeout,
			Transport: browser.SharedTransport,
		},
	}
}

// Fetch retrieves and parses an RSS or Atom feed.
func (r *RSSClient) Fetch(url string) (*Feed, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "tsurf/0.1 (terminal browser)")
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxRSSBytes))
	if err != nil {
		return nil, fmt.Errorf("reading feed body: %w", err)
	}

	// Try RSS first, then Atom.
	feed, err := parseRSS(body)
	if err != nil {
		feed, err = parseAtom(body)
		if err != nil {
			return nil, fmt.Errorf("could not parse feed as RSS or Atom")
		}
	}

	return feed, nil
}

// RSS 2.0 types
type rssRoot struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Author      string `xml:"author"`
	Creator     string `xml:"creator"` // dc:creator
	GUID        string `xml:"guid"`
}

func parseRSS(data []byte) (*Feed, error) {
	var root rssRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	if root.Channel.Title == "" && len(root.Channel.Items) == 0 {
		return nil, fmt.Errorf("empty RSS feed")
	}

	feed := &Feed{
		Title:       root.Channel.Title,
		Description: root.Channel.Description,
		Link:        root.Channel.Link,
	}

	for _, item := range root.Channel.Items {
		author := item.Author
		if author == "" {
			author = item.Creator
		}

		fi := FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: stripHTML(item.Description),
			Author:      author,
			GUID:        item.GUID,
		}

		if item.PubDate != "" {
			if t, err := parseTime(item.PubDate); err == nil {
				fi.Published = t
			}
		}

		feed.Items = append(feed.Items, fi)
	}

	return feed, nil
}

// Atom types
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   string      `xml:"title"`
	Link    []atomLink  `xml:"link"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	Title     string     `xml:"title"`
	Link      []atomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Content   string     `xml:"content"`
	Published string     `xml:"published"`
	Updated   string     `xml:"updated"`
	Author    struct {
		Name string `xml:"name"`
	} `xml:"author"`
	ID string `xml:"id"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func parseAtom(data []byte) (*Feed, error) {
	var af atomFeed
	if err := xml.Unmarshal(data, &af); err != nil {
		return nil, err
	}

	if af.Title == "" && len(af.Entries) == 0 {
		return nil, fmt.Errorf("empty Atom feed")
	}

	feed := &Feed{
		Title: af.Title,
	}

	for _, link := range af.Link {
		if link.Rel == "" || link.Rel == "alternate" {
			feed.Link = link.Href
			break
		}
	}

	for _, entry := range af.Entries {
		link := ""
		for _, l := range entry.Link {
			if l.Rel == "" || l.Rel == "alternate" {
				link = l.Href
				break
			}
		}

		desc := entry.Summary
		if desc == "" {
			desc = entry.Content
		}

		fi := FeedItem{
			Title:       entry.Title,
			Link:        link,
			Description: stripHTML(desc),
			Author:      entry.Author.Name,
			GUID:        entry.ID,
		}

		dateStr := entry.Published
		if dateStr == "" {
			dateStr = entry.Updated
		}
		if dateStr != "" {
			if t, err := parseTime(dateStr); err == nil {
				fi.Published = t
			}
		}

		feed.Items = append(feed.Items, fi)
	}

	return feed, nil
}

// RenderFeed formats a feed for the viewport.
func RenderFeed(feed *Feed) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link

	sb.WriteString(fmt.Sprintf("  📡 %s\n", feed.Title))
	if feed.Description != "" {
		sb.WriteString(fmt.Sprintf("  %s\n", feed.Description))
	}
	sb.WriteString(fmt.Sprintf("  %s\n\n", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))

	for i, item := range feed.Items {
		idx := i + 1
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", idx, item.Title))
		if item.Author != "" {
			sb.WriteString(fmt.Sprintf("       by %s", item.Author))
		}
		if !item.Published.IsZero() {
			sb.WriteString(fmt.Sprintf(" | %s", timeAgo(item.Published)))
		}
		if item.Author != "" || !item.Published.IsZero() {
			sb.WriteString("\n")
		}
		if item.Link != "" {
			sb.WriteString(fmt.Sprintf("       %s\n", item.Link))
			links = append(links, browser.Link{
				Index: idx,
				Text:  item.Title,
				URL:   item.Link,
			})
		}
		if item.Description != "" {
			desc := item.Description
			if len(desc) > 200 {
				desc = desc[:197] + "..."
			}
			sb.WriteString(fmt.Sprintf("       %s\n", desc))
		}
		sb.WriteString("\n")
	}

	return sb.String(), links
}

// stripHTML does a basic removal of HTML tags.
func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

// parseTime tries multiple date formats.
func parseTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"Mon, 02 Jan 2006 15:04:05 GMT",
		"Mon, 02 Jan 2006 15:04:05 +0000",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	s = strings.TrimSpace(s)
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse time: %s", s)
}
