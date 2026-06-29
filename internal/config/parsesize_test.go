package config

import (
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func TestParseEviction(t *testing.T) {
	cases := []struct {
		in   string
		want pb.EvictionPolicy
	}{
		{"lru", pb.EvictionPolicy_EVICTION_POLICY_LRU},
		{"LRU", pb.EvictionPolicy_EVICTION_POLICY_LRU},
		{"fifo", pb.EvictionPolicy_EVICTION_POLICY_FIFO},
		{"FIFO", pb.EvictionPolicy_EVICTION_POLICY_FIFO},
		{"random", pb.EvictionPolicy_EVICTION_POLICY_RANDOM},
		{"RANDOM", pb.EvictionPolicy_EVICTION_POLICY_RANDOM},
		{"", pb.EvictionPolicy_EVICTION_POLICY_LRU},
		{"unknown", pb.EvictionPolicy_EVICTION_POLICY_LRU},
	}
	for _, c := range cases {
		if got := parseEviction(c.in); got != c.want {
			t.Errorf("parseEviction(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseSizeEdgeCases(t *testing.T) {
	// Fractional MB
	got, err := parseSize("1.5MB")
	if err != nil {
		t.Fatalf("parseSize(1.5MB): %v", err)
	}
	want := int64(1.5 * (1 << 20))
	if got != want {
		t.Fatalf("1.5MB = %d, want %d", got, want)
	}

	// Whitespace
	got, err = parseSize("  2GB  ")
	if err != nil {
		t.Fatalf("parseSize with spaces: %v", err)
	}
	if got != 2<<30 {
		t.Fatalf("2GB = %d, want %d", got, 2<<30)
	}

	// Case insensitive (already uppercased in function)
	got, err = parseSize("500mb")
	if err != nil {
		t.Fatalf("parseSize lowercase: %v", err)
	}
	if got != 500<<20 {
		t.Fatalf("500mb = %d", got)
	}
}
