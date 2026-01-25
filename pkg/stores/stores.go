package stores

import (
	redigoredis "github.com/gomodule/redigo/redis"
	"github.com/jackc/pgx/v5/pgxpool"

	cfg "github.com/woogles-io/liwords/pkg/config"
	owstores "github.com/woogles-io/liwords/pkg/omgwords/stores"

	"github.com/woogles-io/liwords/pkg/stores/comments"
	"github.com/woogles-io/liwords/pkg/stores/config"
	"github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/league"
	"github.com/woogles-io/liwords/pkg/stores/mod"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/stores/puzzles"
	"github.com/woogles-io/liwords/pkg/stores/redis"
	"github.com/woogles-io/liwords/pkg/stores/session"
	"github.com/woogles-io/liwords/pkg/stores/soughtgame"
	"github.com/woogles-io/liwords/pkg/stores/stats"
	"github.com/woogles-io/liwords/pkg/stores/tournament"
	"github.com/woogles-io/liwords/pkg/stores/user"
)

type Stores struct {
	UserStore       *user.DBStore
	GameStore       *game.Cache // we need to get rid of this cache
	SoughtGameStore *soughtgame.DBStore
	PresenceStore   *redis.RedisPresenceStore
	ChatStore       *redis.RedisChatStore
	ListStatStore   *stats.DBStore
	NotorietyStore  *mod.DBStore
	TournamentStore *tournament.Cache // this cache too
	ConfigStore     *config.RedisConfigStore
	SessionStore    *session.DBStore
	PuzzleStore     *puzzles.DBStore
	CommentsStore   *comments.DBStore
	LeagueStore     *league.DBStore

	// Refactor this soon:
	GameDocumentStore  *owstores.GameDocumentStore
	AnnotatedGameStore *owstores.DBStore
	// We probably are going to be moving everything to a single queries thingy
	// like this:
	Queries *models.Queries

	// LeagueStandingsUpdater is injected to avoid circular dependencies between
	// pkg/gameplay and pkg/league. It's set during initialization.
	LeagueStandingsUpdater LeagueStandingsUpdater
}

func NewInitializedStores(dbPool *pgxpool.Pool, redisPool *redigoredis.Pool, cfg *cfg.Config) (*Stores, error) {
	stores := &Stores{}
	var err error
	stores.UserStore, err = user.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}

	stores.SessionStore, err = session.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}

	tmpGameStore, err := game.NewDBStore(cfg, stores.UserStore, dbPool)
	if err != nil {
		return nil, err
	}

	stores.GameStore = game.NewCache(tmpGameStore)

	tmpTournamentStore, err := tournament.NewDBStore(cfg, stores.GameStore)
	if err != nil {
		return nil, err
	}
	stores.TournamentStore = tournament.NewCache(tmpTournamentStore)

	stores.SoughtGameStore, err = soughtgame.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}
	stores.ConfigStore = config.NewRedisConfigStore(redisPool)
	stores.ListStatStore, err = stats.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}

	stores.NotorietyStore, err = mod.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}
	stores.PresenceStore = redis.NewRedisPresenceStore(redisPool)

	stores.PuzzleStore, err = puzzles.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}
	stores.GameDocumentStore, err = owstores.NewGameDocumentStore(cfg, redisPool, dbPool)
	if err != nil {
		return nil, err
	}

	stores.AnnotatedGameStore, err = owstores.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}
	stores.CommentsStore, err = comments.NewDBStore(dbPool)
	if err != nil {
		return nil, err
	}

	stores.LeagueStore, err = league.NewDBStore(cfg, dbPool)
	if err != nil {
		return nil, err
	}

	// Initialize ChatStore after LeagueStore is ready
	stores.ChatStore = redis.NewRedisChatStore(redisPool, stores.PresenceStore, stores.TournamentStore, stores.LeagueStore)

	stores.Queries = models.New(dbPool)

	// Note: LeagueStandingsUpdater is set separately after initialization to avoid
	// circular import dependencies. See SetLeagueStandingsUpdater().

	return stores, nil
}

// SetLeagueStandingsUpdater sets the league standings updater implementation.
// This must be called after NewInitializedStores to wire up league standings updates.
// It's separate from initialization to avoid circular import dependencies.
func (s *Stores) SetLeagueStandingsUpdater(updater LeagueStandingsUpdater) {
	s.LeagueStandingsUpdater = updater
}

// Disconnect disconnects from all stores
func (s *Stores) Disconnect() {
	if s.UserStore != nil {
		s.UserStore.Disconnect()
	}
	if s.GameStore != nil {
		s.GameStore.Disconnect()
	}
	if s.SoughtGameStore != nil {
		s.SoughtGameStore.Disconnect()
	}
	if s.ListStatStore != nil {
		s.ListStatStore.Disconnect()
	}
	if s.NotorietyStore != nil {
		s.NotorietyStore.Disconnect()
	}
	if s.TournamentStore != nil {
		s.TournamentStore.Disconnect()
	}
	if s.SessionStore != nil {
		s.SessionStore.Disconnect()
	}
	if s.PuzzleStore != nil {
		s.PuzzleStore.Disconnect()
	}
	if s.CommentsStore != nil {
		s.CommentsStore.Disconnect()
	}
}
