package cache

import (
	"database/sql"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

// index is the SQLite-backed metadata store. Image bytes live on disk; this
// table tracks metadata plus cache bookkeeping (size, usage times).
type index struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS wallpapers (
  id            TEXT PRIMARY KEY,
  source_id     TEXT NOT NULL,
  local_path    TEXT NOT NULL,
  size_bytes    INTEGER NOT NULL DEFAULT 0,
  metadata      BLOB NOT NULL,
  cached_at     INTEGER NOT NULL,
  last_used_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_wallpapers_lru ON wallpapers(last_used_at);
CREATE INDEX IF NOT EXISTS idx_wallpapers_fifo ON wallpapers(cached_at);
`

func openIndex(dbPath string) (*index, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &index{db: db}, nil
}

func (ix *index) close() error { return ix.db.Close() }

func (ix *index) put(m *pb.WallpaperMetadata) error {
	blob, err := proto.Marshal(m)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = ix.db.Exec(
		`INSERT INTO wallpapers (id, source_id, local_path, size_bytes, metadata, cached_at, last_used_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   local_path=excluded.local_path,
		   size_bytes=excluded.size_bytes,
		   metadata=excluded.metadata,
		   last_used_at=excluded.last_used_at`,
		m.Id, m.SourceId, m.LocalPath, m.FileSizeBytes, blob, now, now,
	)
	return err
}

func (ix *index) touch(id string) error {
	_, err := ix.db.Exec(`UPDATE wallpapers SET last_used_at=? WHERE id=?`, time.Now().Unix(), id)
	return err
}

func (ix *index) get(id string) (*pb.WallpaperMetadata, error) {
	var blob []byte
	err := ix.db.QueryRow(`SELECT metadata FROM wallpapers WHERE id=?`, id).Scan(&blob)
	if err != nil {
		return nil, err
	}
	m := &pb.WallpaperMetadata{}
	if err := proto.Unmarshal(blob, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (ix *index) delete(id string) (localPath string, size int64, err error) {
	err = ix.db.QueryRow(`SELECT local_path, size_bytes FROM wallpapers WHERE id=?`, id).Scan(&localPath, &size)
	if err != nil {
		return "", 0, err
	}
	_, err = ix.db.Exec(`DELETE FROM wallpapers WHERE id=?`, id)
	return localPath, size, err
}

func (ix *index) stats() (totalBytes int64, count int32, err error) {
	err = ix.db.QueryRow(`SELECT COALESCE(SUM(size_bytes),0), COUNT(*) FROM wallpapers`).Scan(&totalBytes, &count)
	return
}

// evictionCandidates returns up to limit ids ordered by the policy's priority.
func (ix *index) evictionCandidates(policy pb.EvictionPolicy, limit int) ([]string, error) {
	rows, err := ix.db.Query(`SELECT id FROM wallpapers ORDER BY `+orderClause(policy)+` LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

