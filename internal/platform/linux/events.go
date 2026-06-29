//go:build linux

package linux

import (
	"context"

	"github.com/conallob/peridot/internal/platform"
)

type linuxEvents struct{}

func (e *linuxEvents) Subscribe(ctx context.Context) (<-chan platform.SystemEvent, error) {
	ch := make(chan platform.SystemEvent)
	go func() { <-ctx.Done(); close(ch) }()
	return ch, nil
}
