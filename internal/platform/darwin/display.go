//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/conallob/peridot/internal/platform"
)

type darwinDisplay struct{}

func (d *darwinDisplay) SetWallpaper(ctx context.Context, imagePath string) error {
	script := fmt.Sprintf(
		`tell application "System Events" to set picture of every desktop to %q`,
		imagePath,
	)
	err := runOsascript(ctx, script)
	pruneWallpaperCache()
	return err
}

func (d *darwinDisplay) SetWallpaperForDisplay(ctx context.Context, displayID, imagePath string) error {
	// displayID is the 1-based index of the desktop as reported by System Events.
	script := fmt.Sprintf(
		`tell application "System Events" to set picture of desktop %s to %q`,
		displayID, imagePath,
	)
	err := runOsascript(ctx, script)
	pruneWallpaperCache()
	return err
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

// keepRenders is how many of the newest rendered wallpapers are left in the
// wallpaper agent's cache. It must cover every display's current wallpaper;
// anything the agent still needs is simply re-rendered on demand.
const keepRenders = 8

// minRenderAge protects renders the agent may still be writing or reading.
const minRenderAge = time.Minute

// wallpaperCacheDir returns the directory where macOS's wallpaper agent stores
// decoded, full-resolution BMP renders of every image it has displayed. macOS
// never evicts these, so rotating wallpapers fills the disk.
func wallpaperCacheDir(home string) string {
	return filepath.Join(home, "Library", "Containers", "com.apple.wallpaper.agent",
		"Data", "Library", "Caches", "com.apple.wallpaper.caches",
		"extension-com.apple.wallpaper.extension.image")
}

// pruneWallpaperCache removes stale renders from the wallpaper agent's cache
// (see wallpaperCacheDir). Best-effort: failures are logged, never returned.
func pruneWallpaperCache() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	files, bytes, err := pruneRenders(wallpaperCacheDir(home), keepRenders, minRenderAge, time.Now())
	if err != nil {
		log.Printf("darwin: prune wallpaper cache: %v", err)
		return
	}
	if files > 0 {
		log.Printf("darwin: pruned %d stale wallpaper renders (%d MiB)", files, bytes>>20)
	}
}
