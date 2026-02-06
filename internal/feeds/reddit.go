package feeds

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/vidyasagar/tsurf/internal/browser"
)

const (
	redditTimeout = 10 * time.Second
)

// Reddit URL patterns.
var (
	// Matches reddit.com/r/subreddit/comments/id/... (post detail page)
	redditPostRe = regexp.MustCompile(`(?i)^https?://(?:www\.|old\.|new\.)?reddit\.com/r/(\w+)/comments/(\w+)`)
	// Matches reddit.com/r/subreddit (subreddit listing)
	redditSubRe = regexp.MustCompile(`(?i)^https?://(?:www\.|old\.|new\.)?reddit\.com/r/(\w+)/?(?:\?.*)?$`)
	// Matches reddit.com root (frontpage)
	redditRootRe = regexp.MustCompile(`(?i)^https?://(?:www\.|old\.|new\.)?reddit\.com/?(?:\?.*)?$`)
)

// RedditURLType indicates what kind of Reddit URL was detected.
type RedditURLType int

const (
	RedditURLNone      RedditURLType = iota
	RedditURLFrontpage               // reddit.com
	RedditURLSubreddit               // reddit.com/r/golang
	RedditURLPost                    // reddit.com/r/golang/comments/abc123/...
)

// RedditURLInfo holds parsed info from a Reddit URL.
type RedditURLInfo struct {
	Type      RedditURLType
	Subreddit string // e.g. "golang"
	PostID    string // e.g. "abc123"
	OrigURL   string // original URL
}

// ParseRedditURL checks if a URL is a Reddit URL and extracts info.
func ParseRedditURL(rawURL string) *RedditURLInfo {
	// Normalize the URL first.
	u := strings.TrimSpace(rawURL)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return nil
	}

	host := strings.ToLower(parsed.Hostname())
	if !strings.Contains(host, "reddit.com") {
		return nil
	}

	// Check post URL first (more specific).
	if m := redditPostRe.FindStringSubmatch(u); m != nil {
		return &RedditURLInfo{
			Type:      RedditURLPost,
			Subreddit: m[1],
			PostID:    m[2],
			OrigURL:   u,
		}
	}

	// Check subreddit URL.
	if m := redditSubRe.FindStringSubmatch(u); m != nil {
		return &RedditURLInfo{
			Type:      RedditURLSubreddit,
			Subreddit: m[1],
			OrigURL:   u,
		}
	}

	// Check frontpage.
	if redditRootRe.MatchString(u) {
		return &RedditURLInfo{
			Type:    RedditURLFrontpage,
			OrigURL: u,
		}
	}

	// Fallback: any other reddit.com path, treat as subreddit listing or unknown.
	return &RedditURLInfo{
		Type:    RedditURLNone,
		OrigURL: u,
	}
}

// RedditListing is the top-level Reddit JSON response.
type RedditListing struct {
	Data struct {
		Children []struct {
			Data RedditPost `json:"data"`
		} `json:"children"`
		After string `json:"after"`
	} `json:"data"`
}

// RedditComment represents a Reddit comment.
type RedditComment struct {
	Author     string                `json:"author"`
	Body       string                `json:"body"`
	Score      int                   `json:"score"`
	CreatedUTC float64               `json:"created_utc"`
	Depth      int                   `json:"depth"`
	Replies    *RedditCommentListing `json:"-"`
	RepliesRaw json.RawMessage       `json:"replies"`
}

// RedditCommentListing wraps the comment listing structure.
type RedditCommentListing struct {
	Data struct {
		Children []struct {
			Kind string          `json:"kind"`
			Data json.RawMessage `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// RedditPostDetail holds a post with its comments.
type RedditPostDetail struct {
	Post     RedditPost
	Comments []RedditComment
}

// RedditPost represents a Reddit post.
type RedditPost struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	Selftext    string  `json:"selftext"`
	Author      string  `json:"author"`
	Subreddit   string  `json:"subreddit"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
	IsSelf      bool    `json:"is_self"`
	Domain      string  `json:"domain"`
	Thumbnail   string  `json:"thumbnail"`
}

// RedditClient fetches data from Reddit's JSON API.
type RedditClient struct {
	client *http.Client
}

// NewRedditClient creates a new Reddit API client.
func NewRedditClient() *RedditClient {
	return &RedditClient{
		client: &http.Client{Timeout: redditTimeout},
	}
}

// FetchSubreddit fetches posts from a subreddit.
// sort can be "hot", "new", "top", "rising".
func (r *RedditClient) FetchSubreddit(subreddit string, sort string, limit int) ([]RedditPost, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}
	if sort == "" {
		sort = "hot"
	}

	url := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d&raw_json=1", subreddit, sort, limit)
	return r.fetchPosts(url)
}

// FetchFrontpage fetches Reddit frontpage.
func (r *RedditClient) FetchFrontpage(limit int) ([]RedditPost, error) {
	if limit <= 0 || limit > 50 {
		limit = 25
	}

	url := fmt.Sprintf("https://www.reddit.com/.json?limit=%d&raw_json=1", limit)
	return r.fetchPosts(url)
}

func (r *RedditClient) fetchPosts(url string) ([]RedditPost, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "tsurf/0.1 (terminal browser)")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching reddit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("reddit returned %d: %s", resp.StatusCode, string(body[:200]))
	}

	var listing RedditListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, fmt.Errorf("parsing reddit response: %w", err)
	}

	posts := make([]RedditPost, 0, len(listing.Data.Children))
	for _, child := range listing.Data.Children {
		posts = append(posts, child.Data)
	}

	return posts, nil
}

