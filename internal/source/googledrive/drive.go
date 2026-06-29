package googledrive

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Source serves wallpapers from a Google Drive folder using the Drive API v3.
type Source struct {
	id              string
	displayName     string
	folderID        string
	credentialsPath string
	cacheDir        string

	svc *drive.Service
}

// New constructs a Google Drive source. The Drive service is created lazily on
// first use so that construction does not require network access.
func New(id, displayName, folderID, credentialsPath, cacheDir string) *Source {
	if displayName == "" {
		displayName = "Google Drive"
	}
	return &Source{
		id:              id,
		displayName:     displayName,
		folderID:        folderID,
		credentialsPath: credentialsPath,
		cacheDir:        cacheDir,
	}
}

func (s *Source) ID() string          { return s.id }
func (s *Source) DisplayName() string { return s.displayName }
func (s *Source) Type() pb.SourceType { return pb.SourceType_SOURCE_TYPE_GOOGLE_DRIVE }

func (s *Source) service(ctx context.Context) (*drive.Service, error) {
	if s.svc != nil {
		return s.svc, nil
	}
	opts := []option.ClientOption{option.WithScopes(drive.DriveReadonlyScope)}
	if s.credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(s.credentialsPath))
	}
	svc, err := drive.NewService(ctx, opts...)
	if err != nil {
		return nil, err
	}
	s.svc = svc
	return svc, nil
}

func (s *Source) Catalogue(ctx context.Context) ([]*pb.WallpaperMetadata, error) {
	svc, err := s.service(ctx)
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("'%s' in parents and mimeType contains 'image/' and trashed = false", s.folderID)
	var out []*pb.WallpaperMetadata
	pageToken := ""
	for {
		call := svc.Files.List().Q(q).
			Fields("nextPageToken, files(id, name, size, modifiedTime, createdTime)").
			PageSize(100).Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		res, err := call.Do()
		if err != nil {
			return nil, err
		}
		for _, f := range res.Files {
			m := &pb.WallpaperMetadata{
				Id:            s.id + ":" + f.Id,
				Title:         f.Name,
				SourceId:      s.id,
				OriginUrl:     "gdrive://" + f.Id,
				FileSizeBytes: f.Size,
			}
			if t, err := time.Parse(time.RFC3339, f.CreatedTime); err == nil {
				m.PublishedAt = timestamppb.New(t)
			}
			out = append(out, m)
		}
		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}
	}
	return out, nil
}

func (s *Source) NewSince(ctx context.Context, since time.Time) ([]*pb.WallpaperMetadata, error) {
	all, err := s.Catalogue(ctx)
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

func (s *Source) Fetch(ctx context.Context, m *pb.WallpaperMetadata) (string, error) {
	if m.LocalPath != "" {
		if _, err := os.Stat(m.LocalPath); err == nil {
			return m.LocalPath, nil
		}
	}
	fileID := strings.TrimPrefix(m.OriginUrl, "gdrive://")
	if fileID == "" {
		return "", fmt.Errorf("wallpaper %q missing drive file id", m.Id)
	}
	svc, err := s.service(ctx)
	if err != nil {
		return "", err
	}
	resp, err := svc.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(s.cacheDir, fileID+filepath.Ext(m.Title))
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
	svc, err := s.service(ctx)
	if err != nil {
		return false
	}
	_, err = svc.Files.List().Q(fmt.Sprintf("'%s' in parents", s.folderID)).PageSize(1).Context(ctx).Do()
	return err == nil
}
