package cache

import (
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func openMemIndex(t *testing.T) *index {
	t.Helper()
	ix, err := openIndex(":memory:")
	if err != nil {
		t.Fatalf("openIndex: %v", err)
	}
	t.Cleanup(func() { _ = ix.close() })
	return ix
}

func putMeta(t *testing.T, ix *index, id string, sizeBytes int64) {
	t.Helper()
	if err := ix.put(&pb.WallpaperMetadata{
		Id:            id,
		SourceId:      "src",
		LocalPath:     "/tmp/" + id + ".jpg",
		FileSizeBytes: sizeBytes,
	}); err != nil {
		t.Fatalf("put(%s): %v", id, err)
	}
}

func TestIndexPutGet(t *testing.T) {
	ix := openMemIndex(t)
	putMeta(t, ix, "w1", 1000)

	got, err := ix.get("w1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Id != "w1" {
		t.Fatalf("id = %q, want w1", got.Id)
	}
	if got.FileSizeBytes != 1000 {
		t.Fatalf("size = %d, want 1000", got.FileSizeBytes)
	}
}

func TestIndexGetMissing(t *testing.T) {
	ix := openMemIndex(t)
	if _, err := ix.get("nope"); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestIndexUpsert(t *testing.T) {
	ix := openMemIndex(t)
	putMeta(t, ix, "w1", 100)
	putMeta(t, ix, "w1", 200) // update

	got, _ := ix.get("w1")
	if got.FileSizeBytes != 200 {
		t.Fatalf("size after upsert = %d, want 200", got.FileSizeBytes)
	}

	_, count, _ := ix.stats()
	if count != 1 {
		t.Fatalf("count = %d after upsert, want 1", count)
	}
}

func TestIndexStats(t *testing.T) {
	ix := openMemIndex(t)

	total, count, err := ix.stats()
	if err != nil || total != 0 || count != 0 {
		t.Fatalf("empty stats: total=%d count=%d err=%v", total, count, err)
	}

	putMeta(t, ix, "a", 300)
	putMeta(t, ix, "b", 700)

	total, count, err = ix.stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if total != 1000 {
		t.Fatalf("total = %d, want 1000", total)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestIndexDelete(t *testing.T) {
	ix := openMemIndex(t)
	putMeta(t, ix, "del", 500)

	path, size, err := ix.delete("del")
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if path == "" {
		t.Fatal("delete returned empty path")
	}
	if size != 500 {
		t.Fatalf("delete size = %d, want 500", size)
	}

	_, count, _ := ix.stats()
	if count != 0 {
		t.Fatalf("count = %d after delete, want 0", count)
	}
}

func TestIndexTouch(t *testing.T) {
	ix := openMemIndex(t)
	putMeta(t, ix, "t", 100)

	if err := ix.touch("t"); err != nil {
		t.Fatalf("touch: %v", err)
	}
}

func TestIndexEvictionCandidatesLRU(t *testing.T) {
	ix := openMemIndex(t)
	putMeta(t, ix, "old", 100)
	// Touch "new" so it has a newer last_used_at
	putMeta(t, ix, "new", 100)
	_ = ix.touch("new")

	ids, err := ix.evictionCandidates(pb.EvictionPolicy_EVICTION_POLICY_LRU, 1)
	if err != nil {
		t.Fatalf("evictionCandidates: %v", err)
	}
	if len(ids) != 1 || ids[0] != "old" {
		t.Fatalf("LRU eviction candidate = %v, want [old]", ids)
	}
}

func TestIndexEvictionCandidatesFIFO(t *testing.T) {
	ix := openMemIndex(t)
	putMeta(t, ix, "first", 100)
	putMeta(t, ix, "second", 100)

	ids, err := ix.evictionCandidates(pb.EvictionPolicy_EVICTION_POLICY_FIFO, 1)
	if err != nil {
		t.Fatalf("evictionCandidates: %v", err)
	}
	if len(ids) != 1 || ids[0] != "first" {
		t.Fatalf("FIFO eviction candidate = %v, want [first]", ids)
	}
}

func TestIndexEvictionCandidatesLimit(t *testing.T) {
	ix := openMemIndex(t)
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		putMeta(t, ix, id, 100)
	}

	ids, err := ix.evictionCandidates(pb.EvictionPolicy_EVICTION_POLICY_LRU, 3)
	if err != nil {
		t.Fatalf("evictionCandidates: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("len = %d, want 3", len(ids))
	}
}

func TestIndexEvictionCandidatesRandom(t *testing.T) {
	ix := openMemIndex(t)
	for _, id := range []string{"r1", "r2", "r3"} {
		putMeta(t, ix, id, 100)
	}

	// RANDOM() ordering — just check we get the right count and valid ids.
	ids, err := ix.evictionCandidates(pb.EvictionPolicy_EVICTION_POLICY_RANDOM, 2)
	if err != nil {
		t.Fatalf("evictionCandidates RANDOM: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("len = %d, want 2", len(ids))
	}
}
