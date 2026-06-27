package content

import (
	"strings"
	"testing"
)

func TestRenderHTMLMarkdown(t *testing.T) {
	html, err := RenderHTML(Entry{
		Renderer: RendererMarkdown,
		Raw:      "# Hello\n\nThis is **bold** and [a link](https://example.com).",
	})
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}
	got := string(html)
	if !strings.Contains(got, "<h1") || !strings.Contains(got, "Hello") {
		t.Fatalf("expected h1 with text, got: %s", got)
	}
	if !strings.Contains(got, "<strong>bold</strong>") {
		t.Fatalf("expected bold formatting preserved, got: %s", got)
	}
	if !strings.Contains(got, `href="https://example.com"`) {
		t.Fatalf("expected link preserved, got: %s", got)
	}
}

func TestRenderHTMLMarkdownStripsScript(t *testing.T) {
	html, err := RenderHTML(Entry{
		Renderer: RendererMarkdown,
		Raw:      "# Title\n\n<script>alert('xss')</script>\n\nText after.",
	})
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}
	got := string(html)
	if strings.Contains(got, "<script>") {
		t.Fatalf("script tag must be stripped, got: %s", got)
	}
	if !strings.Contains(got, "Title") || !strings.Contains(got, "Text after") {
		t.Fatalf("expected surrounding content preserved, got: %s", got)
	}
}

func TestRenderHTMLMarkdownImgAllowed(t *testing.T) {
	html, err := RenderHTML(Entry{
		Renderer: RendererMarkdown,
		Raw:      "![alt](https://example.com/logo.png)",
	})
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}
	got := string(html)
	if !strings.Contains(got, "<img") || !strings.Contains(got, "logo.png") {
		t.Fatalf("expected safe img tag preserved, got: %s", got)
	}
}

func TestRenderHTMLRawStripsScript(t *testing.T) {
	html, err := RenderHTML(Entry{
		Renderer: RendererHTML,
		Raw:      `<h1>Title</h1><script>alert('xss')</script><img src="x" onerror="alert(1)">`,
	})
	if err != nil {
		t.Fatalf("render raw html: %v", err)
	}
	got := string(html)
	if strings.Contains(got, "<script>") {
		t.Fatalf("script tag must be stripped, got: %s", got)
	}
	if strings.Contains(got, "onerror") {
		t.Fatalf("onerror handler must be stripped, got: %s", got)
	}
	if !strings.Contains(got, "<h1>Title</h1>") {
		t.Fatalf("expected safe heading preserved, got: %s", got)
	}
}

func TestRenderHTMLRawPreservesStyle(t *testing.T) {
	html, err := RenderHTML(Entry{
		Renderer: RendererHTML,
		Raw:      `<style>.red { color: red; }</style><p class="red" style="font-weight: bold;">Hi</p>`,
	})
	if err != nil {
		t.Fatalf("render raw html: %v", err)
	}
	got := string(html)
	if !strings.Contains(got, "<style>") {
		t.Fatalf("style block must be preserved for raw HTML, got: %s", got)
	}
	if !strings.Contains(got, "class=") {
		t.Fatalf("class attribute must be preserved for raw HTML, got: %s", got)
	}
	if !strings.Contains(got, "style=") {
		t.Fatalf("style attribute must be preserved for raw HTML, got: %s", got)
	}
}

func TestRenderHTMLRawStripsJavascriptLink(t *testing.T) {
	html, err := RenderHTML(Entry{
		Renderer: RendererHTML,
		Raw:      `<a href="javascript:alert(1)">click</a>`,
	})
	if err != nil {
		t.Fatalf("render raw html: %v", err)
	}
	got := string(html)
	if strings.Contains(got, "javascript:") {
		t.Fatalf("javascript: link must be stripped, got: %s", got)
	}
}

// TestRenderHTMLRawAllowsStructureAndMedia 验证 raw-HTML 渲染器放开了
// 低风险的结构交互与媒体元素，避免内容被静默剥离。
func TestRenderHTMLRawAllowsStructureAndMedia(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		mustHave string
	}{
		{"details summary", `<details><summary>more</summary><p>x</p></details>`, "<details>"},
		{"a target blank", `<a href="https://x.com" target="_blank">x</a>`, "target=\"_blank\""},
		{"video controls", `<video controls><source src="v.mp4" type="video/mp4"></video>`, "<video"},
		{"audio controls", `<audio controls src="a.mp3"></audio>`, "<audio"},
		{"picture source", `<picture><source srcset="a.webp" type="image/webp"></picture>`, "<picture"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			html, err := RenderHTML(Entry{Renderer: RendererHTML, Raw: c.raw})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := string(html); !strings.Contains(got, c.mustHave) {
				t.Fatalf("expected %q in output, got: %s", c.mustHave, got)
			}
		})
	}
}

// TestRenderHTMLRawStillRestrictsHighRisk 验证高风险元素仍被剥离。
func TestRenderHTMLRawStillRestrictsHighRisk(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		bad  string
	}{
		{"iframe stripped", `<iframe src="https://evil.com"></iframe>`, "<iframe"},
		{"form stripped", `<form><input name="q"></form>`, "<form"},
		{"script stripped", `<script>alert(1)</script>`, "<script"},
		{"onerror stripped", `<img src="x" onerror="alert(1)">`, "onerror"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			html, err := RenderHTML(Entry{Renderer: RendererHTML, Raw: c.raw})
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := string(html); strings.Contains(got, c.bad) {
				t.Fatalf("expected %q to be stripped, got: %s", c.bad, got)
			}
		})
	}
}

func TestRenderHTMLUnsupported(t *testing.T) {
	if _, err := RenderHTML(Entry{Renderer: RendererType("xml")}); err == nil {
		t.Fatalf("expected error for unsupported renderer")
	}
}
