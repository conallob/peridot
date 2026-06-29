package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/ipc"
	"github.com/spf13/cobra"
)

func statsCmd() *cobra.Command {
	var days int
	var top int
	c := &cobra.Command{
		Use:   "stats",
		Short: "Show wallpaper display statistics",
		RunE: func(_ *cobra.Command, _ []string) error {
			resp, err := ipc.NewClient("").Send(&pb.CommandRequest{Command: &pb.CommandRequest_Stats{
				Stats: &pb.StatsCommand{Days: int32(days), Top: int32(top)},
			}})
			if err != nil {
				return err
			}
			if !resp.Ok {
				return fmt.Errorf("%s", resp.Error)
			}
			payload, ok := resp.Payload.(*pb.CommandResponse_StatsPayload)
			if !ok || payload.StatsPayload == nil {
				return fmt.Errorf("daemon returned no stats")
			}
			renderStats(os.Stdout, payload.StatsPayload)
			return nil
		},
	}
	c.Flags().IntVar(&days, "days", 7, "number of days to include (0 = all time)")
	c.Flags().IntVar(&top, "top", 10, "number of top wallpapers to show")
	return c
}

// bar scales count against max into roughly width filled blocks.
func bar(count, max int32, width int) string {
	if max <= 0 || count <= 0 {
		return ""
	}
	n := int((int64(count) * int64(width)) / int64(max))
	if n < 1 {
		n = 1
	}
	return strings.Repeat("█", n)
}

func renderStats(w *os.File, p *pb.StatsPayload) {
	period := "all time"
	if p.Days > 0 {
		period = fmt.Sprintf("last %d days", p.Days)
	}
	_, _ = fmt.Fprintf(w, "\nPeridot stats — %s\n\n", period)
	_, _ = fmt.Fprintf(w, "  Total displays: %d\n", p.TotalDisplays)

	const rule = "  ─────────────────────────────────────────────────────────"

	// Top wallpapers.
	if len(p.TopWallpapers) > 0 {
		var maxCount int32
		for _, ws := range p.TopWallpapers {
			if ws.DisplayCount > maxCount {
				maxCount = ws.DisplayCount
			}
		}
		_, _ = fmt.Fprintf(w, "\n  Top wallpapers:\n%s\n", rule)
		tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "   #\tTitle\tSource\tCount\tBar")
		for i, ws := range p.TopWallpapers {
			title := "(unknown)"
			source := ""
			if ws.Wallpaper != nil {
				if ws.Wallpaper.Title != "" {
					title = ws.Wallpaper.Title
				}
				source = ws.Wallpaper.SourceId
			}
			_, _ = fmt.Fprintf(tw, "   %d\t%s\t%s\t%d\t%s\n",
				i+1, title, source, ws.DisplayCount, bar(ws.DisplayCount, maxCount, 12))
		}
		_ = tw.Flush()
		_, _ = fmt.Fprintln(w, rule)
	}

	// By source.
	if len(p.SourceBreakdown) > 0 {
		_, _ = fmt.Fprintf(w, "\n  By source:\n%s\n", rule)
		tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "   Source\tCount\t%")
		for _, ss := range p.SourceBreakdown {
			pct := 0
			if p.TotalDisplays > 0 {
				pct = int((int64(ss.DisplayCount)*100 + int64(p.TotalDisplays)/2) / int64(p.TotalDisplays))
			}
			name := ss.DisplayName
			if name == "" {
				name = ss.SourceId
			}
			_, _ = fmt.Fprintf(tw, "   %s\t%d\t%d%%\n", name, ss.DisplayCount, pct)
		}
		_ = tw.Flush()
		_, _ = fmt.Fprintln(w, rule)
	}

	// Hourly heatmap.
	if len(p.HourlyCounts) == 24 {
		var maxHour int32
		for _, c := range p.HourlyCounts {
			if c > maxHour {
				maxHour = c
			}
		}
		_, _ = fmt.Fprintf(w, "\n  Hourly activity (displays per hour):\n")
		for h := 0; h < 24; h++ {
			_, _ = fmt.Fprintf(w, "  %02d %s\n", h, bar(p.HourlyCounts[h], maxHour, 20))
		}
	}
}
