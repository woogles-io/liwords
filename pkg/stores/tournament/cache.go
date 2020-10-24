package tournament

import (
	"context"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	macondo "github.com/domino14/macondo/gen/api/proto/macondo"
)

type backingStore interface {
	Get(ctx context.Context, id string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	Disconnect()
}

const (
	// Increase this if we ever think we might be holding more than
	// 50 tournaments at a time.
	CacheCap = 50
)

// Cache will reside in-memory, and will be per-node.
type Cache struct {
	cache *lru.Cache

	backing backingStore
}

func NewCache(backing backingStore) *Cache {

	lrucache, err := lru.New(CacheCap)
	if err != nil {
		panic(err)
	}

	return &Cache{
		backing: backing,
		cache:   lrucache,
	}
}

// Get gets a tournament from the cache. It loads it into the cache if it's not there.
func (c *Cache) Get(ctx context.Context, id string) (*entity.Tournament, error) {
	tm, ok := c.cache.Get(id)
	if ok && tm != nil {
		return tm.(*entity.Tournament), nil
	}
	log.Info().Str("tournamentid", id).Msg("not-in-cache")
	uncachedTournament, err := c.backing.Get(ctx, id)
	if err == nil {
		c.cache.Add(id, uncachedTournament)
	}
	return uncachedTournament, err
}

// Set sets a tournament in the cache, AND in the backing store. This ensures if the
// node crashes the tournament doesn't just vanish.
func (c *Cache) Set(ctx context.Context, tm *entity.Tournament) error {
	return c.setOrCreate(ctx, tm, false)
}

// Create creates the tournament in the cache as well as the store.
func (c *Cache) Create(ctx context.Context, tm *entity.Tournament) error {
	return c.setOrCreate(ctx, tm, true)
}

func (c *Cache) setOrCreate(ctx context.Context, tm *entity.Tournament, isNew bool) error {
	var err error
	if isNew {
		err = c.backing.Create(ctx, tm)
	} else {
		err = c.backing.Set(ctx, tm)
	}
	if err != nil {
		return err
	}
	c.cache.Add(tm.UUID, tm)
	return nil
}

func (c *Cache) SetTournamentControls(ctx context.Context,
	id string,
	name string,
	lexicon string,
	variant string,
	timeControlName string,
	initialTimeSeconds int32,
	challengeRule macondo.ChallengeRule,
	ratingMode realtime.RatingMode,
	maxOvertimeMinutes int32,
	incrementSeconds int32,
	startTime time.Time) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Name = name
	t.Controls.Lexicon = lexicon
	t.Controls.Rules.VariantName = variant
	t.Controls.InitialTimeSeconds = initialTimeSeconds
	t.Controls.ChallengeRule = challengeRule
	t.Controls.RatingMode = ratingMode
	t.Controls.MaxOvertimeMinutes = maxOvertimeMinutes
	t.Controls.IncrementSeconds = incrementSeconds
	t.StartTime = startTime

	return c.backing.Set(ctx, t)
}

func (c *Cache) AddDirectors(ctx context.Context, id string, directors []string) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	// This is really slow but it shouldn't matter
	// because this operation should be relatively
	// extremely rare.
	indexesToRemove := indexesToRemove(directors, t.Directors)

	for i := len(indexesToRemove) - 1; i >= 0; i-- {
		remove(directors, indexesToRemove[i])
	}

	t.Directors = append(t.Directors, directors...)

	return c.backing.Set(ctx, t)
}

func (c *Cache) RemoveDirectors(ctx context.Context, id string, directors []string) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	indexesToRemove := indexesToRemove(t.Directors, directors)

	for i := len(indexesToRemove) - 1; i >= 0; i-- {
		remove(t.Directors, indexesToRemove[i])
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) AddPlayers(ctx context.Context, id string, players []string) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	indexesToRemove := indexesToRemove(players, t.Players)

	for i := len(indexesToRemove) - 1; i >= 0; i-- {
		remove(players, indexesToRemove[i])
	}

	t.Players = append(t.Players, players...)

	return c.backing.Set(ctx, t)
}

func (c *Cache) RemovePlayers(ctx context.Context, id string, players []string) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	indexesToRemove := indexesToRemove(t.Players, players)

	for i := len(indexesToRemove) - 1; i >= 0; i-- {
		remove(t.Players, indexesToRemove[i])
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) SetPairing(ctx context.Context, id string, playerOneId string, playerTwoId string, round int) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	err = t.TournamentManager.SetPairing(playerOneId, playerTwoId, round)
	if err != nil {
		return err
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) SetResult(ctx context.Context,
	id string,
	playerOneId string,
	playerTwoId string,
	playerOneScore int,
	playerTwoScore int,
	playerOneResult realtime.TournamentGameResult,
	playerTwoResult realtime.TournamentGameResult,
	reason realtime.GameEndReason,
	round int,
	gameIndex int,
	amendment bool) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	err = t.TournamentManager.SubmitResult(round,
		playerOneId,
		playerTwoId,
		playerOneScore,
		playerTwoScore,
		playerOneResult,
		playerTwoResult,
		reason,
		amendment,
		gameIndex)
	if err != nil {
		return err
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) IsRoundComplete(ctx context.Context, id string, round int) (bool, error) {
	t, err := c.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return t.TournamentManager.IsRoundComplete(round)
}

func (c *Cache) IsFinished(ctx context.Context, id string) (bool, error) {
	t, err := c.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return t.TournamentManager.IsFinished()
}

func (c *Cache) StartRound(ctx context.Context, id string, round int) error {
	return nil
}

// Unload unloads the tournament from the cache
func (c *Cache) Unload(ctx context.Context, id string) {
	c.cache.Remove(id)
}

func (c *Cache) Disconnect() {
	c.backing.Disconnect()
}

func indexesToRemove(alteredArray []string, referenceArray []string) []int {
	indexesToRemove := []int{}
	for i, existing := range alteredArray {
		for _, removed := range referenceArray {
			if existing == removed {
				indexesToRemove = append(indexesToRemove, i)
			}
		}
	}
	return indexesToRemove
}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}
