package content

import (
	"bytes"
	"errors"
	"html/template"

	"github.com/yuin/goldmark"
)

var md = goldmark.New()

// RenderHTML 将 Entry 渲染成安全的 HTML 字符串。
func RenderHTML(entry Entry) (template.HTML, error) {
	switch entry.Renderer {
	case RendererMarkdown:
		var buf bytes.Buffer
		if err := md.Convert([]byte(entry.Raw), &buf); err != nil {
			return "", err
		}
		return template.HTML(buf.String()), nil
	case RendererHTML:
		return template.HTML(entry.Raw), nil
	default:
		return "", errors.New("unsupported renderer")
	}
}
