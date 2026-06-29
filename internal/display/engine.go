package display

import (
	"context"
	"fmt"

	"github.com/conallob/peridot/internal/platform"
)

// Engine is a thin wrapper over the platform Display implementation.
type Engine struct {
	d platform.Display
}

// New returns a display Engine bound to the current platform. It returns an
// error if no platform is registered.
func New() (*Engine, error) {
	p := platform.Current()
	if p == nil {
		return nil, fmt.Errorf("no platform registered")
	}
	return &Engine{d: p.Display()}, nil
}

// Set applies imagePath to every display.
func (e *Engine) Set(ctx context.Context, imagePath string) error {
	return e.d.SetWallpaper(ctx, imagePath)
}

// SetForDisplay applies imagePath to a single display.
func (e *Engine) SetForDisplay(ctx context.Context, displayID, imagePath string) error {
	return e.d.SetWallpaperForDisplay(ctx, displayID, imagePath)
}

// Displays enumerates connected displays.
func (e *Engine) Displays(ctx context.Context) ([]platform.DisplayInfo, error) {
	return e.d.Displays(ctx)
}
