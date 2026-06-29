package scheduler

import (
	"fmt"
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func meta(id string) *pb.WallpaperMetadata {
	return &pb.WallpaperMetadata{Id: id, Title: id}
}

func TestHistoryOrderAndCurrent(t *testing.T) {
	h := newHistory()
	if h.current() != nil {
		t.Fatal("empty history current should be nil")
	}

	h.add(meta("a"))
	h.add(meta("b"))
	h.add(meta("c"))

	if cur := h.current(); cur == nil || cur.Id != "c" {
		t.Fatalf("current = %v, want c", cur)
	}

	recent := h.recent(2)
	if len(recent) != 2 {
		t.Fatalf("recent len = %d, want 2", len(recent))
	}
	if recent[0].Wallpaper.Id != "c" || recent[1].Wallpaper.Id != "b" {
		t.Fatalf("recent order wrong: %v %v", recent[0].Wallpaper.Id, recent[1].Wallpaper.Id)
	}
}

func TestHistoryPrev(t *testing.T) {
	h := newHistory()
	h.add(meta("a"))
	h.add(meta("b"))
	h.add(meta("c"))

	if p := h.prev(1); p == nil || p.Id != "b" {
		t.Fatalf("prev(1) = %v, want b", p)
	}
	if p := h.prev(2); p == nil || p.Id != "a" {
		t.Fatalf("prev(2) = %v, want a", p)
	}
	if p := h.prev(99); p != nil {
		t.Fatalf("prev(99) = %v, want nil", p)
	}
}

func TestHistoryCapWraps(t *testing.T) {
	h := newHistory()
	for i := 0; i < historyCap+10; i++ {
		h.add(meta(fmt.Sprintf("item-%d", i)))
	}

	all := h.recent(0) // 0 => all
	if len(all) != historyCap {
		t.Fatalf("history len = %d, want cap %d", len(all), historyCap)
	}
	// Newest first; the most recent added is the last index.
	if all[0].Wallpaper.Id != fmt.Sprintf("item-%d", historyCap+10-1) {
		t.Fatalf("newest = %s, want item-%d", all[0].Wallpaper.Id, historyCap+10-1)
	}
	// Oldest retained entry is index 10 (first 10 evicted).
	if all[len(all)-1].Wallpaper.Id != "item-10" {
		t.Fatalf("oldest retained = %s, want item-10", all[len(all)-1].Wallpaper.Id)
	}
}
