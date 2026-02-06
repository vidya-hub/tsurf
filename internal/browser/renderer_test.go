package browser

import (
	"fmt"
	"testing"
)

func TestRenderBasicHTML(t *testing.T) {
	article := &Article{
		Title:  "Test Page",
		Byline: "By Author",
		Content: `<h1>Test Page</h1>
<p>Hello world. This is a <strong>bold</strong> and <em>italic</em> test.</p>
<p>Here is a <a href="https://example.com">link to example</a> and <a href="https://golang.org">Go website</a>.</p>
<ul>
<li>Item one</li>
<li>Item two</li>
<li>Item three</li>
</ul>
<pre><code class="language-go">func main() {
    fmt.Println("Hello")
}</code></pre>
<blockquote>This is a quote</blockquote>`,
		TextContent: "fallback text",
	}

	page := Render(article, 80)
	fmt.Println("=== RENDERED CONTENT ===")
	fmt.Println(page.Content)
	fmt.Println("=== LINKS ===")
	for _, l := range page.Links {
		fmt.Printf("[%d] %s -> %s\n", l.Index, l.Text, l.URL)
	}

	if len(page.Links) != 2 {
		t.Errorf("Expected 2 links, got %d", len(page.Links))
	}
	if page.Content == "" {
		t.Error("Content should not be empty")
	}
	if page.Title != "Test Page" {
		t.Errorf("Expected title 'Test Page', got '%s'", page.Title)
	}
}

func TestRenderFallbackBasicHTML(t *testing.T) {
	article := &Article{
		Title:       "Fallback Test",
		Content:     `<p>Testing the <a href="https://test.com">fallback renderer</a>.</p>`,
		TextContent: "fallback text",
	}

	page := RenderFallback(article, 80)
	fmt.Println("=== FALLBACK CONTENT ===")
	fmt.Println(page.Content)
	fmt.Println("=== FALLBACK LINKS ===")
	for _, l := range page.Links {
		fmt.Printf("[%d] %s -> %s\n", l.Index, l.Text, l.URL)
	}

	if len(page.Links) != 1 {
		t.Errorf("Expected 1 link, got %d", len(page.Links))
	}
	if page.Content == "" {
		t.Error("Content should not be empty")
	}
}

func TestRenderEmptyArticle(t *testing.T) {
	article := &Article{
		Title:       "",
		Content:     "",
		TextContent: "some text",
	}

	page := Render(article, 80)
	if page == nil {
		t.Error("Page should not be nil")
	}
}

func TestRenderWithTable(t *testing.T) {
	article := &Article{
		Title: "Table Test",
		Content: `<table>
<thead><tr><th>Name</th><th>Value</th></tr></thead>
<tbody>
<tr><td>Foo</td><td>Bar</td></tr>
<tr><td>Baz</td><td>Qux</td></tr>
</tbody>
</table>`,
		TextContent: "table text",
	}

	page := Render(article, 80)
	fmt.Println("=== TABLE CONTENT ===")
	fmt.Println(page.Content)

	if page.Content == "" {
		t.Error("Content should not be empty")
	}
}
