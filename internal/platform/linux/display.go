//go:build linux

package linux

import (
	"context"
	"fmt"

	"github.com/conallob/peridot/internal/platform"
)

type linuxDisplay struct{}

func (d *linuxDisplay) SetWallpaper(ctx context.Context, imagePath string) error {
	return fmt.Errorf("linux wallpaper support: %w", platform.ErrNotImplemented)
}

func (d *linuxDisplay) SetWallpaperForDisplay(ctx context.Context, displayID, imagePath string) error {
	return fmt.Errorf("linux per-display wallpaper support: %w", platform.ErrNotImplemented)
}

func (d *linuxDisplay) Displays(ctx context.Context) ([]platform.DisplayInfo, error) {
	return nil, fmt.Errorf("linux display enumeration: %w", platform.ErrNotImplemented)
}
