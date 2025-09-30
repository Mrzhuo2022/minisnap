package server

import (
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"minisnap/internal/config"
	"minisnap/internal/content"
)

// Server 负责注册 HTTP 路由并处理请求。
type Server struct {
	cfg       config.Config
	store     *content.Store
	mux       *http.ServeMux
	templates *template.Template
	sessions  *sessionStore
}

type entryListItem struct {
	Slug        string
	Renderer    content.RendererType
	Description string
	PublishedAt string
	UpdatedAt   string
	WasUpdated  bool
}

type adminTemplateData struct {
	Title        string
	Action       string
	Content      string
	Renderer     content.RendererType
	Description  string
	PublishedAt  string
	UpdatedAt    string
	SelectedSlug string
}

type libraryTemplateData struct {
	Title         string
	Entries       []entryListItem
	SearchTerm    string
	TotalEntries  int
	FilteredCount int
	HasFilter     bool
}

// New 创建一个 Server 并加载模板。
func New(cfg config.Config, store *content.Store, tplDir string) (*Server, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}

	tpls, err := template.ParseGlob(filepath.Join(tplDir, "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	s := &Server{
		cfg:       cfg,
		store:     store,
		templates: tpls,
		mux:       http.NewServeMux(),
		sessions:  newSessionStore(),
	}
	s.registerRoutes()
	return s, nil
}

// ServeHTTP 实现 http.Handler。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /", s.redirectAdmin)
	s.mux.HandleFunc("GET /healthz", s.health)

	s.mux.HandleFunc("GET /login", s.showLogin)
	s.mux.HandleFunc("POST /login", s.handleLogin)
	s.mux.HandleFunc("POST /logout", s.requireAuth(s.handleLogout))

	s.mux.HandleFunc("GET /admin/library", s.requireAuth(s.showLibrary))
	s.mux.HandleFunc("GET /admin", s.requireAuth(s.showEditor))
	s.mux.HandleFunc("POST /admin", s.requireAuth(s.createEntry))
	s.mux.HandleFunc("POST /admin/preview", s.requireAuth(s.previewEntry))

	s.mux.HandleFunc("GET /{slug}/edit", s.requireAuth(s.showEdit))
	s.mux.HandleFunc("POST /{slug}/edit", s.requireAuth(s.updateEntry))
	s.mux.HandleFunc("POST /{slug}/delete", s.requireAuth(s.deleteEntry))

	s.mux.HandleFunc("GET /{slug}", s.showEntry)
}

func (s *Server) redirectAdmin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.authenticated(r); !ok {
			nextURL := url.QueryEscape(r.URL.RequestURI())
			http.Redirect(w, r, "/login?next="+nextURL, http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *Server) authenticated(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	if !s.sessions.Validate(cookie.Value) {
		return "", false
	}
	return cookie.Value, true
}

func (s *Server) setSession(w http.ResponseWriter) {
	token, expires := s.sessions.Create()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  expires,
	})
}

func (s *Server) clearSession(w http.ResponseWriter, token string) {
	s.sessions.Remove(token)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})
}

func (s *Server) showLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authenticated(r); ok {
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}

	next := r.URL.Query().Get("next")
	s.renderTemplate(w, "login.tmpl", map[string]any{
		"Title": "Login",
		"Next":  next,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderTemplate(w, "login.tmpl", map[string]any{
			"Title": "Login",
			"Error": "Invalid form data",
			"Next":  r.FormValue("next"),
		})
		return
	}

	password := r.FormValue("password")
	if password != s.cfg.AdminPassword {
		s.renderTemplate(w, "login.tmpl", map[string]any{
			"Title": "Login",
			"Error": "Incorrect password",
			"Next":  r.FormValue("next"),
		})
		return
	}

	s.setSession(w)
	next := r.FormValue("next")
	if next == "" {
		next = "/admin"
	} else if !strings.HasPrefix(next, "/") {
		next = "/admin"
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token, ok := s.authenticated(r)
	if ok {
		s.clearSession(w, token)
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) showEditor(w http.ResponseWriter, r *http.Request) {
	data := s.buildEditorData(nil)
	s.renderTemplate(w, "admin.tmpl", data)
}

func (s *Server) createEntry(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	renderer := content.RendererType(r.FormValue("renderer"))
	raw := r.FormValue("content")
	description := r.FormValue("description")

	entry, err := s.store.Create(renderer, raw, description)
	if err != nil {
		slog.Error("create entry", "error", err)
		s.renderError(w, http.StatusBadRequest, err.Error())
		return
	}

	viewURL := fmt.Sprintf("/%s", entry.Slug)
	editURL := fmt.Sprintf("/%s/edit", entry.Slug)
	wasUpdated := !entry.UpdatedAt.IsZero() && !entry.UpdatedAt.Equal(entry.CreatedAt)

	s.renderTemplate(w, "saved.tmpl", map[string]any{
		"Title":       "Entry Saved",
		"ViewURL":     viewURL,
		"EditURL":     editURL,
		"PublishedAt": formatTime(entry.CreatedAt),
		"UpdatedAt":   formatTime(entry.UpdatedAt),
		"WasUpdated":  wasUpdated,
	})
}

func (s *Server) previewEntry(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	renderer := content.RendererType(r.FormValue("renderer"))
	raw := r.FormValue("content")

	// Validate renderer type
	if renderer != content.RendererMarkdown && renderer != content.RendererHTML {
		s.renderError(w, http.StatusBadRequest, "Invalid renderer type")
		return
	}

	// Create a temporary entry for preview
	tempEntry := content.Entry{
		Renderer: renderer,
		Raw:      raw,
	}

	html, err := content.RenderHTML(tempEntry)
	if err != nil {
		slog.Error("render preview", "error", err)
		s.renderError(w, http.StatusInternalServerError, "Render Failed")
		return
	}

	s.renderTemplate(w, "preview.tmpl", map[string]any{
		"Title":            "Preview",
		"HTML":             html,
		"GeneratedAt":      formatTime(time.Now()),
		"AllowThemeSwitch": renderer == content.RendererMarkdown,
	})
}

func (s *Server) showEntry(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	entry, err := s.store.Get(slug)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Not Found")
		return
	}

	html, err := content.RenderHTML(entry)
	if err != nil {
		slog.Error("render entry", "slug", slug, "error", err)
		s.renderError(w, http.StatusInternalServerError, "Render Failed")
		return
	}

	s.renderTemplate(w, "view.tmpl", map[string]any{
		"Title":            entry.Slug,
		"HTML":             html,
		"PublishedAt":      formatTime(entry.CreatedAt),
		"UpdatedAt":        formatTime(entry.UpdatedAt),
		"WasUpdated":       !entry.UpdatedAt.IsZero() && !entry.UpdatedAt.Equal(entry.CreatedAt),
		"AllowThemeSwitch": entry.Renderer == content.RendererMarkdown,
	})
}

func (s *Server) showEdit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	entry, err := s.store.Get(slug)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "Not Found")
		return
	}

	data := s.buildEditorData(&entry)
	s.renderTemplate(w, "admin.tmpl", data)
}

