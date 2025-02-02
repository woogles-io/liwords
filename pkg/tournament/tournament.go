package tournament

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
	"github.com/woogles-io/liwords/pkg/user"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/tournament_service"
)

const MaxDivisionNameLength = 24

var (
	tracer = otel.Tracer("tournament")
)

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

func md5hash(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	return hex.EncodeToString(h.Sum(nil))
}

func HandleTournamentGameEnded(ctx context.Context, ts TournamentStore, us user.Store,
	g *entity.Game, queries *models.Queries) error {

	Results := []ipc.TournamentGameResult{ipc.TournamentGameResult_DRAW,
		ipc.TournamentGameResult_WIN,
		ipc.TournamentGameResult_LOSS}

	p1idx, p2idx := 0, 1
	// WinnerIdx and LoserIdx are both -1 if it's a tie:
	p1result, p2result := Results[g.WinnerIdx+1], Results[g.LoserIdx+1]

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
		g, queries)
}

func NewTournament(ctx context.Context,
	tournamentStore TournamentStore,
	name string,
	description string,
	directors *ipc.TournamentPersons,
	ttype entity.CompetitionType,
	parent string,
	slug string,
) (*entity.Tournament, error) {

	executiveDirector, err := getExecutiveDirector(name, directors)
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
	} else {
		log.Error().Msg("send-msg-tournament-event-chan-nil")
	}
	log.Debug().Str("tid", id).Msg("sent tournament message type: " + wrapped.Type.String())
	return nil
}

func ternary[T any](c bool, i, e T) T {
	if c {
		return i
	}
	return e
}

