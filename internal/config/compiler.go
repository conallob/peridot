package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Compile runs the 3-stage pipeline (parse -> translate -> validate) over the
// TOML file at path and returns the resulting protobuf Config. On success it
// also writes a compiled <path>.lock.pb alongside the source atomically.
func Compile(path string) (*pb.Config, error) {
	raw, err := parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	cfg, err := translate(raw)
	if err != nil {
		return nil, fmt.Errorf("translate: %w", err)
	}
	if err := validate(cfg, DefaultValidators()); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	if err := writeLock(path, cfg); err != nil {
		return nil, fmt.Errorf("write lock: %w", err)
	}
	return cfg, nil
}

// Validate compiles without writing the lock file. Useful for `config validate`.
func Validate(path string) (*pb.Config, error) {
	raw, err := parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	cfg, err := translate(raw)
	if err != nil {
		return nil, fmt.Errorf("translate: %w", err)
	}
	if err := validate(cfg, DefaultValidators()); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}
	return cfg, nil
}

// stage 1: parse
func parse(path string) (*rawConfig, error) {
	var raw rawConfig
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return nil, err
	}
	return &raw, nil
}

// stage 2: translate
func translate(raw *rawConfig) (*pb.Config, error) {
	cfg := &pb.Config{
		Scheduler: &pb.SchedulerConfig{
			Shuffle:    raw.Scheduler.Shuffle,
			PerDisplay: raw.Scheduler.PerDisplay,
		},
		Cache: &pb.CacheConfig{
			MaxItems:      raw.Cache.MaxItems,
			PrefetchAhead: raw.Cache.PrefetchAhead,
			Location:      raw.Cache.Location,
		},
	}

	if raw.Scheduler.Interval != "" {
		d, err := time.ParseDuration(raw.Scheduler.Interval)
		if err != nil {
			return nil, fmt.Errorf("scheduler.interval: %w", err)
		}
		cfg.Scheduler.Interval = durationpb.New(d)
	}
	for _, e := range raw.Scheduler.ChangeOn {
		switch strings.ToLower(e) {
		case "timer":
			cfg.Scheduler.ChangeOn = append(cfg.Scheduler.ChangeOn, pb.ChangeEvent_CHANGE_EVENT_TIMER)
		case "wake":
			cfg.Scheduler.ChangeOn = append(cfg.Scheduler.ChangeOn, pb.ChangeEvent_CHANGE_EVENT_WAKE)
		case "login":
			cfg.Scheduler.ChangeOn = append(cfg.Scheduler.ChangeOn, pb.ChangeEvent_CHANGE_EVENT_LOGIN)
		default:
			return nil, fmt.Errorf("unknown change_on event %q", e)
		}
	}

	if raw.Cache.MaxSize != "" {
		b, err := parseSize(raw.Cache.MaxSize)
		if err != nil {
			return nil, fmt.Errorf("cache.max_size: %w", err)
		}
		cfg.Cache.MaxSizeBytes = b
	}
	cfg.Cache.EvictionPolicy = parseEviction(raw.Cache.EvictionPolicy)

	for i := range raw.Sources {
		s, err := translateSource(&raw.Sources[i])
		if err != nil {
			return nil, err
		}
		cfg.Sources = append(cfg.Sources, s)
	}
	return cfg, nil
}

func translateSource(rs *rawSource) (*pb.SourceConfig, error) {
	sc := &pb.SourceConfig{
		Id:          rs.ID,
		DisplayName: rs.DisplayName,
		Weight:      rs.Weight,
	}
	switch strings.ToLower(rs.Type) {
	case "local":
		sc.Type = pb.SourceType_SOURCE_TYPE_LOCAL
		sc.Config = &pb.SourceConfig_Local{Local: &pb.LocalSourceConfig{
			Path:  rs.Path,
			Watch: rs.Watch,
		}}
	case "digital_blasphemy":
		sc.Type = pb.SourceType_SOURCE_TYPE_DIGITAL_BLASPHEMY
		db := &pb.DigitalBlasphemySourceConfig{
			Username:            rs.Username,
			KeychainService:     rs.KeychainService,
			PreferredResolution: rs.PreferredResolution,
			AutoDownloadNew:     rs.AutoDownloadNew,
		}
		if rs.CheckInterval != "" {
			d, err := time.ParseDuration(rs.CheckInterval)
			if err != nil {
				return nil, fmt.Errorf("source %q check_interval: %w", rs.ID, err)
			}
			db.CheckInterval = durationpb.New(d)
		}
		sc.Config = &pb.SourceConfig_DigitalBlasphemy{DigitalBlasphemy: db}
	case "google_drive":
		sc.Type = pb.SourceType_SOURCE_TYPE_GOOGLE_DRIVE
		sc.Config = &pb.SourceConfig_GoogleDrive{GoogleDrive: &pb.GoogleDriveSourceConfig{
			FolderId:            rs.FolderID,
			CredentialsPath:     rs.CredentialsPath,
			PreferredResolution: rs.PreferredResolution,
		}}
	case "url":
		sc.Type = pb.SourceType_SOURCE_TYPE_URL
		sc.Config = &pb.SourceConfig_Url{Url: &pb.URLSourceConfig{
			Endpoint: rs.Endpoint,
			Headers:  rs.Headers,
		}}
	default:
		return nil, fmt.Errorf("source %q: unknown type %q", rs.ID, rs.Type)
	}
	return sc, nil
}

// stage 3: validate
func validate(cfg *pb.Config, vs []Validator) error {
	for _, v := range vs {
		if err := v.Validate(cfg); err != nil {
			return err
		}
	}
	return nil
}

func writeLock(srcPath string, cfg *pb.Config) error {
	lockPath := srcPath + ".lock.pb"
	data, err := proto.Marshal(cfg)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(lockPath), ".peridot-lock-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, lockPath)
}

// LoadLock reads a compiled .lock.pb file.
func LoadLock(lockPath string) (*pb.Config, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}
	cfg := &pb.Config{}
	if err := proto.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func parseEviction(s string) pb.EvictionPolicy {
	switch strings.ToLower(s) {
	case "lru":
		return pb.EvictionPolicy_EVICTION_POLICY_LRU
	case "fifo":
		return pb.EvictionPolicy_EVICTION_POLICY_FIFO
	case "random":
		return pb.EvictionPolicy_EVICTION_POLICY_RANDOM
	default:
		return pb.EvictionPolicy_EVICTION_POLICY_LRU
	}
}

// parseSize parses sizes like "2GB", "500MB", "1024".
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "GB"):
		mult, s = 1<<30, strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		mult, s = 1<<20, strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		mult, s = 1<<10, strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}
	s = strings.TrimSpace(s)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(n * float64(mult)), nil
}