func (s *Server) updateEntry(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := r.ParseForm(); err != nil {
		s.renderError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	renderer := content.RendererType(r.FormValue("renderer"))
	raw := r.FormValue("content")
	description := r.FormValue("description")

	entry, err := s.store.Update(slug, renderer, raw, description)
	if err != nil {
		slog.Error("update entry", "slug", slug, "error", err)
		s.renderError(w, http.StatusBadRequest, err.Error())
		return
	}

	viewURL := fmt.Sprintf("/%s", entry.Slug)
	wasUpdated := !entry.UpdatedAt.IsZero() && !entry.UpdatedAt.Equal(entry.CreatedAt)
	s.renderTemplate(w, "saved.tmpl", map[string]any{
		"Title":       "Entry Updated",
		"ViewURL":     viewURL,
		"EditURL":     fmt.Sprintf("/%s/edit", entry.Slug),
		"PublishedAt": formatTime(entry.CreatedAt),
		"UpdatedAt":   formatTime(entry.UpdatedAt),
		"WasUpdated":  wasUpdated,
	})
}

func (s *Server) deleteEntry(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		s.renderError(w, http.StatusBadRequest, "Invalid slug")
		return
	}

	if err := s.store.Delete(slug); err != nil {
		if errors.Is(err, content.ErrEntryNotFound) {
			s.renderError(w, http.StatusNotFound, "Not Found")
			return
		}
		slog.Error("delete entry", "slug", slug, "error", err)
		s.renderError(w, http.StatusInternalServerError, "Delete Failed")
		return
	}

	http.Redirect(w, r, "/admin/library", http.StatusFound)
}

func (s *Server) showLibrary(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	items, total, err := s.buildEntryList(search)
	if err != nil {
		slog.Error("list entries", "error", err)
		s.renderError(w, http.StatusInternalServerError, "Failed to load entries")
		return
	}

	s.renderTemplate(w, "library.tmpl", libraryTemplateData{
		Title:         "Content Library",
		Entries:       items,
		SearchTerm:    search,
		TotalEntries:  total,
		FilteredCount: len(items),
		HasFilter:     search != "",
	})
}

func (s *Server) buildEditorData(entry *content.Entry) adminTemplateData {
	data := adminTemplateData{
		Title:    "Create New Entry",
		Action:   "/admin",
		Renderer: content.RendererMarkdown,
	}

	if entry == nil {
		return data
	}

	data.Title = fmt.Sprintf("Edit %s", entry.Slug)
	data.Action = fmt.Sprintf("/%s/edit", entry.Slug)
	data.Content = entry.Raw
	data.Renderer = entry.Renderer
	data.Description = entry.Description
	data.PublishedAt = formatTime(entry.CreatedAt)
	data.UpdatedAt = formatTime(entry.UpdatedAt)
	data.SelectedSlug = entry.Slug
	return data
}

func (s *Server) buildEntryList(searchTerm string) ([]entryListItem, int, error) {
	entries, err := s.store.List()
	if err != nil {
		return nil, 0, err
	}

	total := len(entries)
	search := strings.ToLower(strings.TrimSpace(searchTerm))
	items := make([]entryListItem, 0, total)
	for _, entry := range entries {
		if search != "" {
			if !strings.Contains(strings.ToLower(entry.Slug), search) &&
				!strings.Contains(strings.ToLower(entry.Raw), search) &&
				!strings.Contains(strings.ToLower(entry.Description), search) {
				continue
			}
		}
		description := strings.TrimSpace(entry.Description)
		if description == "" {
			description = summarize(entry.Raw, 140)
		}
		items = append(items, entryListItem{
			Slug:        entry.Slug,
			Renderer:    entry.Renderer,
			Description: description,
			PublishedAt: formatTime(entry.CreatedAt),
			UpdatedAt:   formatTime(entry.UpdatedAt),
			WasUpdated:  !entry.UpdatedAt.IsZero() && !entry.UpdatedAt.Equal(entry.CreatedAt),
		})
	}

	return items, total, nil
}

func summarize(raw string, limit int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	collapsed := strings.Join(strings.Fields(trimmed), " ")
	runes := []rune(collapsed)
	if len(runes) <= limit {
		return collapsed
	}
	return string(runes[:limit]) + "…"
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("render template", "name", name, "error", err)
		http.Error(w, "Template Error", http.StatusInternalServerError)
	}
}

func (s *Server) renderError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}
