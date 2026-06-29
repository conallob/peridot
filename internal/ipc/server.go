package ipc

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	pb "github.com/conallob/peridot/gen/peridot"
)

// Handler processes a CommandRequest and returns a CommandResponse.
type Handler func(ctx context.Context, req *pb.CommandRequest) *pb.CommandResponse

// Server listens on a Unix domain socket and dispatches command requests.
type Server struct {
	path    string
	handler Handler

	mu       sync.Mutex
	listener net.Listener
}

// DefaultSocketPath returns ~/.peridot/peridot.sock.
func DefaultSocketPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".peridot", "peridot.sock")
}

// NewServer creates a server bound to socketPath using handler.
func NewServer(socketPath string, handler Handler) *Server {
	return &Server{path: socketPath, handler: handler}
}

// Serve listens and accepts connections until ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	// Remove a stale socket file from a previous run.
	_ = os.Remove(s.path)

	l, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.listener = l
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return nil
			}
			return err
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	for {
		req := &pb.CommandRequest{}
		if err := ReadFrame(conn, req); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}
		resp := s.handler(ctx, req)
		if resp == nil {
			resp = &pb.CommandResponse{Ok: false, Error: "nil response"}
		}
		if err := WriteFrame(conn, resp); err != nil {
			return
		}
	}
}

// Close stops the server and removes the socket file.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		_ = os.Remove(s.path)
		return err
	}
	return nil
}
