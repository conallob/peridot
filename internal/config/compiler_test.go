package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/protobuf/proto"
)

const validTOML = `
[scheduler]
interval = "30m"
shuffle = true
change_on = ["timer", "wake"]

[cache]
max_size = "2GB"
max_items = 500
eviction_policy = "lru"

[[sources]]
id = "urlsrc"
type = "url"
display_name = "URL Source"
weight = 1.0
endpoint = "https://example.com/random"
`

func writeTOML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	return path
}

func TestCompileValid(t *testing.T) {
	path := writeTOML(t, validTOML)

	cfg, err := Compile(path)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if cfg.Scheduler.Interval.AsDuration().Minutes() != 30 {
		t.Fatalf("interval = %v, want 30m", cfg.Scheduler.Interval.AsDuration())
	}
	if cfg.Cache.MaxSizeBytes != 2<<30 {
		t.Fatalf("max size = %d, want %d", cfg.Cache.MaxSizeBytes, 2<<30)
	}
	if cfg.Cache.EvictionPolicy != pb.EvictionPolicy_EVICTION_POLICY_LRU {
		t.Fatalf("eviction = %v, want LRU", cfg.Cache.EvictionPolicy)
	}
	if len(cfg.Sources) != 1 || cfg.Sources[0].Id != "urlsrc" {
		t.Fatalf("unexpected sources: %v", cfg.Sources)
	}

	// Lock file should have been written and should round-trip.
	lockPath := path + ".lock.pb"
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	loaded := &pb.Config{}
	if err := proto.Unmarshal(data, loaded); err != nil {
		t.Fatalf("unmarshal lock: %v", err)
	}
	if !proto.Equal(cfg, loaded) {
		t.Fatalf("lock mismatch")
	}

	via, err := LoadLock(lockPath)
	if err != nil {
		t.Fatalf("LoadLock: %v", err)
	}
	if !proto.Equal(cfg, via) {
		t.Fatalf("LoadLock mismatch")
	}
}

func TestCompileBadTOML(t *testing.T) {
	path := writeTOML(t, "this is = = not valid toml [[[")
	if _, err := Compile(path); err == nil {
		t.Fatal("expected parse error")
	} else if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileMissingInterval(t *testing.T) {
	toml := `
[cache]
max_size = "1GB"

[[sources]]
id = "s"
type = "url"
endpoint = "https://example.com"
`
	path := writeTOML(t, toml)
	if _, err := Compile(path); err == nil {
		t.Fatal("expected validation error for missing interval")
	} else if !strings.Contains(err.Error(), "validate") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileUnknownSourceType(t *testing.T) {
	toml := `
[scheduler]
interval = "30m"

[[sources]]
id = "s"
type = "nope"
`
	path := writeTOML(t, toml)
	if _, err := Compile(path); err == nil {
		t.Fatal("expected error for unknown source type")
	}
}

func TestValidateNoLockFile(t *testing.T) {
	path := writeTOML(t, validTOML)
	if _, err := Validate(path); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if _, err := os.Stat(path + ".lock.pb"); !os.IsNotExist(err) {
		t.Fatal("Validate should not write a lock file")
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{
		"1024":  1024,
		"1KB":   1 << 10,
		"500MB": 500 << 20,
		"2GB":   2 << 30,
		"1B":    1,
	}
	for in, want := range cases {
		got, err := parseSize(in)
		if err != nil {
			t.Errorf("parseSize(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseSize(%q) = %d, want %d", in, got, want)
		}
	}
	if _, err := parseSize("notasize"); err == nil {
		t.Error("expected error for invalid size")
	}
}
