package tournament

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"

	realtime "github.com/domino14/liwords/rpc/api/proto/realtime"
	pb "github.com/domino14/liwords/rpc/api/proto/tournament_service"
)

type PlayerSorter []RatedPlayer

func (a PlayerSorter) Len() int           { return len(a) }
func (a PlayerSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PlayerSorter) Less(i, j int) bool { return a[i].Rating > a[j].Rating }

type TournamentStore interface {
	Get(context.Context, string) (*entity.Tournament, error)
	GetBySlug(context.Context, string) (*entity.Tournament, error)
	Set(context.Context, *entity.Tournament) error
	Create(context.Context, *entity.Tournament) error
	GetRecentGames(ctx context.Context, tourneyID string, numGames int, offset int) (*pb.RecentGamesResponse, error)
	Unload(context.Context, string)
	SetTournamentEventChan(c chan<- *entity.EventWrapper)
	TournamentEventChan() chan<- *entity.EventWrapper
	ListAllIDs(context.Context) ([]string, error)

	GetRecentClubSessions(ctx context.Context, clubID string, numSessions int, offset int) (*pb.ClubSessionsResponse, error)
	AddRegistrants(ctx context.Context, tid string, userIDs []string, division string) error
	RemoveRegistrants(ctx context.Context, tid string, userIDs []string, division string) error
	ActiveTournamentsFor(ctx context.Context, userID string) ([][2]string, error)
}

func HandleTournamentGameEnded(ctx context.Context, ts TournamentStore, us user.Store,
	g *entity.Game) error {

	Results := []realtime.TournamentGameResult{realtime.TournamentGameResult_DRAW,
		realtime.TournamentGameResult_WIN,
		realtime.TournamentGameResult_LOSS}

	p1idx, p2idx := 0, 1
	p1result, p2result := Results[g.WinnerIdx+1], Results[g.LoserIdx+1]
	if g.History().SecondWentFirst {
		p1idx, p2idx = p2idx, p1idx
		p1result, p2result = p2result, p1result
	}

	return SetResult(ctx,
		ts,
		us,
		g.TournamentData.Id,
		g.TournamentData.Division,
		g.History().Players[p1idx].UserId,
		g.History().Players[p2idx].UserId,
		int(g.History().FinalScores[p1idx]),
		int(g.History().FinalScores[p2idx]),
		p1result,
		p2result,
		g.GameEndReason,
		g.TournamentData.Round,
		g.TournamentData.GameIndex,
		false,
		g)
}

func NewTournament(ctx context.Context,
	tournamentStore TournamentStore,
	name string,
	description string,
	directors *realtime.TournamentPersons,
	ttype entity.CompetitionType,
	parent string,
	slug string,
) (*entity.Tournament, error) {

	executiveDirector, err := getExecutiveDirector(directors)
	if err != nil {
		return nil, err
	}

	id := shortuuid.New()

	entTournament := &entity.Tournament{Name: name,
		Description:       description,
		Directors:         directors,
		ExecutiveDirector: executiveDirector,
		IsStarted:         false,
		Divisions:         map[string]*entity.TournamentDivision{},
		UUID:              id,
		Type:              ttype,
		ParentID:          parent,
		Slug:              slug,
	}

	err = tournamentStore.Create(ctx, entTournament)
	if err != nil {
		return nil, err
	}
	return entTournament, nil
}

// SendTournamentMessage sends updated tournament information on the channel.
func SendTournamentMessage(ctx context.Context, ts TournamentStore, id string) error {

	_, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	eventChannel := ts.TournamentEventChan()

	tdevt, err := TournamentDataResponse(ctx, ts, id)
	if err != nil {
		return err
	}
	log.Debug().Interface("tdevt", tdevt).Msg("sending tournament message")
	wrapped := entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_MESSAGE)
	wrapped.AddAudience(entity.AudTournament, id)
	if eventChannel != nil {
		eventChannel <- wrapped
	}
	log.Debug().Str("tid", id).Msg("sent tournament message")
	return nil
}

