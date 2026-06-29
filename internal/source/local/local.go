package local

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/fsnotify/fsnotify"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".bmp": true, ".webp": true, ".heic": true, ".tiff": true,
}

// LocalDirectorySource serves wallpapers from a directory on disk.
type LocalDirectorySource struct {
	id          string
	displayName string
	path        string
	watch       bool
}

// New constructs a LocalDirectorySource.
func New(id, displayName, path string, watch bool) *LocalDirectorySource {
	if displayName == "" {
		displayName = "Local: " + path
	}
	return &LocalDirectorySource{id: id, displayName: displayName, path: path, watch: watch}
}

func (s *LocalDirectorySource) ID() string          { return s.id }
func (s *LocalDirectorySource) DisplayName() string { return s.displayName }
func (s *LocalDirectorySource) Type() pb.SourceType { return pb.SourceType_SOURCE_TYPE_LOCAL }

func (s *LocalDirectorySource) Catalogue(ctx context.Context) ([]*pb.WallpaperMetadata, error) {
	var out []*pb.WallpaperMetadata
	err := filepath.WalkDir(s.path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		if !imageExts[strings.ToLower(filepath.Ext(p))] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out = append(out, &pb.WallpaperMetadata{
			Id:            idFor(s.id, p),
			Title:         filepath.Base(p),
			SourceId:      s.id,
			LocalPath:     p,
			PublishedAt:   timestamppb.New(info.ModTime()),
			FileSizeBytes: info.Size(),
		})
		return nil
	})
	return out, err
}

func (s *LocalDirectorySource) Fetch(ctx context.Context, m *pb.WallpaperMetadata) (string, error) {
	// Local files are already on disk.
	if m.LocalPath != "" {
		return m.LocalPath, nil
	}
	return "", os.ErrNotExist
}

func (s *LocalDirectorySource) NewSince(ctx context.Context, since time.Time) ([]*pb.WallpaperMetadata, error) {
	all, err := s.Catalogue(ctx)
	if err != nil {
		return nil, err
	}
	var out []*pb.WallpaperMetadata
	for _, m := range all {
		if m.PublishedAt != nil && m.PublishedAt.AsTime().After(since) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *LocalDirectorySource) Healthy(ctx context.Context) bool {
	fi, err := os.Stat(s.path)
	return err == nil && fi.IsDir()
}

// Watch emits a notification on the returned channel whenever the directory's
// contents change. It is a no-op (returns a closed-on-cancel channel) if watch
// was not enabled.
func (s *LocalDirectorySource) Watch(ctx context.Context) (<-chan struct{}, error) {
	ch := make(chan struct{}, 1)
	if !s.watch {
		go func() { <-ctx.Done(); close(ch) }()
		return ch, nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(s.path); err != nil {
		w.Close()
		return nil, err
	}
	go func() {
		defer w.Close()
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-w.Events:
				if !ok {
					return
				}
				select {
				case ch <- struct{}{}:
				default:
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
	return ch, nil
}

func idFor(sourceID, path string) string {
	h := sha1.Sum([]byte(sourceID + "|" + path))
	return hex.EncodeToString(h[:])
}
