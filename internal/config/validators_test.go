package config

import (
	"strings"
	"testing"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestIntervalValidator(t *testing.T) {
	v := IntervalValidator()

	if err := v.Validate(&pb.Config{}); err == nil {
		t.Fatal("expected error for missing interval")
	}

	tooShort := &pb.Config{Scheduler: &pb.SchedulerConfig{Interval: durationpb.New(5 * time.Second)}}
	if err := v.Validate(tooShort); err == nil {
		t.Fatal("expected error for sub-10s interval")
	} else if !strings.Contains(err.Error(), "10s") {
		t.Fatalf("unexpected message: %v", err)
	}

	ok := &pb.Config{Scheduler: &pb.SchedulerConfig{Interval: durationpb.New(30 * time.Minute)}}
	if err := v.Validate(ok); err != nil {
		t.Fatalf("valid interval rejected: %v", err)
	}
}

func TestCacheSizeValidator(t *testing.T) {
	v := CacheSizeValidator()

	if err := v.Validate(&pb.Config{}); err != nil {
		t.Fatalf("nil cache should be valid: %v", err)
	}

	if err := v.Validate(&pb.Config{Cache: &pb.CacheConfig{MaxSizeBytes: -1}}); err == nil {
		t.Fatal("expected error for negative max size")
	}
	if err := v.Validate(&pb.Config{Cache: &pb.CacheConfig{MaxItems: -5}}); err == nil {
		t.Fatal("expected error for negative max items")
	}

	ok := &pb.Config{Cache: &pb.CacheConfig{MaxSizeBytes: 1 << 30, MaxItems: 100}}
	if err := v.Validate(ok); err != nil {
		t.Fatalf("valid cache rejected: %v", err)
	}
}

func TestSourceValidator(t *testing.T) {
	v := SourceValidator()

	missingID := &pb.Config{Sources: []*pb.SourceConfig{{Type: pb.SourceType_SOURCE_TYPE_LOCAL}}}
	if err := v.Validate(missingID); err == nil {
		t.Fatal("expected error for missing source id")
	}

	dup := &pb.Config{Sources: []*pb.SourceConfig{
		{Id: "a", Type: pb.SourceType_SOURCE_TYPE_LOCAL},
		{Id: "a", Type: pb.SourceType_SOURCE_TYPE_LOCAL},
	}}
	if err := v.Validate(dup); err == nil {
		t.Fatal("expected error for duplicate id")
	} else if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("unexpected message: %v", err)
	}

	unknownType := &pb.Config{Sources: []*pb.SourceConfig{{Id: "a"}}}
	if err := v.Validate(unknownType); err == nil {
		t.Fatal("expected error for unspecified type")
	}

	ok := &pb.Config{Sources: []*pb.SourceConfig{
		{Id: "a", Type: pb.SourceType_SOURCE_TYPE_LOCAL},
		{Id: "b", Type: pb.SourceType_SOURCE_TYPE_URL},
	}}
	if err := v.Validate(ok); err != nil {
		t.Fatalf("valid sources rejected: %v", err)
	}
}

func TestResolutionValidator(t *testing.T) {
	v := ResolutionValidator()

	bad := &pb.Config{Sources: []*pb.SourceConfig{{
		Id:     "db",
		Type:   pb.SourceType_SOURCE_TYPE_DIGITAL_BLASPHEMY,
		Config: &pb.SourceConfig_DigitalBlasphemy{DigitalBlasphemy: &pb.DigitalBlasphemySourceConfig{PreferredResolution: "4k"}},
	}}}
	if err := v.Validate(bad); err == nil {
		t.Fatal("expected error for malformed resolution")
	} else if !strings.Contains(err.Error(), "resolution") {
		t.Fatalf("unexpected message: %v", err)
	}

	ok := &pb.Config{Sources: []*pb.SourceConfig{{
		Id:     "db",
		Type:   pb.SourceType_SOURCE_TYPE_DIGITAL_BLASPHEMY,
		Config: &pb.SourceConfig_DigitalBlasphemy{DigitalBlasphemy: &pb.DigitalBlasphemySourceConfig{PreferredResolution: "3840x2160"}},
	}}}
	if err := v.Validate(ok); err != nil {
		t.Fatalf("valid resolution rejected: %v", err)
	}

	empty := &pb.Config{Sources: []*pb.SourceConfig{{
		Id:     "gd",
		Type:   pb.SourceType_SOURCE_TYPE_GOOGLE_DRIVE,
		Config: &pb.SourceConfig_GoogleDrive{GoogleDrive: &pb.GoogleDriveSourceConfig{}},
	}}}
	if err := v.Validate(empty); err != nil {
		t.Fatalf("empty resolution should be valid: %v", err)
	}
}
