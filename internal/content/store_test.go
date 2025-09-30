package content

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCreateAndGet(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	entry, err := store.Create(RendererMarkdown, "# Hello", "Greeting entry")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	if entry.Slug == "" {
		t.Fatalf("expected slug to be generated")
	}

	path := filepath.Join(root, entry.Slug+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}

	loaded, err := store.Get(entry.Slug)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}

	if loaded.Raw != "# Hello" {
		t.Fatalf("expected raw content to match")
	}
	if loaded.Description != "Greeting entry" {
		t.Fatalf("expected description to persist")
	}
}

func TestStoreUpdate(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	entry, err := store.Create(RendererMarkdown, "initial", "first")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	updated, err := store.Update(entry.Slug, RendererHTML, "<h1>Updated</h1>", "updated")
	if err != nil {
		t.Fatalf("update entry: %v", err)
	}

	if updated.Renderer != RendererHTML {
		t.Fatalf("expected renderer to update")
	}

	if updated.Raw != "<h1>Updated</h1>" {
		t.Fatalf("expected raw content to update")
	}
	if updated.Description != "updated" {
		t.Fatalf("expected description to update")
	}
}

func TestStoreList(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first, err := store.Create(RendererMarkdown, "first", "alpha")
	if err != nil {
		t.Fatalf("create first entry: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	second, err := store.Create(RendererHTML, "<p>second</p>", "beta")
	if err != nil {
		t.Fatalf("create second entry: %v", err)
	}

	entries, err := store.List()
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].Slug != second.Slug {
		t.Fatalf("expected most recent entry first")
	}

	found := map[string]bool{}
	for _, entry := range entries {
		found[entry.Slug] = true
	}
	if !found[first.Slug] || !found[second.Slug] {
		t.Fatalf("expected both slugs to be present")
	}
}

func TestStoreDelete(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	entry, err := store.Create(RendererMarkdown, "hello", "desc")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	if err := store.Delete(entry.Slug); err != nil {
		t.Fatalf("delete entry: %v", err)
	}

	if _, err := store.Get(entry.Slug); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("expected get to return ErrEntryNotFound, got %v", err)
	}
}
