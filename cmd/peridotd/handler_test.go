package main

import (
	"context"
	"os"
	"testing"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/cache"
	"github.com/conallob/peridot/internal/display"
	"github.com/conallob/peridot/internal/scheduler"
	"github.com/conallob/peridot/internal/source"
)

// stubSource satisfies source.Source for testing.
type stubSource struct {
	id    string
	items []*pb.WallpaperMetadata
}

func (s *stubSource) ID() string          { return s.id }
func (s *stubSource) DisplayName() string { return s.id + "-display" }
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

func newTestDaemon(t *testing.T, srcs ...source.Source) *daemon {
	t.Helper()
	eng, _ := display.New()
	sched := scheduler.New(scheduler.Options{
		Display:  eng,
		Sources:  srcs,
		Interval: time.Hour,
	})
	dir := t.TempDir()
	mgr, err := cache.New(cache.Options{Dir: dir})
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return &daemon{sched: sched, cache: mgr, sources: srcs}
}

func TestHandlePauseResume(t *testing.T) {
	d := newTestDaemon(t)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_Pause{Pause: &pb.PauseCommand{}}})
	if !resp.Ok {
		t.Fatalf("Pause: %v", resp.Error)
	}
	if !d.sched.Paused() {
		t.Fatal("scheduler should be paused")
	}

	resp = d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_Resume{Resume: &pb.ResumeCommand{}}})
	if !resp.Ok {
		t.Fatalf("Resume: %v", resp.Error)
	}
	if d.sched.Paused() {
		t.Fatal("scheduler should not be paused after resume")
	}
}

func TestHandleSetInterval(t *testing.T) {
	d := newTestDaemon(t)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_SetInterval{
		SetInterval: &pb.SetIntervalCommand{Interval: durationFromDuration(10 * time.Minute)},
	}})
	if !resp.Ok {
		t.Fatalf("SetInterval: %v", resp.Error)
	}
}

func TestHandleHistory(t *testing.T) {
	d := newTestDaemon(t)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_History{
		History: &pb.HistoryCommand{Limit: 5},
	}})
	if !resp.Ok {
		t.Fatalf("History: %v", resp.Error)
	}
	hp, ok := resp.Payload.(*pb.CommandResponse_History)
	if !ok {
		t.Fatal("expected history payload")
	}
	_ = hp.History.Entries
}

func TestHandleStatus(t *testing.T) {
	d := newTestDaemon(t)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_Status{Status: &pb.StatusCommand{}}})
	if !resp.Ok {
		t.Fatalf("Status: %v", resp.Error)
	}
}

func TestHandleListSources(t *testing.T) {
	src := &stubSource{id: "local"}
	d := newTestDaemon(t, src)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_ListSources{ListSources: &pb.ListSourcesCommand{}}})
	if !resp.Ok {
		t.Fatalf("ListSources: %v", resp.Error)
	}
	sp, ok := resp.Payload.(*pb.CommandResponse_Sources)
	if !ok {
		t.Fatal("expected sources payload")
	}
	if len(sp.Sources.Sources) != 1 || sp.Sources.Sources[0].Id != "local" {
		t.Fatalf("sources = %v", sp.Sources.Sources)
	}
}

func TestHandleFetchSource(t *testing.T) {
	img := writeTempImage(t)
	src := &stubSource{id: "s1", items: []*pb.WallpaperMetadata{
		{Id: "w1", SourceId: "s1", LocalPath: img},
	}}
	d := newTestDaemon(t, src)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_FetchSource{
		FetchSource: &pb.FetchSourceCommand{SourceId: "s1"},
	}})
	if !resp.Ok {
		t.Fatalf("FetchSource: %v", resp.Error)
	}
}

func TestHandleFetchSource_NotFound(t *testing.T) {
	d := newTestDaemon(t)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_FetchSource{
		FetchSource: &pb.FetchSourceCommand{SourceId: "nonexistent"},
	}})
	if !resp.Ok {
		t.Fatalf("FetchSource unknown id should still return ok: %v", resp.Error)
	}
	fp := resp.Payload.(*pb.CommandResponse_Fetch)
	if fp.Fetch.Queued != 0 {
		t.Fatalf("queued = %d, want 0", fp.Fetch.Queued)
	}
}

func TestHandleStats(t *testing.T) {
	d := newTestDaemon(t)
	ctx := context.Background()

	_ = d.cache.Log().Record("w1", "s1", "Title")

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_Stats{
		Stats: &pb.StatsCommand{Days: 7, Top: 5},
	}})
	if !resp.Ok {
		t.Fatalf("Stats: %v", resp.Error)
	}
	sp, ok := resp.Payload.(*pb.CommandResponse_StatsPayload)
	if !ok {
		t.Fatal("expected stats payload")
	}
	if sp.StatsPayload.TotalDisplays != 1 {
		t.Fatalf("total = %d, want 1", sp.StatsPayload.TotalDisplays)
	}
}

func TestHandleStats_NoCache(t *testing.T) {
	eng, _ := display.New()
	sched := scheduler.New(scheduler.Options{Display: eng, Interval: time.Hour})
	d := &daemon{sched: sched, cache: nil}
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{Command: &pb.CommandRequest_Stats{
		Stats: &pb.StatsCommand{Days: 7},
	}})
	if resp.Ok {
		t.Fatal("expected error when cache is nil")
	}
}

func TestHandleUnknownCommand(t *testing.T) {
	d := newTestDaemon(t)
	ctx := context.Background()

	resp := d.handle(ctx, &pb.CommandRequest{})
	if resp.Ok {
		t.Fatal("unknown command should return Ok=false")
	}
}

func TestWallpaperResp_Error(t *testing.T) {
	resp := wallpaperResp(nil, context.Canceled)
	if resp.Ok {
		t.Fatal("error should produce Ok=false")
	}
}

func TestWallpaperResp_OK(t *testing.T) {
	m := &pb.WallpaperMetadata{Id: "w1"}
	resp := wallpaperResp(m, nil)
	if !resp.Ok {
		t.Fatalf("wallpaperResp: %v", resp.Error)
	}
}

func TestDurationFromDuration(t *testing.T) {
	d := durationFromDuration(5 * time.Minute)
	if d.AsDuration() != 5*time.Minute {
		t.Fatalf("got %v, want 5m", d.AsDuration())
	}
}

func writeTempImage(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "img*.jpg")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.Write([]byte("fake"))
	_ = f.Close()
	return f.Name()
}