// SendTournamentDivisionMessage sends an updated division on the channel.
func SendTournamentDivisionMessage(ctx context.Context, ts TournamentStore, id string,
	division string) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	eventChannel := ts.TournamentEventChan()

	_, ok := t.Divisions[division]

	var wrapped *entity.EventWrapper
	if ok {
		tdevt, err := TournamentDivisionDataResponse(ctx, ts, id, division)
		if err != nil {
			return err
		}
		log.Debug().Interface("tdevt", tdevt).Msg("sending tournament division message")
		wrapped = entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_DIVISION_MESSAGE)
	} else {
		tddevt := &realtime.TournamentDivisionDeletedResponse{Id: id, Division: division}
		log.Debug().Interface("tdevt", tddevt).Msg("sending tournament division deleted message")
		wrapped = entity.WrapEvent(tddevt, realtime.MessageType_TOURNAMENT_DIVISION_DELETED_MESSAGE)
	}
	wrapped.AddAudience(entity.AudTournament, id)
	if eventChannel != nil {
		eventChannel <- wrapped
	}
	log.Debug().Str("tid:did", id+":"+division).Msg("sent tournament division changed or deleted message")
	return nil
}

func TournamentSetPairingsEvent(ctx context.Context, ts TournamentStore, id string, division string) error {
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func validateTournamentMeta(ttype pb.TType, slug string) (entity.CompetitionType, error) {
	var tt entity.CompetitionType
	switch ttype {
	case pb.TType_CLUB:
		tt = entity.TypeClub
		if !strings.HasPrefix(slug, "/club/") {
			return "", twirp.NewError(twirp.InvalidArgument, "club slug must start with /club/")
		}
	case pb.TType_STANDARD:
		tt = entity.TypeStandard
		if !strings.HasPrefix(slug, "/tournament/") {
			return "", twirp.NewError(twirp.InvalidArgument, "tournament slug must start with /tournament/")
		}
	case pb.TType_LEGACY:
		tt = entity.TypeLegacy
		if !strings.HasPrefix(slug, "/tournament/") {
			return "", twirp.NewError(twirp.InvalidArgument, "tournament slug must start with /tournament/")
		}
	case pb.TType_CHILD:
		tt = entity.TypeChild
		// A Club session type can also be a child tournament (it's essentially just a tournament with a parent ID)
		if !strings.HasPrefix(slug, "/club/") && !strings.HasPrefix(slug, "/tournament/") {
			return "", twirp.NewError(twirp.InvalidArgument, "club-session slug must start with /club/ or /tournament/")
		}
	default:
		return "", twirp.NewError(twirp.InvalidArgument, "invalid tournament type")
	}
	return tt, nil
}

func SetTournamentMetadata(ctx context.Context, ts TournamentStore, id string, name string, description string,
	slug string, ttype entity.CompetitionType) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	t.Lock()
	defer t.Unlock()
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name cannot be blank")
	}
	t.Name = name
	t.Description = description
	t.Slug = slug
	t.Type = ttype

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentMessage(ctx, ts, id)
}

