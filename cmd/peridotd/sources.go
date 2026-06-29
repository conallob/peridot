package main

import (
	"path/filepath"
	"time"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/conallob/peridot/internal/platform"
	"github.com/conallob/peridot/internal/source"
	"github.com/conallob/peridot/internal/source/digitalblasphemy"
	"github.com/conallob/peridot/internal/source/googledrive"
	"github.com/conallob/peridot/internal/source/local"
	urlsrc "github.com/conallob/peridot/internal/source/url"
)

// buildSources constructs source.Source implementations from the compiled config.
func buildSources(cfg *pb.Config, cacheDir string) []source.Source {
	var out []source.Source
	for _, sc := range cfg.Sources {
		switch c := sc.Config.(type) {
		case *pb.SourceConfig_Local:
			out = append(out, local.New(sc.Id, sc.DisplayName, c.Local.Path, c.Local.Watch))
		case *pb.SourceConfig_DigitalBlasphemy:
			db := c.DigitalBlasphemy
			var interval time.Duration
			if db.CheckInterval != nil {
				interval = db.CheckInterval.AsDuration()
			}
			secret := resolveSecret(db.KeychainService, db.Username)
			out = append(out, digitalblasphemy.New(
				sc.Id, sc.DisplayName, db.Username, db.KeychainService,
				db.PreferredResolution, filepath.Join(cacheDir, sc.Id),
				db.AutoDownloadNew, interval, secret,
			))
		case *pb.SourceConfig_GoogleDrive:
			gd := c.GoogleDrive
			out = append(out, googledrive.New(
				sc.Id, sc.DisplayName, gd.FolderId, gd.CredentialsPath,
				filepath.Join(cacheDir, sc.Id),
			))
		case *pb.SourceConfig_Url:
			u := c.Url
			out = append(out, urlsrc.New(sc.Id, sc.DisplayName, u.Endpoint, u.Headers, filepath.Join(cacheDir, sc.Id)))
		}
	}
	return out
}

// resolveSecret looks up a secret from the platform credential store, returning
// an empty string when unavailable (the public feed still works).
func resolveSecret(service, account string) string {
	if service == "" || account == "" {
		return ""
	}
	p := platform.Current()
	if p == nil {
		return ""
	}
	secret, err := p.Credential().Get(service, account)
	if err != nil {
		return ""
	}
	return secret
}
