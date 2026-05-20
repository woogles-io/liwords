package game

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

const (
	archivedVersion = 2
	archivedIdAuth  = "io.woogles" // mirrors pkg/gameplay/game.go:43
	archivedDesc    = "Created with Macondo"
)

// archiveStore is the subset of Cache methods that HistoryArchiver needs.
type archiveStore interface {
	GetTurns(ctx context.Context, gameUUID string) ([]models.GetGameTurnsRow, error)
	CommitArchival(ctx context.Context, gameUUID string, s3Key string) error
	DeleteTurns(ctx context.Context, gameUUID string) error
	SetHistoryS3Key(ctx context.Context, gameUUID string, s3Key string) error
}

// HistoryArchiver assembles a GameHistory from game_turns rows, uploads it to S3
// as gzipped protojson, and atomically sets history_s3_key + deletes the turns
// rows only after a confirmed-successful upload.
type HistoryArchiver struct {
	s3Client *s3.Client
	bucket   string
	store    archiveStore
}

func NewHistoryArchiver(bucket string, s3Client *s3.Client, store archiveStore) *HistoryArchiver {
	return &HistoryArchiver{bucket: bucket, s3Client: s3Client, store: store}
}

// ArchiveAndCleanup is the single entry point for end-of-game archival.
// On any error before a successful upload it returns without touching
// game_turns or history_s3_key.
func (h *HistoryArchiver) ArchiveAndCleanup(ctx context.Context, g *entity.Game) error {
	l := log.Ctx(ctx)

	turns, err := h.store.GetTurns(ctx, g.GameID())
	if err != nil {
		return fmt.Errorf("archive-load-turns: %w", err)
	}

	expected := len(g.History().Events)
	if len(turns) < expected {
		// The game was in flight when dual-write was enabled: turns only cover
		// the post-cutover events. Drop the partial rows and fall back to
		// uploading from the in-memory history so the game is archived immediately
		// rather than waiting for a manual backfill run.
		l.Warn().Str("gameID", g.GameID()).
			Int("got", len(turns)).Int("want", expected).
			Msg("archive-partial-turns: in-flight at dual-write cutover, falling back to bytea upload")
		if err := h.store.DeleteTurns(ctx, g.GameID()); err != nil {
			return fmt.Errorf("archive-discard-partial: %w", err)
		}
		key, err := h.upload(ctx, g.GameID(), g.CreatedAt, g.History())
		if err != nil {
			return fmt.Errorf("archive-upload-fallback: %w", err)
		}
		if err := h.store.SetHistoryS3Key(ctx, g.GameID(), key); err != nil {
			return fmt.Errorf("archive-commit-fallback: %w", err)
		}
		l.Info().Str("gameID", g.GameID()).Str("key", key).Msg("game-archived-fallback")
		return nil
	}
	if len(turns) > expected {
		return fmt.Errorf("archive-extra-turns: have %d rows but history has %d events for %s",
			len(turns), expected, g.GameID())
	}

	assembled, err := assembleHistory(g, turns)
	if err != nil {
		return fmt.Errorf("archive-assemble: %w", err)
	}

	// Transitional: drops out in Phase 4 when games.history bytea is removed.
	// At that point verification becomes a structural self-check (non-empty
	// events, winner_idx ∈ {-1,0,1}, final_scores match last cumulatives).
	if err := verifyHistory(assembled, g.History(), g.GameID()); err != nil {
		return err
	}

	key, err := h.upload(ctx, g.GameID(), g.CreatedAt, assembled)
	if err != nil {
		return fmt.Errorf("archive-upload: %w", err)
	}

	if err := h.store.CommitArchival(ctx, g.GameID(), key); err != nil {
		// Upload succeeded but DB commit failed — the S3 object is orphaned but
		// turns rows are still present, so the backfill script can retry.
		return fmt.Errorf("archive-commit: %w", err)
	}

	l.Info().Str("gameID", g.GameID()).Str("key", key).Msg("game-archived")
	return nil
}

// assembleHistory builds a GameHistory proto from the stored turn rows plus
// metadata from the entity.Game. It does not read g.History() except for two
// transitional fields (RealName and LastKnownRacks) that have no other source
// in the current schema; those reads are removed in Phase 4.
func assembleHistory(g *entity.Game, turns []models.GetGameTurnsRow) (*macondopb.GameHistory, error) {
	events := make([]*macondopb.GameEvent, len(turns))
	for i, t := range turns {
		evt := &macondopb.GameEvent{}
		if err := protojson.Unmarshal(t.Event, evt); err != nil {
			return nil, fmt.Errorf("unmarshal turn %d: %w", t.TurnIdx, err)
		}
		events[i] = evt
	}

	// Transitional: RealName and LastKnownRacks come from the bytea history
	// because they have no dedicated columns/quickdata fields yet.
	storedHistory := g.History()
	players := make([]*macondopb.PlayerInfo, len(g.Quickdata.PlayerInfo))
	for i, pi := range g.Quickdata.PlayerInfo {
		var realName string
		if storedHistory != nil && i < len(storedHistory.Players) {
			realName = storedHistory.Players[i].RealName
		}
		players[i] = &macondopb.PlayerInfo{
			Nickname: pi.Nickname,
			UserId:   pi.UserId,
			RealName: realName,
		}
	}
	var lastKnownRacks []string
	if storedHistory != nil {
		lastKnownRacks = storedHistory.LastKnownRacks
	}

	var (
		lexicon       string
		boardLayout   string
		letterDist    string
		variant       string
		challengeRule macondopb.ChallengeRule
	)
	if g.GameReq != nil && g.GameReq.GameRequest != nil {
		lexicon = g.GameReq.Lexicon
		challengeRule = g.GameReq.ChallengeRule
		if g.GameReq.Rules != nil {
			boardLayout = g.GameReq.Rules.BoardLayoutName
			letterDist = g.GameReq.Rules.LetterDistributionName
			variant = g.GameReq.Rules.VariantName
		}
	}

	return &macondopb.GameHistory{
		Players:            players,
		Events:             events,
		Version:            archivedVersion,
		Lexicon:            lexicon,
		IdAuth:             archivedIdAuth,
		Uid:                g.GameID(),
		Description:        archivedDesc,
		LastKnownRacks:     lastKnownRacks,
		ChallengeRule:      challengeRule,
		PlayState:          macondopb.PlayState_GAME_OVER,
		FinalScores:        g.Quickdata.FinalScores,
		Variant:            variant,
		Winner:             int32(g.WinnerIdx),
		BoardLayout:        boardLayout,
		LetterDistribution: letterDist,
	}, nil
}

