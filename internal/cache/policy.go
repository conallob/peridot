package cache

import pb "github.com/conallob/peridot/gen/peridot"

// orderClause returns the SQL ORDER BY fragment that selects eviction
// candidates first for the given policy.
func orderClause(policy pb.EvictionPolicy) string {
	switch policy {
	case pb.EvictionPolicy_EVICTION_POLICY_LRU:
		return "last_used_at ASC"
	case pb.EvictionPolicy_EVICTION_POLICY_FIFO:
		return "cached_at ASC"
	case pb.EvictionPolicy_EVICTION_POLICY_RANDOM:
		return "RANDOM()"
	default:
		return "last_used_at ASC"
	}
}
