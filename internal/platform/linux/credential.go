//go:build linux

package linux

import (
	"fmt"

	"github.com/conallob/peridot/internal/platform"
)

type linuxCredential struct{}

func (c *linuxCredential) Get(service, account string) (string, error) {
	return "", fmt.Errorf("linux credential store: %w", platform.ErrNotImplemented)
}

func (c *linuxCredential) Set(service, account, secret string) error {
	return fmt.Errorf("linux credential store: %w", platform.ErrNotImplemented)
}

func (c *linuxCredential) Delete(service, account string) error {
	return fmt.Errorf("linux credential store: %w", platform.ErrNotImplemented)
}
