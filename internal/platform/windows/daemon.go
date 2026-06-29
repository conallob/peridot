//go:build windows

package windows

import (
	"fmt"

	"github.com/conallob/peridot/internal/platform"
)

type windowsDaemon struct{}

func (d *windowsDaemon) Install() error {
	return fmt.Errorf("windows daemon install: %w", platform.ErrNotImplemented)
}
func (d *windowsDaemon) Uninstall() error {
	return fmt.Errorf("windows daemon uninstall: %w", platform.ErrNotImplemented)
}
func (d *windowsDaemon) Start() error {
	return fmt.Errorf("windows daemon start: %w", platform.ErrNotImplemented)
}
func (d *windowsDaemon) Stop() error {
	return fmt.Errorf("windows daemon stop: %w", platform.ErrNotImplemented)
}
func (d *windowsDaemon) Restart() error {
	return fmt.Errorf("windows daemon restart: %w", platform.ErrNotImplemented)
}
func (d *windowsDaemon) Status() (string, error) {
	return "", fmt.Errorf("windows daemon status: %w", platform.ErrNotImplemented)
}
