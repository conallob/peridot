//go:build darwin

package darwin

import (
	"context"

	"github.com/conallob/peridot/internal/platform"
)

// darwinEvents is a stub implementation. A full implementation would observe
// NSWorkspace notifications (didWake, sessionDidBecomeActive) via cgo. Here we
// expose a channel that simply closes when the context is cancelled.
type darwinEvents struct{}

func (e *darwinEvents) Subscribe(ctx context.Context) (<-chan platform.SystemEvent, error) {
	ch := make(chan platform.SystemEvent)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}
