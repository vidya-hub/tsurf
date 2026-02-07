package browser

import (
	"fmt"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/vidyasagar/tsurf/internal/theme"
)

// Cached glamour renderer to avoid recreation on every render call.
var (
	cachedRenderer      *glamour.TermRenderer
	cachedRendererWidth int
	rendererMu          sync.Mutex
)

// RenderedPage holds the final terminal-ready output.
type RenderedPage struct {
	Title   string
	Content string // styled terminal text
	Links   []Link
}

// Render converts an Article's HTML content into styled terminal text.
func Render(article *Article, width int) *RenderedPage {
	if width <= 0 {
		width = 80
	}

	// Constrain content width for readability.
	contentWidth := width - 4
	if contentWidth > 100 {
		contentWidth = 100
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return &RenderedPage{
			Title:   article.Title,
			Content: article.TextContent,
			Links:   article.Links,
		}
	}

	// Convert HTML to markdown, collecting links along the way.
	conv := &mdConverter{
		linkIndex: 0,
		links:     nil,
	}

	var md strings.Builder

	// Title.
	if article.Title != "" {
		md.WriteString("# " + article.Title + "\n\n")
	}

	// Byline.
	if article.Byline != "" {
		md.WriteString("*" + article.Byline + "*\n\n")
	}

	md.WriteString("---\n\n")

	// Convert body HTML to markdown.
	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		md.WriteString(conv.convertNode(s, 0))
	})

	// Render markdown with glamour.
	rendered, glamErr := renderWithGlamour(md.String(), contentWidth)
	if glamErr != nil {
		// Fallback: use the raw markdown.
		rendered = md.String()
	}

	return &RenderedPage{
		Title:   article.Title,
		Content: rendered,
		Links:   conv.links,
	}
}

// renderWithGlamour uses glamour to render markdown into styled terminal output.
// Uses a cached renderer to avoid expensive recreation on every call.
func renderWithGlamour(markdown string, width int) (string, error) {
	rendererMu.Lock()
	defer rendererMu.Unlock()

	// Recreate renderer only if width changed or not initialized.
	if cachedRenderer == nil || cachedRendererWidth != width {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return "", err
		}
		cachedRenderer = renderer
		cachedRendererWidth = width
	}

	out, err := cachedRenderer.Render(markdown)
	if err != nil {
		return "", err
	}

	return out, nil
}

// mdConverter converts goquery HTML nodes to markdown.
type mdConverter struct {
	linkIndex int
	links     []Link
}

