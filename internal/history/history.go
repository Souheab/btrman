package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"

	"btrman/internal/manual"
)

const DefaultLimit = 50

type Entry struct {
	Ref        manual.PageRef `json:"ref"`
	Title      string         `json:"title,omitempty"`
	LastOpened time.Time      `json:"last_opened"`
}

type Store struct {
	Path  string
	Limit int
}

func NewStore() (*Store, error) {
	path, err := xdg.StateFile("btrman/history.json")
	if err != nil {
		return nil, err
	}
	return &Store{Path: path, Limit: DefaultLimit}, nil
}

func NewStoreAt(path string, limit int) *Store {
	if limit <= 0 {
		limit = DefaultLimit
	}
	return &Store{Path: path, Limit: limit}
}

func (s *Store) Entries() ([]Entry, error) {
	return s.Load()
}

func (s *Store) Load() ([]Entry, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return capEntries(entries, s.limit()), nil
}

func (s *Store) Add(ref manual.PageRef, title string) error {
	if ref.IsZero() {
		return nil
	}
	entries, err := s.Load()
	if err != nil {
		return err
	}

	entry := Entry{Ref: ref, Title: title, LastOpened: time.Now().UTC()}
	filtered := []Entry{entry}
	for _, existing := range entries {
		if existing.Ref.Key() == ref.Key() {
			continue
		}
		filtered = append(filtered, existing)
	}
	return s.Save(capEntries(filtered, s.limit()))
}

func (s *Store) Save(entries []Entry) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(capEntries(entries, s.limit()), "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.Path, data, 0o644)
}

func (s *Store) limit() int {
	if s.Limit <= 0 {
		return DefaultLimit
	}
	return s.Limit
}

func capEntries(entries []Entry, limit int) []Entry {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}
