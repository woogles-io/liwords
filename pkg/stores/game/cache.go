package game

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	gs "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	cacheLookups     metric.Int64Counter
	cacheLoadDurMs   metric.Float64Histogram
	cacheMetricsOnce sync.Once
)

func initCacheMetrics() {
	meter := otel.Meter("game-cache")
	cacheLookups, _ = meter.Int64Counter(
		"game.cache.lookups",
		metric.WithDescription("Number of game cache lookups"),
	)
	cacheLoadDurMs, _ = meter.Float64Histogram(
		"game.load.duration_ms",
		metric.WithDescription("End-to-end game load latency (hit or miss) in milliseconds"),
		metric.WithUnit("ms"),
	)
}

var errNoID = errors.New("game ID was not defined")

// same as the GameStore in gameplay package, but this gives us a bit more flexibility
// in defining the backing store (i.e. it may not necessarily be a SQL db store)
type backingStore interface {
	Get(ctx context.Context, id string) (*entity.Game, error)
	GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error)
	GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error)
	GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*pb.GameInfoResponses, error)
	GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.GameInfoResponses, error)
	GetRecentCorrespondenceGames(ctx context.Context, username string, numGames int) (*pb.GameInfoResponses, error)
	Set(context.Context, *entity.Game) error
	Create(context.Context, *entity.Game) error
	CreateRaw(context.Context, *entity.Game, pb.GameType) error
	Exists(context.Context, string) (bool, error)
	ListActive(ctx context.Context, tourneyID string, bust bool) (*pb.GameInfoResponses, error)
	ListActiveCorrespondence(ctx context.Context) (*pb.GameInfoResponses, error)
	ListActiveCorrespondenceForUser(ctx context.Context, userID string) (*pb.GameInfoResponses, error)
	ListActiveCorrespondenceForUserAndLeague(ctx context.Context, leagueID uuid.UUID, userID string) (*pb.GameInfoResponses, error)
	ListActiveCorrespondenceRaw(ctx context.Context) ([]models.ListActiveCorrespondenceGamesRow, error)
	Count(ctx context.Context) (int64, error)
	GameEventChan() chan<- *entity.EventWrapper
	SetGameEventChan(ch chan<- *entity.EventWrapper)
	Disconnect()
	SetReady(ctx context.Context, gid string, pidx int) (int, error)
	GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error)
	InsertGamePlayers(ctx context.Context, g *entity.Game) error
	SetTimerModuleCreator(creator TimerModuleCreator)
	SetHistoryFetcher(f HistoryFetcher)
	// StageTurns queues a move's events to be written by the next Set, in the
	// same transaction as the games row. The move path must use this rather
	// than AppendTurns; see DBStore.Set.
	StageTurns(g *entity.Game, startIdx int, events []*macondopb.GameEvent) error
	AppendTurns(ctx context.Context, gameUUID string, startIdx int, events []*macondopb.GameEvent) error
	LockGame(ctx context.Context, gameID string) (*GameLock, error)
	GetTurns(ctx context.Context, gameUUID string) ([]models.GetGameTurnsRow, error)
	DeleteTurns(ctx context.Context, gameUUID string) error
	CommitArchival(ctx context.Context, gameUUID string, s3Key string, archivedTurns int) error
	SetHistoryS3Key(ctx context.Context, gameUUID string, s3Key string) error
}

// TimerModuleCreator is a function that creates a new timer module for a game.
type TimerModuleCreator func() entity.Nower

// Cache no longer caches games. It used to hold up to 400 of them in an LRU,
// per node, which is a per-node answer to a question that now has more than one
// node: a game cached here goes stale the moment another server plays a move on
// it, and it would do so while holding the game lock correctly -- serialized,
// and reading a position from before the move it was serialized against.
//
// What remains is a five-second cache of the *active games listing*, which is a
// different thing: a lobby view that may lag slightly, not a position that must
// not. Everything else passes straight through to the database.
//
// The name is now wrong and should be changed; that is a rename across
// pkg/stores, kept out of this change so the diff stays about behaviour.
type Cache struct {
	sync.RWMutex // used for the activeGames cache.
	activeGames  *pb.GameInfoResponses

	activeGamesTTL         time.Duration
	activeGamesLastUpdated time.Time

	backing backingStore
}

func NewCache(backing backingStore) *Cache {
	c := &Cache{
		backing: backing,
		// Have a non-trivial TTL for the cache of active games.
		// XXX: This might act poorly if the following happens within the TTL:
		//  - active games gets cached
		//  - someone starts playing a game
		//  - new player logs on and fetches active games
		//  - new player will receive the old games and not the new game?
		// One solution: bust the cache or add/subtract directly from cache
		//  when a new game is created/ended.
		// Problem: this won't work for distributed nodes. Once we
		// add multiple nodes we should probably have a Redis cache for a
		// few things (especially game quickdata).
		activeGamesTTL: time.Second * 5,
	}
	return c
}