func (c *mdConverter) convertNode(s *goquery.Selection, depth int) string {
	var sb strings.Builder

	tagName := goquery.NodeName(s)

	switch tagName {
	case "h1":
		sb.WriteString(c.convertHeading(s, 1))
	case "h2":
		sb.WriteString(c.convertHeading(s, 2))
	case "h3":
		sb.WriteString(c.convertHeading(s, 3))
	case "h4":
		sb.WriteString(c.convertHeading(s, 4))
	case "h5":
		sb.WriteString(c.convertHeading(s, 5))
	case "h6":
		sb.WriteString(c.convertHeading(s, 6))
	case "p":
		sb.WriteString(c.convertParagraph(s))
	case "a":
		sb.WriteString(c.convertLink(s))
	case "ul":
		sb.WriteString(c.convertList(s, false, depth))
	case "ol":
		sb.WriteString(c.convertList(s, true, depth))
	case "blockquote":
		sb.WriteString(c.convertBlockquote(s))
	case "pre":
		sb.WriteString(c.convertCodeBlock(s))
	case "code":
		sb.WriteString(c.convertInlineCode(s))
	case "img":
		sb.WriteString(c.convertImage(s))
	case "hr":
		sb.WriteString("\n---\n\n")
	case "table":
		sb.WriteString(c.convertTable(s))
	case "br":
		sb.WriteString("  \n")
	case "strong", "b":
		sb.WriteString("**")
		c.convertInlineChildren(s, &sb)
		sb.WriteString("**")
	case "em", "i":
		sb.WriteString("*")
		c.convertInlineChildren(s, &sb)
		sb.WriteString("*")
	case "div", "article", "section", "main", "header", "footer", "figure", "span":
		s.Children().Each(func(i int, child *goquery.Selection) {
			sb.WriteString(c.convertNode(child, depth))
		})
	case "figcaption":
		text := strings.TrimSpace(s.Text())
		if text != "" {
			sb.WriteString("*" + text + "*\n\n")
		}
	default:
		text := strings.TrimSpace(s.Text())
		if text != "" {
			sb.WriteString(text)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func (c *mdConverter) convertHeading(s *goquery.Selection, level int) string {
	text := strings.TrimSpace(s.Text())
	if text == "" {
		return ""
	}
	prefix := strings.Repeat("#", level) + " "
	return prefix + text + "\n\n"
}

func (c *mdConverter) convertParagraph(s *goquery.Selection) string {
	var sb strings.Builder
	c.convertInlineChildren(s, &sb)
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return ""
	}
	return text + "\n\n"
}

func (c *mdConverter) convertInlineChildren(s *goquery.Selection, sb *strings.Builder) {
	s.Contents().Each(func(i int, child *goquery.Selection) {
		if goquery.NodeName(child) == "#text" {
			sb.WriteString(child.Text())
		} else {
			switch goquery.NodeName(child) {
			case "a":
				sb.WriteString(c.convertLink(child))
			case "strong", "b":
				sb.WriteString("**")
				c.convertInlineChildren(child, sb)
				sb.WriteString("**")
			case "em", "i":
				sb.WriteString("*")
				c.convertInlineChildren(child, sb)
				sb.WriteString("*")
			case "code":
				sb.WriteString("`" + child.Text() + "`")
			case "br":
				sb.WriteString("  \n")
			default:
				c.convertInlineChildren(child, sb)
			}
		}
	})
}

func (c *mdConverter) convertLink(s *goquery.Selection) string {
	href, exists := s.Attr("href")
	text := strings.TrimSpace(s.Text())
	if text == "" {
		text = href
	}

	if !exists || href == "" {
		return text
	}

	c.linkIndex++
	c.links = append(c.links, Link{
		Index: c.linkIndex,
		Text:  text,
		URL:   href,
	})

	// Return markdown link with numbered reference.
	return fmt.Sprintf("[%s](%s) **[%d]**", text, href, c.linkIndex)
}

func (c *mdConverter) convertList(s *goquery.Selection, ordered bool, depth int) string {
	var sb strings.Builder
	itemNum := 0

	indent := strings.Repeat("  ", depth)

	s.Find("> li").Each(func(i int, li *goquery.Selection) {
		itemNum++
		var prefix string
		if ordered {
			prefix = fmt.Sprintf("%s%d. ", indent, itemNum)
		} else {
			prefix = indent + "- "
		}

		var itemSb strings.Builder
		c.convertInlineChildren(li, &itemSb)
		text := strings.TrimSpace(itemSb.String())

		sb.WriteString(prefix + text + "\n")

		// Handle nested lists.
		li.Children().Each(func(j int, child *goquery.Selection) {
			tag := goquery.NodeName(child)
			if tag == "ul" {
				sb.WriteString(c.convertList(child, false, depth+1))
			} else if tag == "ol" {
				sb.WriteString(c.convertList(child, true, depth+1))
			}
		})
	})

	return sb.String() + "\n"
}

func (c *mdConverter) convertBlockquote(s *goquery.Selection) string {
	var sb strings.Builder
	s.Children().Each(func(i int, child *goquery.Selection) {
		content := c.convertNode(child, 0)
		for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
			sb.WriteString("> " + line + "\n")
		}
	})
	sb.WriteString("\n")
	return sb.String()
}

func (c *mdConverter) convertCodeBlock(s *goquery.Selection) string {
	code := s.Find("code")

	// Try to detect language from class.
	lang := ""
	if code.Length() > 0 {
		class, _ := code.Attr("class")
		if strings.Contains(class, "language-") {
			parts := strings.Split(class, "language-")
			if len(parts) > 1 {
				lang = strings.Fields(parts[1])[0]
			}
		}
	}

	text := ""
	if code.Length() > 0 {
		text = code.Text()
	} else {
		text = s.Text()
	}

	return "```" + lang + "\n" + text + "\n```\n\n"
}

func (c *mdConverter) convertInlineCode(s *goquery.Selection) string {
	return "`" + s.Text() + "`"
}

func (c *mdConverter) convertImage(s *goquery.Selection) string {
	alt, _ := s.Attr("alt")
	src, _ := s.Attr("src")

	if alt == "" {
		alt = "image"
	}

	return fmt.Sprintf("![%s](%s)\n\n", alt, src)
}

func (c *mdConverter) convertTable(s *goquery.Selection) string {
	var sb strings.Builder

	// Collect headers.
	var headers []string
	s.Find("thead th, thead td").Each(func(i int, th *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(th.Text()))
	})

	// Collect rows.
	var rows [][]string
	s.Find("tbody tr").Each(func(i int, tr *goquery.Selection) {
		var row []string
		tr.Find("td, th").Each(func(j int, td *goquery.Selection) {
			row = append(row, strings.TrimSpace(td.Text()))
		})
		rows = append(rows, row)
	})

	// If no thead, try first row as header.
	if len(headers) == 0 {
		s.Find("tr").First().Find("th, td").Each(func(i int, cell *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(cell.Text()))
		})
	}

	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	if numCols == 0 {
		return ""
	}

	// Pad headers if needed.
	for len(headers) < numCols {
		headers = append(headers, "")
	}

	// Write markdown table.
	sb.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	separators := make([]string, numCols)
	for i := range separators {
		separators[i] = "---"
	}
	sb.WriteString("| " + strings.Join(separators, " | ") + " |\n")

	for _, row := range rows {
		for len(row) < numCols {
			row = append(row, "")
		}
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// --- Fallback renderer (used if glamour is not available) ---

// RenderFallback is a simple renderer without glamour, used as fallback.
func RenderFallback(article *Article, width int) *RenderedPage {
	if width <= 0 {
		width = 80
	}

	contentWidth := width - 4
	if contentWidth > 100 {
		contentWidth = 100
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return &RenderedPage{
			Title:   article.Title,
			Content: article.TextContent,
			Links:   article.Links,
		}
	}

	r := &fallbackRenderer{
		width:     contentWidth,
		linkIndex: 0,
		links:     nil,
	}

	var sb strings.Builder

	// Render title.
	if article.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Current.Heading).
			Width(contentWidth).
			MarginBottom(1)
		sb.WriteString(titleStyle.Render(article.Title))
		sb.WriteString("\n\n")
	}

	// Render byline.
	if article.Byline != "" {
		bylineStyle := lipgloss.NewStyle().
			Italic(true).
			Foreground(theme.Current.TextDim)
		sb.WriteString(bylineStyle.Render(article.Byline))
		sb.WriteString("\n\n")
	}

	// Separator.
	sepStyle := lipgloss.NewStyle().Foreground(theme.Current.Border)
	sb.WriteString(sepStyle.Render(strings.Repeat("─", min(contentWidth, 60))))
	sb.WriteString("\n\n")

	// Render body.
	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		sb.WriteString(r.renderNode(s))
	})

	return &RenderedPage{
		Title:   article.Title,
		Content: sb.String(),
		Links:   r.links,
	}
}

