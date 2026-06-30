package digitalblasphemy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func TestExt(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://example.com/img.jpg", ".jpg"},
		{"https://example.com/img.PNG", ".PNG"},
		{"https://example.com/img", ".jpg"},           // no ext → default
		{"https://example.com/img.toolongext", ".jpg"}, // too long → default
	}
	for _, c := range cases {
		if got := ext(c.url); got != c.want {
			t.Errorf("ext(%q) = %q, want %q", c.url, got, c.want)
		}
	}
}

func newDBSource(t *testing.T, srv *httptest.Server) *Source {
	t.Helper()
	s := New("db", "", "user", "", "1920x1080", t.TempDir(), false, 0, "pass")
	s.http = srv.Client()
	return s
}

func TestFetch_Success(t *testing.T) {
	body := []byte("fake-db-image")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	s := newDBSource(t, srv)
	m := &pb.WallpaperMetadata{
		Id:        "db:abc123",
		SourceId:  "db",
		Title:     "Night Sky",
		OriginUrl: srv.URL + "/nightsky.jpg",
	}
	path, err := s.Fetch(context.Background(), m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if path == "" {
		t.Fatal("empty path")
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(body) {
		t.Fatal("file content mismatch")
	}
}

func TestFetch_AlreadyCached(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make HTTP request for cached file")
	}))
	defer srv.Close()

	s := newDBSource(t, srv)
	// Write a fake cached file.
	cached := filepath.Join(s.cacheDir, "cached.jpg")
	if err := os.WriteFile(cached, []byte("cached"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m := &pb.WallpaperMetadata{LocalPath: cached, OriginUrl: srv.URL + "/img.jpg"}
	path, err := s.Fetch(context.Background(), m)
	if err != nil {
		t.Fatalf("Fetch cached: %v", err)
	}
	if path != cached {
		t.Fatalf("path = %q, want %q", path, cached)
	}
}

func TestFetch_MissingOriginURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	s := newDBSource(t, srv)

	_, err := s.Fetch(context.Background(), &pb.WallpaperMetadata{Id: "x"})
	if err == nil {
		t.Fatal("expected error for missing origin URL")
	}
}

func TestFetch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	s := newDBSource(t, srv)

	m := &pb.WallpaperMetadata{OriginUrl: srv.URL + "/img.jpg"}
	_, err := s.Fetch(context.Background(), m)
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected 403 error, got %v", err)
	}
}
