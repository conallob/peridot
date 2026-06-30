package config

import (
	"testing"

	pb "github.com/conallob/peridot/gen/peridot"
)

func TestTranslateSource_Local(t *testing.T) {
	sc, err := translateSource(&rawSource{
		ID:   "loc",
		Type: "local",
		Path: "/tmp/walls",
	})
	if err != nil {
		t.Fatalf("translateSource local: %v", err)
	}
	if sc.Type != pb.SourceType_SOURCE_TYPE_LOCAL {
		t.Fatalf("type = %v", sc.Type)
	}
	if sc.GetLocal().Path != "/tmp/walls" {
		t.Fatalf("path = %q", sc.GetLocal().Path)
	}
}

func TestTranslateSource_URL(t *testing.T) {
	sc, err := translateSource(&rawSource{
		ID:       "u",
		Type:     "url",
		Endpoint: "https://example.com/img",
		Headers:  map[string]string{"X-Key": "val"},
	})
	if err != nil {
		t.Fatalf("translateSource url: %v", err)
	}
	if sc.Type != pb.SourceType_SOURCE_TYPE_URL {
		t.Fatalf("type = %v", sc.Type)
	}
	if sc.GetUrl().Endpoint != "https://example.com/img" {
		t.Fatalf("endpoint = %q", sc.GetUrl().Endpoint)
	}
	if sc.GetUrl().Headers["X-Key"] != "val" {
		t.Fatalf("headers = %v", sc.GetUrl().Headers)
	}
}

func TestTranslateSource_DigitalBlasphemy(t *testing.T) {
	sc, err := translateSource(&rawSource{
		ID:                  "db",
		Type:                "digital_blasphemy",
		Username:            "user",
		PreferredResolution: "1920x1080",
		CheckInterval:       "1h",
		AutoDownloadNew:     true,
	})
	if err != nil {
		t.Fatalf("translateSource db: %v", err)
	}
	if sc.Type != pb.SourceType_SOURCE_TYPE_DIGITAL_BLASPHEMY {
		t.Fatalf("type = %v", sc.Type)
	}
	db := sc.GetDigitalBlasphemy()
	if db.Username != "user" {
		t.Fatalf("username = %q", db.Username)
	}
	if db.CheckInterval == nil || db.CheckInterval.AsDuration().Hours() != 1 {
		t.Fatalf("check_interval = %v", db.CheckInterval)
	}
}

func TestTranslateSource_DigitalBlasphemy_BadInterval(t *testing.T) {
	_, err := translateSource(&rawSource{
		ID:            "db",
		Type:          "digital_blasphemy",
		CheckInterval: "not-a-duration",
	})
	if err == nil {
		t.Fatal("expected error for bad check_interval")
	}
}

func TestTranslateSource_GoogleDrive(t *testing.T) {
	sc, err := translateSource(&rawSource{
		ID:              "gd",
		Type:            "google_drive",
		FolderID:        "folder123",
		CredentialsPath: "/path/to/creds.json",
	})
	if err != nil {
		t.Fatalf("translateSource gdrive: %v", err)
	}
	if sc.Type != pb.SourceType_SOURCE_TYPE_GOOGLE_DRIVE {
		t.Fatalf("type = %v", sc.Type)
	}
	gd := sc.GetGoogleDrive()
	if gd.FolderId != "folder123" {
		t.Fatalf("folder_id = %q", gd.FolderId)
	}
}

func TestTranslateSource_Unknown(t *testing.T) {
	_, err := translateSource(&rawSource{ID: "x", Type: "unsupported"})
	if err == nil {
		t.Fatal("expected error for unknown source type")
	}
}

func TestTranslate_ChangeOnEvents(t *testing.T) {
	raw := &rawConfig{
		Scheduler: rawScheduler{
			Interval: "30m",
			ChangeOn: []string{"timer", "wake", "login"},
		},
	}
	cfg, err := translate(raw)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if len(cfg.Scheduler.ChangeOn) != 3 {
		t.Fatalf("change_on len = %d, want 3", len(cfg.Scheduler.ChangeOn))
	}
}

func TestTranslate_UnknownChangeOn(t *testing.T) {
	raw := &rawConfig{
		Scheduler: rawScheduler{
			Interval: "30m",
			ChangeOn: []string{"unknown_event"},
		},
	}
	if _, err := translate(raw); err == nil {
		t.Fatal("expected error for unknown change_on event")
	}
}
