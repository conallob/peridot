package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func metaWithPath(p string) *pb.WallpaperMetadata {
	return &pb.WallpaperMetadata{LocalPath: p}
}

func writeFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestCatalogueFindsImages(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.jpg")
	writeFile(t, dir, "b.png")
	writeFile(t, dir, "c.webp")
	writeFile(t, dir, "notes.txt")
	writeFile(t, dir, "archive.zip")
	writeFile(t, dir, "no-ext")

	// Nested image should also be found.
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, sub, "d.JPG")

	s := New("local1", "", dir, false)
	got, err := s.Catalogue(context.Background())
	if err != nil {
		t.Fatalf("Catalogue: %v", err)
	}
	if len(got) != 4 {
		names := make([]string, len(got))
		for i, m := range got {
			names[i] = m.Title
		}
		t.Fatalf("found %d images %v, want 4 (jpg, png, webp, JPG)", len(got), names)
	}
	for _, m := range got {
		if m.SourceId != "local1" {
			t.Errorf("source id = %q, want local1", m.SourceId)
		}
		if m.Id == "" {
			t.Error("empty id")
		}
		if m.LocalPath == "" {
			t.Error("empty local path")
		}
	}
}

func TestCatalogueIgnoresNonImages(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "readme.md")
	writeFile(t, dir, "data.json")

	s := New("s", "", dir, false)
	got, err := s.Catalogue(context.Background())
	if err != nil {
		t.Fatalf("Catalogue: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("found %d images, want 0", len(got))
	}
}

func TestHealthy(t *testing.T) {
	dir := t.TempDir()
	s := New("s", "", dir, false)
	if !s.Healthy(context.Background()) {
		t.Error("expected healthy for existing dir")
	}

	missing := New("s", "", filepath.Join(dir, "does-not-exist"), false)
	if missing.Healthy(context.Background()) {
		t.Error("expected unhealthy for missing dir")
	}

	// A file path (not a directory) should be unhealthy.
	writeFile(t, dir, "afile.jpg")
	fileSrc := New("s", "", filepath.Join(dir, "afile.jpg"), false)
	if fileSrc.Healthy(context.Background()) {
		t.Error("expected unhealthy for non-directory path")
	}
}

func TestDefaultDisplayName(t *testing.T) {
	s := New("id", "", "/some/path", false)
	if s.DisplayName() != "Local: /some/path" {
		t.Errorf("display name = %q", s.DisplayName())
	}
	s2 := New("id", "Custom", "/p", false)
	if s2.DisplayName() != "Custom" {
		t.Errorf("display name = %q, want Custom", s2.DisplayName())
	}
}

func TestFetchLocal(t *testing.T) {
	s := New("s", "", "/p", false)
	p, err := s.Fetch(context.Background(), metaWithPath("/p/a.jpg"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if p != "/p/a.jpg" {
		t.Errorf("fetch path = %q", p)
	}
	if _, err := s.Fetch(context.Background(), metaWithPath("")); err == nil {
		t.Error("expected error for empty local path")
	}
}
