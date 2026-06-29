package digitalblasphemy

import (
	"context"
	"crypto/sha1"
	"encoding/hex"

	pb "github.com/conallob/peridot/gen/peridot"
	"github.com/mmcdole/gofeed"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fetchFeed parses the Digital Blasphemy RSS feed into wallpaper metadata.
func (s *Source) fetchFeed(ctx context.Context) ([]*pb.WallpaperMetadata, error) {
	fp := gofeed.NewParser()
	if s.username != "" && s.secret != "" {
		fp.AuthConfig = &gofeed.Auth{Username: s.username, Password: s.secret}
	}
	feed, err := fp.ParseURLWithContext(rssURL, ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*pb.WallpaperMetadata, 0, len(feed.Items))
	for _, item := range feed.Items {
		m := &pb.WallpaperMetadata{
			Id:        idFor(s.id, item.GUID, item.Link),
			Title:     item.Title,
			SourceId:  s.id,
			OriginUrl: imageURL(item),
		}
		if item.PublishedParsed != nil {
			m.PublishedAt = timestamppb.New(*item.PublishedParsed)
		}
		out = append(out, m)
	}
	return out, nil
}

// imageURL extracts a usable image URL from a feed item, preferring an
// enclosure, then a media:content, then the item link.
func imageURL(item *gofeed.Item) string {
	for _, e := range item.Enclosures {
		if e != nil && e.URL != "" {
			return e.URL
		}
	}
	if item.Image != nil && item.Image.URL != "" {
		return item.Image.URL
	}
	return item.Link
}

func idFor(parts ...string) string {
	h := sha1.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
