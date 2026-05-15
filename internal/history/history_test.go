package history

import (
	"os"
	"path/filepath"
	"testing"

	"btrman/internal/manual"
)

func TestStoreAddDeduplicatesAndCaps(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	store := NewStoreAt(path, 2)
	if err := store.Add(manual.PageRef{Name: "ls", Section: "1"}, "ls title"); err != nil {
		t.Fatalf("add ls: %v", err)
	}
	if err := store.Add(manual.PageRef{Name: "printf", Section: "1"}, "printf title"); err != nil {
		t.Fatalf("add printf: %v", err)
	}
	if err := store.Add(manual.PageRef{Name: "grep", Section: "1"}, "grep title"); err != nil {
		t.Fatalf("add grep: %v", err)
	}
	if err := store.Add(manual.PageRef{Name: "printf", Section: "1"}, "printf title"); err != nil {
		t.Fatalf("re-add printf: %v", err)
	}

	entries, err := store.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Ref.String() != "printf(1)" || entries[1].Ref.String() != "grep(1)" {
		t.Fatalf("unexpected order: %#v", entries)
	}
}

func TestLoadMissingHistoryReturnsEmpty(t *testing.T) {
	store := NewStoreAt(filepath.Join(t.TempDir(), "missing", "history.json"), 10)
	entries, err := store.Load()
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestSaveCreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "history.json")
	store := NewStoreAt(path, 10)
	if err := store.Save([]Entry{{Ref: manual.PageRef{Name: "man", Section: "1"}}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat saved history: %v", err)
	}
}
