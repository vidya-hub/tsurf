package feeds

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/browser"
)

const (
	githubTimeout  = 15 * time.Second
	maxGitHubBytes = 2 * 1024 * 1024 // 2MB limit for GitHub responses
)

// GitHub URL patterns.
var (
	// Matches github.com/owner/repo/issues/123
	githubIssueRe = regexp.MustCompile(`(?i)^https?://(?:www\.)?github\.com/([^/]+)/([^/]+)/issues/(\d+)`)
	// Matches github.com/owner/repo/pull/456
	githubPRRe = regexp.MustCompile(`(?i)^https?://(?:www\.)?github\.com/([^/]+)/([^/]+)/pull/(\d+)`)
	// Matches github.com/owner/repo (but not special paths like /issues, /pulls, /settings, etc.)
	githubRepoRe = regexp.MustCompile(`(?i)^https?://(?:www\.)?github\.com/([^/]+)/([^/]+)/?(?:\?.*)?$`)
	// Matches gist.github.com/user/id
	githubGistRe = regexp.MustCompile(`(?i)^https?://gist\.github\.com/([^/]+)/([a-f0-9]+)`)
	// Matches github.com/username (single path segment, not a reserved word)
	githubUserRe = regexp.MustCompile(`(?i)^https?://(?:www\.)?github\.com/([^/]+)/?(?:\?.*)?$`)

	// Reserved GitHub paths that are not usernames
	githubReservedPaths = map[string]bool{
		"about": true, "pricing": true, "enterprise": true, "team": true,
		"explore": true, "topics": true, "trending": true, "collections": true,
		"events": true, "sponsors": true, "login": true, "join": true,
		"settings": true, "notifications": true, "new": true, "organizations": true,
		"marketplace": true, "features": true, "security": true, "codespaces": true,
		"copilot": true, "actions": true, "packages": true, "pulls": true,
		"issues": true, "discussions": true, "search": true, "stars": true,
	}
)

// GitHubURLType indicates what kind of GitHub URL was detected.
type GitHubURLType int

const (
	GitHubURLNone  GitHubURLType = iota
	GitHubURLRepo                // github.com/owner/repo
	GitHubURLIssue               // github.com/owner/repo/issues/123
	GitHubURLPR                  // github.com/owner/repo/pull/456
	GitHubURLGist                // gist.github.com/user/id
	GitHubURLUser                // github.com/username
)

// GitHubURLInfo holds parsed info from a GitHub URL.
type GitHubURLInfo struct {
	Type    GitHubURLType
	Owner   string // repo owner or gist owner
	Repo    string // repo name
	Number  int    // issue or PR number
	GistID  string // gist ID
	User    string // username for profile pages
	OrigURL string
}

// ParseGitHubURL checks if a URL is a GitHub URL and extracts info.
func ParseGitHubURL(rawURL string) *GitHubURLInfo {
	u := strings.TrimSpace(rawURL)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "https://" + u
	}

	parsed, err := url.Parse(u)
	if err != nil {
		return nil
	}

	host := strings.ToLower(parsed.Hostname())

	// Check for gist.github.com first
	if host == "gist.github.com" {
		if m := githubGistRe.FindStringSubmatch(u); m != nil {
			return &GitHubURLInfo{
				Type:    GitHubURLGist,
				Owner:   m[1],
				GistID:  m[2],
				OrigURL: u,
			}
		}
		return nil
	}

	// Must be github.com
	if host != "github.com" && host != "www.github.com" {
		return nil
	}

	// Check issue URL first (more specific)
	if m := githubIssueRe.FindStringSubmatch(u); m != nil {
		num := 0
		fmt.Sscanf(m[3], "%d", &num)
		return &GitHubURLInfo{
			Type:    GitHubURLIssue,
			Owner:   m[1],
			Repo:    m[2],
			Number:  num,
			OrigURL: u,
		}
	}

	// Check PR URL
	if m := githubPRRe.FindStringSubmatch(u); m != nil {
		num := 0
		fmt.Sscanf(m[3], "%d", &num)
		return &GitHubURLInfo{
			Type:    GitHubURLPR,
			Owner:   m[1],
			Repo:    m[2],
			Number:  num,
			OrigURL: u,
		}
	}

	// Check repo URL (owner/repo with no additional path)
	if m := githubRepoRe.FindStringSubmatch(u); m != nil {
		// Ensure it's not a reserved path
		if !githubReservedPaths[strings.ToLower(m[1])] {
			return &GitHubURLInfo{
				Type:    GitHubURLRepo,
				Owner:   m[1],
				Repo:    m[2],
				OrigURL: u,
			}
		}
	}

	// Check user profile URL (just /username)
	if m := githubUserRe.FindStringSubmatch(u); m != nil {
		username := m[1]
		if !githubReservedPaths[strings.ToLower(username)] {
			return &GitHubURLInfo{
				Type:    GitHubURLUser,
				User:    username,
				OrigURL: u,
			}
		}
	}

	return nil
}

