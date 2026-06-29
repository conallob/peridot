package cache

import (
	"time"
)

// DisplayLog persists wallpaper display events to SQLite.
// The table is created in the same DB as the cache index.

const displayLogSchema = `
CREATE TABLE IF NOT EXISTS display_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    wallpaper_id TEXT NOT NULL,
    source_id    TEXT NOT NULL,
    title        TEXT NOT NULL,
    displayed_at INTEGER NOT NULL  -- Unix timestamp (seconds)
);
CREATE INDEX IF NOT EXISTS idx_display_log_at ON display_log(displayed_at);
`

// DisplayLog records and aggregates wallpaper display events. It shares the
// SQLite connection owned by the cache index.
type DisplayLog struct {
	idx *index
}

// NewDisplayLog returns a DisplayLog backed by idx's connection.
func NewDisplayLog(idx *index) *DisplayLog { return &DisplayLog{idx: idx} }

// Record writes a display event. Called by the scheduler on every wallpaper change.
func (dl *DisplayLog) Record(wallpaperID, sourceID, title string) error {
	_, err := dl.idx.db.Exec(
		`INSERT INTO display_log (wallpaper_id, source_id, title, displayed_at) VALUES (?, ?, ?, ?)`,
		wallpaperID, sourceID, title, time.Now().Unix(),
	)
	return err
}

// WallpaperDisplayStat is a per-wallpaper display aggregate.
type WallpaperDisplayStat struct {
	WallpaperID  string
	SourceID     string
	Title        string
	DisplayCount int32
}

// SourceDisplayStat is a per-source display aggregate.
type SourceDisplayStat struct {
	SourceID     string
	DisplayCount int32
}

// QueryStats returns aggregate stats for the last `days` days (0 = all time).
// Returns total displays, the top N wallpapers, the per-source breakdown, and
// a 24-element hourly histogram (index = hour 0-23).
func (dl *DisplayLog) QueryStats(days int, topN int) (
	total int32,
	topWallpapers []WallpaperDisplayStat,
	sourceCounts []SourceDisplayStat,
	hourlyCounts [24]int32,
	err error,
) {
	if topN <= 0 {
		topN = 10
	}
	var cutoff int64
	hasCutoff := days > 0
	if hasCutoff {
		cutoff = time.Now().AddDate(0, 0, -days).Unix()
	}
	where := ""
	args := []any{}
	if hasCutoff {
		where = " WHERE displayed_at > ?"
		args = append(args, cutoff)
	}

	// 1. Total count.
	if err = dl.idx.db.QueryRow(`SELECT COUNT(*) FROM display_log`+where, args...).Scan(&total); err != nil {
		return
	}

	// 2. Top wallpapers.
	topArgs := append(append([]any{}, args...), topN)
	rows, qerr := dl.idx.db.Query(
		`SELECT wallpaper_id, source_id, MAX(title) AS title, COUNT(*) AS c
		 FROM display_log`+where+`
		 GROUP BY wallpaper_id
		 ORDER BY c DESC, title ASC
		 LIMIT ?`, topArgs...)
	if qerr != nil {
		err = qerr
		return
	}
	for rows.Next() {
		var w WallpaperDisplayStat
		if err = rows.Scan(&w.WallpaperID, &w.SourceID, &w.Title, &w.DisplayCount); err != nil {
			_ = rows.Close()
			return
		}
		topWallpapers = append(topWallpapers, w)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return
	}
	_ = rows.Close()

	// 3. Source breakdown.
	srows, serr := dl.idx.db.Query(
		`SELECT source_id, COUNT(*) AS c
		 FROM display_log`+where+`
		 GROUP BY source_id
		 ORDER BY c DESC`, args...)
	if serr != nil {
		err = serr
		return
	}
	for srows.Next() {
		var s SourceDisplayStat
		if err = srows.Scan(&s.SourceID, &s.DisplayCount); err != nil {
			_ = srows.Close()
			return
		}
		sourceCounts = append(sourceCounts, s)
	}
	if err = srows.Err(); err != nil {
		_ = srows.Close()
		return
	}
	_ = srows.Close()

	// 4. Hourly histogram.
	hrows, herr := dl.idx.db.Query(
		`SELECT CAST(strftime('%H', datetime(displayed_at, 'unixepoch')) AS INTEGER) AS hour, COUNT(*) AS c
		 FROM display_log`+where+`
		 GROUP BY hour`, args...)
	if herr != nil {
		err = herr
		return
	}
	for hrows.Next() {
		var hour int
		var c int32
		if err = hrows.Scan(&hour, &c); err != nil {
			_ = hrows.Close()
			return
		}
		if hour >= 0 && hour < 24 {
			hourlyCounts[hour] = c
		}
	}
	if err = hrows.Err(); err != nil {
		_ = hrows.Close()
		return
	}
	_ = hrows.Close()

	return
}
