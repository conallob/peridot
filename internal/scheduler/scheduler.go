package scheduler

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/display"
	"github.com/conallob/peridot/internal/platform"
	"github.com/conallob/peridot/internal/source"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Scheduler rotates wallpapers on a timer and in response to system events.
type Scheduler struct {
	display *display.Engine
	events  platform.Events

	mu       sync.Mutex
	sources  []source.Source
	interval time.Duration
	shuffle  bool

	paused  atomic.Bool
	history *history

	resetCh  chan struct{}
	lastTick time.Time
}

// Options configures a Scheduler.
type Options struct {
	Display  *display.Engine
	Events   platform.Events
	Sources  []source.Source
	Interval time.Duration
	Shuffle  bool
}

// New creates a Scheduler.
func New(opts Options) *Scheduler {
	interval := opts.Interval
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &Scheduler{
		display:  opts.Display,
		events:   opts.Events,
		sources:  opts.Sources,
		interval: interval,
		shuffle:  opts.Shuffle,
		history:  newHistory(),
		resetCh:  make(chan struct{}, 1),
	}
}

// Run drives the scheduler until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	var eventCh <-chan platform.SystemEvent
	if s.events != nil {
		if ch, err := s.events.Subscribe(ctx); err == nil {
			eventCh = ch
		}
	}

	// Apply an initial wallpaper immediately.
	_ = s.advance(ctx)

	timer := time.NewTimer(s.currentInterval())
	defer timer.Stop()
	s.lastTick = time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if !s.paused.Load() {
				_ = s.advance(ctx)
			}
			s.lastTick = time.Now()
			timer.Reset(s.currentInterval())
		case <-s.resetCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			s.lastTick = time.Now()
			timer.Reset(s.currentInterval())
		case ev, ok := <-eventCh:
			if !ok {
				eventCh = nil
				continue
			}
			if (ev == platform.EventWake || ev == platform.EventLogin) && !s.paused.Load() {
				_ = s.advance(ctx)
			}
		}
	}
}

func (s *Scheduler) currentInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

// advance selects and displays the next wallpaper.
func (s *Scheduler) advance(ctx context.Context) error {
	m, err := s.pick(ctx)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("no wallpapers available")
	}
	path := m.LocalPath
	if path == "" {
		return errors.New("selected wallpaper has no local path")
	}
	if err := s.display.Set(ctx, path); err != nil {
		return err
	}
	m.CachedAt = timestamppb.New(time.Now())
	s.history.add(m)
	return nil
}

// pick chooses a wallpaper from the configured sources.
func (s *Scheduler) pick(ctx context.Context) (*pb.WallpaperMetadata, error) {
	s.mu.Lock()
	srcs := append([]source.Source(nil), s.sources...)
	s.mu.Unlock()

	var pool []*pb.WallpaperMetadata
	for _, src := range srcs {
		items, err := src.Catalogue(ctx)
		if err != nil {
			continue
		}
		pool = append(pool, items...)
	}
	if len(pool) == 0 {
		return nil, nil
	}
	idx := 0
	if s.shuffle {
		idx = rand.Intn(len(pool))
	} else {
		idx = int(time.Now().UnixNano()) % len(pool)
		if idx < 0 {
			idx = -idx
		}
	}
	chosen := pool[idx]
	// Ensure the image is available locally.
	for _, src := range srcs {
		if src.ID() == chosen.SourceId {
			if p, err := src.Fetch(ctx, chosen); err == nil {
				chosen.LocalPath = p
			}
			break
		}
	}
	return chosen, nil
}

// Next forces an immediate advance.
func (s *Scheduler) Next(ctx context.Context) (*pb.WallpaperMetadata, error) {
	if err := s.advance(ctx); err != nil {
		return nil, err
	}
	s.kick()
	return s.history.current(), nil
}

// Prev re-displays the previous wallpaper from history.
func (s *Scheduler) Prev(ctx context.Context) (*pb.WallpaperMetadata, error) {
	m := s.history.prev(1)
	if m == nil {
		return nil, errors.New("no previous wallpaper")
	}
	if m.LocalPath != "" {
		if err := s.display.Set(ctx, m.LocalPath); err != nil {
			return nil, err
		}
	}
	s.history.add(m)
	s.kick()
	return m, nil
}

// Pause stops automatic rotation.
func (s *Scheduler) Pause() { s.paused.Store(true) }

// Resume re-enables automatic rotation and resets the timer.
func (s *Scheduler) Resume() {
	s.paused.Store(false)
	s.kick()
}

// Paused reports whether rotation is paused.
func (s *Scheduler) Paused() bool { return s.paused.Load() }

// SetInterval updates the rotation interval and resets the timer.
func (s *Scheduler) SetInterval(d time.Duration) {
	if d <= 0 {
		return
	}
	s.mu.Lock()
	s.interval = d
	s.mu.Unlock()
	s.kick()
}

// SetSources replaces the source list (used on config hot-reload).
func (s *Scheduler) SetSources(srcs []source.Source) {
	s.mu.Lock()
	s.sources = srcs
	s.mu.Unlock()
}

// History returns up to limit recent entries.
func (s *Scheduler) History(limit int) []*pb.HistoryEntry { return s.history.recent(limit) }

// Current returns the currently displayed wallpaper.
func (s *Scheduler) Current() *pb.WallpaperMetadata { return s.history.current() }

// NextChangeIn returns the approximate time until the next automatic change.
func (s *Scheduler) NextChangeIn() time.Duration {
	if s.paused.Load() {
		return 0
	}
	elapsed := time.Since(s.lastTick)
	remaining := s.currentInterval() - elapsed
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

func (s *Scheduler) kick() {
	select {
	case s.resetCh <- struct{}{}:
	default:
	}
}
