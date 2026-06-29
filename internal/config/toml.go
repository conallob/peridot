package config

// rawConfig is the intermediate representation parsed directly from TOML before
// translation/validation into the protobuf Config.
type rawConfig struct {
	Scheduler rawScheduler `toml:"scheduler"`
	Cache     rawCache     `toml:"cache"`
	Sources   []rawSource  `toml:"sources"`
}

type rawScheduler struct {
	Interval   string   `toml:"interval"` // e.g. "30m", "1h"
	Shuffle    bool     `toml:"shuffle"`
	PerDisplay bool     `toml:"per_display"`
	ChangeOn   []string `toml:"change_on"` // "timer", "wake", "login"
}

type rawCache struct {
	MaxSize        string `toml:"max_size"` // e.g. "2GB"
	MaxItems       int32  `toml:"max_items"`
	PrefetchAhead  int32  `toml:"prefetch_ahead"`
	EvictionPolicy string `toml:"eviction_policy"` // "lru", "fifo", "random"
	Location       string `toml:"location"`
}

type rawSource struct {
	ID          string  `toml:"id"`
	Type        string  `toml:"type"` // "local","digital_blasphemy","google_drive","url"
	DisplayName string  `toml:"display_name"`
	Weight      float32 `toml:"weight"`

	// Local
	Path  string `toml:"path"`
	Watch bool   `toml:"watch"`

	// DigitalBlasphemy
	Username            string `toml:"username"`
	KeychainService     string `toml:"keychain_service"`
	CheckInterval       string `toml:"check_interval"`
	AutoDownloadNew     bool   `toml:"auto_download_new"`
	PreferredResolution string `toml:"preferred_resolution"`

	// GoogleDrive
	FolderID        string `toml:"folder_id"`
	CredentialsPath string `toml:"credentials_path"`

	// URL
	Endpoint string            `toml:"endpoint"`
	Headers  map[string]string `toml:"headers"`
}
