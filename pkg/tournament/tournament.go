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

const MaxDivisionNameLength = 24

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
	RemoveRegistrantsForTournament(ctx context.Context, tid string) error
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
		ExtraMeta:         &entity.TournamentMeta{},
	}

	err = tournamentStore.Create(ctx, entTournament)
	if err != nil {
		return nil, err
	}
	return entTournament, nil
}

// SendTournamentMessage sends updated tournament information on the channel.
func SendTournamentMessage(ctx context.Context, ts TournamentStore, id string, wrapped *entity.EventWrapper) error {

	_, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	eventChannel := ts.TournamentEventChan()

	wrapped.AddAudience(entity.AudTournament, id)
	if eventChannel != nil {
		eventChannel <- wrapped
	}
	log.Debug().Str("tid", id).Msg("sent tournament message type: " + string(wrapped.Type))
	return nil
}

func SetTournamentMetadata(ctx context.Context, ts TournamentStore, meta *pb.TournamentMetadata) error {

	ttype, err := validateTournamentMeta(meta.Type, meta.Slug)
	if err != nil {
		return err
	}

	t, err := ts.Get(ctx, meta.Id)
	if err != nil {
		return err
	}

	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", meta.Id)
	}

	t.Lock()
	defer t.Unlock()
	name := strings.TrimSpace(meta.Name)
	if name == "" {
		return errors.New("name cannot be blank")
	}
	t.Name = name
	t.Description = meta.Description
	t.Slug = meta.Slug
	t.Type = ttype
	t.ExtraMeta = &entity.TournamentMeta{
		Disclaimer:                meta.Disclaimer,
		TileStyle:                 meta.TileStyle,
		BoardStyle:                meta.BoardStyle,
		DefaultClubSettings:       meta.DefaultClubSettings,
		FreeformClubSettingFields: meta.FreeformClubSettingFields,
		Password:                  meta.Password,
		Logo:                      meta.Logo,
		Color:                     meta.Color,
		PrivateAnalysis:           meta.PrivateAnalysis,
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	tdevt, err := TournamentDataResponse(ctx, ts, meta.Id)
	if err != nil {
		return err
	}
	wrapped := entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_MESSAGE)
	return SendTournamentMessage(ctx, ts, meta.Id, wrapped)
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

	if !t.IsStarted {
		return errors.New("cannot set controls for a single round before the tournament has started")
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division manager null for division %s", division)
	}

	currentRound := divisionObject.DivisionManager.GetCurrentRound()
	if round < currentRound+1 {
		return fmt.Errorf("cannot set single round controls for non-future round %d since current round is %d", round, currentRound)
	}

	newControls, err := divisionObject.DivisionManager.SetSingleRoundControls(round, controls)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	wrapped := entity.WrapEvent(&realtime.DivisionRoundControls{
		Id:            id,
		Division:      division,
		RoundControls: []*realtime.RoundControl{newControls},
	}, realtime.MessageType_TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func SetRoundControls(ctx context.Context, ts TournamentStore, id string, division string, roundControls []*realtime.RoundControl) error {

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
		return errors.New("cannot set division round controls after it has started")
	}

	pairingsResp, newDivisionRoundControls, err := divisionObject.DivisionManager.SetRoundControls(roundControls)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	wrapped := entity.WrapEvent(&realtime.DivisionRoundControls{Id: id, Division: division,
		RoundControls:     newDivisionRoundControls,
		DivisionPairings:  pairingsResp.DivisionPairings,
		DivisionStandings: pairingsResp.DivisionStandings},
		realtime.MessageType_TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func SetDivisionControls(ctx context.Context, ts TournamentStore, id string, division string, controls *realtime.DivisionControls) error {

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

	newDivisionControls, standings, err := divisionObject.DivisionManager.SetDivisionControls(controls)
	if err != nil {
		return err
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	resp := &realtime.DivisionControlsResponse{
		Id:                id,
		Division:          division,
		DivisionControls:  newDivisionControls,
		DivisionStandings: standings,
	}
	wrapped := entity.WrapEvent(resp, realtime.MessageType_TOURNAMENT_DIVISION_CONTROLS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
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

	if len(division) == 0 || len(division) > MaxDivisionNameLength {
		return errors.New("your division name is too long or too short")
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
	tdevt, err := t.Divisions[division].DivisionManager.GetXHRResponse()
	if err != nil {
		return err
	}
	tdevt.Id = id
	tdevt.Division = division
	wrapped := entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_DIVISION_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
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

	if len(t.Divisions[division].DivisionManager.GetPlayers().GetPersons()) > 0 {
		return fmt.Errorf("cannot remove division %s since it has at least one player in it", division)
	}

	delete(t.Divisions, division)
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	tddevt := &realtime.TournamentDivisionDeletedResponse{Id: id, Division: division}
	wrapped := entity.WrapEvent(tddevt, realtime.MessageType_TOURNAMENT_DIVISION_DELETED_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func constructFullID(ctx context.Context, us user.Store, user string) (string, string, error) {
	u, err := us.Get(ctx, user)
	if err != nil {
		return "", "", fmt.Errorf("full ID for player %s could not be constructed: %s", user, err.Error())
	}
	return u.TournamentID(), u.UUID, nil
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

	for _, director := range directors.Persons {
		fullID, _, err := constructFullID(ctx, us, director.Id)
		if err != nil {
			return err
		}
		director.Id = fullID
	}

	// Not very efficient but there's only gonna be like
	// a maximum of maybe 10 existing directors

	for _, newDirector := range directors.Persons {
		if newDirector.Id == t.ExecutiveDirector || newDirector.Rating == 0 {
			return fmt.Errorf("cannot add another executive director %s, %d", newDirector.Id, newDirector.Rating)
		}
		for _, existingDirector := range t.Directors.Persons {
			if newDirector.Id == existingDirector.Id {
				return fmt.Errorf("director %s already exists", newDirector.Id)
			}
		}
	}

	t.Directors.Persons = append(t.Directors.Persons, directors.Persons...)

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	tdevt, err := TournamentDataResponse(ctx, ts, id)
	if err != nil {
		return err
	}

	wrapped := entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
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

	for _, newDirector := range directors.Persons {
		newDirectorfullID, _, err := constructFullID(ctx, us, newDirector.Id)
		if err != nil {
			return err
		}
		newDirector.Id = newDirectorfullID
	}

	newDirectors, err := removeTournamentPersons(t.Directors, directors, true)
	if err != nil {
		return err
	}

	t.Directors = newDirectors

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	tdevt, err := TournamentDataResponse(ctx, ts, id)
	if err != nil {
		return err
	}
	wrapped := entity.WrapEvent(tdevt, realtime.MessageType_TOURNAMENT_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
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
	for _, player := range players.Persons {
		fullID, UUID, err := constructFullID(ctx, us, player.Id)
		if err != nil {
			return err
		}
		player.Id = fullID
		userUUIDs = append(userUUIDs, UUID)
	}

	pairingsResp, err := divisionObject.DivisionManager.AddPlayers(players)
	if err != nil {
		return err
	}

	allCurrentPlayers := divisionObject.DivisionManager.GetPlayers()

	err = ts.AddRegistrants(ctx, t.UUID, userUUIDs, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	pairingsResp.Id = id
	pairingsResp.Division = division

	addPlayersMessage := &realtime.PlayersAddedOrRemovedResponse{Id: id,
		Division:          division,
		Players:           allCurrentPlayers,
		DivisionPairings:  pairingsResp.DivisionPairings,
		DivisionStandings: pairingsResp.DivisionStandings}
	wrapped := entity.WrapEvent(addPlayersMessage, realtime.MessageType_TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
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
	userUUIDs := []string{}
	for _, player := range players.Persons {
		fullID, UUID, err := constructFullID(ctx, us, player.Id)
		if err != nil {
			return err
		}
		player.Id = fullID
		userUUIDs = append(userUUIDs, UUID)
	}

	pairingsResp, err := divisionObject.DivisionManager.RemovePlayers(players)
	if err != nil {
		return err
	}

	allCurrentPlayers := divisionObject.DivisionManager.GetPlayers()

	err = ts.RemoveRegistrants(ctx, t.UUID, userUUIDs, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	pairingsResp.Id = id
	pairingsResp.Division = division

	removePlayersMessage := &realtime.PlayersAddedOrRemovedResponse{Id: id,
		Division:          division,
		Players:           allCurrentPlayers,
		DivisionPairings:  pairingsResp.DivisionPairings,
		DivisionStandings: pairingsResp.DivisionStandings}
	wrapped := entity.WrapEvent(removePlayersMessage, realtime.MessageType_TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
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

	pairingsResponse := []*realtime.Pairing{}
	standingsResponse := make(map[int32]*realtime.RoundStandings)

	for _, pairing := range pairings {
		divisionObject, ok := t.Divisions[division]

		if !ok {
			return fmt.Errorf("division %s does not exist", division)
		}

		if divisionObject.DivisionManager == nil {
			return fmt.Errorf("division %s does not have enough players or controls to set pairings", division)
		}

		pairingsResp, err := divisionObject.DivisionManager.SetPairing(pairing.PlayerOneId, pairing.PlayerTwoId, int(pairing.Round), pairing.SelfPlayResult)
		if err != nil {
			return err
		}
		pairingsResponse = combinePairingsResponses(pairingsResponse, pairingsResp.DivisionPairings)
		standingsResponse = combineStandingsResponses(standingsResponse, pairingsResp.DivisionStandings)
	}
	err = possiblyEndTournament(ctx, ts, t, division)
	if err != nil {
		return err
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	pairingsMessage := PairingsToResponse(id, division, pairingsResponse, standingsResponse)
	wrapped := entity.WrapEvent(pairingsMessage, realtime.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
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

	log.Debug().Str("playerOneId", playerOneId).Str("playerTwoId", playerTwoId).Msg("tSetResult")

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
			Division:  division,
			GameIndex: int32(gameIndex),
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

	log.Debug().Str("p1", p1.Username).Str("p2", p2.Username).Msg("after-get-by-uuid")

	gid := ""
	if g != nil {
		gid = g.GameID()
	}

	pairingsResp, err := divisionObject.DivisionManager.SubmitResult(round,
		p1.TournamentID(),
		p2.TournamentID(),
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

	err = possiblyEndTournament(ctx, ts, t, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	pairingsResp.Id = id
	pairingsResp.Division = division
	wrapped := entity.WrapEvent(pairingsResp, realtime.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func possiblyEndTournament(ctx context.Context, ts TournamentStore, t *entity.Tournament,
	division string) error {

	divisionObject := t.Divisions[division]
	ended, err := divisionObject.DivisionManager.IsFinished()
	if err != nil {
		return err
	}
	allended := ended
	if ended {
		for dname, div := range t.Divisions {
			if dname == division {
				continue
				// no need to check again
			}
			dended, err := div.DivisionManager.IsFinished()
			if err != nil {
				return err
			}
			if !dended {
				allended = false
				break
			}
		}
	}
	if allended {
		t.IsFinished = true
		err := ts.RemoveRegistrantsForTournament(ctx, t.UUID)
		if err != nil {
			return err
		}
	}
	return nil
}

func startTournamentChecks(t *entity.Tournament) error {
	if t.IsFinished {
		return fmt.Errorf("tournament %s has finished", t.Name)
	}

	if len(t.Divisions) == 0 {
		return fmt.Errorf("cannot start tournament %s with no divisions", t.Name)
	}

	return nil
}

func startDivisionChecks(t *entity.Tournament, division string, round int) error {
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	if divisionObject.DivisionManager == nil {
		return fmt.Errorf("division %s does not have enough players or controls to set pairings", division)
	}

	dm := divisionObject.DivisionManager

	if dm.GetDivisionControls().GameRequest == nil {
		return fmt.Errorf("no division game controls have been set for division %v", division)
	}

	if round != dm.GetCurrentRound()+1 {
		return fmt.Errorf("incorrect start round for division %v", division)
	}

	err := dm.IsRoundStartable()
	if err != nil {
		return err
	}
	return nil
}

func sendDivisionStart(ts TournamentStore, tuuid string, division string, round int) error {

	// Send code that sends signal to all tournament players that backend
	// is now accepting "ready" messages for this round.
	eventChannel := ts.TournamentEventChan()
	evt := &realtime.TournamentRoundStarted{
		TournamentId: tuuid,
		Division:     division,
		Round:        int32(round),
		// GameIndex: int32(0) -- fix this when we have other types of tournaments
		// add timestamp deadline here as well at some point
	}
	wrapped := entity.WrapEvent(evt, realtime.MessageType_TOURNAMENT_ROUND_STARTED)

	// Send it to everyone in this division across the app.
	wrapped.AddAudience(entity.AudChannel, DivisionChannelName(tuuid, division))
	// Also send it to the tournament realm.
	wrapped.AddAudience(entity.AudTournament, tuuid)
	if eventChannel != nil {
		eventChannel <- wrapped
	}
	log.Debug().Interface("evt", evt).Msg("sent-tournament-round-started")
	return nil
}

func StartAllRoundCountdowns(ctx context.Context, ts TournamentStore, id string, round int) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	err = startTournamentChecks(t)
	if err != nil {
		return err
	}

	for division := range t.Divisions {
		err = startDivisionChecks(t, division, round)
		if err != nil {
			return err
		}
	}

	for division := range t.Divisions {
		err := t.Divisions[division].DivisionManager.StartRound(false)
		if err != nil {
			return err
		}
	}
	t.IsStarted = true
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	for division := range t.Divisions {
		err := sendDivisionStart(ts, t.UUID, division, round)
		if err != nil {
			return err
		}
	}

	return nil
}

func StartRoundCountdown(ctx context.Context, ts TournamentStore, id string,
	division string, round int) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	err = startTournamentChecks(t)
	if err != nil {
		return err
	}

	err = startDivisionChecks(t, division, round)
	if err != nil {
		return err
	}

	err = t.Divisions[division].DivisionManager.StartRound(false)
	if err != nil {
		return err
	}
	t.IsStarted = true
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	return sendDivisionStart(ts, t.UUID, division, round)
}

// DivisionChannelName returns a channel name that can be used
// for sending communications regarding a tournament and division.
func DivisionChannelName(tid, division string) string {
	// We encode to b64 because division can contain spaces.
	return base64.URLEncoding.EncodeToString([]byte(tid + ":" + division))
}

func PairRound(ctx context.Context, ts TournamentStore, id string, division string, round int, preserveByes bool) error {
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
	if round < currentRound+1 {
		return fmt.Errorf("cannot repair non-future round %d since current round is %d", round, currentRound)
	}

	pairingsResp, err := divisionObject.DivisionManager.PairRound(round, preserveByes)

	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	pairingsResp.Id = id
	pairingsResp.Division = division
	wrapped := entity.WrapEvent(pairingsResp, realtime.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)

	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func DeletePairings(ctx context.Context, ts TournamentStore, id string, division string, round int) error {
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
		return errors.New("cannot erase pairings before the tournament has started")
	}

	currentRound := divisionObject.DivisionManager.GetCurrentRound()
	if round < currentRound+1 {
		return fmt.Errorf("cannot erase pairings for non-future round %d since current round is %d", round, currentRound)
	}

	err = divisionObject.DivisionManager.DeletePairings(round)

	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	wrapped := entity.WrapEvent(&realtime.DivisionPairingsDeletedResponse{Id: id,
		Division: division,
		Round:    int32(round)}, realtime.MessageType_TOURNAMENT_DIVISION_PAIRINGS_DELETED_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func IsStarted(ctx context.Context, ts TournamentStore, id string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if t.IsFinished {
		return false, fmt.Errorf("tournament %s has finished", id)
	}

	return t.IsStarted, nil
}

func IsRoundComplete(ctx context.Context, ts TournamentStore, id string, division string, round int) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if t.IsFinished {
		return false, fmt.Errorf("tournament %s has finished", id)
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
	wrapped := entity.WrapEvent(&realtime.TournamentFinishedResponse{Id: id}, realtime.MessageType_TOURNAMENT_FINISHED_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func IsFinished(ctx context.Context, ts TournamentStore, id string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, errors.New("cannot check if tournament is finished before the tournament has started")
	}
	return t.IsFinished, nil
}

func GetXHRResponse(ctx context.Context, ts TournamentStore, id string) (*realtime.FullTournamentDivisions, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	response := &realtime.FullTournamentDivisions{Divisions: make(map[string]*realtime.TournamentDivisionDataResponse),
		Started: t.IsStarted}
	for divisionKey, division := range t.Divisions {
		if division.DivisionManager == nil {
			return nil, fmt.Errorf("division %s has nil division manager, cannot get division message", divisionKey)
		}
		xhr, err := division.DivisionManager.GetXHRResponse()
		if err != nil {
			return nil, err
		}
		xhr.Id = id
		xhr.Division = divisionKey
		response.Divisions[divisionKey] = xhr
	}
	return response, nil
}

func SetReadyForGame(ctx context.Context, ts TournamentStore, t *entity.Tournament,
	playerID, connID, division string,
	round, gameIndex int, unready bool) ([]string, bool, error) {

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return nil, false, fmt.Errorf("tournament %s has finished", t.Name)
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
		return fmt.Errorf("tournament %s has finished", t.Name)
	}

	_, ok := t.Divisions[division]
	if !ok {
		return fmt.Errorf("division %s does not exist", division)
	}

	pairing, err := t.Divisions[division].DivisionManager.ClearReadyStates(userID, round, gameIndex)
	if err != nil {
		return err
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	pairingsMessage := PairingsToResponse(t.UUID, division, pairing, make(map[int32]*realtime.RoundStandings))
	wrapped := entity.WrapEvent(pairingsMessage, realtime.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)
	return SendTournamentMessage(ctx, ts, t.UUID, wrapped)
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

func PairingsToResponse(id string, division string, pairings []*realtime.Pairing, standings map[int32]*realtime.RoundStandings) *realtime.DivisionPairingsResponse {
	// This is quite simple for now
	// This function is here in case this structure
	// gets more complicated later
	return &realtime.DivisionPairingsResponse{Id: id,
		Division:          division,
		DivisionPairings:  pairings,
		DivisionStandings: standings}
}

func getExecutiveDirector(directors *realtime.TournamentPersons) (string, error) {
	err := errors.New("tournament must have exactly one executive director")
	executiveDirector := ""
	for _, director := range directors.Persons {
		if director.Rating == 0 {
			if executiveDirector != "" {
				return "", err
			} else {
				executiveDirector = director.Id
			}
		}
	}
	if executiveDirector == "" {
		return "", err
	}
	return executiveDirector, nil
}

func removeTournamentPersons(persons *realtime.TournamentPersons, personsToRemove *realtime.TournamentPersons, areDirectors bool) (*realtime.TournamentPersons, error) {
	indexesToRemove := []int{}
	for _, personToRemove := range personsToRemove.Persons {
		present := false
		for index, person := range persons.Persons {
			if personToRemove.Id == person.Id {
				if person.Rating == 0 && areDirectors {
					return nil, fmt.Errorf("cannot remove the executive director: %s", person.Id)
				}
				present = true
				indexesToRemove = append(indexesToRemove, index)
				break
			}
		}
		if !present {
			return nil, fmt.Errorf("person %s does not exist", personToRemove.Id)
		}
	}

	sort.Ints(indexesToRemove)

	for i := len(indexesToRemove) - 1; i >= 0; i-- {
		idx := indexesToRemove[i]
		persons.Persons[len(persons.Persons)-1], persons.Persons[idx] = persons.Persons[idx], persons.Persons[len(persons.Persons)-1]
		persons.Persons = persons.Persons[:len(persons.Persons)-1]
	}
	return persons, nil
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

func CheckIn(ctx context.Context, ts TournamentStore, tid string, playerid string) error {
	return errors.New("not implemented")
}

func UncheckIn(ctx context.Context, ts TournamentStore, tid string) error {
	return errors.New("not implemented")
}

func DisassociateClubGame(ctx context.Context, ts TournamentStore, tid, gid string) error {
	t, err := ts.Get(ctx, tid)
	if err != nil {
		return errors.New("tournament does not exist")
	}

}

/*
func CheckIn(ctx context.Context, ts TournamentStore, tid string, playerid string) error {
	t, err := ts.Get(ctx, tid)
	if err != nil {
		return err
	}
	t.Lock()
	defer t.Unlock()

	found := false
	// really a player should only be in one division but we allow this so let's
	// make it correct for now
	divisionsFound := []string{}
	for dname, d := range t.Divisions {
		if _, ok := d.Players.Persons[playerid]; ok {
			err := d.DivisionManager.SetCheckedIn(playerid)
			if err != nil {
				return err
			}
			found = true
			divisionsFound = append(divisionsFound, dname)
		}
	}
	if !found {
		return errors.New("user not in this tournament")
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	for _, d := range divisionsFound {
		err = SendTournamentDivisionMessage(ctx, ts, t.UUID, d)
		if err != nil {
			return err
		}
	}

	return nil
}

func UncheckIn(ctx context.Context, ts TournamentStore, tid string) error {
	t, err := ts.Get(ctx, tid)
	if err != nil {
		return err
	}
	t.Lock()
	defer t.Unlock()

	for dname, d := range t.Divisions {
		d.DivisionManager.ClearCheckedIn()
		err = SendTournamentDivisionMessage(ctx, ts, t.UUID, dname)
		if err != nil {
			return err
		}
	}
	return ts.Set(ctx, t)
}

*/
