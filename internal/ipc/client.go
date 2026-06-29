package ipc

import (
	"net"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
)

// Client sends command requests to a running daemon over the Unix socket.
type Client struct {
	path string
}

// NewClient returns a client targeting socketPath. If empty, the default path
// is used.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath()
	}
	return &Client{path: socketPath}
}

// Send dials the daemon, writes req, and returns the response.
func (c *Client) Send(req *pb.CommandRequest) (*pb.CommandResponse, error) {
	conn, err := net.DialTimeout("unix", c.path, 5*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	if err := WriteFrame(conn, req); err != nil {
		return nil, err
	}
	resp := &pb.CommandResponse{}
	if err := ReadFrame(conn, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
