package darwin

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

// pruneRenders deletes all but the newest keep *.bmp files in dir, skipping
// files modified within minAge of now. It only touches regular files with a
// .bmp extension (never symlinks, never cacheVersion.db) and returns the
// number of files and bytes removed.
func pruneRenders(dir string, keep int, minAge time.Duration, now time.Time) (files int, bytes int64, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	type render struct {
		path string
		size int64
		mod  time.Time
	}
	var renders []render
	for _, e := range entries {
		if !e.Type().IsRegular() || filepath.Ext(e.Name()) != ".bmp" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		renders = append(renders, render{filepath.Join(dir, e.Name()), info.Size(), info.ModTime()})
	}
	sort.Slice(renders, func(i, j int) bool { return renders[i].mod.After(renders[j].mod) })
	if keep < 0 {
		keep = 0
	}
	for i, r := range renders {
		if i < keep || now.Sub(r.mod) < minAge {
			continue
		}
		if os.Remove(r.path) == nil {
			files++
			bytes += r.size
		}
	}
	return files, bytes, nil
}