func SetTournamentMetadata(ctx context.Context, ts TournamentStore, meta *pb.TournamentMetadata,
	merge bool) error {

	var err error
	var ttype entity.CompetitionType
	if meta.Slug != "" {
		ttype, err = validateTournamentMeta(meta.Type, meta.Slug)
		if err != nil {
			return err
		}
	}

	t, err := ts.Get(ctx, meta.Id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()
	name := strings.TrimSpace(meta.Name)
	if name == "" && !merge {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_EMPTY_NAME, t.Name)
	}
	log.Info().Interface("t", t).Msg("tournament-before-set-meta")

	// Not all fields need to be specified if we're merging.
	t.Name = ternary(merge && name == "", t.Name, name)
	t.Description = ternary(merge && meta.Description == "", t.Description, meta.Description)
	t.Slug = ternary(merge && meta.Slug == "", t.Slug, meta.Slug)
	t.Type = ternary(merge && meta.Type == pb.TType_STANDARD, t.Type, ttype)
	t.ExtraMeta = &entity.TournamentMeta{
		Disclaimer: ternary(merge && meta.Disclaimer == "" && t.ExtraMeta != nil,
			t.ExtraMeta.Disclaimer, meta.Disclaimer),
		TileStyle: ternary(merge && meta.TileStyle == "" && t.ExtraMeta != nil,
			t.ExtraMeta.TileStyle, meta.TileStyle),
		BoardStyle: ternary(merge && meta.BoardStyle == "" && t.ExtraMeta != nil,
			t.ExtraMeta.BoardStyle, meta.BoardStyle),
		DefaultClubSettings: ternary(merge && meta.DefaultClubSettings == nil && t.ExtraMeta != nil,
			t.ExtraMeta.DefaultClubSettings, meta.DefaultClubSettings),
		FreeformClubSettingFields: ternary(merge && meta.FreeformClubSettingFields == nil && t.ExtraMeta != nil,
			t.ExtraMeta.FreeformClubSettingFields, meta.FreeformClubSettingFields),
		Password: ternary(merge && meta.Password == "" && t.ExtraMeta != nil,
			t.ExtraMeta.Password, meta.Password),
		Logo: ternary(merge && meta.Logo == "" && t.ExtraMeta != nil,
			t.ExtraMeta.Logo, meta.Logo),
		Color: ternary(merge && meta.Color == "" && t.ExtraMeta != nil,
			t.ExtraMeta.Color, meta.Color),
		PrivateAnalysis: ternary(merge && !meta.PrivateAnalysis && t.ExtraMeta != nil,
			t.ExtraMeta.PrivateAnalysis, meta.PrivateAnalysis),
		IRLMode: ternary(merge && !meta.IrlMode && t.ExtraMeta != nil,
			t.ExtraMeta.IRLMode, meta.IrlMode),
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	tdevt, err := TournamentDataResponse(ctx, ts, meta.Id)
	if err != nil {
		return err
	}
	wrapped := entity.WrapEvent(tdevt, ipc.MessageType_TOURNAMENT_MESSAGE)
	return SendTournamentMessage(ctx, ts, meta.Id, wrapped)
}

func SetSingleRoundControls(ctx context.Context, ts TournamentStore, id string, division string, round int, controls *ipc.RoundControl) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	if !t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NOT_STARTED, t.Name, division)
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if divisionObject.DivisionManager == nil {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, division)
	}

	currentRound := divisionObject.DivisionManager.GetCurrentRound()
	if round < currentRound+1 {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_SET_NON_FUTURE_ROUND_CONTROLS, t.Name, division, strconv.Itoa(currentRound+1))
	}

	newControls, err := divisionObject.DivisionManager.SetSingleRoundControls(round, controls)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	wrapped := entity.WrapEvent(&ipc.DivisionRoundControls{
		Id:            id,
		Division:      division,
		RoundControls: []*ipc.RoundControl{newControls},
	}, ipc.MessageType_TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func SetRoundControls(ctx context.Context, ts TournamentStore, id string, division string, roundControls []*ipc.RoundControl) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_SET_ROUND_CONTROLS_AFTER_START, t.Name, division, "tournament")
	}

	pairingsResp, newDivisionRoundControls, err := divisionObject.DivisionManager.SetRoundControls(roundControls)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	wrapped := entity.WrapEvent(&ipc.DivisionRoundControls{Id: id, Division: division,
		RoundControls:     newDivisionRoundControls,
		DivisionPairings:  pairingsResp.DivisionPairings,
		DivisionStandings: pairingsResp.DivisionStandings},
		ipc.MessageType_TOURNAMENT_DIVISION_ROUND_CONTROLS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func SetDivisionControls(ctx context.Context, ts TournamentStore, id string, division string, controls *ipc.DivisionControls) error {

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	newDivisionControls, standings, err := divisionObject.DivisionManager.SetDivisionControls(controls)
	if err != nil {
		return err
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	resp := &ipc.DivisionControlsResponse{
		Id:                id,
		Division:          division,
		DivisionControls:  newDivisionControls,
		DivisionStandings: standings,
	}
	wrapped := entity.WrapEvent(resp, ipc.MessageType_TOURNAMENT_DIVISION_CONTROLS_MESSAGE)
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
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	if t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_ADD_DIVISION_AFTER_START, t.Name, division)
	}

	if len(division) == 0 || len(division) > MaxDivisionNameLength {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_INVALID_DIVISION_NAME, t.Name, division)
	}

	_, ok := t.Divisions[division]

	if ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_DIVISION_ALREADY_EXISTS, t.Name, division)
	}

	t.Divisions[division] = &entity.TournamentDivision{DivisionManager: NewClassicDivision(t.Name, division)}

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
	wrapped := entity.WrapEvent(tdevt, ipc.MessageType_TOURNAMENT_DIVISION_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func RenameDivision(ctx context.Context, ts TournamentStore, id string, division, newName string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	if t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_ADD_DIVISION_AFTER_START, t.Name, division)
	}

	div, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	div.DivisionManager.ChangeName(newName)
	t.Divisions[newName] = div
	delete(t.Divisions, division)

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	// Send a deletion and then an addition.

	tddevt := &ipc.TournamentDivisionDeletedResponse{Id: id, Division: division}
	wrapped := entity.WrapEvent(tddevt, ipc.MessageType_TOURNAMENT_DIVISION_DELETED_MESSAGE)
	err = SendTournamentMessage(ctx, ts, id, wrapped)
	if err != nil {
		return err
	}

	tdevt, err := t.Divisions[newName].DivisionManager.GetXHRResponse()
	if err != nil {
		return err
	}
	tdevt.Id = id
	tdevt.Division = newName
	wrapped = entity.WrapEvent(tdevt, ipc.MessageType_TOURNAMENT_DIVISION_MESSAGE)
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
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	_, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_DIVISION_REMOVAL_AFTER_START, t.Name, division)
	}

	if len(t.Divisions[division].DivisionManager.GetPlayers().GetPersons()) > 0 {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_DIVISION_REMOVAL_EXISTING_PLAYERS, t.Name, division)
	}

	delete(t.Divisions, division)
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	tddevt := &ipc.TournamentDivisionDeletedResponse{Id: id, Division: division}
	wrapped := entity.WrapEvent(tddevt, ipc.MessageType_TOURNAMENT_DIVISION_DELETED_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func constructFullID(tournamentName string, divisionName string, ctx context.Context, us user.Store, user string) (string, string, error) {
	u, err := us.Get(ctx, user)
	if err != nil {
		return "", "", entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_PLAYER_ID_CONSTRUCTION, tournamentName, divisionName, user)
	}
	return u.TournamentID(), u.UUID, nil
}

