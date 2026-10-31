package game

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/matryer/is"
	"google.golang.org/protobuf/encoding/protojson"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// stallingArchiveStore serves the given turn rows, then holds the commit until
// release is closed.
type stallingArchiveStore struct {
	turns   []models.GetGameTurnsRow
	release chan struct{}
	mu      sync.Mutex
	key     string
}

func (s *stallingArchiveStore) GetTurns(context.Context, string) ([]models.GetGameTurnsRow, error) {
	return s.turns, nil
}

func (s *stallingArchiveStore) CommitArchival(_ context.Context, _ string, key string, _ int) error {
	<-s.release
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key = key
	return nil
}

func (s *stallingArchiveStore) SetHistoryS3Key(context.Context, string, string) error {
	return nil
}

func (s *stallingArchiveStore) committedKey() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.key
}

func TestArchiveInBackgroundReturnsAtOnceAndWaitWaits(t *testing.T) {
	is := is.New(t)

	var puts sync.Map
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if r.Method == http.MethodPut {
			puts.Store(r.URL.Path, true)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := s3.New(s3.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(srv.URL),
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider("k", "s", ""),
	})
	mcg := newMacondoGame([2]*entity.User{{UUID: "u1", Username: "one"}, {UUID: "u2", Username: "two"}})
	mcg.StartGame()
	g := entity.NewGame(mcg, &pb.GameRequest{
		Lexicon: "NWL20",
		Rules:   &pb.GameRules{BoardLayoutName: "CrosswordGame", LetterDistributionName: "english"},
	})
	g.Quickdata = &entity.Quickdata{PlayerInfo: []*pb.PlayerInfo{
		{Nickname: "one", UserId: "u1"}, {Nickname: "two", UserId: "u2"},
	}}
	g.History().PlayState = macondopb.PlayState_GAME_OVER
	g.CreatedAt = time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	g.History().Uid = "bgArchive1"
	evt := &macondopb.GameEvent{Type: macondopb.GameEvent_TILE_PLACEMENT_MOVE}
	g.History().Events = append(g.History().Events, evt)
	raw, err := protojson.Marshal(evt)
	is.NoErr(err)

	store := &stallingArchiveStore{
		turns:   []models.GetGameTurnsRow{{TurnIdx: 0, Event: raw}},
		release: make(chan struct{}),
	}
	archiver := NewHistoryArchiver("games-bucket", client, store)

	archiver.ArchiveInBackground(context.Background(), g)

	// The upload goes ahead without the caller, and the commit is still held.
	short, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	is.Equal(archiver.Wait(short), context.DeadlineExceeded)

	close(store.release)
	is.NoErr(archiver.Wait(context.Background()))

	want := "games/2026/10/bgArchive1.json.gz"
	is.Equal(store.committedKey(), want)
	_, uploaded := puts.Load("/games-bucket/" + want)
	is.True(uploaded)
}
