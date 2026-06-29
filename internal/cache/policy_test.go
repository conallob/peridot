package cache

import (
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func TestOrderClause(t *testing.T) {
	cases := []struct {
		policy pb.EvictionPolicy
		want   string
	}{
		{pb.EvictionPolicy_EVICTION_POLICY_LRU, "last_used_at ASC"},
		{pb.EvictionPolicy_EVICTION_POLICY_FIFO, "cached_at ASC"},
		{pb.EvictionPolicy_EVICTION_POLICY_RANDOM, "RANDOM()"},
		{pb.EvictionPolicy_EVICTION_POLICY_UNSPECIFIED, "last_used_at ASC"},
	}
	for _, c := range cases {
		if got := orderClause(c.policy); got != c.want {
			t.Errorf("orderClause(%v) = %q, want %q", c.policy, got, c.want)
		}
	}
}
