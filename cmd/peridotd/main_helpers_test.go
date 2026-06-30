package main

import (
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func TestLoadConfig_Missing(t *testing.T) {
	cfg := loadConfig("/no/such/path/config.toml")
	if cfg == nil {
		t.Fatal("loadConfig should return defaults on missing file")
	}
	if cfg.Scheduler == nil {
		t.Fatal("default config should have Scheduler")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Scheduler == nil || cfg.Cache == nil {
		t.Fatal("defaultConfig missing Scheduler or Cache")
	}
}

func TestCacheMaxBytes(t *testing.T) {
	if cacheMaxBytes(&pb.Config{}) != 0 {
		t.Fatal("nil Cache should return 0")
	}
	cfg := &pb.Config{Cache: &pb.CacheConfig{MaxSizeBytes: 1 << 30}}
	if cacheMaxBytes(cfg) != 1<<30 {
		t.Fatalf("cacheMaxBytes = %d", cacheMaxBytes(cfg))
	}
}

func TestCacheMaxItems(t *testing.T) {
	if cacheMaxItems(&pb.Config{}) != 0 {
		t.Fatal("nil Cache should return 0")
	}
	cfg := &pb.Config{Cache: &pb.CacheConfig{MaxItems: 42}}
	if cacheMaxItems(cfg) != 42 {
		t.Fatalf("cacheMaxItems = %d", cacheMaxItems(cfg))
	}
}

func TestCachePolicy(t *testing.T) {
	if cachePolicy(&pb.Config{}) != pb.EvictionPolicy_EVICTION_POLICY_LRU {
		t.Fatal("nil Cache should default to LRU")
	}
	cfg := &pb.Config{Cache: &pb.CacheConfig{EvictionPolicy: pb.EvictionPolicy_EVICTION_POLICY_FIFO}}
	if cachePolicy(cfg) != pb.EvictionPolicy_EVICTION_POLICY_FIFO {
		t.Fatalf("cachePolicy = %v", cachePolicy(cfg))
	}
}
