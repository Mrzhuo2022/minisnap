package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"minisnap/internal/config"
	"minisnap/internal/content"
)

// TestStaticAssetsServed 验证 /-/static/ 路由能提供 base.css 和 theme.js，
// 且不会被 GET /{slug} 抢占。
func TestStaticAssetsServed(t *testing.T) {
	store, err := content.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	cfg := config.Config{AdminPassword: "testpass"}
	srv, err := New(cfg, store, "../../templates")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	cases := []struct {
		path        string
		wantStatus  int
		wantContent string
	}{
		{"/-/static/base.css", http.StatusOK, "--accent"},
		{"/-/static/theme.js", http.StatusOK, "minisnap.theme"},
		{"/-/static/missing.css", http.StatusNotFound, ""},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != c.wantStatus {
			t.Errorf("%s: status = %d, want %d", c.path, w.Code, c.wantStatus)
			continue
		}
		if c.wantContent != "" && !strings.Contains(w.Body.String(), c.wantContent) {
			t.Errorf("%s: body missing %q", c.path, c.wantContent)
		}
	}
}

// TestStaticRouteDoesNotShadowSlug 验证多段静态路径不与单段 slug 路由冲突。
func TestStaticRouteDoesNotShadowSlug(t *testing.T) {
	store, err := content.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	cfg := config.Config{AdminPassword: "testpass"}
	srv, err := New(cfg, store, "../../templates")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	entry, err := store.Create(content.RendererMarkdown, "# x", "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// 真实 slug 仍能访问
	req := httptest.NewRequest(http.MethodGet, "/"+entry.Slug, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("slug route: status = %d, want 200", w.Code)
	}
}
