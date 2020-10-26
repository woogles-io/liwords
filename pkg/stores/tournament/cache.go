package tournament

import (
	"context"
	"errors"
	"sort"

	"github.com/domino14/liwords/pkg/entity"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
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

func (c *Cache) SetTournamentControls(ctx context.Context, id string, name string, description string, controls *entity.TournamentControls) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	if t.IsStarted {
		return errors.New("Cannot change tournament controls after it has started.")
	}

	t.Name = name
	t.Description = description
	t.Controls = controls

	return c.backing.Set(ctx, t)
}

func (c *Cache) AddDirectors(ctx context.Context, id string, directors *entity.TournamentPersons) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	err = entity.AddDirectors(t, directors)
	if err != nil {
		return err
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) RemoveDirectors(ctx context.Context, id string, directors *entity.TournamentPersons) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	err = entity.RemoveDirectors(t, directors)
	if err != nil {
		return err
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) AddPlayers(ctx context.Context, id string, players *entity.TournamentPersons) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	err = entity.AddPlayers(t, players)
	if err != nil {
		return err
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) RemovePlayers(ctx context.Context, id string, players *entity.TournamentPersons) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	err = entity.RemovePlayers(t, players)
	if err != nil {
		return err
	}

	return c.backing.Set(ctx, t)
}

func (c *Cache) SetPairing(ctx context.Context, id string, playerOneId string, playerTwoId string, round int) error {

	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	if !t.IsStarted {
		return errors.New("Cannot set tournament pairings before the tournament has started.")
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

	if !t.IsStarted {
		return errors.New("Cannot set tournament results before the tournament has started.")
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

func (c *Cache) IsStarted(ctx context.Context, id string) (bool, error) {
	t, err := c.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return t.IsStarted, nil
}

func (c *Cache) IsRoundComplete(ctx context.Context, id string, round int) (bool, error) {
	t, err := c.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, errors.New("Cannot check if round is complete before the tournament has started.")
	}

	return t.TournamentManager.IsRoundComplete(round)
}

func (c *Cache) IsFinished(ctx context.Context, id string) (bool, error) {
	t, err := c.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, errors.New("Cannot check if tournament is finished before the tournament has started.")
	}

	return t.TournamentManager.IsFinished()
}

func (c *Cache) StartRound(ctx context.Context, id string, round int) error {
	t, err := c.Get(ctx, id)
	if err != nil {
		return err
	}

	if round == 0 {
		if t.IsStarted {
			return errors.New("The tournament has already been started.")
		}
		startTournament(t)
	}

	err = t.TournamentManager.StartRound(round)
	if err != nil {
		return err
	}

	return c.backing.Set(ctx, t)
}

// Unload unloads the tournament from the cache
func (c *Cache) Unload(ctx context.Context, id string) {
	c.cache.Remove(id)
}

func (c *Cache) Disconnect() {
	c.backing.Disconnect()
}

func startTournament(t *entity.Tournament) error {
	rankedPlayers := rankPlayers(t.Players)

	if t.Controls.Type == entity.ClassicTournamentType {
		tm, err := entity.NewTournamentClassic(rankedPlayers,
			t.Controls.NumberOfRounds, t.Controls.PairingMethods, t.Controls.GamesPerRound)
		if err != nil {
			return err
		}
		t.TournamentManager = tm
	} else {
		return errors.New("Only Classic Tournaments have been implemented")
	}
	t.IsStarted = true
	return nil
}

func rankPlayers(players *entity.TournamentPersons) []string {
	// Sort players by descending int (which is probably rating)
	var values []int
	for _, v := range players.Persons {
		values = append(values, v)
	}
	sort.Ints(values)
	reversedPlayersMap := reverseMap(players.Persons)
	rankedPlayers := []string{}
	for i := len(values) - 1; i >= 0; i-- {
		rankedPlayers = append(rankedPlayers, reversedPlayersMap[values[i]])
	}
	return rankedPlayers
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

func reverseMap(m map[string]int) map[int]string {
	n := make(map[int]string)
	for k, v := range m {
		n[v] = k
	}
	return n
}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}
