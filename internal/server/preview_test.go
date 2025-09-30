package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"minisnap/internal/config"
	"minisnap/internal/content"
)

func TestPreviewEntry(t *testing.T) {
	// Create a temporary content store
	store, err := content.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create server instance
	cfg := config.Config{AdminPassword: "testpass"}
	server, err := New(cfg, store, "../../templates")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	tests := []struct {
		name        string
		renderer    string
		content     string
		expectError bool
		expectHTML  bool
	}{
		{
			name:        "markdown preview",
			renderer:    "markdown",
			content:     "# Test\nThis is **bold**",
			expectError: false,
			expectHTML:  true,
		},
		{
			name:        "html preview",
			renderer:    "html",
			content:     "<h1>Test</h1><p>This is <strong>bold</strong></p>",
			expectError: false,
			expectHTML:  true,
		},
		{
			name:        "invalid renderer",
			renderer:    "invalid",
			content:     "test content",
			expectError: true,
			expectHTML:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create form data
			formData := url.Values{}
			formData.Set("renderer", tt.renderer)
			formData.Set("content", tt.content)

			// Create request
			req := httptest.NewRequest("POST", "/admin/preview", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Mock authentication by setting session
			server.setSession(httptest.NewRecorder())

			// Create response recorder
			w := httptest.NewRecorder()

			// Call the handler
			server.previewEntry(w, req)

			if tt.expectError {
				if w.Code == http.StatusOK {
					t.Errorf("expected error response, got status %d", w.Code)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", w.Code)
				}

				if tt.expectHTML {
					body := w.Body.String()
					if !strings.Contains(body, "Preview mode") {
						t.Error("expected preview template with 'Preview mode' text")
					}
					if !strings.Contains(body, "Content is not saved yet") {
						t.Error("expected preview template with 'Content is not saved yet' text")
					}
				}
			}
		})
	}
}
