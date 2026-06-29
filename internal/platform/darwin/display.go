//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/conallob/peridot/internal/platform"
)

type darwinDisplay struct{}

func (d *darwinDisplay) SetWallpaper(ctx context.Context, imagePath string) error {
	script := fmt.Sprintf(
		`tell application "System Events" to set picture of every desktop to %q`,
		imagePath,
	)
	return runOsascript(ctx, script)
}

func (d *darwinDisplay) SetWallpaperForDisplay(ctx context.Context, displayID, imagePath string) error {
	// displayID is the 1-based index of the desktop as reported by System Events.
	script := fmt.Sprintf(
		`tell application "System Events" to set picture of desktop %s to %q`,
		displayID, imagePath,
	)
	return runOsascript(ctx, script)
}

func (d *darwinDisplay) Displays(ctx context.Context) ([]platform.DisplayInfo, error) {
	script := `tell application "System Events" to return count of desktops`
	out, err := osascriptOutput(ctx, script)
	if err != nil {
		return nil, err
	}
	n := 0
	fmt.Sscanf(strings.TrimSpace(out), "%d", &n)
	if n <= 0 {
		n = 1
	}
	infos := make([]platform.DisplayInfo, 0, n)
	for i := 1; i <= n; i++ {
		infos = append(infos, platform.DisplayInfo{
			ID:   fmt.Sprintf("%d", i),
			Name: fmt.Sprintf("Display %d", i),
			Main: i == 1,
		})
	}
	return infos, nil
}

func runOsascript(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func osascriptOutput(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("osascript: %w", err)
	}
	return string(out), nil
}
