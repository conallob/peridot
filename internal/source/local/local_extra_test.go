package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestID(t *testing.T) {
	dir := t.TempDir()
	s := New("my-id", "My Local", dir, false)
	if s.ID() != "my-id" {
		t.Fatalf("ID = %q", s.ID())
	}
}

func TestType(t *testing.T) {
	dir := t.TempDir()
	s := New("id", "name", dir, false)
	if s.Type() == 0 {
		t.Fatal("Type should be non-zero")
	}
}

func TestNewSince(t *testing.T) {
	dir := t.TempDir()
	writeImg(t, dir, "wall.jpg")

	s := New("local", "Local", dir, false)
	all, _ := s.Catalogue(context.Background())
	if len(all) == 0 {
		t.Fatal("expected at least one item")
	}

	// Anything from the past should include our file.
	got, err := s.NewSince(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("NewSince: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("NewSince(epoch) should return all items")
	}

	// Far-future cutoff returns nothing.
	got, err = s.NewSince(context.Background(), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("NewSince future: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("NewSince(future) = %d items, want 0", len(got))
	}
}

func writeImg(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestCatalogueNonExistentDir(t *testing.T) {
	s := New("id", "name", "/no/such/dir", false)
	items, err := s.Catalogue(context.Background())
	// Should error or return empty — just should not panic.
	_ = items
	_ = err
}

func TestFetchAlreadyCached(t *testing.T) {
	dir := t.TempDir()
	img := writeImg(t, dir, "img.jpg")
	s := New("id", "name", dir, false)

	items, err := s.Catalogue(context.Background())
	if err != nil || len(items) == 0 {
		t.Fatalf("Catalogue: %v items=%d", err, len(items))
	}
	items[0].LocalPath = img

	got, err := s.Fetch(context.Background(), items[0])
	if err != nil {
		t.Fatalf("Fetch cached: %v", err)
	}
	if got != img {
		t.Fatalf("Fetch = %q, want %q", got, img)
	}
}
