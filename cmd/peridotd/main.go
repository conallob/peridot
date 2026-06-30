// Command peridotd is the peridot wallpaper rotation daemon.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/cache"
	"github.com/conallob/peridot/internal/config"
	"github.com/conallob/peridot/internal/display"
	"github.com/conallob/peridot/internal/ipc"
	"github.com/conallob/peridot/internal/platform"
	"github.com/conallob/peridot/internal/scheduler"
	// The platform implementation for the current OS is registered via the
	// build-tagged platform_<os>.go files in this package.
)

// Injected at build time via -ldflags "-X main.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	log.SetPrefix("peridotd: ")
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)

	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".peridot")
	configPath := filepath.Join(baseDir, "config.toml")
	if v := os.Getenv("PERIDOT_CONFIG"); v != "" {
		configPath = v
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if platform.Current() == nil {
		log.Fatal("no platform support compiled in for this OS")
	}

	cfg := loadConfig(configPath)

	cacheDir := filepath.Join(baseDir, "cache")
	if cfg.Cache != nil && cfg.Cache.Location != "" {
		cacheDir = cfg.Cache.Location
	}
	cacheMgr, err := cache.New(cache.Options{
		Dir:      cacheDir,
		MaxBytes: cacheMaxBytes(cfg),
		MaxItems: cacheMaxItems(cfg),
		Policy:   cachePolicy(cfg),
	})
	if err != nil {
		log.Fatalf("init cache: %v", err)
	}
	defer func() { _ = cacheMgr.Close() }()

	engine, err := display.New()
	if err != nil {
		log.Fatalf("init display: %v", err)
	}

	srcs := buildSources(cfg, cacheDir)

	interval := 30 * time.Minute
	shuffle := false
	if cfg.Scheduler != nil {
		if cfg.Scheduler.Interval != nil {
			interval = cfg.Scheduler.Interval.AsDuration()
		}
		shuffle = cfg.Scheduler.Shuffle
	}

	sched := scheduler.New(scheduler.Options{
		Display:  engine,
		Events:   platform.Current().Events(),
		Sources:  srcs,
		Interval: interval,
		Shuffle:  shuffle,
		Log:      cacheMgr.Log(),
	})

	d := &daemon{sched: sched, cache: cacheMgr, sources: srcs}

	// Config hot-reload.
	if w, err := config.NewWatcher(configPath); err == nil {
		go w.Watch(ctx, func(newCfg *pb.Config) {
			log.Printf("config reloaded")
			newSrcs := buildSources(newCfg, cacheDir)
			d.sources = newSrcs
			sched.SetSources(newSrcs)
			if newCfg.Scheduler != nil && newCfg.Scheduler.Interval != nil {
				sched.SetInterval(newCfg.Scheduler.Interval.AsDuration())
			}
		}, func(err error) {
			log.Printf("config reload error: %v", err)
		})
	}

	// IPC server.
	srv := ipc.NewServer(ipc.DefaultSocketPath(), d.handle)
	go func() {
		if err := srv.Serve(ctx); err != nil {
			log.Printf("ipc server: %v", err)
		}
	}()

	// Scheduler loop.
	go func() {
		if err := sched.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("scheduler: %v", err)
		}
	}()

	log.Printf("peridotd %s (commit %s, built %s) started (platform=%s, config=%s)", version, commit, date, platform.Current().Name(), configPath)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Printf("shutting down")
	cancel()
	_ = srv.Close()
	time.Sleep(100 * time.Millisecond)
}

// loadConfig compiles the config, falling back to defaults if it is missing.
func loadConfig(path string) *pb.Config {
	if _, err := os.Stat(path); err != nil {
		log.Printf("no config at %s; using defaults", path)
		return defaultConfig()
	}
	cfg, err := config.Compile(path)
	if err != nil {
		log.Printf("config error (%v); using defaults", err)
		return defaultConfig()
	}
	return cfg
}

func defaultConfig() *pb.Config {
	return &pb.Config{
		Scheduler: &pb.SchedulerConfig{},
		Cache:     &pb.CacheConfig{},
	}
}

func cacheMaxBytes(cfg *pb.Config) int64 {
	if cfg.Cache != nil {
		return cfg.Cache.MaxSizeBytes
	}
	return 0
}

func cacheMaxItems(cfg *pb.Config) int32 {
	if cfg.Cache != nil {
		return cfg.Cache.MaxItems
	}
	return 0
}

func cachePolicy(cfg *pb.Config) pb.EvictionPolicy {
	if cfg.Cache != nil {
		return cfg.Cache.EvictionPolicy
	}
	return pb.EvictionPolicy_EVICTION_POLICY_LRU
}
