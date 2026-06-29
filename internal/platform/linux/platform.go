//go:build linux

package linux

import "github.com/conallob/peridot/internal/platform"

type linuxPlatform struct{}

func init() { platform.Register(&linuxPlatform{}) }

func (p *linuxPlatform) Display() platform.Display       { return &linuxDisplay{} }
func (p *linuxPlatform) Events() platform.Events         { return &linuxEvents{} }
func (p *linuxPlatform) Credential() platform.Credential { return &linuxCredential{} }
func (p *linuxPlatform) Daemon() platform.Daemon         { return &linuxDaemon{} }
func (p *linuxPlatform) Name() string                    { return "linux" }
