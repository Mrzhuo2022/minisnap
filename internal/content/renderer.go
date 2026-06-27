package content

import (
	"bytes"
	"errors"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

var md = goldmark.New()

// markdownPolicy 用于消毒 Markdown 渲染产物。
// 保留富文本格式（标题、段落、列表、表格、图片、链接、强调等），
// 剥离 <script>、内联事件处理器、javascript: 链接等危险内容。
//
// 这里在 NewPolicy() 基础上手动构建白名单（等价于 UGFPolicy 的覆盖范围），
// 而非直接调用 bluemonday.UGFPolicy()，以保证工具链兼容性。
var markdownPolicy = buildMarkdownPolicy()

// htmlPolicy 用于消毒原始 HTML 渲染产物。
// 在 markdownPolicy 基础上放宽：额外允许 <style> 块及 style/class 属性，
// 便于高级排版，但仍然剥离 <script>、on* 事件、javascript: 链接。
var htmlPolicy = buildHTMLPolicy()

func buildMarkdownPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	// 全局：标准属性（id/name/title/dir 等）与标准 URL 协议校验
	p.AllowStandardAttributes()
	p.AllowStandardURLs()

	// 结构与分节
	p.AllowElements("article", "aside", "figure", "section", "summary")

	// 标题
	p.AllowElements("h1", "h2", "h3", "h4", "h5", "h6", "hgroup")

	// 内容分组与分隔
	p.AllowElements("br", "div", "hr", "p", "span", "wbr")
	p.AllowAttrs("cite").OnElements("blockquote", "q")

	// 链接
	p.AllowAttrs("href").OnElements("a")

	// 短语元素（内联语义）
	p.AllowElements(
		"abbr", "acronym", "cite", "code", "dfn", "em",
		"figcaption", "mark", "s", "samp", "strong", "sub", "sup", "var",
	)
	p.AllowAttrs("datetime").OnElements("time")

	// 样式元素（不含 <style> 块）
	p.AllowElements("b", "i", "pre", "small", "strike", "tt", "u")

	// 列表、表格、图片
	p.AllowLists()
	p.AllowTables()
	p.AllowImages()

	return p
}

func buildHTMLPolicy() *bluemonday.Policy {
	// raw-HTML 在 markdown 白名单基础上放宽，面向想完全控制 HTML 的可信管理员。
	//
	// 放开范围：样式（<style> 块 + style/class 属性）、结构交互（<details>/<summary>）、
	// 链接（target，bluemonday 会自动补 rel="noopener"）、媒体（<video>/<audio>/
	// <source>/<track>/<picture>）。
	//
	// 仍然限制：iframe/embed/object（点击劫持风险）、form/input/button（钓鱼/CSRF 风险），
	// 以及 <script>、on* 事件处理器、javascript: 链接等。
	//
	// AllowUnsafe(true) 仅用于保留 <style> 块（bluemonday 默认剥离无法安全处理的元素），
	// markdownPolicy 不开启，保持最严格。
	p := buildMarkdownPolicy()
	p.AllowUnsafe(true)

	// 样式
	p.AllowElements("style")
	p.AllowAttrs("style").Globally()
	p.AllowAttrs("class").Globally()

	// 结构与交互
	p.AllowElements("details")
	p.AllowAttrs("open").OnElements("details")
	p.AllowElements("summary")

	// 链接：允许新窗口打开（bluemonday 对 target=_blank 自动补 rel="noopener"）
	p.AllowAttrs("target").OnElements("a")

	// 媒体
	p.AllowElements("video", "audio", "source", "track", "picture")
	p.AllowAttrs("src", "poster", "preload", "controls", "loop", "muted", "autoplay").OnElements("video")
	p.AllowAttrs("width", "height").OnElements("video")
	p.AllowAttrs("src", "preload", "controls", "loop", "muted", "autoplay").OnElements("audio")
	p.AllowAttrs("src", "srcset", "sizes", "type", "media").OnElements("source")
	p.AllowAttrs("src", "kind", "srclang", "label").OnElements("track")

	return p
}

// RenderHTML 将 Entry 渲染成已消毒的安全 HTML 字符串。
func RenderHTML(entry Entry) (template.HTML, error) {
	switch entry.Renderer {
	case RendererMarkdown:
		var buf bytes.Buffer
		if err := md.Convert([]byte(entry.Raw), &buf); err != nil {
			return "", err
		}
		return template.HTML(markdownPolicy.Sanitize(buf.String())), nil
	case RendererHTML:
		return template.HTML(htmlPolicy.Sanitize(entry.Raw)), nil
	default:
		return "", errors.New("unsupported renderer")
	}
}
