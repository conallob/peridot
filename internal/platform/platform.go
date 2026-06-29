package platform

import (
	"context"
	"errors"

	pb "github.com/conallob/peridot/gen/peridot"
)

// ErrNotImplemented is returned by platform operations that are not supported
// on the current operating system.
var ErrNotImplemented = errors.New("peridot: operation not implemented on this platform")

// DisplayInfo describes a single connected display.
type DisplayInfo struct {
	ID   string
	Name string
	Main bool
}

// SystemEvent represents an OS-level event the scheduler may react to.
type SystemEvent int

const (
	EventUnknown SystemEvent = iota
	EventWake
	EventLogin
	EventDisplayChange
)

// Display sets desktop wallpapers.
type Display interface {
	// SetWallpaper sets the wallpaper on all displays.
	SetWallpaper(ctx context.Context, imagePath string) error
	// SetWallpaperForDisplay sets the wallpaper on a single display.
	SetWallpaperForDisplay(ctx context.Context, displayID, imagePath string) error
	// Displays enumerates connected displays.
	Displays(ctx context.Context) ([]DisplayInfo, error)
}

// Events delivers OS-level system events.
type Events interface {
	// Subscribe returns a channel that receives system events until ctx is done.
	Subscribe(ctx context.Context) (<-chan SystemEvent, error)
}

// Credential provides secure secret storage (e.g. macOS Keychain).
type Credential interface {
	Get(service, account string) (string, error)
	Set(service, account, secret string) error
	Delete(service, account string) error
}

// Daemon manages OS service installation and lifecycle.
type Daemon interface {
	Install() error
	Uninstall() error
	Start() error
	Stop() error
	Restart() error
	Status() (string, error)
}

// Platform aggregates all OS-specific capabilities.
type Platform interface {
	Display() Display
	Events() Events
	Credential() Credential
	Daemon() Daemon
	Name() string
}

var current Platform

// Register installs the active platform implementation. It is called from the
// platform-specific init() guarded by build tags.
func Register(p Platform) { current = p }

// Current returns the registered platform implementation, or nil.
func Current() Platform { return current }

// ConfigSourceType is re-exported for convenience in higher layers.
type _ = pb.SourceType