func SetSingleRoundControls(ctx context.Context, ts TournamentStore, id string, division string, round int, controls *realtime.RoundControl) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if round < 0 || round >= len(divisionObject.Controls.RoundControls) {
		return fmt.Errorf("round number %d out or range for division %s", round, division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division manager null for division %s", division)
	}

	err = divisionObject.DivisionManager.SetSingleRoundControls(round, controls)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func SetTournamentControls(ctx context.Context, ts TournamentStore, id string, division string, controls *realtime.TournamentControls) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if t.IsStarted {
		return errors.New("cannot change tournament controls after it has started")
	}

	divisionObject.Controls = controls

	err = createDivisionManager(t, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func AddDivision(ctx context.Context, ts TournamentStore, id string, division string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	if t.IsStarted {
		return errors.New("cannot add division after the tournament has started")
	}

	_, ok := t.Divisions[division]

	if ok {
		return fmt.Errorf("division %s already exists", division)
	}

	t.Divisions[division] = &entity.TournamentDivision{DivisionManager: NewClassicDivision()}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func RemoveDivision(ctx context.Context, ts TournamentStore, id string, division string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	_, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if t.IsStarted {
		return fmt.Errorf("cannot remove division %s after the tournament has started", division)
	}

	delete(t.Divisions, division)
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func constructFullID(ctx context.Context, us user.Store, user string) (string, error) {
	name := player.Name
	u, err := us.Get(ctx, player)
	if err != nil {
		return fmt.Errorf("full ID for player %s could not be constructed: %s", player, err.Error())
	}
	return u.UUID + ":" + u.Username
}

func AddDirectors(ctx context.Context, ts TournamentStore, us user.Store, id string, directors *realtime.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	// Only perform the add operation if all persons can be added.

	for director := range directors.Persons {
		fullID := constructFullID(ctx, us, player.Name)
		if err != nil {
			return err
		}
		director.Name = fullID
	}

	// Not very efficient but there's only gonna be like
	// a maximum of maybe 10 existing directors

	for idx, newDirector := range directors.Persons {
		if newDirector.Name == t.ExecutiveDirector || newDirector.Rating == 0 {
			return errors.New("cannot add another executive director %s, %d", newDirector.Name, newDirector.Rating)
		}
		for _, existingDirector := range t.Directors {
			if newDirector.Name == existingDirector.Name {
				return errors.New("director %s already exists", newDirector.Name)
			}
		}
	}

	t.Directors.Persons = append(t.Directors.Persons, directors.Persons...)

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func RemoveDirectors(ctx context.Context, ts TournamentStore, us user.Store, id string, directors *realtime.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	// Not very efficient but there's only gonna be like
	// a maximum of maybe 10 existing directors

	indexesToRemove := []int{}
	for idx, newDirector := range directors.Persons {
		index := -1
		for idx, existingDirector := range t.Directors {
			if newDirector.Name == existingDirector.Name {
				if existingDirector.Rating == 0 {
					return errors.New("cannot remove the executive director: %s", existingDirector.Name)
				}
				index = idx
				break
			}
		}
		if index == -1 {
			return fmt.Errorf("director %s does not exist", newDirector.Name)
		} else {
			indexesToRemove = append(indexesToRemove, index)
			da := t.Directors.Persons
			da[len(da)-1], da[i] = da[i], da[len(da)-1]
    		t.Directors.Persons = da[:len(da)-1]
		}
	}

	for i := len(t.Directors.Persons) - 1; i >= 0; i-- {
		da := t.Directors.Persons
		da[len(da)-1], da[i] = da[i], da[len(da)-1]
		t.Directors.Persons = da[:len(da)-1]
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func AddPlayers(ctx context.Context, ts TournamentStore, us user.Store, id string, division string, players *realtime.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division %s does not have a division manager", division)
	}

	// Only perform the add operation if all persons can be added.

	userUUIDs := []string{}
	for player := range players.Persons {
		fullID := constructFullID(ctx, us, player.Name)
		if err != nil {
			return err
		}
		player.Name = fullID
		userUUIDs = append(userUUIDs, u.UUID)
	}

	err = divisionObject.DivisionManager.AddPlayers(players)
	if err != nil {
		return err
	}

	err = ts.AddRegistrants(ctx, t.UUID, userUUIDs, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func RemovePlayers(ctx context.Context, ts TournamentStore, us user.Store, id string, division string, players *realtime.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division %s does not have a division manager", division)
	}

	// Only perform the add operation if all persons can be added.

	for player := range players.Persons {
		fullID, err := constructFullID(ctx, us, player.Name)
		if err != nil {
			return err
		}
		player.Name = fullID
	}

	err = divisionObject.DivisionManager.RemovePlayers(players)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}
}

// SetPairings is only called by the API
func SetPairings(ctx context.Context, ts TournamentStore, id string, division string, pairings []*pb.TournamentPairingRequest) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	for _, pairing := range pairings {
		divisionObject, ok := t.Divisions[division]

		if !ok {
			return fmt.Errorf("division %s does not exist", division)
		}

		if divisionObject.DivisionManager == nil {
			return fmt.Errorf("division %s does not have enough players or controls to set pairings", division)
		}

		err = divisionObject.DivisionManager.SetPairing(pairing.PlayerOneId, pairing.PlayerTwoId, int(pairing.Round), pairing.IsForfeit)
		if err != nil {
			return err
		}
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return TournamentSetPairingsEvent(ctx, ts, id, division)
}

// SetResult sets the result for the game. Note: playerOne and playerTwo
// went first and second, respectively.
func SetResult(ctx context.Context,
	ts TournamentStore,
	us user.Store,
	id string,
	division string,
	playerOneId string, // user UUID
	playerTwoId string, // user UUID
	playerOneScore int,
	playerTwoScore int,
	playerOneResult realtime.TournamentGameResult,
	playerTwoResult realtime.TournamentGameResult,
	reason realtime.GameEndReason,
	round int,
	gameIndex int,
	amendment bool,
	g *entity.Game) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	testMode := false
	if strings.HasSuffix(os.Args[0], ".test") {
		testMode = true
	}

	if !testMode && (t.Type == entity.TypeClub || t.Type == entity.TypeLegacy) {
		// This game was played in a legacy "Clubhouse".
		// This is a tournament of "club" type (note, not a club *session*) or
		// a tournament of "legacy" type.
		// This is a casual type of tournament game with no defined divisions, pairings,
		// game settings, etc, so we bypass all of this code and just send a
		// tournament game ended message.

		p1user, err := us.GetByUUID(ctx, playerOneId)
		if err != nil {
			return err
		}
		p2user, err := us.GetByUUID(ctx, playerTwoId)
		if err != nil {
			return err
		}

		players := []*realtime.TournamentGameEndedEvent_Player{
			{Username: p1user.Username, Score: int32(playerOneScore), Result: playerOneResult},
			{Username: p2user.Username, Score: int32(playerTwoScore), Result: playerTwoResult},
		}

		gameID := ""
		if g != nil {
			gameID = g.GameID()
		}
		tevt := &realtime.TournamentGameEndedEvent{
			GameId:    gameID,
			Players:   players,
			EndReason: reason,
			Round:     int32(round),
			Time:      time.Now().Unix(),
		}
		log.Debug().Interface("tevt", tevt).Msg("sending legacy tournament game ended evt")
		wrapped := entity.WrapEvent(tevt, realtime.MessageType_TOURNAMENT_GAME_ENDED_EVENT)
		wrapped.AddAudience(entity.AudTournament, id)
		evtChan := ts.TournamentEventChan()
		if evtChan != nil {
			evtChan <- wrapped
		}
		log.Debug().Str("tid", id).Msg("sent legacy tournament game ended event")
		return nil
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if !t.IsStarted {
		return errors.New("cannot set tournament results before the tournament has started")
	}

	// We need to send the division manager the "full" user ID, so look that up here.
	p1, err := us.GetByUUID(ctx, playerOneId)
	if err != nil {
		return err
	}
	p2, err := us.GetByUUID(ctx, playerTwoId)
	if err != nil {
		return err
	}

	gid := ""
	if g != nil {
		gid = g.GameID()
	}

	err = divisionObject.DivisionManager.SubmitResult(round,
		p1.UUID+":"+p1.Username,
		p2.UUID+":"+p2.Username,
		playerOneScore,
		playerTwoScore,
		playerOneResult,
		playerTwoResult,
		reason,
		amendment,
		gameIndex,
		gid)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func StartAllRoundCountdowns(ctx context.Context, ts TournamentStore, id string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	// Do not lock, StartRound will do that

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	for division := range t.Divisions {
		t.IsStarted = true
		err := StartRoundCountdown(ctx, ts, id, division, round,false)
		if err != nil {
			return err
		}
	}
	if !t.IsStarted {
		return fmt.Errorf("cannot start tournament %s with no divisions", t.Name)
	}
	return ts.Set(ctx, t)
}

// DivisionChannelName returns a channel name that can be used
// for sending communications regarding a tournament and division.
func DivisionChannelName(tid, division string) string {
	// We encode to b64 because division can contain spaces.
	return string(base64.URLEncoding.EncodeToString([]byte(tid + ":" + division)))
}

func StartRoundCountdown(ctx context.Context, ts TournamentStore, id string,
	division string, round int, save bool) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division %s does not have enough players or controls to set pairings", division)
	}

/*	We will deal with auto start at a later date
	if manual && divisionObject.Controls.AutoStart && divisionObject.DivisionManager.IsStarted() {
		return fmt.Errorf("division %s has autostart enabled and cannot be manually started", division)
	}*/

	if !t.IsStarted {
		return fmt.Errorf("cannot start division %s before starting the tournament", t.Name)
	}

	ready, err := divisionObject.DivisionManager.IsRoundReady(round)
	if err != nil {
		return err
	}

	if ready {
		err = divisionObject.DivisionManager.StartRound()
		if err != nil {
			return err
		}
		// Send code that sends signal to all tournament players that backend
		// is now accepting "ready" messages for this round.
		eventChannel := ts.TournamentEventChan()
		evt := &realtime.TournamentRoundStarted{
			TournamentId: id,
			Division:     division,
			Round:        int32(round),
			// GameIndex: int32(0) -- fix this when we have other types of tournaments
			// add timestamp deadline here as well at some point
		}
		wrapped := entity.WrapEvent(evt, realtime.MessageType_TOURNAMENT_ROUND_STARTED)

		// Send it to everyone in this division across the app.
		wrapped.AddAudience(entity.AudChannel, DivisionChannelName(id, division))
		// Also send it to the tournament realm.
		wrapped.AddAudience(entity.AudTournament, id)
		if eventChannel != nil {
			eventChannel <- wrapped
		}
		divisionObject.DivisionManager.SetLastStarted(evt)
		log.Debug().Interface("evt", evt).Msg("sent-tournament-round-started")
	} else {
		return fmt.Errorf("division %s round %d is not ready to be started", division, round)
	}
	if save {
		err = ts.Set(ctx, t)
		if err != nil {
			return err
		}
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func PairRound(ctx context.Context, ts TournamentStore, id string, division string, round int) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if !t.IsStarted {
		return errors.New("cannot pair a round before the tournament has started")
	}

	currentRound := divisionObject.DivisionManager.GetCurrentRound()
	if round < currentRound + 1 {
		return errors.New("cannot repair non-future round %d because current round is %d", round, currentRound)
	}

	err = divisionObject.DivisionManager.PairRound(round)

	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentDivisionMessage(ctx, ts, id, division)
}

func IsStarted(ctx context.Context, ts TournamentStore, id string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}
	return t.IsStarted, nil
}

func IsRoundComplete(ctx context.Context, ts TournamentStore, id string, division string, round int) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	if !t.IsStarted {
		return false, errors.New("cannot check if round is complete before the tournament has started")
	}

	_, ok := t.Divisions[division]

	if !ok {
		return false, fmt.Errorf("division %s does not exist", division)
	}

	return t.Divisions[division].DivisionManager.IsRoundComplete(round)
}

func SetFinished(ctx context.Context, ts TournamentStore, id string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	if t.IsFinished {
		return fmt.Errorf("tournament %s has already finished", id)
	}

	if !t.IsStarted {
		return errors.New("cannot finish the tournament before the tournament has started")
	}

	for divisionKey, division := range t.Divisions {
		if division.DivisionManager == nil {
			return fmt.Errorf("division %s has nil division manager, cannot finish tournament", divisionKey)
		}
		finished, err := division.DivisionManager.IsFinished()
		if err != nil {
			return nil
		}
		if !finished {
			return fmt.Errorf("cannot finish tournament, division %s is not done", divisionKey)
		}
	}

	t.IsFinished = true

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return SendTournamentMessage(ctx, ts, id)
}

func IsFinished(ctx context.Context, ts TournamentStore, id string, division string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, errors.New("cannot check if tournament is finished before the tournament has started")
	}

	_, ok := t.Divisions[division]

	if !ok {
		return false, fmt.Errorf("division %s does not exist", division)
	}

	return t.Divisions[division].DivisionManager.IsFinished()
}

func SetReadyForGame(ctx context.Context, ts TournamentStore, t *entity.Tournament,
	playerID, connID, division string,
	round, gameIndex int, unready bool) ([]string, bool, error) {

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	_, ok := t.Divisions[division]
	if !ok {
		return nil, false, fmt.Errorf("division %s does not exist", division)
	}

	connIDs, bothReady, err := t.Divisions[division].DivisionManager.SetReadyForGame(playerID, connID, round, gameIndex, unready)
	if err != nil {
		return nil, false, err
	}
	return connIDs, bothReady, ts.Set(ctx, t)
}

func ClearReadyStates(ctx context.Context, ts TournamentStore, t *entity.Tournament,
	division, userID string, round, gameIndex int) error {

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", id)
	}

	_, ok := t.Divisions[division]
	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	err := t.Divisions[division].DivisionManager.ClearReadyStates(userID, round, gameIndex)
	if err != nil {
		return err
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	return SendTournamentDivisionMessage(ctx, ts, t.UUID, division)
}

func TournamentDataResponse(ctx context.Context, ts TournamentStore, id string) (*realtime.TournamentDataResponse, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return &realtime.TournamentDataResponse{Id: t.UUID,
		Name:              t.Name,
		Description:       t.Description,
		ExecutiveDirector: t.ExecutiveDirector,
		Directors:         t.Directors,
		IsStarted:         t.IsStarted}, nil
}

func TournamentDivisionDataResponse(ctx context.Context, ts TournamentStore,
	id string, division string) (*realtime.TournamentDivisionDataResponse, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return nil, fmt.Errorf("division %s does not exist", division)
	}

	response := &realtime.TournamentDivisionDataResponse{}
	if divisionObject.DivisionManager != nil {
		response, err = divisionObject.DivisionManager.ToResponse()
		if err != nil {
			return nil, err
		}
	}
	response.Id = id
	response.DivisionId = division
	// response.Controls = divisionObject.Controls
	log.Debug().
		Str("division", division).
		Str("id", id).Msg("tournament-division-data-response")
	return response, nil
}

func getExecutiveDirector(directors *realtime.TournamentPersons) (string, error) {
	err := errors.New("tournament must have exactly one executive director")
	ed := ""
	for k, v := range directors.Persons {
		if v == 0 {
			if ed != "" {
				return "", err
			} else {
				ed = k
			}
		}
	}
	if ed == "" {
		return "", err
	}
	return ed, nil
}