// RenderRedditPosts formats Reddit posts for the viewport.
func RenderRedditPosts(posts []RedditPost, title string) (string, []browser.Link) {
	var result string
	var links []browser.Link

	result += fmt.Sprintf("  🤖 %s\n", title)
	result += fmt.Sprintf("  %s\n\n", "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for i, post := range posts {
		ago := timeAgo(time.Unix(int64(post.CreatedUTC), 0))

		link := post.URL
		if post.IsSelf {
			link = "https://www.reddit.com" + post.Permalink
		}

		idx := i + 1
		result += fmt.Sprintf("  [%d] %s\n", idx, post.Title)
		result += fmt.Sprintf("       r/%s | %d pts | %s | %d comments\n", post.Subreddit, post.Score, ago, post.NumComments)
		result += fmt.Sprintf("       %s\n\n", link)

		links = append(links, browser.Link{
			Index: idx,
			Text:  post.Title,
			URL:   link,
		})
	}

	return result, links
}

// FetchPostDetail fetches a Reddit post with comments using the .json API.
func (r *RedditClient) FetchPostDetail(subreddit, postID string) (*RedditPostDetail, error) {
	jsonURL := fmt.Sprintf("https://www.reddit.com/r/%s/comments/%s.json?raw_json=1&limit=100", subreddit, postID)

	req, err := http.NewRequest(http.MethodGet, jsonURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "tsurf/0.1 (terminal browser)")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching reddit post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("reddit returned %d: %s", resp.StatusCode, string(body[:min(200, len(body))]))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Reddit returns an array of 2 listings: [post_listing, comments_listing]
	var listings []json.RawMessage
	if err := json.Unmarshal(body, &listings); err != nil {
		return nil, fmt.Errorf("parsing reddit JSON array: %w", err)
	}

	if len(listings) < 2 {
		return nil, fmt.Errorf("unexpected reddit response format (got %d listings)", len(listings))
	}

	// Parse the post from the first listing.
	var postListing RedditListing
	if err := json.Unmarshal(listings[0], &postListing); err != nil {
		return nil, fmt.Errorf("parsing post listing: %w", err)
	}

	if len(postListing.Data.Children) == 0 {
		return nil, fmt.Errorf("no post found")
	}

	detail := &RedditPostDetail{
		Post: postListing.Data.Children[0].Data,
	}

	// Parse comments from the second listing.
	var commentListing RedditCommentListing
	if err := json.Unmarshal(listings[1], &commentListing); err != nil {
		return nil, fmt.Errorf("parsing comment listing: %w", err)
	}

	detail.Comments = parseComments(commentListing, 0)

	return detail, nil
}

// parseComments recursively parses Reddit comments from a listing.
func parseComments(listing RedditCommentListing, depth int) []RedditComment {
	var comments []RedditComment

	for _, child := range listing.Data.Children {
		if child.Kind != "t1" {
			continue // skip non-comment entries (e.g. "more" objects)
		}

		var rawComment struct {
			Author     string          `json:"author"`
			Body       string          `json:"body"`
			Score      int             `json:"score"`
			CreatedUTC float64         `json:"created_utc"`
			Depth      int             `json:"depth"`
			Replies    json.RawMessage `json:"replies"`
		}

		if err := json.Unmarshal(child.Data, &rawComment); err != nil {
			continue
		}

		comment := RedditComment{
			Author:     rawComment.Author,
			Body:       rawComment.Body,
			Score:      rawComment.Score,
			CreatedUTC: rawComment.CreatedUTC,
			Depth:      rawComment.Depth,
		}

		comments = append(comments, comment)

		// Parse nested replies if they exist (replies can be "" or a listing object).
		if len(rawComment.Replies) > 0 && string(rawComment.Replies) != `""` && string(rawComment.Replies) != "null" {
			var replyListing RedditCommentListing
			if err := json.Unmarshal(rawComment.Replies, &replyListing); err == nil {
				nested := parseComments(replyListing, depth+1)
				comments = append(comments, nested...)
			}
		}
	}

	return comments
}

