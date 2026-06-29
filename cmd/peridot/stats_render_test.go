package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func captureRender(p *pb.StatsPayload) string {
	// renderStats writes to *os.File; use a real pipe to capture it.
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	renderStats(w, p)
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestRenderStats_AllTime(t *testing.T) {
	out := captureRender(&pb.StatsPayload{Days: 0, TotalDisplays: 42})
	if !strings.Contains(out, "all time") {
		t.Fatalf("expected 'all time', got: %s", out)
	}
	if !strings.Contains(out, "42") {
		t.Fatalf("expected total 42, got: %s", out)
	}
}

func TestRenderStats_Days(t *testing.T) {
	out := captureRender(&pb.StatsPayload{Days: 7, TotalDisplays: 10})
	if !strings.Contains(out, "last 7 days") {
		t.Fatalf("expected 'last 7 days', got: %s", out)
	}
}

func TestRenderStats_TopWallpapers(t *testing.T) {
	payload := &pb.StatsPayload{
		Days:          7,
		TotalDisplays: 5,
		TopWallpapers: []*pb.WallpaperStat{
			{Wallpaper: &pb.WallpaperMetadata{Title: "Sunset", SourceId: "local"}, DisplayCount: 5},
			{Wallpaper: &pb.WallpaperMetadata{Title: "Forest", SourceId: "drive"}, DisplayCount: 2},
		},
	}
	out := captureRender(payload)
	if !strings.Contains(out, "Sunset") {
		t.Fatalf("expected 'Sunset', got: %s", out)
	}
	if !strings.Contains(out, "Forest") {
		t.Fatalf("expected 'Forest', got: %s", out)
	}
}

func TestRenderStats_TopWallpaperNilMeta(t *testing.T) {
	payload := &pb.StatsPayload{
		Days:          1,
		TotalDisplays: 1,
		TopWallpapers: []*pb.WallpaperStat{
			{Wallpaper: nil, DisplayCount: 1},
		},
	}
	out := captureRender(payload)
	if !strings.Contains(out, "unknown") {
		t.Fatalf("expected '(unknown)' for nil wallpaper, got: %s", out)
	}
}

func TestRenderStats_SourceBreakdown(t *testing.T) {
	payload := &pb.StatsPayload{
		Days:          7,
		TotalDisplays: 10,
		SourceBreakdown: []*pb.SourceStat{
			{SourceId: "local", DisplayName: "My Local", DisplayCount: 7},
			{SourceId: "drive", DisplayName: "", DisplayCount: 3},
		},
	}
	out := captureRender(payload)
	if !strings.Contains(out, "My Local") {
		t.Fatalf("expected 'My Local', got: %s", out)
	}
	if !strings.Contains(out, "drive") {
		t.Fatalf("expected fallback to SourceId 'drive', got: %s", out)
	}
}

func TestRenderStats_HourlyHeatmap(t *testing.T) {
	counts := make([]int32, 24)
	counts[9] = 5
	counts[14] = 10
	payload := &pb.StatsPayload{
		Days:          7,
		TotalDisplays: 15,
		HourlyCounts:  counts,
	}
	out := captureRender(payload)
	if !strings.Contains(out, "Hourly") {
		t.Fatalf("expected hourly section, got: %s", out)
	}
	if !strings.Contains(out, "09") {
		t.Fatalf("expected hour 09, got: %s", out)
	}
}

func TestBar(t *testing.T) {
	if b := bar(0, 10, 20); b != "" {
		t.Fatalf("bar(0,...) = %q, want empty", b)
	}
	if b := bar(10, 0, 20); b != "" {
		t.Fatalf("bar(x,0,...) = %q, want empty (max=0)", b)
	}
	full := bar(10, 10, 10)
	if len([]rune(full)) != 10 {
		t.Fatalf("full bar width = %d, want 10", len([]rune(full)))
	}
	half := bar(5, 10, 10)
	if len([]rune(half)) < 1 {
		t.Fatalf("half bar should have at least 1 block")
	}
}
