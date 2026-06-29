package url

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Source is a generic HTTP source. Each call to the endpoint is expected to
// return a single image (e.g. a "random wallpaper" endpoint). The catalogue
// therefore contains a single synthetic entry that resolves at fetch time.
type Source struct {
	id          string
	displayName string
	endpoint    string
	headers     map[string]string
	cacheDir    string

	http *http.Client
}

// New constructs a URL source.
func New(id, displayName, endpoint string, headers map[string]string, cacheDir string) *Source {
	if displayName == "" {
		displayName = "URL: " + endpoint
	}
	return &Source{
		id:          id,
		displayName: displayName,
		endpoint:    endpoint,
		headers:     headers,
		cacheDir:    cacheDir,
		http:        &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Source) ID() string          { return s.id }
func (s *Source) DisplayName() string { return s.displayName }
func (s *Source) Type() pb.SourceType { return pb.SourceType_SOURCE_TYPE_URL }

func (s *Source) Catalogue(ctx context.Context) ([]*pb.WallpaperMetadata, error) {
	return []*pb.WallpaperMetadata{{
		Id:          idFor(s.id, s.endpoint),
		Title:       s.displayName,
		SourceId:    s.id,
		OriginUrl:   s.endpoint,
		PublishedAt: timestamppb.New(time.Now()),
	}}, nil
}

func (s *Source) NewSince(ctx context.Context, since time.Time) ([]*pb.WallpaperMetadata, error) {
	return s.Catalogue(ctx)
}

func (s *Source) Fetch(ctx context.Context, m *pb.WallpaperMetadata) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, nil)
	if err != nil {
		return "", err
	}
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("url source %s: status %d", s.endpoint, resp.StatusCode)
	}

	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(s.cacheDir, fmt.Sprintf("%s-%d.img", m.Id, time.Now().UnixNano()))
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

func (s *Source) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, s.endpoint, nil)
	if err != nil {
		return false
	}
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}

func idFor(parts ...string) string {
	h := sha1.New()
	for _, p := range parts {
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}
