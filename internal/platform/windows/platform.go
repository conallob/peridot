//go:build windows

package windows

import "github.com/conallob/peridot/internal/platform"

type windowsPlatform struct{}

func init() { platform.Register(&windowsPlatform{}) }

func (p *windowsPlatform) Display() platform.Display       { return &windowsDisplay{} }
func (p *windowsPlatform) Events() platform.Events         { return &windowsEvents{} }
func (p *windowsPlatform) Credential() platform.Credential { return &windowsCredential{} }
func (p *windowsPlatform) Daemon() platform.Daemon         { return &windowsDaemon{} }
func (p *windowsPlatform) Name() string                    { return "windows" }
