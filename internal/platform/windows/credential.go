//go:build windows

package windows

import (
	"fmt"

	"github.com/conallob/peridot/internal/platform"
)

type windowsCredential struct{}

func (c *windowsCredential) Get(service, account string) (string, error) {
	return "", fmt.Errorf("windows credential store: %w", platform.ErrNotImplemented)
}

func (c *windowsCredential) Set(service, account, secret string) error {
	return fmt.Errorf("windows credential store: %w", platform.ErrNotImplemented)
}

func (c *windowsCredential) Delete(service, account string) error {
	return fmt.Errorf("windows credential store: %w", platform.ErrNotImplemented)
}
