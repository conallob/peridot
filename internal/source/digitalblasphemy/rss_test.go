package digitalblasphemy

import (
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestImageURL_Enclosure(t *testing.T) {
	item := &gofeed.Item{
		Enclosures: []*gofeed.Enclosure{{URL: "https://example.com/img.jpg"}},
		Link:       "https://example.com/link",
	}
	if got := imageURL(item); got != "https://example.com/img.jpg" {
		t.Fatalf("imageURL = %q, want enclosure URL", got)
	}
}

func TestImageURL_Image(t *testing.T) {
	item := &gofeed.Item{
		Image: &gofeed.Image{URL: "https://example.com/cover.jpg"},
		Link:  "https://example.com/link",
	}
	if got := imageURL(item); got != "https://example.com/cover.jpg" {
		t.Fatalf("imageURL = %q, want image URL", got)
	}
}

func TestImageURL_FallsBackToLink(t *testing.T) {
	item := &gofeed.Item{Link: "https://example.com/link"}
	if got := imageURL(item); got != "https://example.com/link" {
		t.Fatalf("imageURL = %q, want link", got)
	}
}

func TestImageURL_EmptyEnclosureSkipped(t *testing.T) {
	item := &gofeed.Item{
		Enclosures: []*gofeed.Enclosure{{URL: ""}},
		Link:       "https://example.com/fallback",
	}
	if got := imageURL(item); got != "https://example.com/fallback" {
		t.Fatalf("imageURL = %q, want fallback link", got)
	}
}

func TestIdFor_Deterministic(t *testing.T) {
	a := idFor("src", "guid-1", "https://example.com")
	b := idFor("src", "guid-1", "https://example.com")
	if a != b {
		t.Fatal("idFor not deterministic")
	}
}

func TestIdFor_Distinct(t *testing.T) {
	a := idFor("src", "guid-1", "https://a.com")
	b := idFor("src", "guid-1", "https://b.com")
	if a == b {
		t.Fatal("different URLs produced same id")
	}
}

func TestNewSource_DefaultDisplayName(t *testing.T) {
	s := New("id", "", "user", "", "1920x1080", "/tmp", false, 0, "")
	if s.DisplayName() != "Digital Blasphemy" {
		t.Fatalf("DisplayName = %q", s.DisplayName())
	}
}

func TestNewSource_Identity(t *testing.T) {
	s := New("db-id", "My DB", "user", "", "1920x1080", "/tmp", true, 0, "pass")
	if s.ID() != "db-id" {
		t.Fatalf("ID = %q", s.ID())
	}
	if s.DisplayName() != "My DB" {
		t.Fatalf("DisplayName = %q", s.DisplayName())
	}
}
