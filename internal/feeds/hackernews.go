package feeds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

const (
	hnBaseURL     = "https://hacker-news.firebaseio.com/v0"
	hnMaxItems    = 30
	hnTimeout     = 10 * time.Second
	hnConcurrency = 10          // parallel fetches
	hnMaxBodySize = 1024 * 1024 // 1MB limit per API response
)

// HNStory represents a Hacker News story.
type HNStory struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Text        string `json:"text"` // for Ask HN etc.
	By          string `json:"by"`
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"` // comment count
	Time        int64  `json:"time"`
	Type        string `json:"type"`
	Kids        []int  `json:"kids"`
}

// HNComment represents a HN comment.
type HNComment struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	By      string `json:"by"`
	Time    int64  `json:"time"`
	Kids    []int  `json:"kids"`
	Parent  int    `json:"parent"`
	Dead    bool   `json:"dead"`
	Deleted bool   `json:"deleted"`
}

// HNClient fetches data from the Hacker News API.
type HNClient struct {
	client *http.Client
}

// NewHNClient creates a new HN API client using the shared transport.
func NewHNClient() *HNClient {
	return &HNClient{
		client: &http.Client{
			Transport: browser.SharedTransport,
			Timeout:   hnTimeout,
		},
	}
}

// TopStories fetches the top stories.
func (h *HNClient) TopStories(limit int) ([]HNStory, error) {
	return h.fetchStories("topstories", limit)
}

// NewStories fetches the newest stories.
func (h *HNClient) NewStories(limit int) ([]HNStory, error) {
	return h.fetchStories("newstories", limit)
}

// BestStories fetches the best stories.
func (h *HNClient) BestStories(limit int) ([]HNStory, error) {
	return h.fetchStories("beststories", limit)
}

// AskStories fetches Ask HN stories.
func (h *HNClient) AskStories(limit int) ([]HNStory, error) {
	return h.fetchStories("askstories", limit)
}

// ShowStories fetches Show HN stories.
func (h *HNClient) ShowStories(limit int) ([]HNStory, error) {
	return h.fetchStories("showstories", limit)
}

// FetchComments fetches comments for a story (top-level only) in parallel.
func (h *HNClient) FetchComments(story *HNStory, limit int) ([]HNComment, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	kids := story.Kids
	if len(kids) > limit {
		kids = kids[:limit]
	}

	// Fetch comments in parallel
	type result struct {
		idx     int
		comment HNComment
		ok      bool
	}

	results := make(chan result, len(kids))
	sem := make(chan struct{}, hnConcurrency)

	var wg sync.WaitGroup
	for i, id := range kids {
		wg.Add(1)
		go func(idx, commentID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var comment HNComment
			if err := h.fetchItem(commentID, &comment); err != nil {
				results <- result{idx: idx, ok: false}
				return
			}
			if comment.Deleted || comment.Dead {
				results <- result{idx: idx, ok: false}
				return
			}
			results <- result{idx: idx, comment: comment, ok: true}
		}(i, id)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and sort by original order
	comments := make([]HNComment, 0, len(kids))
	collected := make(map[int]HNComment)
	for r := range results {
		if r.ok {
			collected[r.idx] = r.comment
		}
	}
	for i := 0; i < len(kids); i++ {
		if c, ok := collected[i]; ok {
			comments = append(comments, c)
		}
	}

	return comments, nil
}

func (h *HNClient) fetchStories(endpoint string, limit int) ([]HNStory, error) {
	if limit <= 0 || limit > hnMaxItems {
		limit = hnMaxItems
	}

	url := fmt.Sprintf("%s/%s.json", hnBaseURL, endpoint)
	resp, err := h.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, hnMaxBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("parsing IDs: %w", err)
	}

	if len(ids) > limit {
		ids = ids[:limit]
	}

	// Fetch stories in parallel with bounded concurrency
	type storyResult struct {
		idx   int
		story HNStory
		ok    bool
	}

	results := make(chan storyResult, len(ids))
	sem := make(chan struct{}, hnConcurrency)

	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func(idx, storyID int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var story HNStory
			if err := h.fetchItem(storyID, &story); err != nil {
				results <- storyResult{idx: idx, ok: false}
				return
			}
			results <- storyResult{idx: idx, story: story, ok: true}
		}(i, id)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results preserving order
	collected := make(map[int]HNStory)
	for r := range results {
		if r.ok {
			collected[r.idx] = r.story
		}
	}

	stories := make([]HNStory, 0, len(ids))
	for i := 0; i < len(ids); i++ {
		if s, ok := collected[i]; ok {
			stories = append(stories, s)
		}
	}

	return stories, nil
}

func (h *HNClient) fetchItem(id int, v interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/item/%d.json", hnBaseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(io.LimitReader(resp.Body, hnMaxBodySize)).Decode(v)
}

// RenderHNStories formats HN stories as readable content for the viewport.
func RenderHNStories(stories []HNStory, title string) (string, []browser.Link) {
	r := &storyRenderer{stories: stories, title: title}
	return r.render()
}

type storyRenderer struct {
	stories []HNStory
	title   string
}

func (r *storyRenderer) render() (string, []browser.Link) {
	var sb strings.Builder
	links := make([]browser.Link, 0, len(r.stories))

	sb.WriteString(fmt.Sprintf("  🔥 %s\n", r.title))
	sb.WriteString("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Sort by score descending.
	sorted := make([]HNStory, len(r.stories))
	copy(sorted, r.stories)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Score > sorted[j].Score
	})

	for i, story := range sorted {
		ago := timeAgo(time.Unix(story.Time, 0))
		url := story.URL
		if url == "" {
			url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)
		}

		idx := i + 1
		sb.WriteString(fmt.Sprintf("  [%d] %s\n", idx, story.Title))
		sb.WriteString(fmt.Sprintf("       %d points | %s | %d comments\n", story.Score, ago, story.Descendants))
		sb.WriteString(fmt.Sprintf("       %s\n\n", url))

		links = append(links, browser.Link{
			Index: idx,
			Text:  story.Title,
			URL:   url,
		})
	}

	return sb.String(), links
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
