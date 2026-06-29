package ipc

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/protobuf/proto"
)

func TestRoundTripCommandRequest(t *testing.T) {
	req := &pb.CommandRequest{
		Command: &pb.CommandRequest_Next{Next: &pb.NextCommand{}},
	}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, req); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	got := &pb.CommandRequest{}
	if err := ReadFrame(&buf, got); err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !proto.Equal(req, got) {
		t.Fatalf("round-trip mismatch: got %v want %v", got, req)
	}
}

func TestRoundTripCommandResponse(t *testing.T) {
	resp := &pb.CommandResponse{Ok: true, Error: ""}
	var buf bytes.Buffer
	if err := WriteFrame(&buf, resp); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}
	got := &pb.CommandResponse{}
	if err := ReadFrame(&buf, got); err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if !proto.Equal(resp, got) {
		t.Fatalf("round-trip mismatch: got %v want %v", got, resp)
	}
}

func TestReadFrameTooLarge(t *testing.T) {
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], maxFrameSize+1)
	r := bytes.NewReader(hdr[:])
	err := ReadFrame(r, &pb.CommandResponse{})
	if err == nil {
		t.Fatal("expected error for oversize frame")
	}
	if !strings.Contains(err.Error(), "frame too large") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadFramePartialHeader(t *testing.T) {
	// Only two bytes of the 4-byte header.
	r := bytes.NewReader([]byte{0x00, 0x01})
	err := ReadFrame(r, &pb.CommandResponse{})
	if err == nil {
		t.Fatal("expected error for partial header")
	}
}

func TestReadFramePartialBody(t *testing.T) {
	// Header claims 10 bytes, but body is short.
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], 10)
	data := append(hdr[:], []byte{0x01, 0x02, 0x03}...)
	r := bytes.NewReader(data)
	err := ReadFrame(r, &pb.CommandResponse{})
	if err == nil {
		t.Fatal("expected error for partial body")
	}
	if err != io.ErrUnexpectedEOF {
		t.Logf("got error: %v (acceptable)", err)
	}
}

func TestReadFrameEmpty(t *testing.T) {
	err := ReadFrame(bytes.NewReader(nil), &pb.CommandResponse{})
	if err == nil {
		t.Fatal("expected error reading from empty reader")
	}
}
