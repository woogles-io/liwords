package broadcasts

// Broadcast feeds, stored.
//
// A broadcast's pairings, standings and ratings come from a feed the
// director's site serves (TSH's tourney.js, for instance). The poller is the
// only thing that fetches it: it claims each due broadcast in the database
// (ClaimBroadcastForPolling), so with several servers each feed is fetched
// once per interval, and stores the feed in broadcast_feeds, gzipped, under a
// hash of its content.
//
// Everything else reads the stored copy. Each server keeps parsed feeds in
// memory under that hash, and checks it against the database before using
// one, so a server never serves a feed another server has since replaced. A
// broadcast whose feed URL has gone away keeps serving its last stored feed.
//
// The short-lived RPC response caches are keyed by the same hash and by
// broadcasts.games_version, which goes up on every claim, unclaim, slot
// change and finished annotation, so they are correct on every server too.
// Their TTLs only bound how stale live scores of games in progress get.

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sort"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// parsedFeedCacheSize is how many parsed feeds a server keeps. A parsed feed
// is a few MB at most, and only a broadcast or two is live at a time.
const parsedFeedCacheSize = 16

// parsedFeed is a stored feed, parsed.
type parsedFeed struct {
	hash      string
	divisions map[string]*FeedData // division name → data ("" for single-division formats)
	names     []string             // division names, sorted
}

// firstDivision is the division pages show when none is asked for.
func (f *parsedFeed) firstDivision() string {
	if f == nil || len(f.names) == 0 {
		return ""
	}
	return f.names[0]
}

// division returns one division's data, or the first division's when division
// is empty. It is nil if there is no feed or no such division.
func (f *parsedFeed) division(division string) *FeedData {
	if f == nil {
		return nil
	}
	if division == "" {
		division = f.firstDivision()
	}
	return f.divisions[division]
}

// broadcastVersion is what a broadcast's cached responses depend on.
type broadcastVersion struct {
	feedHash string // "" when nothing is stored yet
	games    int64
}

// rpcCacheKey keys the response caches. division and round are empty for
// the responses that cover the whole broadcast.
type rpcCacheKey struct {
	slug     string
	division string
	round    int32
	version  broadcastVersion
}

func (k rpcCacheKey) String() string {
	return fmt.Sprintf("%s|%s|%d|%s|%d", k.slug, k.division, k.round, k.version.feedHash, k.version.games)
}

func newParsedFeedCache() *lru.Cache[string, *parsedFeed] {
	c, err := lru.New[string, *parsedFeed](parsedFeedCacheSize)
	if err != nil {
		panic(err) // only for a size <= 0
	}
	return c
}

// version reads what the broadcast's cached responses depend on.
func (bs *BroadcastService) version(ctx context.Context, slug string) (broadcastVersion, error) {
	row, err := bs.queries.GetBroadcastVersion(ctx, slug)
	if err != nil {
		return broadcastVersion{}, err
	}
	return broadcastVersion{feedHash: row.FeedHash, games: row.GamesVersion}, nil
}

// feed returns the broadcast's stored feed, parsed, or nil if none is stored
// or it can't be read.
func (bs *BroadcastService) feed(ctx context.Context, slug string) *parsedFeed {
	v, err := bs.version(ctx, slug)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Err(err).Str("slug", slug).Msg("broadcast-feed-version-failed")
		}
		return nil
	}
	return bs.feedAt(ctx, slug, v.feedHash)
}

// feedAt returns the stored feed with the given hash, parsed. It uses this
// server's parsed copy when that has the same hash, and otherwise loads and
// parses the stored feed once, whatever the number of concurrent callers.
func (bs *BroadcastService) feedAt(ctx context.Context, slug, hash string) *parsedFeed {
	if hash == "" {
		return nil
	}
	ctx, span := otel.Tracer("broadcast-feed").Start(ctx, "broadcast.feed.load")
	defer span.End()
	span.SetAttributes(attribute.String("broadcast.slug", slug))

	if f, ok := bs.parsedFeeds.Get(slug); ok && f.hash == hash {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		return f
	}
	span.SetAttributes(attribute.Bool("cache.hit", false))

	v, err, _ := bs.feedSF.Do(slug+"|"+hash, func() (any, error) {
		if f, ok := bs.parsedFeeds.Get(slug); ok && f.hash == hash {
			return f, nil
		}
		row, err := bs.queries.GetBroadcastFeedBySlug(ctx, slug)
		if err != nil {
			return nil, err
		}
		raw, err := gunzip(row.RawGz)
		if err != nil {
			return nil, err
		}
		span.SetAttributes(
			attribute.Int("feed.gz_bytes", len(row.RawGz)),
			attribute.Int("feed.bytes", len(raw)))
		divs, err := parseFeed(raw, row.Format)
		if err != nil {
			return nil, err
		}
		// The stored feed may be newer than the hash asked for; cache it
		// under its own.
		f := newParsedFeed(row.ContentHash, divs)
		bs.parsedFeeds.Add(slug, f)
		return f, nil
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if !errors.Is(err, pgx.ErrNoRows) {
			log.Err(err).Str("slug", slug).Msg("broadcast-feed-load-failed")
		}
		return nil
	}
	return v.(*parsedFeed) //nolint:forcetypeassert
}

func newParsedFeed(hash string, divs map[string]*FeedData) *parsedFeed {
	names := make([]string, 0, len(divs))
	for name := range divs {
		names = append(names, name)
	}
	sort.Strings(names)
	return &parsedFeed{hash: hash, divisions: divs, names: names}
}

// parseFeed parses every division of a raw feed.
func parseFeed(raw []byte, format string) (map[string]*FeedData, error) {
	p, err := NewFeedParser(format)
	if err != nil {
		return nil, err
	}
	tp, ok := p.(*TSHNewtParser)
	if !ok {
		// Formats without divisions go under "".
		fd, err := p.Parse(raw)
		if err != nil {
			return nil, err
		}
		return map[string]*FeedData{"": fd}, nil
	}
	names, err := tp.ListDivisions(raw)
	if err != nil {
		return nil, err
	}
	divs := make(map[string]*FeedData, len(names))
	for _, name := range names {
		fd, err := tp.ParseDivision(raw, name)
		if err != nil {
			log.Err(err).Str("division", name).Msg("feed: failed to parse division")
			continue
		}
		divs[name] = fd
	}
	if len(divs) == 0 && len(names) > 0 {
		return nil, fmt.Errorf("no division of %d could be parsed", len(names))
	}
	return divs, nil
}

// feedHash identifies a feed's content as parsed: the same bytes in another
// format are another feed.
func feedHash(format string, raw []byte) string {
	h := sha256.New()
	io.WriteString(h, format)
	h.Write([]byte{0})
	h.Write(raw)
	return hex.EncodeToString(h.Sum(nil))
}

func gzipBytes(b []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func gunzip(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// gamesChanged records that a broadcast's claims, slots or annotations
// changed, so every server's cached responses for it are out of date.
func (bs *BroadcastService) gamesChanged(ctx context.Context, broadcastID int32) {
	if err := bs.queries.BumpBroadcastGamesVersion(ctx, broadcastID); err != nil {
		// The response caches' TTLs still bound how stale they get.
		log.Err(err).Int32("broadcastID", broadcastID).Msg("broadcast-games-version-bump-failed")
	}
}
