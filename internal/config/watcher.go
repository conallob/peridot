package config

import (
	"context"
	"path/filepath"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/fsnotify/fsnotify"
)

// Watcher reloads and recompiles a config file when it changes on disk.
type Watcher struct {
	path string
	w    *fsnotify.Watcher
}

// NewWatcher creates a config watcher for path.
func NewWatcher(path string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	// Watch the parent directory so we still catch atomic rename-replace writes.
	if err := w.Add(filepath.Dir(path)); err != nil {
		w.Close()
		return nil, err
	}
	return &Watcher{path: path, w: w}, nil
}

// Watch invokes onReload with the freshly compiled config whenever the file
// changes, until ctx is cancelled. Compilation errors are passed to onError.
func (cw *Watcher) Watch(ctx context.Context, onReload func(*pb.Config), onError func(error)) {
	defer cw.w.Close()
	target := filepath.Clean(cw.path)
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-cw.w.Events:
			if !ok {
				return
			}
			if filepath.Clean(ev.Name) != target {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			cfg, err := Compile(cw.path)
			if err != nil {
				if onError != nil {
					onError(err)
				}
				continue
			}
			if onReload != nil {
				onReload(cfg)
			}
		case err, ok := <-cw.w.Errors:
			if !ok {
				return
			}
			if onError != nil {
				onError(err)
			}
		}
	}
}

// Close stops watching.
func (cw *Watcher) Close() error { return cw.w.Close() }
