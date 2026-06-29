package cache

import (
	"os"
	"path/filepath"
	"sync"

	pb "github.com/conallob/peridot/gen/peridot"
)

// Manager coordinates the on-disk image cache and its SQLite metadata index,
// enforcing size/item limits via the configured eviction policy.
type Manager struct {
	mu         sync.Mutex
	ix         *index
	displayLog *DisplayLog
	dir        string
	maxBytes int64
	maxItems int32
	policy   pb.EvictionPolicy
}

// Options configures a cache Manager.
type Options struct {
	Dir      string
	MaxBytes int64
	MaxItems int32
	Policy   pb.EvictionPolicy
}

// New opens (creating if needed) a cache rooted at opts.Dir.
func New(opts Options) (*Manager, error) {
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, err
	}
	ix, err := openIndex(filepath.Join(opts.Dir, "index.db"))
	if err != nil {
		return nil, err
	}
	return &Manager{
		ix:         ix,
		displayLog: NewDisplayLog(ix),
		dir:        opts.Dir,
		maxBytes:   opts.MaxBytes,
		maxItems:   opts.MaxItems,
		policy:     opts.Policy,
	}, nil
}

// Log returns the display-event log backed by the same SQLite connection.
func (m *Manager) Log() *DisplayLog { return m.displayLog }

// Close releases the index.
func (m *Manager) Close() error { return m.ix.close() }

// Dir returns the cache root directory (sources may store images under it).
func (m *Manager) Dir() string { return m.dir }

// Put records (or updates) a cached wallpaper and enforces limits afterwards.
func (m *Manager) Put(meta *pb.WallpaperMetadata) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if meta.FileSizeBytes == 0 && meta.LocalPath != "" {
		if fi, err := os.Stat(meta.LocalPath); err == nil {
			meta.FileSizeBytes = fi.Size()
		}
	}
	if err := m.ix.put(meta); err != nil {
		return err
	}
	return m.enforce()
}

// Get returns cached metadata and marks it as recently used.
func (m *Manager) Get(id string) (*pb.WallpaperMetadata, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	meta, err := m.ix.get(id)
	if err != nil {
		return nil, err
	}
	_ = m.ix.touch(id)
	return meta, nil
}

// Has reports whether id is present in the cache.
func (m *Manager) Has(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.ix.get(id)
	return err == nil
}

// Status returns current cache utilisation.
func (m *Manager) Status() (*pb.CacheStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bytes, count, err := m.ix.stats()
	if err != nil {
		return nil, err
	}
	return &pb.CacheStatus{
		CurrentSizeBytes: bytes,
		MaxSizeBytes:     m.maxBytes,
		ItemCount:        count,
		MaxItems:         m.maxItems,
		EvictionPolicy:   m.policy,
	}, nil
}

// enforce evicts entries until size/item limits are satisfied. Caller holds mu.
func (m *Manager) enforce() error {
	for {
		bytes, count, err := m.ix.stats()
		if err != nil {
			return err
		}
		overSize := m.maxBytes > 0 && bytes > m.maxBytes
		overItems := m.maxItems > 0 && count > m.maxItems
		if !overSize && !overItems {
			return nil
		}
		ids, err := m.ix.evictionCandidates(m.policy, 1)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if err := m.evict(ids[0]); err != nil {
			return err
		}
	}
}

// evict removes a single entry from the index and deletes its image file.
func (m *Manager) evict(id string) error {
	path, _, err := m.ix.delete(id)
	if err != nil {
		return err
	}
	if path != "" {
		_ = os.Remove(path)
	}
	return nil
}
