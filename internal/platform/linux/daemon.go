//go:build linux

package linux

import (
	"fmt"

	"github.com/conallob/peridot/internal/platform"
)

type linuxDaemon struct{}

func (d *linuxDaemon) Install() error {
	return fmt.Errorf("linux daemon install: %w", platform.ErrNotImplemented)
}
func (d *linuxDaemon) Uninstall() error {
	return fmt.Errorf("linux daemon uninstall: %w", platform.ErrNotImplemented)
}
func (d *linuxDaemon) Start() error {
	return fmt.Errorf("linux daemon start: %w", platform.ErrNotImplemented)
}
func (d *linuxDaemon) Stop() error {
	return fmt.Errorf("linux daemon stop: %w", platform.ErrNotImplemented)
}
func (d *linuxDaemon) Restart() error {
	return fmt.Errorf("linux daemon restart: %w", platform.ErrNotImplemented)
}
func (d *linuxDaemon) Status() (string, error) {
	return "", fmt.Errorf("linux daemon status: %w", platform.ErrNotImplemented)
}
