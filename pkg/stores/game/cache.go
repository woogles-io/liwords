package game

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	gs "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

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
	ListFrozenGameIDs(ctx context.Context) ([]string, error)
	Count(ctx context.Context) (int64, error)
	GameEventChan() chan<- *entity.EventWrapper
	SetGameEventChan(ch chan<- *entity.EventWrapper)
	Disconnect()
	SetReady(ctx context.Context, gid string, pidx int) (int, error)
	GetHistory(ctx context.Context, id string) (*macondopb.GameHistory, error)
	InsertGamePlayers(ctx context.Context, g *entity.Game) error
	SetTimerModuleCreator(creator TimerModuleCreator)
}

// TimerModuleCreator is a function that creates a new timer module for a game.
type TimerModuleCreator func() entity.Nower

const (
	// Assume every game takes up roughly 50KB in memory
	// This is roughly 200 MB and allows for 4000 simultaneous games.
	// We will want to increase this as we grow (as well as the size of our container)

	// Note: above is overly optimistic.
	// It seems each cache slot is taking about 750kB.
	// That's in addition to about 200MB base.
	// Reduced cache cap accordingly.
	CacheCap = 400

	// GameLockExpiration is how long a game lock can be idle before cleanup.
	// This should be longer than any possible request duration.
	GameLockExpiration = 30 * time.Minute

	// GameLockCleanupInterval is how often we check for expired locks.
	GameLockCleanupInterval = 5 * time.Minute
)

// gameLock holds a mutex and tracks when it was last used.
type gameLock struct {
	mu         sync.Mutex
	lastAccess time.Time
}

// Cache will reside in-memory, and will be per-node. If we add more nodes
// we will need to make sure only the right nodes respond to game requests.
type Cache struct {
	sync.RWMutex // used for the activeGames cache.
	cache        *lru.Cache
	activeGames  *pb.GameInfoResponses

	activeGamesTTL         time.Duration
	activeGamesLastUpdated time.Time

	backing backingStore

	// gameLocks provides per-game-ID locks for correspondence games.
	// Since correspondence games bypass the cache and each Get() returns a new
	// instance, the game's internal Lock() doesn't protect against concurrent
	// access from multiple goroutines. This map ensures only one goroutine can
	// work on a correspondence game at a time.
	gameLocks   map[string]*gameLock
	gameLocksMu sync.Mutex

	// stopCleanup is used to signal the cleanup goroutine to stop.
	stopCleanup chan struct{}
}

func NewCache(backing backingStore) *Cache {

	lrucache, err := lru.New(CacheCap)
	if err != nil {
		panic(err)
	}

	c := &Cache{
		backing: backing,
		cache:   lrucache,

		// games:   make(map[string]*entity.Game),
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

		gameLocks:   make(map[string]*gameLock),
		stopCleanup: make(chan struct{}),
	}

	// Start the cleanup goroutine
	go c.cleanupExpiredLocks()

	return c
}

// Unload unloads the game from the cache
func (c *Cache) Unload(ctx context.Context, id string) {
	c.cache.Remove(id)
	// Let's also expire the active games cache. The only time we ever
	// call Unload is when a game is over - so we don't want to go back
	// to the lobby and still show our game as active.
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
	// Check if we already have it in cache (correspondence games won't be cached)
	g, ok := c.cache.Get(id)
	if ok && g != nil {
		return g.(*entity.Game), nil
	}

	// Recheck after locking, to ensure it is still not there.
	c.Lock()
	defer c.Unlock()
	g, ok = c.cache.Get(id)
	if ok && g != nil {
		return g.(*entity.Game), nil
	}
	log.Debug().Str("gameid", id).Msg("not-in-cache")
	uncachedGame, err := c.backing.Get(ctx, id)
	if err == nil && !uncachedGame.IsCorrespondence() {
		// Only add to cache if it's not a correspondence game
		c.cache.Add(id, uncachedGame)
	}
	return uncachedGame, err

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
	if err != nil {
		return err
	}
	// Only add to cache if it's not a correspondence game
	if !game.IsCorrespondence() {
		c.cache.Add(gameID, game)
	}
	return nil
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

func (c *Cache) CachedCount(ctx context.Context) int {
	return c.cache.Len()
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

// FreezeAllGames freezes all active games in cache for maintenance.
// It calculates and persists the current time remaining for each game.
// Returns the number of games frozen and any error encountered.
func (c *Cache) FreezeAllGames(ctx context.Context) (int, error) {
	keys := c.cache.Keys()
	count := 0
	for _, key := range keys {
		g, ok := c.cache.Get(key)
		if !ok || g == nil {
			continue
		}
		game := g.(*entity.Game)
		// Only freeze games that are actively playing
		if game.Playing() != macondopb.PlayState_PLAYING {
			continue
		}
		game.FreezeTimers()
		if err := c.backing.Set(ctx, game); err != nil {
			log.Error().Err(err).Str("gameID", game.GameID()).Msg("failed to freeze game")
			continue
		}
		count++
	}
	return count, nil
}

// ListFrozenGameIDs returns the UUIDs of games that were frozen for maintenance.
func (c *Cache) ListFrozenGameIDs(ctx context.Context) ([]string, error) {
	return c.backing.ListFrozenGameIDs(ctx)
}

// LockGame acquires a lock for the given game ID.
// This is used to serialize access to correspondence games which bypass the cache.
// The caller MUST call UnlockGame when done.
func (c *Cache) LockGame(gameID string) {
	c.gameLocksMu.Lock()
	gl, ok := c.gameLocks[gameID]
	if !ok {
		gl = &gameLock{}
		c.gameLocks[gameID] = gl
	}
	c.gameLocksMu.Unlock()

	gl.mu.Lock()

	// Update last access time while holding the lock
	c.gameLocksMu.Lock()
	gl.lastAccess = time.Now()
	c.gameLocksMu.Unlock()
}

// UnlockGame releases the lock for the given game ID.
func (c *Cache) UnlockGame(gameID string) {
	c.gameLocksMu.Lock()
	gl, ok := c.gameLocks[gameID]
	c.gameLocksMu.Unlock()

	if ok {
		gl.mu.Unlock()
	}
}

// cleanupExpiredLocks periodically removes locks that haven't been accessed recently.
// This prevents memory leaks from accumulating locks for games that are no longer active.
func (c *Cache) cleanupExpiredLocks() {
	ticker := time.NewTicker(GameLockCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.gameLocksMu.Lock()
			now := time.Now()
			for gameID, gl := range c.gameLocks {
				// Only remove if the lock is not currently held and has expired.
				// TryLock returns true if we acquired the lock (meaning it wasn't held).
				if gl.mu.TryLock() {
					if now.Sub(gl.lastAccess) > GameLockExpiration {
						delete(c.gameLocks, gameID)
					}
					gl.mu.Unlock()
				}
			}
			c.gameLocksMu.Unlock()
		case <-c.stopCleanup:
			return
		}
	}
}

// StopCleanup stops the background lock cleanup goroutine.
// This should be called when the cache is being shut down.
func (c *Cache) StopCleanup() {
	close(c.stopCleanup)
}
