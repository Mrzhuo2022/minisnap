package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"minisnap/internal/config"
	"minisnap/internal/content"
)

// TestViewPageRendering 验证 Markdown 阅读页渲染出字号控件、主题切换入口、
// 静态资源引用以及 FOUC 脚本。
func TestViewPageRendering(t *testing.T) {
	store, err := content.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	cfg := config.Config{AdminPassword: "testpass"}
	srv, err := New(cfg, store, "../../templates")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	entry, err := store.Create(content.RendererMarkdown, "# Hello\n\nbody text", "")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+entry.Slug, nil)
	req.SetPathValue("slug", entry.Slug)
	w := httptest.NewRecorder()
	srv.showEntry(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()

	for _, want := range []string{
		`id="font-down"`,            // 字号减
		`id="font-up"`,              // 字号加
		`data-theme-toggle`,         // 主题切换按钮（theme.js 接管）
		`href="/-/static/base.css"`, // 公共 CSS
		`src="/-/static/theme.js"`,  // 公共 JS
		`minisnap.theme`,            // FOUC 主题脚本
		`minisnap.font`,             // FOUC 字号脚本
		`data-font="`,               // data-font 默认值
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in view page output", want)
		}
	}
}

// TestViewPageCanEditForAuthenticated 验证已登录访客看到编辑入口，匿名访客看不到。
func TestViewPageCanEditForAuthenticated(t *testing.T) {
	store, err := content.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	cfg := config.Config{AdminPassword: "testpass"}
	srv, err := New(cfg, store, "../../templates")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	entry, err := store.Create(content.RendererMarkdown, "# Hi", "")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	// 匿名访客：无编辑入口
	req := httptest.NewRequest(http.MethodGet, "/"+entry.Slug, nil)
	req.SetPathValue("slug", entry.Slug)
	w := httptest.NewRecorder()
	srv.showEntry(w, req)
	if strings.Contains(w.Body.String(), "CanEdit") || strings.Contains(w.Body.String(), `href="/`+entry.Slug+`/edit"`) {
		t.Errorf("anonymous visitor should not see edit link")
	}

	// 已登录访客：有编辑入口。通过真实登录设置 session cookie。
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=testpass"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginW := httptest.NewRecorder()
	srv.handleLogin(loginW, loginReq)
	cookie := loginW.Result().Cookies()
	if len(cookie) == 0 {
		t.Fatalf("expected session cookie after login")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/"+entry.Slug, nil)
	req2.AddCookie(cookie[0])
	req2.SetPathValue("slug", entry.Slug)
	w2 := httptest.NewRecorder()
	srv.showEntry(w2, req2)
	if !strings.Contains(w2.Body.String(), `href="/`+entry.Slug+`/edit"`) {
		t.Errorf("authenticated visitor should see edit link, body: %s", w2.Body.String()[:min(200, len(w2.Body.String()))])
	}
}