// RenderPostDetail formats a Reddit post with comments for the viewport.
func RenderPostDetail(detail *RedditPostDetail) (string, []browser.Link) {
	var result string
	var links []browser.Link

	post := detail.Post
	ago := timeAgo(time.Unix(int64(post.CreatedUTC), 0))

	result += fmt.Sprintf("  🤖 r/%s\n", post.Subreddit)
	result += "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	// Post title and metadata.
	result += fmt.Sprintf("  %s\n", post.Title)
	result += fmt.Sprintf("  👤 u/%s | %d pts | %s | 💬 %d comments\n", post.Author, post.Score, ago, post.NumComments)

	// External link if not a self post.
	linkIdx := 1
	if !post.IsSelf && post.URL != "" {
		result += fmt.Sprintf("  [%d] 🔗 %s\n", linkIdx, post.URL)
		links = append(links, browser.Link{
			Index: linkIdx,
			Text:  post.Title,
			URL:   post.URL,
		})
		linkIdx++
	}
	result += "\n"

	// Self text.
	if post.Selftext != "" {
		// Word wrap the self text.
		wrapped := wordWrap(post.Selftext, 76)
		for _, line := range strings.Split(wrapped, "\n") {
			result += fmt.Sprintf("  %s\n", line)
		}
		result += "\n"
	}

	// Comments section.
	result += "  ── Comments ────────────────────────────\n\n"

	if len(detail.Comments) == 0 {
		result += "  No comments yet.\n"
	}

	for _, comment := range detail.Comments {
		if comment.Author == "" || comment.Body == "" {
			continue
		}

		indent := strings.Repeat("  ", comment.Depth)
		cAgo := timeAgo(time.Unix(int64(comment.CreatedUTC), 0))

		// Comment header.
		result += fmt.Sprintf("  %s👤 u/%s | %d pts | %s\n", indent, comment.Author, comment.Score, cAgo)

		// Comment body with word wrapping.
		maxWidth := 76 - (comment.Depth * 2)
		if maxWidth < 30 {
			maxWidth = 30
		}
		wrapped := wordWrap(comment.Body, maxWidth)
		for _, line := range strings.Split(wrapped, "\n") {
			result += fmt.Sprintf("  %s%s\n", indent, line)
		}
		result += "\n"
	}

	return result, links
}

// FetchURL auto-detects a Reddit URL type and fetches/renders it.
// Returns content, title, links, and any error.
func (r *RedditClient) FetchURL(info *RedditURLInfo) (string, string, []browser.Link, error) {
	switch info.Type {
	case RedditURLPost:
		detail, err := r.FetchPostDetail(info.Subreddit, info.PostID)
		if err != nil {
			return "", "", nil, err
		}
		content, links := RenderPostDetail(detail)
		title := fmt.Sprintf("r/%s - %s", detail.Post.Subreddit, truncate(detail.Post.Title, 40))
		return content, title, links, nil

	case RedditURLSubreddit:
		posts, err := r.FetchSubreddit(info.Subreddit, "hot", 25)
		if err != nil {
			return "", "", nil, err
		}
		title := fmt.Sprintf("r/%s - Hot", info.Subreddit)
		content, links := RenderRedditPosts(posts, title)
		return content, title, links, nil

	case RedditURLFrontpage:
		posts, err := r.FetchFrontpage(25)
		if err != nil {
			return "", "", nil, err
		}
		title := "Reddit - Front Page"
		content, links := RenderRedditPosts(posts, title)
		return content, title, links, nil

	default:
		return "", "", nil, fmt.Errorf("unsupported Reddit URL type")
	}
}

// wordWrap wraps text at the given width.
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			result.WriteString("\n")
			continue
		}

		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result.WriteString("\n")
			continue
		}

		lineLen := 0
		for i, word := range words {
			wLen := len(word)
			if i > 0 && lineLen+1+wLen > width {
				result.WriteString("\n")
				lineLen = 0
			} else if i > 0 {
				result.WriteString(" ")
				lineLen++
			}
			result.WriteString(word)
			lineLen += wLen
		}
		result.WriteString("\n")
	}

	return strings.TrimRight(result.String(), "\n")
}

// truncate shortens a string to max length with "..." suffix.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
