package scheduler

import (
	"sync"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const historyCap = 50

// history is a fixed-size ring buffer of recently displayed wallpapers.
type history struct {
	mu      sync.Mutex
	entries []*pb.HistoryEntry
}

func newHistory() *history {
	return &history{entries: make([]*pb.HistoryEntry, 0, historyCap)}
}

func (h *history) add(m *pb.WallpaperMetadata) {
	h.mu.Lock()
	defer h.mu.Unlock()
	e := &pb.HistoryEntry{Wallpaper: m, DisplayedAt: timestamppb.New(time.Now())}
	h.entries = append(h.entries, e)
	if len(h.entries) > historyCap {
		h.entries = h.entries[len(h.entries)-historyCap:]
	}
}

// recent returns up to limit most-recent entries, newest first.
func (h *history) recent(limit int) []*pb.HistoryEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	if limit <= 0 || limit > len(h.entries) {
		limit = len(h.entries)
	}
	out := make([]*pb.HistoryEntry, 0, limit)
	for i := len(h.entries) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, h.entries[i])
	}
	return out
}

// prev returns the entry n steps back from the most recent (1 = previous).
func (h *history) prev(n int) *pb.WallpaperMetadata {
	h.mu.Lock()
	defer h.mu.Unlock()
	idx := len(h.entries) - 1 - n
	if idx < 0 || idx >= len(h.entries) {
		return nil
	}
	return h.entries[idx].Wallpaper
}

func (h *history) current() *pb.WallpaperMetadata {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.entries) == 0 {
		return nil
	}
	return h.entries[len(h.entries)-1].Wallpaper
}