// Unload unloads the game from the cache
// Unload expires the active games listing. There is no game to unload any
// more, but a finished game must stop appearing in the lobby immediately rather
// than for up to another five seconds.
func (c *Cache) Unload(ctx context.Context, id string) {
	c.Lock()
	defer c.Unlock()
	c.activeGamesLastUpdated = time.Time{}
}

func (c *Cache) GameEventChan() chan<- *entity.EventWrapper {
	return c.backing.GameEventChan()
}

// SetGameEventChan sets the game event channel to the passed in channel.
func (c *Cache) SetGameEventChan(ch chan<- *entity.EventWrapper) {
	c.backing.SetGameEventChan(ch)
}

// Get gets a game from the cache.. it loads it into the cache if it's not there.
// Correspondence games bypass the cache and always go to the DB.
func (c *Cache) Get(ctx context.Context, id string) (*entity.Game, error) {
	cacheMetricsOnce.Do(initCacheMetrics)
	start := time.Now()

	tracer := otel.Tracer("game-cache")
	ctx, span := tracer.Start(ctx, "cache.Get")
	defer span.End()

	// Every load reads the database. Correspondence games always did -- they
	// were never cached, because each request needs the position as it is now
	// -- and this is that path for every game.
	g, err := c.backing.Get(ctx, id)
	cacheLoadDurMs.Record(ctx, float64(time.Since(start).Milliseconds()))
	return g, err
}

// Just call the DB implementation for now
func (c *Cache) GetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	return c.backing.GetRematchStreak(ctx, originalRequestId)
}

func (c *Cache) GetRecentGames(ctx context.Context, username string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	return c.backing.GetRecentGames(ctx, username, numGames, offset)
}

func (c *Cache) GetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.GameInfoResponses, error) {
	return c.backing.GetRecentTourneyGames(ctx, tourneyID, numGames, offset)
}

func (c *Cache) GetRecentCorrespondenceGames(ctx context.Context, username string, numGames int) (*pb.GameInfoResponses, error) {
	return c.backing.GetRecentCorrespondenceGames(ctx, username, numGames)
}

// Similar to get but does not unmarshal the stats and timers and does
// not play the game
func (c *Cache) GetMetadata(ctx context.Context, id string) (*pb.GameInfoResponse, error) {
	return c.backing.GetMetadata(ctx, id)
}

// Set sets a game in the cache, AND in the backing store. This ensures if the
// node crashes the game doesn't just vanish.
func (c *Cache) Set(ctx context.Context, game *entity.Game) error {
	return c.setOrCreate(ctx, game, false)
}

// Create creates the game in the cache as well as the store.
func (c *Cache) Create(ctx context.Context, game *entity.Game) error {
	return c.setOrCreate(ctx, game, true)
}

// CreateRaw creates the game in the store only.
func (c *Cache) CreateRaw(ctx context.Context, game *entity.Game, gt pb.GameType) error {
	return c.backing.CreateRaw(ctx, game, gt)
}

func (c *Cache) Exists(ctx context.Context, id string) (bool, error) {
	return c.backing.Exists(ctx, id)
}

func (c *Cache) setOrCreate(ctx context.Context, game *entity.Game, isNew bool) error {
	gameID := game.History().Uid
	if gameID == "" {
		return errNoID
	}
	var err error
	if isNew {
		err = c.backing.Create(ctx, game)
	} else {
		err = c.backing.Set(ctx, game)
	}
	return err
}

// ListActive lists all active games in the given tournament ID (optional) or
// site-wide if not provided. If `bust` is true, we will always query the backing
// store.
func (c *Cache) ListActive(ctx context.Context, tourneyID string, bust bool) (*pb.GameInfoResponses, error) {
	if tourneyID == "" && !bust {
		return c.listAllActive(ctx)
	}
	// Otherwise don't worry about caching; this list should be comparatively smaller.
	return c.backing.ListActive(ctx, tourneyID, bust)
}

func (c *Cache) listAllActive(ctx context.Context) (*pb.GameInfoResponses, error) {
	c.RLock()
	if time.Since(c.activeGamesLastUpdated) < c.activeGamesTTL {
		log.Debug().Msg("returning active games from cache")
		c.RUnlock()
		return c.activeGames, nil
	}
	c.RUnlock()
	log.Debug().Msg("active games not in cache, fetching from backing")

	games, err := c.backing.ListActive(ctx, "", false)
	if err == nil {
		c.Lock()
		c.activeGames = games
		c.activeGamesLastUpdated = time.Now()
		c.Unlock()
	}
	return games, err
}

