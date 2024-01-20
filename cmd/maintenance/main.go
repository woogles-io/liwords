package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
)

const BlogURL = "https://blog.woogles.io"

const (
	BlogRSSFeedURL   = BlogURL + "/index.xml"
	BlogSearchString = BlogURL
)

var AdminAPIKey = os.Getenv("ADMIN_API_KEY")
var WooglesAPIBasePath = os.Getenv("WOOGLES_API_BASE_PATH")

// A set of maintenance functions on Woogles that can run at some given
// cadence.

// go run . blogrssupdater,foo,bar,baz
func main() {
	if len(os.Args) < 2 {
		panic("need one comma-separated list of commands")
	}
	commands := strings.Split(os.Args[1], ",")
	log.Info().Interface("commands", commands).Msg("starting maintenance")
	for _, command := range commands {
		switch strings.ToLower(command) {
		case "blogrssupdater":
			err := BlogRssUpdater()
			log.Err(err).Msg("ran blogRssUpdater")

		default:
			log.Error().Str("command", command).Msg("command not recognized")
		}
	}
}

func WooglesAPIRequest(service, rpc string, bts []byte) (*http.Response, error) {
	path, err := url.JoinPath(WooglesAPIBasePath, service, rpc)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", path, bytes.NewReader(bts))
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Api-Key", AdminAPIKey)
	req.Header.Add("Content-Type", "application/json")
	return http.DefaultClient.Do(req)

}

// BlogRssUpdater updates the announcements homepage if a new blog post is found
// It subscribes to our blog RSS feed.
func BlogRssUpdater() error {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(BlogRSSFeedURL)
	if err != nil {
		return err
	}
	if len(feed.Items) < 1 {
		return errors.New("unexpected feed length")
	}
	authors := feed.Items[0].Authors
	authorsArr := make([]string, 0, len(authors))
	for _, a := range authors {
		authorsArr = append(authorsArr, a.Name)
	}
	authorsPrint := strings.Join(authorsArr, ", ")
	emoji := ""

	switch {
	case strings.Contains(feed.Items[0].Link, "/posts/"):
		emoji = "✏️"
	case strings.Contains(feed.Items[0].Link, "/guides/"):
		emoji = "📜"
	case strings.Contains(feed.Items[0].Link, "/articles/"):
		emoji = "📰"
	}

	img := feed.Items[0].Custom["image"]

	annobody := feed.Items[0].Description + " (Click to read more)"
	if img != "" {
		annobody = "![Image](" + img + ")\n\n" + annobody
	}
	b := &pb.SetSingleAnnouncementRequest{
		Announcement: &pb.Announcement{
			Title: emoji + " " + feed.Items[0].Title + " - written by " + authorsPrint,
			Body:  annobody,
			Link:  feed.Items[0].Link},
		LinkSearchString: BlogSearchString,
	}

	bts, err := protojson.Marshal(b)
	if err != nil {
		return err
	}
	resp, err := WooglesAPIRequest(
		"config_service.ConfigService",
		"SetSingleAnnouncement",
		bts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Info().Str("body", string(body)).Msg("received")

	return nil
}