type fallbackRenderer struct {
	width     int
	linkIndex int
	links     []Link
}

func (r *fallbackRenderer) renderNode(s *goquery.Selection) string {
	var sb strings.Builder

	tagName := goquery.NodeName(s)

	switch tagName {
	case "h1":
		sb.WriteString(r.renderHeading(s, 1))
	case "h2":
		sb.WriteString(r.renderHeading(s, 2))
	case "h3":
		sb.WriteString(r.renderHeading(s, 3))
	case "h4", "h5", "h6":
		sb.WriteString(r.renderHeading(s, 4))
	case "p":
		sb.WriteString(r.renderParagraph(s))
	case "a":
		sb.WriteString(r.renderLink(s))
	case "ul":
		sb.WriteString(r.renderList(s, false))
	case "ol":
		sb.WriteString(r.renderList(s, true))
	case "blockquote":
		sb.WriteString(r.renderBlockquote(s))
	case "pre":
		sb.WriteString(r.renderCodeBlock(s))
	case "code":
		sb.WriteString(r.renderInlineCode(s))
	case "img":
		sb.WriteString(r.renderImage(s))
	case "hr":
		sb.WriteString(r.renderHR())
	case "table":
		sb.WriteString(r.renderTable(s))
	case "br":
		sb.WriteString("\n")
	case "div", "article", "section", "main", "header", "footer", "figure", "figcaption", "span":
		s.Children().Each(func(i int, child *goquery.Selection) {
			sb.WriteString(r.renderNode(child))
		})
	default:
		text := strings.TrimSpace(s.Text())
		if text != "" {
			sb.WriteString(text)
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func (r *fallbackRenderer) renderHeading(s *goquery.Selection, level int) string {
	text := strings.TrimSpace(s.Text())
	if text == "" {
		return ""
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Heading)

	var prefix string
	switch level {
	case 1:
		style = style.
			MarginTop(1).
			MarginBottom(1).
			Underline(true)
	case 2:
		prefix = "## "
		style = style.MarginBottom(1)
	case 3:
		prefix = "### "
		style = style.MarginBottom(1)
	default:
		prefix = "#### "
	}

	return style.Render(prefix+text) + "\n\n"
}

func (r *fallbackRenderer) renderParagraph(s *goquery.Selection) string {
	var sb strings.Builder
	r.renderInline(s, &sb)
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return ""
	}

	// Word-wrap the paragraph text.
	wrapped := wrapText(text, r.width)

	style := lipgloss.NewStyle().
		Foreground(theme.Current.Text)

	return style.Render(wrapped) + "\n\n"
}

func (r *fallbackRenderer) renderInline(s *goquery.Selection, sb *strings.Builder) {
	s.Contents().Each(func(i int, child *goquery.Selection) {
		if goquery.NodeName(child) == "#text" {
			sb.WriteString(child.Text())
		} else {
			switch goquery.NodeName(child) {
			case "a":
				sb.WriteString(r.renderLink(child))
			case "strong", "b":
				style := lipgloss.NewStyle().Bold(true)
				sb.WriteString(style.Render(child.Text()))
			case "em", "i":
				style := lipgloss.NewStyle().Italic(true)
				sb.WriteString(style.Render(child.Text()))
			case "code":
				sb.WriteString(r.renderInlineCode(child))
			case "br":
				sb.WriteString("\n")
			default:
				r.renderInline(child, sb)
			}
		}
	})
}

func (r *fallbackRenderer) renderLink(s *goquery.Selection) string {
	href, exists := s.Attr("href")
	text := strings.TrimSpace(s.Text())
	if text == "" {
		text = href
	}

	if !exists || href == "" {
		return text
	}

	r.linkIndex++
	r.links = append(r.links, Link{
		Index: r.linkIndex,
		Text:  text,
		URL:   href,
	})

	linkStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Link).
		Underline(true)

	indexStyle := lipgloss.NewStyle().
		Foreground(theme.Current.LinkIndex).
		Bold(true)

	return linkStyle.Render(text) + indexStyle.Render(fmt.Sprintf(" [%d]", r.linkIndex))
}

func (r *fallbackRenderer) renderList(s *goquery.Selection, ordered bool) string {
	var sb strings.Builder
	itemNum := 0

	s.Find("> li").Each(func(i int, li *goquery.Selection) {
		itemNum++
		var prefix string
		if ordered {
			prefix = fmt.Sprintf("  %d. ", itemNum)
		} else {
			prefix = "  • "
		}

		prefixStyle := lipgloss.NewStyle().Foreground(theme.Current.Accent)
		textStyle := lipgloss.NewStyle().Foreground(theme.Current.Text)

		var itemSb strings.Builder
		r.renderInline(li, &itemSb)
		text := strings.TrimSpace(itemSb.String())

		sb.WriteString(prefixStyle.Render(prefix))
		sb.WriteString(textStyle.Render(text))
		sb.WriteString("\n")
	})

	return sb.String() + "\n"
}

func (r *fallbackRenderer) renderBlockquote(s *goquery.Selection) string {
	text := strings.TrimSpace(s.Text())
	if text == "" {
		return ""
	}

	quoteStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Quote).
		Italic(true).
		PaddingLeft(2).
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(theme.Current.Accent)

	return quoteStyle.Render(text) + "\n\n"
}

