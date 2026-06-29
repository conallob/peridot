package source

import (
	"context"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
)

// Source is a provider of wallpaper images.
type Source interface {
	ID() string
	DisplayName() string
	Type() pb.SourceType
	// Catalogue returns all currently known wallpapers (metadata only).
	Catalogue(ctx context.Context) ([]*pb.WallpaperMetadata, error)
	// Fetch downloads (if necessary) the image for m and returns its local path.
	Fetch(ctx context.Context, m *pb.WallpaperMetadata) (localPath string, err error)
	// NewSince returns wallpapers published after since.
	NewSince(ctx context.Context, since time.Time) ([]*pb.WallpaperMetadata, error)
	// Healthy reports whether the source is currently reachable.
	Healthy(ctx context.Context) bool
}
