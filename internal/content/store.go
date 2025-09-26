package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"minisnap/internal/slug"
)

var ErrEntryNotFound = errors.New("entry not found")

// RendererType 表示内容渲染器。
type RendererType string

const (
	RendererMarkdown RendererType = "markdown"
	RendererHTML     RendererType = "html"
)

// Entry 表示存储在磁盘上的一篇内容。
type Entry struct {
	Slug        string       `json:"slug"`
	Renderer    RendererType `json:"renderer"`
	Raw         string       `json:"raw"`
	Description string       `json:"description,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Store 负责将 Entry 持久化到文件系统。
type Store struct {
	root string
	mu   sync.RWMutex
}

// NewStore 创建一个指向指定目录的 Store，目录不存在会自动创建。
func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("content root cannot be empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create content dir: %w", err)
	}
	return &Store{root: root}, nil
}

// Create 新建一篇内容并返回持久化后的 Entry。
func (s *Store) Create(renderer RendererType, raw string, description string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateRenderer(renderer); err != nil {
		return Entry{}, err
	}
	description = strings.TrimSpace(description)

	var slugID string
	for i := 0; i < 5; i++ {
		candidate := slug.New()
		if _, err := os.Stat(s.entryPath(candidate)); errors.Is(err, os.ErrNotExist) {
			slugID = candidate
			break
		}
	}
	if slugID == "" {
		return Entry{}, errors.New("unable to allocate unique slug")
	}

	entry := Entry{
		Slug:        slugID,
		Renderer:    renderer,
		Raw:         raw,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.persist(entry); err != nil {
		return Entry{}, err
	}

	return entry, nil
}

// Update 覆盖现有内容。
func (s *Store) Update(slugID string, renderer RendererType, raw string, description string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateRenderer(renderer); err != nil {
		return Entry{}, err
	}
	description = strings.TrimSpace(description)

	existing, err := s.read(slugID)
	if err != nil {
		return Entry{}, err
	}

	existing.Renderer = renderer
	existing.Raw = raw
	existing.Description = description
	existing.UpdatedAt = time.Now().UTC()

	if err := s.persist(existing); err != nil {
		return Entry{}, err
	}

	return existing, nil
}

// Get 读取指定 slug 的内容。
func (s *Store) Get(slugID string) (Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.read(slugID)
}

// List 返回所有内容，按创建时间倒序排列。
func (s *Store) List() ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := os.ReadDir(s.root)
	if err != nil {
		return nil, fmt.Errorf("read content dir: %w", err)
	}

	entries := make([]Entry, 0, len(files))
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if filepath.Ext(f.Name()) != ".json" {
			continue
		}
		slugID := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		entry, err := s.read(slugID)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].CreatedAt.Equal(entries[j].CreatedAt) {
			return entries[i].Slug > entries[j].Slug
		}
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	return entries, nil
}

func (s *Store) entryPath(slugID string) string {
	return filepath.Join(s.root, fmt.Sprintf("%s.json", slugID))
}

func (s *Store) persist(entry Entry) error {
	path := s.entryPath(entry.Slug)
	tmpPath := path + ".tmp"

	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&entry); err != nil {
		_ = file.Close()
		return fmt.Errorf("encode entry: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func (s *Store) read(slugID string) (Entry, error) {
	path := s.entryPath(slugID)
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Entry{}, ErrEntryNotFound
		}
		return Entry{}, fmt.Errorf("open entry: %w", err)
	}
	defer file.Close()

	var entry Entry
	if err := json.NewDecoder(file).Decode(&entry); err != nil {
		return Entry{}, fmt.Errorf("decode entry: %w", err)
	}
	return entry, nil
}

// Delete 移除指定 slug 的内容。
func (s *Store) Delete(slugID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.entryPath(slugID)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrEntryNotFound
		}
		return fmt.Errorf("delete entry: %w", err)
	}
	return nil
}

func validateRenderer(renderer RendererType) error {
	switch renderer {
	case RendererMarkdown, RendererHTML:
		return nil
	default:
		return fmt.Errorf("unsupported renderer: %s", renderer)
	}
}
