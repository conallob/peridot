//go:build darwin

package darwin

import "github.com/conallob/peridot/internal/platform"

// darwinCredential is a stub. A production implementation would call the
// Security framework (SecItem*) via cgo to talk to the login Keychain.
type darwinCredential struct{}

func (c *darwinCredential) Get(service, account string) (string, error) {
	return "", platform.ErrNotImplemented
}

func (c *darwinCredential) Set(service, account, secret string) error {
	return platform.ErrNotImplemented
}

func (c *darwinCredential) Delete(service, account string) error {
	return platform.ErrNotImplemented
}
