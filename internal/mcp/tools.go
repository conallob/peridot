package mcp

import (
	"context"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/protobuf/types/known/durationpb"
)

// registerTools wires all peridot MCP tools onto srv, bridging to the daemon.
func registerTools(srv *server.MCPServer, b *bridge) {
	srv.AddTool(
		mcp.NewTool("peridot_next", mcp.WithDescription("Advance to the next wallpaper.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_Next{Next: &pb.NextCommand{}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_prev", mcp.WithDescription("Return to the previous wallpaper.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_Prev{Prev: &pb.PrevCommand{}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_pause", mcp.WithDescription("Pause automatic wallpaper rotation.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_Pause{Pause: &pb.PauseCommand{}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_resume", mcp.WithDescription("Resume automatic wallpaper rotation.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_Resume{Resume: &pb.ResumeCommand{}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_status", mcp.WithDescription("Get current daemon status, including the active wallpaper and cache stats.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_Status{Status: &pb.StatusCommand{}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_history",
			mcp.WithDescription("List recently displayed wallpapers."),
			mcp.WithNumber("limit", mcp.Description("Maximum number of entries to return.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			limit := int32(req.GetInt("limit", 10))
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_History{History: &pb.HistoryCommand{Limit: limit}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_list_sources", mcp.WithDescription("List configured wallpaper sources and their health.")),
		func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_ListSources{ListSources: &pb.ListSourcesCommand{}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_fetch_source",
			mcp.WithDescription("Trigger a fetch for new wallpapers from a source."),
			mcp.WithString("source_id", mcp.Required(), mcp.Description("ID of the source to fetch.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := req.RequireString("source_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_FetchSource{FetchSource: &pb.FetchSourceCommand{SourceId: id}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("peridot_set_interval",
			mcp.WithDescription("Set the wallpaper rotation interval."),
			mcp.WithString("interval", mcp.Required(), mcp.Description("Go duration string, e.g. \"30m\" or \"1h\".")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			s, err := req.RequireString("interval")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			d, err := time.ParseDuration(s)
			if err != nil {
				return mcp.NewToolResultError("invalid interval: " + err.Error()), nil
			}
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_SetInterval{SetInterval: &pb.SetIntervalCommand{Interval: durationpb.New(d)}}})
		},
	)

	srv.AddTool(
		mcp.NewTool("wallpaper_stats",
			mcp.WithDescription("Show wallpaper display statistics: totals, top wallpapers, per-source breakdown, and an hourly histogram."),
			mcp.WithNumber("days", mcp.Description("Number of days to include (0 = all time). Default 7.")),
			mcp.WithNumber("top", mcp.Description("Number of top wallpapers to return. Default 10.")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			days := int32(req.GetInt("days", 7))
			top := int32(req.GetInt("top", 10))
			return b.call(&pb.CommandRequest{Command: &pb.CommandRequest_Stats{Stats: &pb.StatsCommand{Days: days, Top: top}}})
		},
	)
}

// call sends req and renders the response as a JSON tool result.
func (b *bridge) call(req *pb.CommandRequest) (*mcp.CallToolResult, error) {
	resp, err := b.send(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(jsonResponse(resp)), nil
}
