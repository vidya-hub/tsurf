package feeds

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

const (
	hnBaseURL  = "https://hacker-news.firebaseio.com/v0"
	hnMaxItems = 30
	hnTimeout  = 10 * time.Second
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

// NewHNClient creates a new HN API client.
func NewHNClient() *HNClient {
	return &HNClient{
		client: &http.Client{Timeout: hnTimeout},
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

// FetchComments fetches comments for a story (top-level only).
func (h *HNClient) FetchComments(story *HNStory, limit int) ([]HNComment, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	kids := story.Kids
	if len(kids) > limit {
		kids = kids[:limit]
	}

	comments := make([]HNComment, 0, len(kids))
	for _, id := range kids {
		var comment HNComment
		err := h.fetchItem(id, &comment)
		if err != nil {
			continue
		}
		if !comment.Deleted && !comment.Dead {
			comments = append(comments, comment)
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

	body, err := io.ReadAll(resp.Body)
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

	stories := make([]HNStory, 0, len(ids))
	for _, id := range ids {
		var story HNStory
		if err := h.fetchItem(id, &story); err != nil {
			continue
		}
		stories = append(stories, story)
	}

	return stories, nil
}

func (h *HNClient) fetchItem(id int, v interface{}) error {
	url := fmt.Sprintf("%s/item/%d.json", hnBaseURL, id)
	resp, err := h.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(v)
}

// RenderStories formats HN stories as readable content for the viewport.
func RenderHNStories(stories []HNStory, title string) (string, []browser.Link) {
	r := &storyRenderer{stories: stories, title: title}
	return r.render()
}

type storyRenderer struct {
	stories []HNStory
	title   string
}

func (r *storyRenderer) render() (string, []browser.Link) {
	var result string
	var links []browser.Link

	result += fmt.Sprintf("  🔥 %s\n", r.title)
	result += fmt.Sprintf("  %s\n\n", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

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
		result += fmt.Sprintf("  [%d] %s\n", idx, story.Title)
		result += fmt.Sprintf("       %d points | %s | %d comments\n", story.Score, ago, story.Descendants)
		result += fmt.Sprintf("       %s\n\n", url)

		links = append(links, browser.Link{
			Index: idx,
			Text:  story.Title,
			URL:   url,
		})
	}

	return result, links
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
