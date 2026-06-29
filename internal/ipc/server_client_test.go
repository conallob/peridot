package ipc

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
)

func startTestServer(t *testing.T, handler Handler) (socketPath string) {
	t.Helper()
	socketPath = filepath.Join(t.TempDir(), "test.sock")
	srv := NewServer(socketPath, handler)
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan struct{})
	go func() {
		// Signal readiness after a small delay (listener set up).
		time.AfterFunc(20*time.Millisecond, func() { close(ready) })
		_ = srv.Serve(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		_ = srv.Close()
	})
	<-ready
	return socketPath
}

func TestClientSend_OKResponse(t *testing.T) {
	path := startTestServer(t, func(_ context.Context, req *pb.CommandRequest) *pb.CommandResponse {
		return &pb.CommandResponse{Ok: true}
	})

	client := NewClient(path)
	resp, err := client.Send(&pb.CommandRequest{Command: &pb.CommandRequest_Next{Next: &pb.NextCommand{}}})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("resp.Ok = false")
	}
}

func TestClientSend_ErrorResponse(t *testing.T) {
	path := startTestServer(t, func(_ context.Context, req *pb.CommandRequest) *pb.CommandResponse {
		return &pb.CommandResponse{Ok: false, Error: "something went wrong"}
	})

	client := NewClient(path)
	resp, err := client.Send(&pb.CommandRequest{Command: &pb.CommandRequest_Pause{Pause: &pb.PauseCommand{}}})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.Ok {
		t.Fatal("expected Ok=false")
	}
	if resp.Error != "something went wrong" {
		t.Fatalf("Error = %q", resp.Error)
	}
}

func TestClientSend_MultipleRequests(t *testing.T) {
	var count int
	path := startTestServer(t, func(_ context.Context, req *pb.CommandRequest) *pb.CommandResponse {
		count++
		return &pb.CommandResponse{Ok: true}
	})

	client := NewClient(path)
	for i := 0; i < 5; i++ {
		if _, err := client.Send(&pb.CommandRequest{Command: &pb.CommandRequest_Next{Next: &pb.NextCommand{}}}); err != nil {
			t.Fatalf("Send %d: %v", i, err)
		}
	}
}

func TestClientSend_NoServer(t *testing.T) {
	client := NewClient(filepath.Join(t.TempDir(), "nonexistent.sock"))
	_, err := client.Send(&pb.CommandRequest{Command: &pb.CommandRequest_Next{Next: &pb.NextCommand{}}})
	if err == nil {
		t.Fatal("expected error dialing nonexistent socket")
	}
}

func TestServer_NilHandlerResponse(t *testing.T) {
	path := startTestServer(t, func(_ context.Context, req *pb.CommandRequest) *pb.CommandResponse {
		return nil
	})

	client := NewClient(path)
	resp, err := client.Send(&pb.CommandRequest{Command: &pb.CommandRequest_Next{Next: &pb.NextCommand{}}})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if resp.Ok {
		t.Fatal("nil handler response should produce Ok=false")
	}
}

func TestDefaultSocketPath(t *testing.T) {
	p := DefaultSocketPath()
	if p == "" {
		t.Fatal("DefaultSocketPath returned empty string")
	}
}
