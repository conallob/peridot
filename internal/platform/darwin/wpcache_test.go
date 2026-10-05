package darwin

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneRenders(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	write := func(name string, age time.Duration) {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("1234"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	write("old1.bmp", 5*time.Hour)
	write("old2.bmp", 4*time.Hour)
	write("new.bmp", time.Hour)
	write("fresh.bmp", time.Second)
	write("cacheVersion.db", 9*time.Hour)

	files, bytes, err := pruneRenders(dir, 2, time.Minute, now)
	if err != nil {
		t.Fatal(err)
	}
	if files != 2 || bytes != 8 {
		t.Fatalf("removed %d files / %d bytes, want 2 / 8", files, bytes)
	}
	for _, name := range []string{"fresh.bmp", "new.bmp", "cacheVersion.db"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s should remain: %v", name, err)
		}
	}
}

func TestPruneRenders_MissingDir(t *testing.T) {
	if _, _, err := pruneRenders(filepath.Join(t.TempDir(), "nope"), 1, 0, time.Now()); err != nil {
		t.Fatalf("missing dir should be a no-op, got %v", err)
	}
}
