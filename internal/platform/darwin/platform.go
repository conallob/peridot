//go:build darwin

package darwin

import "github.com/conallob/peridot/internal/platform"

type darwinPlatform struct {
	display    *darwinDisplay
	events     *darwinEvents
	credential *darwinCredential
	daemon     *darwinDaemon
}

func init() {
	platform.Register(&darwinPlatform{
		display:    &darwinDisplay{},
		events:     &darwinEvents{},
		credential: &darwinCredential{},
		daemon:     &darwinDaemon{},
	})
}

func (p *darwinPlatform) Display() platform.Display       { return p.display }
func (p *darwinPlatform) Events() platform.Events         { return p.events }
func (p *darwinPlatform) Credential() platform.Credential { return p.credential }
func (p *darwinPlatform) Daemon() platform.Daemon         { return p.daemon }
func (p *darwinPlatform) Name() string                    { return "darwin" }