func AddDirectors(ctx context.Context, ts TournamentStore, us user.Store, id string, directors *ipc.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, "")
	}
	// Only perform the add operation if all persons can be added.

	for _, director := range directors.Persons {
		fullID, _, err := constructFullID(t.Name, "", ctx, us, director.Id)
		if err != nil {
			return err
		}
		director.Id = fullID
	}

	// Not very efficient but there's only gonna be like
	// a maximum of maybe 10 existing directors

	for _, newDirector := range directors.Persons {
		if newDirector.Id == t.ExecutiveDirector || newDirector.Rating == 0 {
			return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_EXECUTIVE_DIRECTOR_EXISTS, t.Name, "", newDirector.Id)
		}
		for _, existingDirector := range t.Directors.Persons {
			if newDirector.Id == existingDirector.Id {
				return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_DIRECTOR_EXISTS, t.Name, "", newDirector.Id)
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

	wrapped := entity.WrapEvent(tdevt, ipc.MessageType_TOURNAMENT_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func RemoveDirectors(ctx context.Context, ts TournamentStore, us user.Store, id string, directors *ipc.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, "")
	}

	for _, newDirector := range directors.Persons {
		newDirectorfullID, _, err := constructFullID(t.Name, "", ctx, us, newDirector.Id)
		if err != nil {
			return err
		}
		newDirector.Id = newDirectorfullID
	}

	newDirectors, err := removeTournamentPersons(t.Name, "", t.Directors, directors, true)
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
	wrapped := entity.WrapEvent(tdevt, ipc.MessageType_TOURNAMENT_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func AddPlayers(ctx context.Context, ts TournamentStore, us user.Store, id string, division string, players *ipc.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}
	existingPlayers := map[string]string{}
	for k, v := range t.Divisions {
		if k == division {
			continue
		}
		dp := v.DivisionManager.GetPlayers()
		for _, p := range dp.Persons {
			existingPlayers[p.Id] = k
		}
	}

	if divisionObject.DivisionManager == nil {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, division)
	}

	// Only perform the add operation if all persons can be added.

	userUUIDs := []string{}
	for _, player := range players.Persons {
		var UUID string
		var fullID string
		var err error
		if t.ExtraMeta.IRLMode {
			// Use a deterministic "uuid"
			UUID = md5hash(player.Id)
			fullID = UUID + ":" + player.Id
		} else {
			fullID, UUID, err = constructFullID(t.Name, division, ctx, us, player.Id)
			if err != nil {
				return err
			}
		}
		if dname, ok := existingPlayers[fullID]; ok {
			return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_PLAYER_ALREADY_EXISTS, t.Name, dname, fullID)
		}
		player.Id = fullID
		userUUIDs = append(userUUIDs, UUID)
	}

	pairingsResp, err := divisionObject.DivisionManager.AddPlayers(players)
	if err != nil {
		return err
	}

	allCurrentPlayers := divisionObject.DivisionManager.GetPlayers()

	if !t.ExtraMeta.IRLMode {
		err = ts.AddRegistrants(ctx, t.UUID, userUUIDs, division)
		if err != nil {
			return err
		}
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	pairingsResp.Id = id
	pairingsResp.Division = division

	addPlayersMessage := &ipc.PlayersAddedOrRemovedResponse{Id: id,
		Division:          division,
		Players:           allCurrentPlayers,
		DivisionPairings:  pairingsResp.DivisionPairings,
		DivisionStandings: pairingsResp.DivisionStandings}
	wrapped := entity.WrapEvent(addPlayersMessage, ipc.MessageType_TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func RemovePlayers(ctx context.Context, ts TournamentStore, us user.Store, id string, division string, players *ipc.TournamentPersons) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if divisionObject.DivisionManager == nil {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, division)
	}

	// Only perform the remove operation if all persons can be removed.
	userUUIDs := []string{}
	for _, player := range players.Persons {
		var UUID string
		var fullID string
		var err error
		if t.ExtraMeta.IRLMode {
			UUID = md5hash(player.Id)
			fullID = UUID + ":" + player.Id
		} else {
			fullID, UUID, err = constructFullID(t.Name, division, ctx, us, player.Id)
			if err != nil {
				return err
			}
		}
		player.Id = fullID
		userUUIDs = append(userUUIDs, UUID)
	}

	pairingsResp, err := divisionObject.DivisionManager.RemovePlayers(players)
	if err != nil {
		return err
	}

	allCurrentPlayers := divisionObject.DivisionManager.GetPlayers()

	if !t.ExtraMeta.IRLMode {
		err = ts.RemoveRegistrants(ctx, t.UUID, userUUIDs, division)
		if err != nil {
			return err
		}
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	pairingsResp.Id = id
	pairingsResp.Division = division

	removePlayersMessage := &ipc.PlayersAddedOrRemovedResponse{Id: id,
		Division:          division,
		Players:           allCurrentPlayers,
		DivisionPairings:  pairingsResp.DivisionPairings,
		DivisionStandings: pairingsResp.DivisionStandings}
	wrapped := entity.WrapEvent(removePlayersMessage, ipc.MessageType_TOURNAMENT_DIVISION_PLAYER_CHANGE_MESSAGE)
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
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	pairingsResponse := []*ipc.Pairing{}
	standingsResponse := make(map[int32]*ipc.RoundStandings)

	for _, pairing := range pairings {
		divisionObject, ok := t.Divisions[division]

		if !ok {
			return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
		}

		if divisionObject.DivisionManager == nil {
			// Not enough players or rounds to make a manager, most likely
			return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, division)
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
	wrapped := entity.WrapEvent(pairingsMessage, ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)
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
	playerOneResult ipc.TournamentGameResult,
	playerTwoResult ipc.TournamentGameResult,
	reason ipc.GameEndReason,
	round int,
	gameIndex int,
	amendment bool,
	g *entity.Game,
	queries *models.Queries) error {
	ctx, span := tracer.Start(ctx, "set-result")
	defer span.End()

	log.Debug().Str("playerOneId", playerOneId).Str("playerTwoId", playerTwoId).Msg("tSetResult")

	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	span.AddEvent("lock", trace.WithAttributes(attribute.String("tid", id)))

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
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

		players := []*ipc.TournamentGameEndedEvent_Player{
			{Username: p1user.Username, Score: int32(playerOneScore), Result: playerOneResult},
			{Username: p2user.Username, Score: int32(playerTwoScore), Result: playerTwoResult},
		}

		gameID := ""
		if g != nil {
			gameID = g.GameID()
		}
		tevt := &ipc.TournamentGameEndedEvent{
			GameId:    gameID,
			Players:   players,
			EndReason: reason,
			Round:     int32(round),
			Division:  division,
			GameIndex: int32(gameIndex),
			Time:      time.Now().Unix(),
		}
		log.Debug().Interface("tevt", tevt).Msg("sending legacy tournament game ended evt")
		wrapped := entity.WrapEvent(tevt, ipc.MessageType_TOURNAMENT_GAME_ENDED_EVENT)
		wrapped.AddAudience(entity.AudTournament, id)
		evtChan := ts.TournamentEventChan()
		if evtChan != nil {
			evtChan <- wrapped
		} else {
			log.Error().Msg("set-result-tournament-event-chan-nil")
		}
		log.Debug().Str("tid", id).Msg("sent legacy tournament game ended event")
		return nil
	}

	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if !t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NOT_STARTED, t.Name, division)
	}

	// We need to send the division manager the "full" user ID, so look that up here.
	var p1TID, p2TID string
	if t.ExtraMeta.IRLMode {
		// note this is only the first part of the full tournamentt ID in this case
		// we do not look up the player in the DB since IRLMode does not use
		// Woogles IDs
		p1TID = playerOneId
		p2TID = playerTwoId
	} else {
		p1, err := us.GetByUUID(ctx, playerOneId)
		if err != nil {
			return err
		}
		p2, err := us.GetByUUID(ctx, playerTwoId)
		if err != nil {
			return err
		}
		log.Debug().Str("p1", p1.Username).Str("p2", p2.Username).Msg("after-get-by-uuid")
		p1TID = p1.TournamentID()
		p2TID = p2.TournamentID()
	}

	gid := ""
	if g != nil {
		gid = g.GameID()
	}
	span.AddEvent("about-to-submit-result")

	pairingsResp, err := divisionObject.DivisionManager.SubmitResult(round,
		p1TID,
		p2TID,
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
	span.AddEvent("result-submitted")

	err = possiblyEndTournament(ctx, ts, t, division)
	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	span.AddEvent("ts-set")

	stats, err := computeTourneyGameStats(ctx, p1TID, p2TID, playerOneScore, playerTwoScore,
		playerOneResult, playerTwoResult, amendment, g)
	if err != nil {
		return err
	}

	span.AddEvent("set-stats")

	pairingsResp.Id = id
	pairingsResp.Division = division
	wrapped := entity.WrapEvent(pairingsResp, ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func computeTourneyGameStats(ctx context.Context, p1TID, p2TID string, p1Score, p2Score int,
	p1Result, p2Result ipc.TournamentGameResult, amendment bool, g *entity.Game) (*entity.Stats, error) {

	cfg, err := config.GetConfig(ctx)
	if err != nil {
		return nil, err
	}
	var gameStats *entity.Stats
	// Would have been computed by performEndgameDuties in end.go
	if g != nil && g.Stats != nil {
		gameStats = g.Stats
	}
	// hmmm.
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
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, "")
	}

	if len(t.Divisions) == 0 {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NO_DIVISIONS, t.Name, "")
	}

	return nil
}

