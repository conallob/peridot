package digitalblasphemy

import (
	"context"
	"net/http"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
)

const rssURL = "https://digitalblasphemy.com/rss.xml"

// Source implements source.Source for the Digital Blasphemy gallery via its RSS
// feed. Authentication (for member resolutions) uses a username and a secret
// resolved from the platform credential store.
type Source struct {
	id                  string
	displayName         string
	username            string
	keychainService     string
	preferredResolution string
	autoDownloadNew     bool
	checkInterval       time.Duration
	cacheDir            string

	http   *http.Client
	secret string // resolved password/token; may be empty for public feed
}

// New constructs a Digital Blasphemy source.
func New(id, displayName, username, keychainService, preferredResolution, cacheDir string, autoDownloadNew bool, checkInterval time.Duration, secret string) *Source {
	if displayName == "" {
		displayName = "Digital Blasphemy"
	}
	return &Source{
		id:                  id,
		displayName:         displayName,
		username:            username,
		keychainService:     keychainService,
		preferredResolution: preferredResolution,
		autoDownloadNew:     autoDownloadNew,
		checkInterval:       checkInterval,
		cacheDir:            cacheDir,
		secret:              secret,
		http:                &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Source) ID() string          { return s.id }
func (s *Source) DisplayName() string { return s.displayName }
func (s *Source) Type() pb.SourceType { return pb.SourceType_SOURCE_TYPE_DIGITAL_BLASPHEMY }

func (s *Source) Catalogue(ctx context.Context) ([]*pb.WallpaperMetadata, error) {
	return s.fetchFeed(ctx)
}

func (s *Source) NewSince(ctx context.Context, since time.Time) ([]*pb.WallpaperMetadata, error) {
	all, err := s.fetchFeed(ctx)
	if err != nil {
		return nil, err
	}
	var out []*pb.WallpaperMetadata
	for _, m := range all {
		if m.PublishedAt != nil && m.PublishedAt.AsTime().After(since) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Source) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rssURL, nil)
	if err != nil {
		return false
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}