// verifyHistory performs a full deep-equality check between the assembled-from-
// turns history and the in-memory history decoded from games.history. Any
// mismatch means either the turn rows are corrupt or the assembly has a bug.
func verifyHistory(assembled, expected *macondopb.GameHistory, gameID string) error {
	if proto.Equal(assembled, expected) {
		return nil
	}
	diff := cmp.Diff(expected, assembled,
		protocmp.Transform(),
		cmpopts.EquateEmpty(),
	)
	log.Error().Str("gameID", gameID).Str("diff", diff).Msg("archive-verify-mismatch")
	return fmt.Errorf("archive-verify-mismatch for game %s", gameID)
}

// upload gzip-compresses the protojson-marshaled history and PUTs it to S3.
// The object key is partitioned by the game's creation date, not the archive
// date, so games stay in the bucket corresponding to when they were played.
// Returns the object key on success.
func (h *HistoryArchiver) upload(ctx context.Context, gameUUID string, createdAt time.Time, hist *macondopb.GameHistory) (string, error) {
	raw, err := protojson.Marshal(hist)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}

	t := createdAt.UTC()
	if t.IsZero() {
		// Annotated/imported games may have a NULL created_at; fall back to now.
		log.Ctx(ctx).Warn().Str("gameID", gameUUID).Msg("archive-no-created-at, falling back to now")
		t = time.Now().UTC()
	}
	key := fmt.Sprintf("games/%d/%02d/%s.json.gz", t.Year(), t.Month(), gameUUID)

	uploader := manager.NewUploader(h.s3Client)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(h.bucket),
		Key:             aws.String(key),
		Body:            bytes.NewReader(buf.Bytes()),
		ContentType:     aws.String("application/json"),
		ContentEncoding: aws.String("gzip"),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

// Fetch downloads the game history at s3Key, decompresses, and unmarshals it.
// It is the symmetric read side of upload. Used by DBStore.Get and GetHistory
// when history_s3_key is set.
func (h *HistoryArchiver) Fetch(ctx context.Context, s3Key string) (*macondopb.GameHistory, error) {
	tracer := otel.Tracer("game-store")
	ctx, span := tracer.Start(ctx, "game.FetchFromS3")
	defer span.End()
	span.SetAttributes(attribute.String("s3.key", s3Key))

	downloader := manager.NewDownloader(h.s3Client)
	buf := manager.NewWriteAtBuffer(nil)
	_, err := downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(h.bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("s3-fetch %s: %w", s3Key, err)
	}

	gr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("s3-fetch-gzip %s: %w", s3Key, err)
	}
	defer gr.Close()

	raw, err := io.ReadAll(gr)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("s3-fetch-read %s: %w", s3Key, err)
	}

	hist := &macondopb.GameHistory{}
	if err := protojson.Unmarshal(raw, hist); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("s3-fetch-unmarshal %s: %w", s3Key, err)
	}
	span.SetAttributes(attribute.Int("history.events_count", len(hist.Events)))
	return hist, nil
}

// ArchiveBytea archives a game whose history is stored as a proto binary bytea
// (i.e. games that predate the game_turns dual-write). It decodes the bytea,
// re-encodes as protojson, gzips, uploads to S3, then sets history_s3_key.
// The output format is identical to ArchiveAndCleanup so S3 objects are
// indistinguishable regardless of which code path produced them.
func (h *HistoryArchiver) ArchiveBytea(ctx context.Context, gameUUID string, createdAt time.Time, historyBytes []byte) (string, error) {
	hist := &macondopb.GameHistory{}
	if err := proto.Unmarshal(historyBytes, hist); err != nil {
		return "", fmt.Errorf("backfill-unmarshal: %w", err)
	}
	key, err := h.upload(ctx, gameUUID, createdAt, hist)
	if err != nil {
		return "", err
	}
	if err := h.store.SetHistoryS3Key(ctx, gameUUID, key); err != nil {
		return key, fmt.Errorf("backfill-set-key: %w", err)
	}
	return key, nil
}
