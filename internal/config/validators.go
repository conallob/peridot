package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
)

// Validator inspects a compiled Config and returns an error if it is invalid.
type Validator interface {
	Validate(cfg *pb.Config) error
}

// ValidatorFunc adapts a function to the Validator interface.
type ValidatorFunc func(cfg *pb.Config) error

func (f ValidatorFunc) Validate(cfg *pb.Config) error { return f(cfg) }

// IntervalValidator ensures the scheduler interval is sane.
func IntervalValidator() Validator {
	return ValidatorFunc(func(cfg *pb.Config) error {
		if cfg.Scheduler == nil || cfg.Scheduler.Interval == nil {
			return fmt.Errorf("scheduler.interval is required")
		}
		d := cfg.Scheduler.Interval.AsDuration()
		if d < 10*time.Second {
			return fmt.Errorf("scheduler.interval must be >= 10s, got %s", d)
		}
		return nil
	})
}

// CacheSizeValidator ensures cache limits are positive and consistent.
func CacheSizeValidator() Validator {
	return ValidatorFunc(func(cfg *pb.Config) error {
		if cfg.Cache == nil {
			return nil
		}
		if cfg.Cache.MaxSizeBytes < 0 {
			return fmt.Errorf("cache.max_size must be >= 0")
		}
		if cfg.Cache.MaxItems < 0 {
			return fmt.Errorf("cache.max_items must be >= 0")
		}
		return nil
	})
}

// SourceValidator ensures each source has an ID and a known type.
func SourceValidator() Validator {
	return ValidatorFunc(func(cfg *pb.Config) error {
		seen := map[string]bool{}
		for i, s := range cfg.Sources {
			if s.Id == "" {
				return fmt.Errorf("sources[%d]: id is required", i)
			}
			if seen[s.Id] {
				return fmt.Errorf("duplicate source id %q", s.Id)
			}
			seen[s.Id] = true
			if s.Type == pb.SourceType_SOURCE_TYPE_UNSPECIFIED {
				return fmt.Errorf("source %q: unknown or missing type", s.Id)
			}
		}
		return nil
	})
}

// PathValidator ensures local source paths and cache locations exist (or their
// parent exists for the cache).
func PathValidator() Validator {
	return ValidatorFunc(func(cfg *pb.Config) error {
		for _, s := range cfg.Sources {
			if local := s.GetLocal(); local != nil {
				if local.Path == "" {
					return fmt.Errorf("source %q: local.path is required", s.Id)
				}
				if fi, err := os.Stat(local.Path); err != nil || !fi.IsDir() {
					return fmt.Errorf("source %q: path %q is not a directory", s.Id, local.Path)
				}
			}
		}
		return nil
	})
}

// ResolutionValidator validates preferred_resolution strings like "3840x2160".
func ResolutionValidator() Validator {
	return ValidatorFunc(func(cfg *pb.Config) error {
		check := func(id, res string) error {
			if res == "" {
				return nil
			}
			parts := strings.SplitN(strings.ToLower(res), "x", 2)
			if len(parts) != 2 {
				return fmt.Errorf("source %q: invalid resolution %q (want WxH)", id, res)
			}
			return nil
		}
		for _, s := range cfg.Sources {
			if db := s.GetDigitalBlasphemy(); db != nil {
				if err := check(s.Id, db.PreferredResolution); err != nil {
					return err
				}
			}
			if gd := s.GetGoogleDrive(); gd != nil {
				if err := check(s.Id, gd.PreferredResolution); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DefaultValidators returns the standard validation pipeline.
func DefaultValidators() []Validator {
	return []Validator{
		IntervalValidator(),
		CacheSizeValidator(),
		SourceValidator(),
		PathValidator(),
		ResolutionValidator(),
	}
}