func startDivisionChecks(t *entity.Tournament, division string, round int) error {
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if divisionObject.DivisionManager == nil {
		// division does not have enough players or controls to set pairings
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, division)
	}

	dm := divisionObject.DivisionManager

	if dm.GetDivisionControls().GameRequest == nil {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_GAME_CONTROLS_NOT_SET, t.Name, division)
	}

	if round != dm.GetCurrentRound()+1 {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_INCORRECT_START_ROUND, t.Name, division, strconv.Itoa(round+1), strconv.Itoa(dm.GetCurrentRound()+1))
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
	evt := &ipc.TournamentRoundStarted{
		TournamentId: tuuid,
		Division:     division,
		Round:        int32(round),
		// GameIndex: int32(0) -- fix this when we have other types of tournaments
		// add timestamp deadline here as well at some point
	}
	wrapped := entity.WrapEvent(evt, ipc.MessageType_TOURNAMENT_ROUND_STARTED)

	// Send it to everyone in this division across the app.
	wrapped.AddAudience(entity.AudChannel, DivisionChannelName(tuuid, division))
	// Also send it to the tournament realm.
	wrapped.AddAudience(entity.AudTournament, tuuid)
	if eventChannel != nil {
		eventChannel <- wrapped
	} else {
		log.Error().Msg("send-divstart-tournament-event-chan-nil")
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
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if !t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NOT_STARTED, t.Name, division)
	}

	currentRound := divisionObject.DivisionManager.GetCurrentRound()
	if round < currentRound+1 {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_PAIR_NON_FUTURE_ROUND, t.Name, division, strconv.Itoa(round+1), strconv.Itoa(currentRound+1))
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
	wrapped := entity.WrapEvent(pairingsResp, ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)

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
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}
	divisionObject, ok := t.Divisions[division]

	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	if !t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NOT_STARTED, t.Name, division)
	}

	currentRound := divisionObject.DivisionManager.GetCurrentRound()
	if round < currentRound+1 {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_DELETE_NON_FUTURE_ROUND, t.Name, division, strconv.Itoa(round+1), strconv.Itoa(currentRound+1))
	}

	err = divisionObject.DivisionManager.DeletePairings(round)

	if err != nil {
		return err
	}

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}

	wrapped := entity.WrapEvent(&ipc.DivisionPairingsDeletedResponse{Id: id,
		Division: division,
		Round:    int32(round)}, ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_DELETED_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func IsStarted(ctx context.Context, ts TournamentStore, id string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if t.IsFinished {
		return true, nil
	}

	return t.IsStarted, nil
}

func IsRoundComplete(ctx context.Context, ts TournamentStore, id string, division string, round int) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if t.IsFinished {
		return true, nil
	}

	if !t.IsStarted {
		return false, nil
	}

	_, ok := t.Divisions[division]

	if !ok {
		return false, entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	return t.Divisions[division].DivisionManager.IsRoundComplete(round)
}

func SetFinished(ctx context.Context, ts TournamentStore, id string) error {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return err
	}

	if t.IsFinished {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, "")
	}

	if !t.IsStarted {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NOT_STARTED, t.Name, "")
	}

	for d, division := range t.Divisions {
		if division.DivisionManager == nil {
			return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, "")
		}

		finished, err := division.DivisionManager.IsFinished()
		if err != nil {
			return nil
		}
		if !finished {
			return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_DIVISION_NOT_FINISHED, t.Name, d)
		}
	}

	t.IsFinished = true

	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	wrapped := entity.WrapEvent(&ipc.TournamentFinishedResponse{Id: id}, ipc.MessageType_TOURNAMENT_FINISHED_MESSAGE)
	return SendTournamentMessage(ctx, ts, id, wrapped)
}