// ListActiveCorrespondence lists all active correspondence games.
// Don't cache correspondence game lists, always query DB.
func (c *Cache) ListActiveCorrespondence(ctx context.Context) (*pb.GameInfoResponses, error) {
	return c.backing.ListActiveCorrespondence(ctx)
}

// ListActiveCorrespondenceForUser lists active correspondence games for a specific user.
// Don't cache correspondence game lists, always query DB.
func (c *Cache) ListActiveCorrespondenceForUser(ctx context.Context, userID string) (*pb.GameInfoResponses, error) {
	return c.backing.ListActiveCorrespondenceForUser(ctx, userID)
}

// ListActiveCorrespondenceForUserAndLeague lists active correspondence games for a specific user in a specific league.
// Don't cache correspondence game lists, always query DB.
func (c *Cache) ListActiveCorrespondenceForUserAndLeague(ctx context.Context, leagueID uuid.UUID, userID string) (*pb.GameInfoResponses, error) {
	return c.backing.ListActiveCorrespondenceForUserAndLeague(ctx, leagueID, userID)
}

// ListActiveCorrespondenceRaw returns raw DB rows for correspondence games.
// This is used by the adjudication process to check timeouts without loading full games.
// Don't cache correspondence game lists, always query DB.
func (c *Cache) ListActiveCorrespondenceRaw(ctx context.Context) ([]models.ListActiveCorrespondenceGamesRow, error) {
	return c.backing.ListActiveCorrespondenceRaw(ctx)
}

func (c *Cache) Count(ctx context.Context) (int64, error) {
	return c.backing.Count(ctx)
}

// CachedCount reports zero: no games are held in memory any more. Kept only so
// the interface in pkg/gameplay does not have to change in this commit.
func (c *Cache) CachedCount(ctx context.Context) int {
	return 0
}

func (c *Cache) Disconnect() {
	c.backing.Disconnect()
}

func (c *Cache) SetReady(ctx context.Context, gid string, pidx int) (int, error) {
	return c.backing.SetReady(ctx, gid, pidx)
}

func (c *Cache) GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error) {
	return c.backing.GetHistory(ctx, id)
}

func (c *Cache) InsertGamePlayers(ctx context.Context, g *entity.Game) error {
	return c.backing.InsertGamePlayers(ctx, g)
}

func (c *Cache) SetTimerModuleCreator(creator TimerModuleCreator) {
	c.backing.SetTimerModuleCreator(creator)
}

// SetHistoryFetcher wires the S3 history reader into the backing DB store.
func (c *Cache) SetHistoryFetcher(f HistoryFetcher) {
	c.backing.SetHistoryFetcher(f)
}

func (c *Cache) StageTurns(g *entity.Game, startIdx int, events []*macondopb.GameEvent) error {
	return c.backing.StageTurns(g, startIdx, events)
}

func (c *Cache) AppendTurns(ctx context.Context, gameUUID string, startIdx int, events []*macondopb.GameEvent) error {
	return c.backing.AppendTurns(ctx, gameUUID, startIdx, events)
}

func (c *Cache) GetTurns(ctx context.Context, gameUUID string) ([]models.GetGameTurnsRow, error) {
	return c.backing.GetTurns(ctx, gameUUID)
}

func (c *Cache) DeleteTurns(ctx context.Context, gameUUID string) error {
	return c.backing.DeleteTurns(ctx, gameUUID)
}

func (c *Cache) CommitArchival(ctx context.Context, gameUUID string, s3Key string, archivedTurns int) error {
	return c.backing.CommitArchival(ctx, gameUUID, s3Key, archivedTurns)
}

func (c *Cache) SetHistoryS3Key(ctx context.Context, gameUUID string, s3Key string) error {
	return c.backing.SetHistoryS3Key(ctx, gameUUID, s3Key)
}

// LockGame acquires the game's lock, serializing everything that plays a move
// on it -- across app servers, not merely across goroutines here.
//
// It used to be a process-local map of mutexes, which was correct while there
// was one server and became a liability the moment there might be two: two
// processes would each take their own mutex, each load the same position, and
// each save a move onto it, the second discarding the first.
//
// The caller MUST release it. See lock.go for what is being held.
func (c *Cache) LockGame(ctx context.Context, gameID string) (*GameLock, error) {
	return c.backing.LockGame(ctx, gameID)
}

// StopCleanup is retained as a no-op so shutdown code does not have to change.
// There is no longer a background goroutine to stop.
func (c *Cache) StopCleanup() {}
