//go:build windows

package windows

import (
	"context"
	"fmt"

	"github.com/conallob/peridot/internal/platform"
)

type windowsDisplay struct{}

func (d *windowsDisplay) SetWallpaper(ctx context.Context, imagePath string) error {
	return fmt.Errorf("windows wallpaper support: %w", platform.ErrNotImplemented)
}

func (d *windowsDisplay) SetWallpaperForDisplay(ctx context.Context, displayID, imagePath string) error {
	return fmt.Errorf("windows per-display wallpaper support: %w", platform.ErrNotImplemented)
}

func (d *windowsDisplay) Displays(ctx context.Context) ([]platform.DisplayInfo, error) {
	return nil, fmt.Errorf("windows display enumeration: %w", platform.ErrNotImplemented)
}