func IsFinished(ctx context.Context, ts TournamentStore, id string) (bool, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !t.IsStarted {
		return false, nil
	}
	return t.IsFinished, nil
}

func GetXHRResponse(ctx context.Context, ts TournamentStore, id string) (*ipc.FullTournamentDivisions, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	t.RLock()
	defer t.RUnlock()

	response := &ipc.FullTournamentDivisions{Divisions: make(map[string]*ipc.TournamentDivisionDataResponse),
		Started: t.IsStarted}
	for divisionKey, division := range t.Divisions {
		if division.DivisionManager == nil {
			return nil, entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NIL_DIVISION_MANAGER, t.Name, divisionKey)
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
		return nil, false, entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}

	_, ok := t.Divisions[division]
	if !ok {
		return nil, false, entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
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
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_FINISHED, t.Name, division)
	}

	_, ok := t.Divisions[division]
	if !ok {
		return entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_DIVISION, t.Name, division)
	}

	pairing, err := t.Divisions[division].DivisionManager.ClearReadyStates(userID, round, gameIndex)
	if err != nil {
		return err
	}
	err = ts.Set(ctx, t)
	if err != nil {
		return err
	}
	pairingsMessage := PairingsToResponse(t.UUID, division, pairing, make(map[int32]*ipc.RoundStandings))
	wrapped := entity.WrapEvent(pairingsMessage, ipc.MessageType_TOURNAMENT_DIVISION_PAIRINGS_MESSAGE)
	return SendTournamentMessage(ctx, ts, t.UUID, wrapped)
}

