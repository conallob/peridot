package scheduler

import (
	"context"
	"testing"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/display"
	"github.com/conallob/peridot/internal/source"
)

// stubSource is a minimal in-memory source for testing.
type stubSource struct {
	id    string
	items []*pb.WallpaperMetadata
}

func (s *stubSource) ID() string          { return s.id }
func (s *stubSource) DisplayName() string { return s.id }
func (s *stubSource) Type() pb.SourceType { return pb.SourceType_SOURCE_TYPE_LOCAL }
func (s *stubSource) Catalogue(_ context.Context) ([]*pb.WallpaperMetadata, error) {
	return s.items, nil
}
func (s *stubSource) NewSince(_ context.Context, _ time.Time) ([]*pb.WallpaperMetadata, error) {
	return s.items, nil
}
func (s *stubSource) Fetch(_ context.Context, m *pb.WallpaperMetadata) (string, error) {
	return m.LocalPath, nil
}
func (s *stubSource) Healthy(_ context.Context) bool { return true }

func newTestScheduler(srcs ...source.Source) *Scheduler {
	eng, _ := display.New()
	return New(Options{
		Display:  eng,
		Interval: time.Hour,
		Sources:  srcs,
	})
}

func TestSchedulerPauseResume(t *testing.T) {
	s := newTestScheduler()
	if s.Paused() {
		t.Fatal("should not be paused initially")
	}
	s.Pause()
	if !s.Paused() {
		t.Fatal("should be paused after Pause()")
	}
	s.Resume()
	if s.Paused() {
		t.Fatal("should not be paused after Resume()")
	}
}

func TestSchedulerSetInterval(t *testing.T) {
	s := newTestScheduler()
	s.SetInterval(5 * time.Minute)
	if s.currentInterval() != 5*time.Minute {
		t.Fatalf("interval = %v, want 5m", s.currentInterval())
	}
	// Zero/negative should be ignored.
	s.SetInterval(0)
	if s.currentInterval() != 5*time.Minute {
		t.Fatalf("interval changed on 0 input: %v", s.currentInterval())
	}
}

func TestSchedulerSetSources(t *testing.T) {
	s := newTestScheduler()
	src := &stubSource{id: "new"}
	s.SetSources([]source.Source{src})
}

func TestSchedulerHistoryAndCurrent(t *testing.T) {
	s := newTestScheduler()
	if s.Current() != nil {
		t.Fatal("Current() should be nil before any advance")
	}
	if h := s.History(10); len(h) != 0 {
		t.Fatalf("History should be empty, got %d", len(h))
	}
}

func TestSchedulerNextChangeIn_Paused(t *testing.T) {
	s := newTestScheduler()
	s.Pause()
	if d := s.NextChangeIn(); d != 0 {
		t.Fatalf("NextChangeIn while paused = %v, want 0", d)
	}
}

func TestSchedulerNextChangeIn_Running(t *testing.T) {
	s := newTestScheduler()
	s.lastTick = time.Now()
	d := s.NextChangeIn()
	if d <= 0 || d > time.Hour {
		t.Fatalf("NextChangeIn = %v, want (0, 1h]", d)
	}
}

func TestSchedulerNew_DefaultInterval(t *testing.T) {
	eng, _ := display.New()
	s := New(Options{Display: eng, Interval: 0})
	if s.currentInterval() != 30*time.Minute {
		t.Fatalf("default interval = %v, want 30m", s.currentInterval())
	}
}

func TestSchedulerRecord_NilLog(t *testing.T) {
	s := newTestScheduler()
	// Should not panic when log is nil.
	s.record(&pb.WallpaperMetadata{Id: "x", SourceId: "s", Title: "t"})
	s.record(nil)
}
