package main

import (
	"context"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/cache"
	"github.com/conallob/peridot/internal/scheduler"
	"github.com/conallob/peridot/internal/source"
	"google.golang.org/protobuf/types/known/durationpb"
)

// daemon holds the live runtime objects the IPC handler operates on.
type daemon struct {
	sched   *scheduler.Scheduler
	cache   *cache.Manager
	sources []source.Source
}

// handle dispatches a CommandRequest to the appropriate runtime action.
func (d *daemon) handle(ctx context.Context, req *pb.CommandRequest) *pb.CommandResponse {
	switch c := req.Command.(type) {
	case *pb.CommandRequest_Next:
		m, err := d.sched.Next(ctx)
		return wallpaperResp(m, err)
	case *pb.CommandRequest_Prev:
		m, err := d.sched.Prev(ctx)
		return wallpaperResp(m, err)
	case *pb.CommandRequest_Pause:
		d.sched.Pause()
		return &pb.CommandResponse{Ok: true}
	case *pb.CommandRequest_Resume:
		d.sched.Resume()
		return &pb.CommandResponse{Ok: true}
	case *pb.CommandRequest_Status:
		return d.status()
	case *pb.CommandRequest_History:
		limit := int(c.History.Limit)
		return &pb.CommandResponse{Ok: true, Payload: &pb.CommandResponse_History{
			History: &pb.HistoryPayload{Entries: d.sched.History(limit)},
		}}
	case *pb.CommandRequest_SetInterval:
		if c.SetInterval.Interval != nil {
			d.sched.SetInterval(c.SetInterval.Interval.AsDuration())
		}
		return &pb.CommandResponse{Ok: true}
	case *pb.CommandRequest_ListSources:
		return &pb.CommandResponse{Ok: true, Payload: &pb.CommandResponse_Sources{
			Sources: &pb.SourcesPayload{Sources: d.sourceStatuses(ctx)},
		}}
	case *pb.CommandRequest_FetchSource:
		queued := d.fetchSource(ctx, c.FetchSource.SourceId)
		return &pb.CommandResponse{Ok: true, Payload: &pb.CommandResponse_Fetch{
			Fetch: &pb.FetchPayload{Queued: queued},
		}}
	case *pb.CommandRequest_Stats:
		return d.stats(c.Stats)
	default:
		return &pb.CommandResponse{Ok: false, Error: "unknown command"}
	}
}

func (d *daemon) status() *pb.CommandResponse {
	var cacheStatus *pb.CacheStatus
	if d.cache != nil {
		if cs, err := d.cache.Status(); err == nil {
			cacheStatus = cs
		}
	}
	return &pb.CommandResponse{Ok: true, Payload: &pb.CommandResponse_Status{
		Status: &pb.StatusPayload{
			Current:      d.sched.Current(),
			Paused:       d.sched.Paused(),
			NextChangeIn: durationFromDuration(d.sched.NextChangeIn()),
			Cache:        cacheStatus,
			Sources:      d.sourceStatuses(context.Background()),
		},
	}}
}

func (d *daemon) stats(c *pb.StatsCommand) *pb.CommandResponse {
	if d.cache == nil {
		return &pb.CommandResponse{Ok: false, Error: "no cache/display log available"}
	}
	days := int(c.Days)
	topN := int(c.Top)
	if topN <= 0 {
		topN = 10
	}
	total, top, sources, hourly, err := d.cache.Log().QueryStats(days, topN)
	if err != nil {
		return &pb.CommandResponse{Ok: false, Error: err.Error()}
	}

	names := make(map[string]string, len(d.sources))
	for _, s := range d.sources {
		names[s.ID()] = s.DisplayName()
	}

	topWallpapers := make([]*pb.WallpaperStat, 0, len(top))
	for _, w := range top {
		topWallpapers = append(topWallpapers, &pb.WallpaperStat{
			Wallpaper: &pb.WallpaperMetadata{
				Id:       w.WallpaperID,
				Title:    w.Title,
				SourceId: w.SourceID,
			},
			DisplayCount: w.DisplayCount,
		})
	}

	breakdown := make([]*pb.SourceStat, 0, len(sources))
	for _, s := range sources {
		display := names[s.SourceID]
		if display == "" {
			display = s.SourceID
		}
		breakdown = append(breakdown, &pb.SourceStat{
			SourceId:     s.SourceID,
			DisplayName:  display,
			DisplayCount: s.DisplayCount,
		})
	}

	hourly24 := make([]int32, 24)
	for i := 0; i < 24; i++ {
		hourly24[i] = hourly[i]
	}

	return &pb.CommandResponse{Ok: true, Payload: &pb.CommandResponse_StatsPayload{
		StatsPayload: &pb.StatsPayload{
			TotalDisplays:   total,
			Days:            c.Days,
			TopWallpapers:   topWallpapers,
			SourceBreakdown: breakdown,
			HourlyCounts:    hourly24,
		},
	}}
}

func (d *daemon) sourceStatuses(ctx context.Context) []*pb.SourceStatus {
	out := make([]*pb.SourceStatus, 0, len(d.sources))
	for _, s := range d.sources {
		out = append(out, &pb.SourceStatus{
			Id:          s.ID(),
			DisplayName: s.DisplayName(),
			Type:        s.Type(),
			Healthy:     s.Healthy(ctx),
		})
	}
	return out
}

func (d *daemon) fetchSource(ctx context.Context, id string) int32 {
	for _, s := range d.sources {
		if s.ID() != id {
			continue
		}
		items, err := s.Catalogue(ctx)
		if err != nil {
			return 0
		}
		var queued int32
		for _, m := range items {
			if path, err := s.Fetch(ctx, m); err == nil {
				m.LocalPath = path
				if d.cache != nil {
					_ = d.cache.Put(m)
				}
				queued++
			}
		}
		return queued
	}
	return 0
}

func wallpaperResp(m *pb.WallpaperMetadata, err error) *pb.CommandResponse {
	if err != nil {
		return &pb.CommandResponse{Ok: false, Error: err.Error()}
	}
	return &pb.CommandResponse{Ok: true, Payload: &pb.CommandResponse_Wallpaper{
		Wallpaper: &pb.WallpaperPayload{Current: m},
	}}
}

func durationFromDuration(d time.Duration) *durationpb.Duration {
	return durationpb.New(d)
}
