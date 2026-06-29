package mcp

import (
	"fmt"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/ipc"
	"google.golang.org/protobuf/encoding/protojson"
)

// bridge forwards MCP tool calls to the daemon over the IPC client.
type bridge struct {
	client *ipc.Client
}

func newBridge(socketPath string) *bridge {
	return &bridge{client: ipc.NewClient(socketPath)}
}

// send dispatches a CommandRequest and returns the response, surfacing the
// daemon-level error if ok is false.
func (b *bridge) send(req *pb.CommandRequest) (*pb.CommandResponse, error) {
	resp, err := b.client.Send(req)
	if err != nil {
		return nil, fmt.Errorf("daemon unreachable: %w", err)
	}
	if !resp.Ok {
		return resp, fmt.Errorf("daemon error: %s", resp.Error)
	}
	return resp, nil
}

// jsonResponse renders any CommandResponse payload as JSON text.
func jsonResponse(resp *pb.CommandResponse) string {
	b, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(resp)
	if err != nil {
		return resp.String()
	}
	return string(b)
}
