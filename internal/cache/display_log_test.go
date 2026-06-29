package cache

import (
	"testing"
	"time"
)

// newTestLog returns a DisplayLog backed by an in-memory SQLite DB.
func newTestLog(t *testing.T) *DisplayLog {
	t.Helper()
	ix, err := openIndex(":memory:")
	if err != nil {
		t.Fatalf("openIndex: %v", err)
	}
	t.Cleanup(func() { _ = ix.close() })
	return NewDisplayLog(ix)
}

// insertAt inserts a display event at a specific time (bypassing Record's now()).
func insertAt(t *testing.T, dl *DisplayLog, id, source, title string, at time.Time) {
	t.Helper()
	if _, err := dl.idx.db.Exec(
		`INSERT INTO display_log (wallpaper_id, source_id, title, displayed_at) VALUES (?, ?, ?, ?)`,
		id, source, title, at.Unix(),
	); err != nil {
		t.Fatalf("insertAt: %v", err)
	}
}

func TestRecordInserts(t *testing.T) {
	dl := newTestLog(t)
	if err := dl.Record("w1", "s1", "Title"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	total, _, _, _, err := dl.QueryStats(0, 10)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
}

func TestQueryStatsTotalAndTop(t *testing.T) {
	dl := newTestLog(t)
	now := time.Now()
	for i := 0; i < 5; i++ {
		insertAt(t, dl, "popular", "db", "Sanctuary III", now)
	}
	for i := 0; i < 2; i++ {
		insertAt(t, dl, "mid", "local", "Twilight Zone", now)
	}
	insertAt(t, dl, "rare", "gdrive", "Lone", now)

	total, top, sources, _, err := dl.QueryStats(0, 10)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if total != 8 {
		t.Fatalf("total = %d, want 8", total)
	}
	if len(top) != 3 {
		t.Fatalf("len(top) = %d, want 3", len(top))
	}
	// Sorted by count desc.
	if top[0].WallpaperID != "popular" || top[0].DisplayCount != 5 {
		t.Fatalf("top[0] = %+v, want popular/5", top[0])
	}
	if top[1].WallpaperID != "mid" || top[1].DisplayCount != 2 {
		t.Fatalf("top[1] = %+v, want mid/2", top[1])
	}
	if top[2].DisplayCount != 1 {
		t.Fatalf("top[2] count = %d, want 1", top[2].DisplayCount)
	}
	// Source breakdown sorted desc.
	if len(sources) != 3 || sources[0].SourceID != "db" || sources[0].DisplayCount != 5 {
		t.Fatalf("sources[0] = %+v, want db/5", sources[0])
	}
}

func TestQueryStatsTopLimit(t *testing.T) {
	dl := newTestLog(t)
	now := time.Now()
	for i := 0; i < 5; i++ {
		insertAt(t, dl, string(rune('a'+i)), "s", "T", now)
	}
	_, top, _, _, err := dl.QueryStats(0, 2)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if len(top) != 2 {
		t.Fatalf("len(top) = %d, want 2", len(top))
	}
}

func TestQueryStatsHourly(t *testing.T) {
	dl := newTestLog(t)
	// Use a fixed UTC time so the hour bucket is deterministic.
	base := time.Date(2026, 1, 2, 9, 30, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		insertAt(t, dl, "w", "s", "T", base)
	}
	insertAt(t, dl, "w", "s", "T", time.Date(2026, 1, 2, 14, 0, 0, 0, time.UTC))

	_, _, _, hourly, err := dl.QueryStats(0, 10)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if hourly[9] != 3 {
		t.Fatalf("hourly[9] = %d, want 3", hourly[9])
	}
	if hourly[14] != 1 {
		t.Fatalf("hourly[14] = %d, want 1", hourly[14])
	}
	if hourly[0] != 0 {
		t.Fatalf("hourly[0] = %d, want 0", hourly[0])
	}
}

func TestQueryStatsDaysFilter(t *testing.T) {
	dl := newTestLog(t)
	now := time.Now()
	insertAt(t, dl, "recent", "s", "Recent", now.AddDate(0, 0, -1))
	insertAt(t, dl, "old", "s", "Old", now.AddDate(0, 0, -30))

	total, top, _, _, err := dl.QueryStats(7, 10)
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1 (old entry excluded)", total)
	}
	if len(top) != 1 || top[0].WallpaperID != "recent" {
		t.Fatalf("top = %+v, want only recent", top)
	}

	// All time includes both.
	totalAll, _, _, _, err := dl.QueryStats(0, 10)
	if err != nil {
		t.Fatalf("QueryStats all: %v", err)
	}
	if totalAll != 2 {
		t.Fatalf("totalAll = %d, want 2", totalAll)
	}
}
