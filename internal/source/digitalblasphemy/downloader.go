package digitalblasphemy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	pb "github.com/conallob/peridot/gen/peridot"
)

// Fetch downloads the wallpaper image to the source cache directory and returns
// its local path. If already downloaded, the cached path is returned.
func (s *Source) Fetch(ctx context.Context, m *pb.WallpaperMetadata) (string, error) {
	if m.LocalPath != "" {
		if _, err := os.Stat(m.LocalPath); err == nil {
			return m.LocalPath, nil
		}
	}
	if m.OriginUrl == "" {
		return "", fmt.Errorf("wallpaper %q has no origin URL", m.Id)
	}
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(s.cacheDir, m.Id+ext(m.OriginUrl))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.OriginUrl, nil)
	if err != nil {
		return "", err
	}
	if s.username != "" && s.secret != "" {
		req.SetBasicAuth(s.username, s.secret)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: status %d", m.OriginUrl, resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	m.LocalPath = dest
	return dest, nil
}

func ext(u string) string {
	e := filepath.Ext(u)
	if len(e) > 5 || e == "" {
		return ".jpg"
	}
	return e
}
