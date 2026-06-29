package url

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func newTestSource(t *testing.T, srv *httptest.Server, headers map[string]string) *Source {
	t.Helper()
	dir := t.TempDir()
	s := New("test-id", "Test Source", srv.URL, headers, dir)
	s.http = srv.Client()
	return s
}

func TestNew_DefaultDisplayName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	dir := t.TempDir()
	s := New("id", "", srv.URL, nil, dir)
	if !strings.HasPrefix(s.DisplayName(), "URL:") {
		t.Fatalf("DisplayName = %q, want URL: prefix", s.DisplayName())
	}
}

func TestIdentity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	s := newTestSource(t, srv, nil)

	if s.ID() != "test-id" {
		t.Fatalf("ID = %q", s.ID())
	}
	if s.DisplayName() != "Test Source" {
		t.Fatalf("DisplayName = %q", s.DisplayName())
	}
}

func TestCatalogue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	s := newTestSource(t, srv, nil)

	items, err := s.Catalogue(context.Background())
	if err != nil {
		t.Fatalf("Catalogue: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len = %d, want 1", len(items))
	}
	if items[0].SourceId != "test-id" {
		t.Fatalf("SourceId = %q", items[0].SourceId)
	}
}

func TestNewSince_AlwaysReturnsOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	s := newTestSource(t, srv, nil)

	items, err := s.NewSince(context.Background(), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("NewSince: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("NewSince len = %d, want 1 (always re-fetches)", len(items))
	}
}

func TestFetch_Success(t *testing.T) {
	body := []byte("fake-image-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	s := newTestSource(t, srv, nil)

	items, _ := s.Catalogue(context.Background())
	path, err := s.Fetch(context.Background(), items[0])
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if path == "" {
		t.Fatal("Fetch returned empty path")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("file content mismatch")
	}
}

func TestFetch_ForwardsHeaders(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Api-Key")
		_, _ = w.Write([]byte("img"))
	}))
	defer srv.Close()

	s := newTestSource(t, srv, map[string]string{"X-Api-Key": "secret"})
	items, _ := s.Catalogue(context.Background())
	_, err := s.Fetch(context.Background(), items[0])
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if gotHeader != "secret" {
		t.Fatalf("header = %q, want secret", gotHeader)
	}
}

func TestFetch_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	s := newTestSource(t, srv, nil)

	items, _ := s.Catalogue(context.Background())
	_, err := s.Fetch(context.Background(), items[0])
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v, want 403 mention", err)
	}
}

func TestHealthy_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	s := newTestSource(t, srv, nil)

	if !s.Healthy(context.Background()) {
		t.Fatal("Healthy should return true for 200")
	}
}

func TestHealthy_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	s := newTestSource(t, srv, nil)

	if s.Healthy(context.Background()) {
		t.Fatal("Healthy should return false for 500")
	}
}

func TestHealthy_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	dir := t.TempDir()
	s := New("id", "name", srv.URL, nil, dir)
	srv.Close() // closed before we call Healthy

	if s.Healthy(context.Background()) {
		t.Fatal("Healthy should return false on network error")
	}
}

func TestIDFor_Deterministic(t *testing.T) {
	a := idFor("src", "https://example.com")
	b := idFor("src", "https://example.com")
	if a != b {
		t.Fatal("idFor not deterministic")
	}
	c := idFor("src", "https://other.com")
	if a == c {
		t.Fatal("different inputs produced same id")
	}
}