// --- API Response Structs ---

// GitHubRepo represents a GitHub repository.
type GitHubRepo struct {
	Name            string         `json:"name"`
	FullName        string         `json:"full_name"`
	Description     string         `json:"description"`
	HTMLURL         string         `json:"html_url"`
	StargazersCount int            `json:"stargazers_count"`
	ForksCount      int            `json:"forks_count"`
	OpenIssuesCount int            `json:"open_issues_count"`
	WatchersCount   int            `json:"watchers_count"`
	Language        string         `json:"language"`
	License         *GitHubLicense `json:"license"`
	Topics          []string       `json:"topics"`
	DefaultBranch   string         `json:"default_branch"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	PushedAt        time.Time      `json:"pushed_at"`
	Archived        bool           `json:"archived"`
	Fork            bool           `json:"fork"`
	Private         bool           `json:"private"`
	Owner           *GitHubUser    `json:"owner"`
}

// GitHubLicense represents a license.
type GitHubLicense struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	SPDXID string `json:"spdx_id"`
}

// GitHubUser represents a GitHub user.
type GitHubUser struct {
	Login       string    `json:"login"`
	Name        string    `json:"name"`
	Bio         string    `json:"bio"`
	HTMLURL     string    `json:"html_url"`
	AvatarURL   string    `json:"avatar_url"`
	Followers   int       `json:"followers"`
	Following   int       `json:"following"`
	PublicRepos int       `json:"public_repos"`
	PublicGists int       `json:"public_gists"`
	Company     string    `json:"company"`
	Location    string    `json:"location"`
	Blog        string    `json:"blog"`
	CreatedAt   time.Time `json:"created_at"`
	Type        string    `json:"type"` // "User" or "Organization"
}

// GitHubLabel represents an issue/PR label.
type GitHubLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// GitHubIssue represents a GitHub issue.
type GitHubIssue struct {
	Number    int           `json:"number"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	State     string        `json:"state"` // "open" or "closed"
	HTMLURL   string        `json:"html_url"`
	User      *GitHubUser   `json:"user"`
	Labels    []GitHubLabel `json:"labels"`
	Comments  int           `json:"comments"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	ClosedAt  *time.Time    `json:"closed_at"`
}

// GitHubPR represents a GitHub pull request.
type GitHubPR struct {
	Number    int           `json:"number"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	State     string        `json:"state"` // "open" or "closed"
	HTMLURL   string        `json:"html_url"`
	User      *GitHubUser   `json:"user"`
	Labels    []GitHubLabel `json:"labels"`
	Comments  int           `json:"comments"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	ClosedAt  *time.Time    `json:"closed_at"`
	MergedAt  *time.Time    `json:"merged_at"`
	Merged    bool          `json:"merged"`
	Draft     bool          `json:"draft"`
	Head      *GitHubBranch `json:"head"`
	Base      *GitHubBranch `json:"base"`
	Additions int           `json:"additions"`
	Deletions int           `json:"deletions"`
	Commits   int           `json:"commits"`
}

// GitHubBranch represents a branch reference in a PR.
type GitHubBranch struct {
	Ref  string      `json:"ref"`
	SHA  string      `json:"sha"`
	Repo *GitHubRepo `json:"repo"`
}

// GitHubGist represents a GitHub gist.
type GitHubGist struct {
	ID          string                    `json:"id"`
	Description string                    `json:"description"`
	HTMLURL     string                    `json:"html_url"`
	Public      bool                      `json:"public"`
	Owner       *GitHubUser               `json:"owner"`
	Files       map[string]GitHubGistFile `json:"files"`
	CreatedAt   time.Time                 `json:"created_at"`
	UpdatedAt   time.Time                 `json:"updated_at"`
	Comments    int                       `json:"comments"`
}

// GitHubGistFile represents a file in a gist.
type GitHubGistFile struct {
	Filename string `json:"filename"`
	Type     string `json:"type"`
	Language string `json:"language"`
	RawURL   string `json:"raw_url"`
	Size     int    `json:"size"`
	Content  string `json:"content"`
}

// GitHubReadme represents the README API response.
type GitHubReadme struct {
	Name     string `json:"name"`
	Content  string `json:"content"` // base64 encoded
	Encoding string `json:"encoding"`
}

// --- GitHub Client ---

// GitHubClient fetches data from GitHub's API.
type GitHubClient struct {
	client *http.Client
}

// NewGitHubClient creates a new GitHub API client.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		client: &http.Client{
			Timeout:   githubTimeout,
			Transport: browser.SharedTransport,
		},
	}
}

// doRequest performs an authenticated GitHub API request.
func (g *GitHubClient) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "tsurf/0.1 (terminal browser)")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found (404)")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("rate limited or forbidden (403)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub returned %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxGitHubBytes))
}

// FetchRepo fetches repository information.
func (g *GitHubClient) FetchRepo(owner, repo string) (*GitHubRepo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	body, err := g.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result GitHubRepo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing repo response: %w", err)
	}
	return &result, nil
}

// FetchReadme fetches and decodes the repository README.
func (g *GitHubClient) FetchReadme(owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repo)
	body, err := g.doRequest(url)
	if err != nil {
		// README not found is not an error, just return empty
		return "", nil
	}

	var readme GitHubReadme
	if err := json.Unmarshal(body, &readme); err != nil {
		return "", nil
	}

	if readme.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(readme.Content)
		if err != nil {
			return "", nil
		}
		return string(decoded), nil
	}

	return readme.Content, nil
}

// FetchIssue fetches an issue.
func (g *GitHubClient) FetchIssue(owner, repo string, number int) (*GitHubIssue, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", owner, repo, number)
	body, err := g.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result GitHubIssue
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing issue response: %w", err)
	}
	return &result, nil
}

// FetchPR fetches a pull request.
func (g *GitHubClient) FetchPR(owner, repo string, number int) (*GitHubPR, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, repo, number)
	body, err := g.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result GitHubPR
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing PR response: %w", err)
	}
	return &result, nil
}

// FetchGist fetches a gist.
func (g *GitHubClient) FetchGist(id string) (*GitHubGist, error) {
	url := fmt.Sprintf("https://api.github.com/gists/%s", id)
	body, err := g.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result GitHubGist
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing gist response: %w", err)
	}
	return &result, nil
}

// FetchUser fetches a user profile.
func (g *GitHubClient) FetchUser(username string) (*GitHubUser, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	body, err := g.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result GitHubUser
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing user response: %w", err)
	}
	return &result, nil
}

// FetchUserRepos fetches a user's public repositories.
func (g *GitHubClient) FetchUserRepos(username string, limit int) ([]GitHubRepo, error) {
	if limit <= 0 || limit > 30 {
		limit = 10
	}
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated&per_page=%d", username, limit)
	body, err := g.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result []GitHubRepo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing repos response: %w", err)
	}
	return result, nil
}

// --- Rendering Functions ---

// RenderRepo renders a repository with its README.
func RenderRepo(repo *GitHubRepo, readme string, width int) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link
	linkIdx := 1

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58a6ff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
	statStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))
	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a371f7"))

	// Header
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %s %s/%s", repoIcon(repo), repo.Owner.Login, repo.Name)))
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  " + strings.Repeat("─", min(width-4, 60))))
	sb.WriteString("\n\n")

	// Description
	if repo.Description != "" {
		wrapped := wordWrap(repo.Description, min(width-4, 76))
		for _, line := range strings.Split(wrapped, "\n") {
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n")
	}

	// Stats line
	stats := fmt.Sprintf("  %s %s  %s %s  %s %s",
		statStyle.Render("★"), formatNumber(repo.StargazersCount),
		statStyle.Render("⑂"), formatNumber(repo.ForksCount),
		statStyle.Render("◉"), formatNumber(repo.OpenIssuesCount))
	sb.WriteString(stats + "\n")

	// Language and License
	var meta []string
	if repo.Language != "" {
		meta = append(meta, fmt.Sprintf("● %s", repo.Language))
	}
	if repo.License != nil && repo.License.Name != "" {
		meta = append(meta, repo.License.Name)
	}
	if repo.Archived {
		meta = append(meta, "📦 Archived")
	}
	if repo.Fork {
		meta = append(meta, "⑂ Fork")
	}
	if len(meta) > 0 {
		sb.WriteString("  " + dimStyle.Render(strings.Join(meta, " │ ")) + "\n")
	}

	// Topics
	if len(repo.Topics) > 0 {
		topicsStr := tagStyle.Render(strings.Join(repo.Topics, ", "))
		sb.WriteString("  " + dimStyle.Render("Tags: ") + topicsStr + "\n")
	}

	// Updated time
	sb.WriteString("  " + dimStyle.Render(fmt.Sprintf("Updated %s", timeAgo(repo.PushedAt))) + "\n")

	// Links
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  [%d] %s\n", linkIdx, repo.HTMLURL))
	links = append(links, browser.Link{Index: linkIdx, Text: "Repository", URL: repo.HTMLURL})
	linkIdx++

	// README section
	if readme != "" {
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("  ─── README ───────────────────────────────────────────"))
		sb.WriteString("\n\n")

		// Render README with glamour
		rendered, err := renderMarkdown(readme, width-4)
		if err != nil {
			sb.WriteString("  " + wordWrap(readme, min(width-4, 76)) + "\n")
		} else {
			// Indent the rendered content
			for _, line := range strings.Split(rendered, "\n") {
				sb.WriteString("  " + line + "\n")
			}
		}
	}

	return sb.String(), links
}

// RenderIssue renders a GitHub issue.
func RenderIssue(issue *GitHubIssue, owner, repo string, width int) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link
	linkIdx := 1

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
	openStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950")).Bold(true)
	closedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149")).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a371f7"))

	// State badge
	stateStr := openStyle.Render("OPEN")
	if issue.State == "closed" {
		stateStr = closedStyle.Render("CLOSED")
	}

	// Header
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s #%d %s\n", stateStr, issue.Number, titleStyle.Render(issue.Title)))
	sb.WriteString(dimStyle.Render("  " + strings.Repeat("─", min(width-4, 60))))
	sb.WriteString("\n\n")

	// Meta
	author := "unknown"
	if issue.User != nil {
		author = issue.User.Login
	}
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  @%s opened %s │ %d comments", author, timeAgo(issue.CreatedAt), issue.Comments)))
	sb.WriteString("\n")

	// Labels
	if len(issue.Labels) > 0 {
		var labelNames []string
		for _, l := range issue.Labels {
			labelNames = append(labelNames, l.Name)
		}
		sb.WriteString("  " + labelStyle.Render(strings.Join(labelNames, ", ")) + "\n")
	}

	sb.WriteString("\n")

	// Body
	if issue.Body != "" {
		rendered, err := renderMarkdown(issue.Body, width-4)
		if err != nil {
			wrapped := wordWrap(issue.Body, min(width-4, 76))
			for _, line := range strings.Split(wrapped, "\n") {
				sb.WriteString("  " + line + "\n")
			}
		} else {
			for _, line := range strings.Split(rendered, "\n") {
				sb.WriteString("  " + line + "\n")
			}
		}
	} else {
		sb.WriteString(dimStyle.Render("  No description provided.") + "\n")
	}

	// Link
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  [%d] %s\n", linkIdx, issue.HTMLURL))
	links = append(links, browser.Link{Index: linkIdx, Text: "View on GitHub", URL: issue.HTMLURL})

	return sb.String(), links
}

// RenderPR renders a GitHub pull request.
func RenderPR(pr *GitHubPR, owner, repo string, width int) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link
	linkIdx := 1

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
	openStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950")).Bold(true)
	mergedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a371f7")).Bold(true)
	closedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149")).Bold(true)
	draftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e")).Bold(true)
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950"))
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a371f7"))

	// State badge
	var stateStr string
	if pr.Merged {
		stateStr = mergedStyle.Render("MERGED")
	} else if pr.Draft {
		stateStr = draftStyle.Render("DRAFT")
	} else if pr.State == "closed" {
		stateStr = closedStyle.Render("CLOSED")
	} else {
		stateStr = openStyle.Render("OPEN")
	}

	// Header
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  %s #%d %s\n", stateStr, pr.Number, titleStyle.Render(pr.Title)))
	sb.WriteString(dimStyle.Render("  " + strings.Repeat("─", min(width-4, 60))))
	sb.WriteString("\n\n")

	// Branch info
	if pr.Head != nil && pr.Base != nil {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("  %s → %s", pr.Head.Ref, pr.Base.Ref)))
		sb.WriteString("\n")
	}

	// Meta
	author := "unknown"
	if pr.User != nil {
		author = pr.User.Login
	}
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  @%s opened %s │ %d comments", author, timeAgo(pr.CreatedAt), pr.Comments)))
	sb.WriteString("\n")

	// Stats
	sb.WriteString(fmt.Sprintf("  %s │ %s │ %s\n",
		fmt.Sprintf("%d commits", pr.Commits),
		addStyle.Render(fmt.Sprintf("+%d", pr.Additions)),
		delStyle.Render(fmt.Sprintf("-%d", pr.Deletions))))

	// Labels
	if len(pr.Labels) > 0 {
		var labelNames []string
		for _, l := range pr.Labels {
			labelNames = append(labelNames, l.Name)
		}
		sb.WriteString("  " + labelStyle.Render(strings.Join(labelNames, ", ")) + "\n")
	}

	sb.WriteString("\n")

	// Body
	if pr.Body != "" {
		rendered, err := renderMarkdown(pr.Body, width-4)
		if err != nil {
			wrapped := wordWrap(pr.Body, min(width-4, 76))
			for _, line := range strings.Split(wrapped, "\n") {
				sb.WriteString("  " + line + "\n")
			}
		} else {
			for _, line := range strings.Split(rendered, "\n") {
				sb.WriteString("  " + line + "\n")
			}
		}
	} else {
		sb.WriteString(dimStyle.Render("  No description provided.") + "\n")
	}

	// Link
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  [%d] %s\n", linkIdx, pr.HTMLURL))
	links = append(links, browser.Link{Index: linkIdx, Text: "View on GitHub", URL: pr.HTMLURL})

	return sb.String(), links
}

// RenderGist renders a GitHub gist.
func RenderGist(gist *GitHubGist, width int) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link
	linkIdx := 1

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58a6ff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
	fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))

	// Header
	sb.WriteString("\n")
	owner := "anonymous"
	if gist.Owner != nil {
		owner = gist.Owner.Login
	}
	visibility := "secret"
	if gist.Public {
		visibility = "public"
	}
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  📋 Gist by @%s (%s)", owner, visibility)))
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  " + strings.Repeat("─", min(width-4, 60))))
	sb.WriteString("\n\n")

	// Description
	if gist.Description != "" {
		sb.WriteString("  " + wordWrap(gist.Description, min(width-4, 76)) + "\n\n")
	}

	// Meta
	sb.WriteString(dimStyle.Render(fmt.Sprintf("  Created %s │ %d comments", timeAgo(gist.CreatedAt), gist.Comments)))
	sb.WriteString("\n\n")

	// Files
	for filename, file := range gist.Files {
		sb.WriteString(fileStyle.Render(fmt.Sprintf("  ─── %s (%s, %d bytes) ───", filename, file.Language, file.Size)))
		sb.WriteString("\n\n")

		if file.Content != "" {
			// Render content (could be code or markdown)
			lines := strings.Split(file.Content, "\n")
			maxLines := 50 // Limit displayed lines
			for i, line := range lines {
				if i >= maxLines {
					sb.WriteString(dimStyle.Render(fmt.Sprintf("  ... (%d more lines)", len(lines)-maxLines)))
					sb.WriteString("\n")
					break
				}
				sb.WriteString("  " + line + "\n")
			}
		}
		sb.WriteString("\n")

		// Link to raw file
		sb.WriteString(fmt.Sprintf("  [%d] Raw: %s\n", linkIdx, file.RawURL))
		links = append(links, browser.Link{Index: linkIdx, Text: filename, URL: file.RawURL})
		linkIdx++
	}

	// Main link
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  [%d] %s\n", linkIdx, gist.HTMLURL))
	links = append(links, browser.Link{Index: linkIdx, Text: "View on GitHub", URL: gist.HTMLURL})

	return sb.String(), links
}

// RenderUser renders a GitHub user profile.
func RenderUser(user *GitHubUser, repos []GitHubRepo, width int) (string, []browser.Link) {
	var sb strings.Builder
	var links []browser.Link
	linkIdx := 1

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58a6ff"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8b949e"))
	statStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f0883e"))

	// Header
	sb.WriteString("\n")
	icon := "👤"
	if user.Type == "Organization" {
		icon = "🏢"
	}
	displayName := user.Login
	if user.Name != "" {
		displayName = fmt.Sprintf("%s (@%s)", user.Name, user.Login)
	}
	sb.WriteString(titleStyle.Render(fmt.Sprintf("  %s %s", icon, displayName)))
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  " + strings.Repeat("─", min(width-4, 60))))
	sb.WriteString("\n\n")

	// Bio
	if user.Bio != "" {
		wrapped := wordWrap(user.Bio, min(width-4, 76))
		for _, line := range strings.Split(wrapped, "\n") {
			sb.WriteString("  " + line + "\n")
		}
		sb.WriteString("\n")
	}

	// Stats
	sb.WriteString(fmt.Sprintf("  %s %d followers  %s %d following  %s %d repos\n",
		statStyle.Render("●"), user.Followers,
		statStyle.Render("●"), user.Following,
		statStyle.Render("●"), user.PublicRepos))

	// Additional info
	var info []string
	if user.Company != "" {
		info = append(info, "🏢 "+user.Company)
	}
	if user.Location != "" {
		info = append(info, "📍 "+user.Location)
	}
	if user.Blog != "" {
		info = append(info, "🔗 "+user.Blog)
	}
	if len(info) > 0 {
		sb.WriteString("  " + dimStyle.Render(strings.Join(info, " │ ")) + "\n")
	}

	sb.WriteString("  " + dimStyle.Render(fmt.Sprintf("Joined %s", timeAgo(user.CreatedAt))) + "\n")

	// Profile link
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("  [%d] %s\n", linkIdx, user.HTMLURL))
	links = append(links, browser.Link{Index: linkIdx, Text: "GitHub Profile", URL: user.HTMLURL})
	linkIdx++

	// Repositories
	if len(repos) > 0 {
		sb.WriteString("\n")
		sb.WriteString(dimStyle.Render("  ─── Recent Repositories ───────────────────────────────"))
		sb.WriteString("\n\n")

		for _, r := range repos {
			desc := r.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			if desc == "" {
				desc = dimStyle.Render("No description")
			}

			sb.WriteString(fmt.Sprintf("  [%d] %s", linkIdx, r.Name))
			if r.Language != "" {
				sb.WriteString(dimStyle.Render(fmt.Sprintf(" (%s)", r.Language)))
			}
			sb.WriteString(fmt.Sprintf(" ★%d\n", r.StargazersCount))
			sb.WriteString("      " + desc + "\n\n")

			links = append(links, browser.Link{Index: linkIdx, Text: r.Name, URL: r.HTMLURL})
			linkIdx++
		}
	}

	return sb.String(), links
}

// FetchURL auto-detects a GitHub URL type and fetches/renders it.
func (g *GitHubClient) FetchURL(info *GitHubURLInfo, width int) (string, string, []browser.Link, error) {
	switch info.Type {
	case GitHubURLRepo:
		repo, err := g.FetchRepo(info.Owner, info.Repo)
		if err != nil {
			return "", "", nil, err
		}
		readme, _ := g.FetchReadme(info.Owner, info.Repo) // Ignore readme errors
		content, links := RenderRepo(repo, readme, width)
		title := fmt.Sprintf("%s/%s - GitHub", repo.Owner.Login, repo.Name)
		return content, title, links, nil

	case GitHubURLIssue:
		issue, err := g.FetchIssue(info.Owner, info.Repo, info.Number)
		if err != nil {
			return "", "", nil, err
		}
		content, links := RenderIssue(issue, info.Owner, info.Repo, width)
		title := fmt.Sprintf("#%d: %s", issue.Number, truncate(issue.Title, 40))
		return content, title, links, nil

	case GitHubURLPR:
		pr, err := g.FetchPR(info.Owner, info.Repo, info.Number)
		if err != nil {
			return "", "", nil, err
		}
		content, links := RenderPR(pr, info.Owner, info.Repo, width)
		title := fmt.Sprintf("PR #%d: %s", pr.Number, truncate(pr.Title, 40))
		return content, title, links, nil

	case GitHubURLGist:
		gist, err := g.FetchGist(info.GistID)
		if err != nil {
			return "", "", nil, err
		}
		content, links := RenderGist(gist, width)
		desc := gist.Description
		if desc == "" {
			desc = "Gist"
		}
		title := fmt.Sprintf("Gist: %s", truncate(desc, 40))
		return content, title, links, nil

	case GitHubURLUser:
		user, err := g.FetchUser(info.User)
		if err != nil {
			return "", "", nil, err
		}
		repos, _ := g.FetchUserRepos(info.User, 10) // Ignore repo fetch errors
		content, links := RenderUser(user, repos, width)
		displayName := user.Login
		if user.Name != "" {
			displayName = user.Name
		}
		title := fmt.Sprintf("%s - GitHub", displayName)
		return content, title, links, nil

	default:
		return "", "", nil, fmt.Errorf("unsupported GitHub URL type")
	}
}

// --- Helper Functions ---

// renderMarkdown renders markdown content using glamour.
func renderMarkdown(content string, width int) (string, error) {
	if width < 40 {
		width = 40
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(content)
}

// repoIcon returns an icon for a repository.
func repoIcon(repo *GitHubRepo) string {
	if repo.Archived {
		return "📦"
	}
	if repo.Fork {
		return "⑂"
	}
	if repo.Private {
		return "🔒"
	}
	return "📁"
}

// formatNumber formats large numbers with K/M suffix.
func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// min returns the smaller of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