func TournamentDataResponse(ctx context.Context, ts TournamentStore, id string) (*ipc.TournamentDataResponse, error) {
	t, err := ts.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// no lock needed; only gets called while already locked.
	return &ipc.TournamentDataResponse{Id: t.UUID,
		Name:              t.Name,
		Description:       t.Description,
		ExecutiveDirector: t.ExecutiveDirector,
		Directors:         t.Directors,
		IsStarted:         t.IsStarted}, nil
}

func PairingsToResponse(id string, division string, pairings []*ipc.Pairing, standings map[int32]*ipc.RoundStandings) *ipc.DivisionPairingsResponse {
	// This is quite simple for now
	// This function is here in case this structure
	// gets more complicated later
	return &ipc.DivisionPairingsResponse{Id: id,
		Division:          division,
		DivisionPairings:  pairings,
		DivisionStandings: standings}
}

func getExecutiveDirector(tournamentName string, directors *ipc.TournamentPersons) (string, error) {
	err := entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NOT_EXACTLY_ONE_EXECUTIVE_DIRECTOR, tournamentName, "")
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

func removeTournamentPersons(tournamentName string, divisionName string, persons *ipc.TournamentPersons, personsToRemove *ipc.TournamentPersons, areDirectors bool) (*ipc.TournamentPersons, error) {
	indexesToRemove := []int{}
	for _, personToRemove := range personsToRemove.Persons {
		present := false
		for index, person := range persons.Persons {
			if personToRemove.Id == person.Id {
				if person.Rating == 0 && areDirectors {
					return nil, entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_EXECUTIVE_DIRECTOR_REMOVAL, tournamentName, divisionName, person.Id)
				}
				present = true
				indexesToRemove = append(indexesToRemove, index)
				break
			}
		}
		if !present {
			return nil, entity.NewWooglesError(ipc.WooglesError_TOURNAMENT_NONEXISTENT_PLAYER, tournamentName, divisionName, "0", personToRemove.Id, "removeTournamentPersons")
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
			return "", apiserver.InvalidArg("club slug must start with /club/")
		}
	case pb.TType_STANDARD:
		tt = entity.TypeStandard
		if !strings.HasPrefix(slug, "/tournament/") {
			return "", apiserver.InvalidArg("tournament slug must start with /tournament/")
		}
	case pb.TType_LEGACY:
		tt = entity.TypeLegacy
		if !strings.HasPrefix(slug, "/tournament/") {
			return "", apiserver.InvalidArg("tournament slug must start with /tournament/")
		}
	case pb.TType_CHILD:
		tt = entity.TypeChild
		// A Club session type can also be a child tournament (it's essentially just a tournament with a parent ID)
		if !strings.HasPrefix(slug, "/club/") && !strings.HasPrefix(slug, "/tournament/") {
			return "", apiserver.InvalidArg("club-session slug must start with /club/ or /tournament/")
		}
	default:
		return "", apiserver.InvalidArg("invalid tournament type")
	}
	return tt, nil
}

func CheckIn(ctx context.Context, ts TournamentStore, tid string, playerid string) error {
	return errors.New("not implemented")
}

func UncheckIn(ctx context.Context, ts TournamentStore, tid string) error {
	return errors.New("not implemented")
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