func (r *fallbackRenderer) renderCodeBlock(s *goquery.Selection) string {
	code := s.Find("code").Text()
	if code == "" {
		code = s.Text()
	}

	codeStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Code).
		Background(theme.Current.CodeBg).
		Padding(1, 2).
		Width(r.width)

	return codeStyle.Render(code) + "\n\n"
}

func (r *fallbackRenderer) renderInlineCode(s *goquery.Selection) string {
	style := lipgloss.NewStyle().
		Foreground(theme.Current.Code).
		Background(theme.Current.CodeBg).
		Padding(0, 1)

	return style.Render(s.Text())
}

func (r *fallbackRenderer) renderImage(s *goquery.Selection) string {
	alt, _ := s.Attr("alt")
	src, _ := s.Attr("src")

	if alt == "" {
		alt = "image"
	}

	imgStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextDim).
		Italic(true)

	return imgStyle.Render(fmt.Sprintf("[IMG: %s] (%s)", alt, src)) + "\n\n"
}

func (r *fallbackRenderer) renderHR() string {
	style := lipgloss.NewStyle().Foreground(theme.Current.Border)
	return "\n" + style.Render(strings.Repeat("─", min(r.width, 60))) + "\n\n"
}

func (r *fallbackRenderer) renderTable(s *goquery.Selection) string {
	var sb strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Heading).
		Padding(0, 1)

	cellStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Border)

	var headers []string
	s.Find("thead th, thead td").Each(func(i int, th *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(th.Text()))
	})

	var rows [][]string
	s.Find("tbody tr").Each(func(i int, tr *goquery.Selection) {
		var row []string
		tr.Find("td, th").Each(func(j int, td *goquery.Selection) {
			row = append(row, strings.TrimSpace(td.Text()))
		})
		rows = append(rows, row)
	})

	if len(headers) == 0 && len(rows) > 0 {
		s.Find("tr").First().Find("th, td").Each(func(i int, cell *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(cell.Text()))
		})
	}

	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	if numCols == 0 {
		return ""
	}

	colWidths := make([]int, numCols)
	for i, h := range headers {
		if len(h) > colWidths[i] {
			colWidths[i] = len(h)
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < numCols && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	if len(headers) > 0 {
		for i, h := range headers {
			sb.WriteString(headerStyle.Width(colWidths[i] + 2).Render(h))
		}
		sb.WriteString("\n")
		for _, w := range colWidths {
			sb.WriteString(borderStyle.Render(strings.Repeat("─", w+2)))
		}
		sb.WriteString("\n")
	}

	for _, row := range rows {
		for i := 0; i < numCols; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			sb.WriteString(cellStyle.Width(colWidths[i] + 2).Render(cell))
		}
		sb.WriteString("\n")
	}

	return sb.String() + "\n"
}

// wrapText wraps a string at the given width, breaking at word boundaries.
func wrapText(text string, width int) string {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
