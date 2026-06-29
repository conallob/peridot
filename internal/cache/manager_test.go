package cache

import (
	"os"
	"path/filepath"
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func newTestManager(t *testing.T, opts Options) *Manager {
	t.Helper()
	if opts.Dir == "" {
		opts.Dir = t.TempDir()
	}
	m, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = m.Close() })
	return m
}

func fakeMeta(id string, sizeBytes int64) *pb.WallpaperMetadata {
	return &pb.WallpaperMetadata{
		Id:            id,
		SourceId:      "src",
		Title:         id,
		LocalPath:     "",
		FileSizeBytes: sizeBytes,
	}
}

func TestManagerPutGetHas(t *testing.T) {
	m := newTestManager(t, Options{})

	meta := fakeMeta("w1", 1000)
	if err := m.Put(meta); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if !m.Has("w1") {
		t.Fatal("Has returned false after Put")
	}
	got, err := m.Get("w1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Id != "w1" {
		t.Fatalf("Get id = %s, want w1", got.Id)
	}

	if m.Has("nonexistent") {
		t.Fatal("Has returned true for nonexistent id")
	}
}

func TestManagerStatus(t *testing.T) {
	m := newTestManager(t, Options{MaxBytes: 10_000, MaxItems: 100, Policy: pb.EvictionPolicy_EVICTION_POLICY_LRU})

	if err := m.Put(fakeMeta("a", 500)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := m.Put(fakeMeta("b", 300)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	st, err := m.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.ItemCount != 2 {
		t.Fatalf("ItemCount = %d, want 2", st.ItemCount)
	}
	if st.CurrentSizeBytes != 800 {
		t.Fatalf("CurrentSizeBytes = %d, want 800", st.CurrentSizeBytes)
	}
	if st.MaxSizeBytes != 10_000 {
		t.Fatalf("MaxSizeBytes = %d, want 10000", st.MaxSizeBytes)
	}
}

func TestManagerEvictsByItemLimit(t *testing.T) {
	m := newTestManager(t, Options{MaxItems: 3, Policy: pb.EvictionPolicy_EVICTION_POLICY_LRU})

	for _, id := range []string{"a", "b", "c", "d"} {
		if err := m.Put(fakeMeta(id, 100)); err != nil {
			t.Fatalf("Put %s: %v", id, err)
		}
	}

	st, err := m.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.ItemCount > 3 {
		t.Fatalf("ItemCount = %d after limit 3 exceeded", st.ItemCount)
	}
}

func TestManagerEvictsByByteLimit(t *testing.T) {
	m := newTestManager(t, Options{MaxBytes: 500, Policy: pb.EvictionPolicy_EVICTION_POLICY_FIFO})

	for _, id := range []string{"x1", "x2", "x3"} {
		if err := m.Put(fakeMeta(id, 200)); err != nil {
			t.Fatalf("Put %s: %v", id, err)
		}
	}

	st, err := m.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st.CurrentSizeBytes > 500 {
		t.Fatalf("cache size %d exceeds limit 500", st.CurrentSizeBytes)
	}
}

func TestManagerEvictsFileOnDisk(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, Options{Dir: dir, MaxItems: 1, Policy: pb.EvictionPolicy_EVICTION_POLICY_LRU})

	// Create a real file that should be deleted on eviction.
	imgPath := filepath.Join(dir, "img_a.jpg")
	if err := os.WriteFile(imgPath, []byte("fake-image"), 0o644); err != nil {
		t.Fatalf("write img: %v", err)
	}
	metaA := fakeMeta("evict-a", 100)
	metaA.LocalPath = imgPath
	if err := m.Put(metaA); err != nil {
		t.Fatalf("Put a: %v", err)
	}

	// Put a second item; the first should be evicted and its file removed.
	if err := m.Put(fakeMeta("evict-b", 100)); err != nil {
		t.Fatalf("Put b: %v", err)
	}

	if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
		t.Fatal("evicted file should have been deleted from disk")
	}
}

func TestManagerFileSizeAutoDetect(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, Options{Dir: dir})

	imgPath := filepath.Join(dir, "auto.jpg")
	content := []byte("12345678901234567890") // 20 bytes
	if err := os.WriteFile(imgPath, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	meta := &pb.WallpaperMetadata{Id: "auto", SourceId: "s", LocalPath: imgPath}
	if err := m.Put(meta); err != nil {
		t.Fatalf("Put: %v", err)
	}

	st, _ := m.Status()
	if st.CurrentSizeBytes != 20 {
		t.Fatalf("auto-detected size = %d, want 20", st.CurrentSizeBytes)
	}
}

func TestManagerPutUpdatesExisting(t *testing.T) {
	m := newTestManager(t, Options{})

	if err := m.Put(fakeMeta("dup", 100)); err != nil {
		t.Fatalf("Put 1: %v", err)
	}
	updated := fakeMeta("dup", 200)
	updated.Title = "Updated"
	if err := m.Put(updated); err != nil {
		t.Fatalf("Put 2: %v", err)
	}

	got, err := m.Get("dup")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Updated" {
		t.Fatalf("title = %q, want Updated", got.Title)
	}

	st, _ := m.Status()
	if st.ItemCount != 1 {
		t.Fatalf("ItemCount = %d after upsert, want 1", st.ItemCount)
	}
}

func TestManagerLog(t *testing.T) {
	m := newTestManager(t, Options{})
	if m.Log() == nil {
		t.Fatal("Log() should not be nil")
	}
}

func TestManagerDir(t *testing.T) {
	dir := t.TempDir()
	m := newTestManager(t, Options{Dir: dir})
	if m.Dir() != dir {
		t.Fatalf("Dir() = %q, want %q", m.Dir(), dir)
	}
}
