package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/opentelemetry/tracing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/user"
	pkgtournament "github.com/woogles-io/liwords/pkg/tournament"
)

// Legacy ipc types that have probably since changed are
// copied here so we don't have to think about conflicts

type OldTournamentPersons struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id       string           `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Division string           `protobuf:"bytes,2,opt,name=division,proto3" json:"division,omitempty"`
	Persons  map[string]int32 `protobuf:"bytes,3,rep,name=persons,proto3" json:"persons,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

type OldPlayerRoundInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Players     []string                   `protobuf:"bytes,1,rep,name=players,proto3" json:"players,omitempty"`
	Games       []*ipc.TournamentGame      `protobuf:"bytes,2,rep,name=games,proto3" json:"games,omitempty"` // can be a list, for elimination tourneys
	Outcomes    []ipc.TournamentGameResult `protobuf:"varint,3,rep,packed,name=outcomes,proto3,enum=liwords.TournamentGameResult" json:"outcomes,omitempty"`
	ReadyStates []string                   `protobuf:"bytes,4,rep,name=ready_states,json=readyStates,proto3" json:"ready_states,omitempty"`
}

type OldPlayerProperties struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Removed   bool  `protobuf:"varint,1,opt,name=removed,proto3" json:"removed,omitempty"`
	Rating    int32 `protobuf:"varint,2,opt,name=rating,proto3" json:"rating,omitempty"`
	CheckedIn bool  `protobuf:"varint,3,opt,name=checked_in,json=checkedIn,proto3" json:"checked_in,omitempty"`
}

type OldTournamentRoundStarted struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TournamentId string                 `protobuf:"bytes,1,opt,name=tournament_id,json=tournamentId,proto3" json:"tournament_id,omitempty"`
	Division     string                 `protobuf:"bytes,2,opt,name=division,proto3" json:"division,omitempty"`
	Round        int32                  `protobuf:"varint,3,opt,name=round,proto3" json:"round,omitempty"`
	GameIndex    int32                  `protobuf:"varint,4,opt,name=game_index,json=gameIndex,proto3" json:"game_index,omitempty"` // for matchplay type rounds etc.
	Deadline     *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=deadline,proto3" json:"deadline,omitempty"`
}

type OldPlayerStanding struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Player  string `protobuf:"bytes,1,opt,name=player,proto3" json:"player,omitempty"`
	Wins    int32  `protobuf:"varint,2,opt,name=wins,proto3" json:"wins,omitempty"`
	Losses  int32  `protobuf:"varint,3,opt,name=losses,proto3" json:"losses,omitempty"`
	Draws   int32  `protobuf:"varint,4,opt,name=draws,proto3" json:"draws,omitempty"`
	Spread  int32  `protobuf:"varint,5,opt,name=spread,proto3" json:"spread,omitempty"`
	Removed bool   `protobuf:"varint,6,opt,name=removed,proto3" json:"removed,omitempty"`
}

type OldRoundStandings struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Standings []*OldPlayerStanding `protobuf:"bytes,1,rep,name=standings,proto3" json:"standings,omitempty"`
}

type OldTournamentDivisionDataResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	DivisionId string   `protobuf:"bytes,2,opt,name=division_id,json=divisionId,proto3" json:"division_id,omitempty"`
	Players    []string `protobuf:"bytes,3,rep,name=players,proto3" json:"players,omitempty"`
	// DEPRECIATED
	// TournamentControls controls = 4;
	// In the future we will add different
	// types of tournament divisions and only
	// one will be non nil
	Division []string `protobuf:"bytes,5,rep,name=division,proto3" json:"division,omitempty"`
	// DEPRECIATED
	PlayerIndexMap    map[string]int32               `protobuf:"bytes,6,rep,name=player_index_map,json=playerIndexMap,proto3" json:"player_index_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	PairingMap        map[string]*OldPlayerRoundInfo `protobuf:"bytes,7,rep,name=pairing_map,json=pairingMap,proto3" json:"pairing_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	CurrentRound      int32                          `protobuf:"varint,8,opt,name=current_round,json=currentRound,proto3" json:"current_round,omitempty"`
	Standings         map[int32]*OldRoundStandings   `protobuf:"bytes,9,rep,name=standings,proto3" json:"standings,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	PlayersProperties []*OldPlayerProperties         `protobuf:"bytes,10,rep,name=players_properties,json=playersProperties,proto3" json:"players_properties,omitempty"`
	Finished          bool                           `protobuf:"varint,11,opt,name=finished,proto3" json:"finished,omitempty"`
}

type OldTournamentControls struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Division      string                 `protobuf:"bytes,2,opt,name=division,proto3" json:"division,omitempty"`
	GameRequest   *ipc.GameRequest       `protobuf:"bytes,3,opt,name=game_request,json=gameRequest,proto3" json:"game_request,omitempty"`
	RoundControls []*ipc.RoundControl    `protobuf:"bytes,4,rep,name=round_controls,json=roundControls,proto3" json:"round_controls,omitempty"`
	Type          int32                  `protobuf:"varint,5,opt,name=type,proto3" json:"type,omitempty"`
	StartTime     *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	AutoStart     bool                   `protobuf:"varint,7,opt,name=auto_start,json=autoStart,proto3" json:"auto_start,omitempty"`
}

// Legacy types that need to be migrated

type OldClassicDivision struct {
	Matrix     [][]string                    `json:"matrix"`
	PairingMap map[string]OldPlayerRoundInfo `json:"pairingMap"`
	// By convention, players should look like userUUID:username
	Players           []string                          `json:"players"`
	PlayersProperties []OldPlayerProperties             `json:"playerProperties"`
	PlayerIndexMap    map[string]int32                  `json:"pidxMap"`
	RoundControls     []*ipc.RoundControl               `json:"roundCtrls"`
	CurrentRound      int                               `json:"currentRound"`
	AutoStart         bool                              `json:"autoStart"`
	LastStarted       OldTournamentRoundStarted         `json:"lastStarted"`
	Response          OldTournamentDivisionDataResponse `json:"response"`
}

type OldTournamentDivision struct {
	Players            OldTournamentPersons  `json:"players"`
	Controls           OldTournamentControls `json:"controls"`
	ManagerType        entity.TournamentType `json:"mgrType"`
	DivisionRawMessage json.RawMessage       `json:"json"`
	DivisionManager    OldClassicDivision    `json:"-"`
}

type OldTournament struct {
	sync.RWMutex
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	AliasOf     string `json:"aliasOf"`
	URL         string `json:"url"`
	// XXX: Investigate above.
	ExecutiveDirector string                            `json:"execDirector"`
	Directors         OldTournamentPersons              `json:"directors"`
	IsStarted         bool                              `json:"started"`
	IsFinished        bool                              `json:"finished"`
	Divisions         map[string]*OldTournamentDivision `json:"divs"`
	Type              entity.CompetitionType            `json:"type"`
	ParentID          string                            `json:"parent"`
	Slug              string                            `json:"slug"`
}

type DBStore struct {
	cfg                 *config.Config
	db                  *gorm.DB
	tournamentEventChan chan<- *entity.EventWrapper
	gameStore           gameplay.GameStore
}

type oldtournament struct {
	gorm.Model
	UUID              string `gorm:"uniqueIndex"`
	Name              string
	Description       string
	AliasOf           string
	Directors         datatypes.JSON
	ExecutiveDirector string
	IsStarted         bool
	IsFinished        bool
	Divisions         datatypes.JSON
	// Slug looks like /tournament/abcdef, /club/madison, /club/madison/2020-04-20
	Slug string `gorm:"uniqueIndex:,expression:lower(slug)"`
	// Type is tournament, club, session, and maybe other things.
	Type string
	// Parent is a tournament parent ID.
	Parent string `gorm:"index"`
}

func (oldtournament) TableName() string {
	return "tournaments"
}

func oldDatabaseObjectToEntity(ctx context.Context, s *DBStore, id string) (*OldTournament, error) {
	tm := &oldtournament{}
	ctxDB := s.db.WithContext(ctx)
	if result := ctxDB.Where("uuid = ?", id).First(tm); result.Error != nil {
		return nil, result.Error
	}

	var divisions map[string]*OldTournamentDivision
	err := json.Unmarshal(tm.Divisions, &divisions)
	if err != nil {
		return nil, err
	}

	for _, division := range divisions {
		if division.ManagerType == entity.ClassicTournamentType {
			var classicDivision OldClassicDivision
			err = json.Unmarshal(division.DivisionRawMessage, &classicDivision)
			if err != nil {
				return nil, err
			}
			division.DivisionManager = classicDivision
			division.DivisionRawMessage = nil
		} else {
			return nil, fmt.Errorf("Unknown division manager type: %d", division.ManagerType)
		}
	}

	var directors OldTournamentPersons
	err = json.Unmarshal(tm.Directors, &directors)
	if err != nil {
		return nil, err
	}

	tme := &OldTournament{UUID: tm.UUID,
		Name:              tm.Name,
		Description:       tm.Description,
		AliasOf:           tm.AliasOf,
		Directors:         directors,
		ExecutiveDirector: tm.ExecutiveDirector,
		IsStarted:         tm.IsStarted,
		IsFinished:        tm.IsFinished,
		Divisions:         divisions,
		Type:              entity.CompetitionType(tm.Type),
		ParentID:          tm.Parent,
		Slug:              tm.Slug,
	}
	log.Debug().Msg("return-full")

	return tme, nil
}

// Direct copy from tournament store

type tournament struct {
	gorm.Model
	UUID              string `gorm:"uniqueIndex"`
	Name              string
	Description       string
	AliasOf           string
	Directors         datatypes.JSON
	ExecutiveDirector string
	IsStarted         bool
	IsFinished        bool
	Divisions         datatypes.JSON
	// Slug looks like /tournament/abcdef, /club/madison, /club/madison/2020-04-20
	Slug string `gorm:"uniqueIndex:,expression:lower(slug)"`
	// Type is tournament, club, session, and maybe other things.
	Type string
	// Parent is a tournament parent ID.
	Parent string `gorm:"index"`
}

type registrant struct {
	UserID       string `gorm:"uniqueIndex:idx_registrant;index:idx_user"`
	TournamentID string `gorm:"uniqueIndex:idx_registrant"`
	DivisionID   string `gorm:"uniqueIndex:idx_registrant"`
}

func NewDBStore(config *config.Config, gs gameplay.GameStore) (*DBStore, error) {
	db, err := gorm.Open(postgres.Open(config.DBConnDSN), &gorm.Config{Logger: common.GormLogger})
	if err != nil {
		return nil, err
	}
	if err := db.Use(tracing.NewPlugin()); err != nil {
		return nil, err
	}
	return &DBStore{db: db, gameStore: gs, cfg: config}, nil
}

func (s *DBStore) ListAllIDs(ctx context.Context) ([]string, error) {
	var tids []struct{ UUID string }
	ctxDB := s.db.WithContext(ctx)

	result := ctxDB.Table("tournaments").Select("uuid").Order("created_at").Scan(&tids)
	ids := make([]string, len(tids))
	for idx, tid := range tids {
		ids[idx] = tid.UUID
	}
	return ids, result.Error
}

func (s *DBStore) Set(ctx context.Context, tm *entity.Tournament) error {
	dbt, err := s.toDBObj(tm)
	if err != nil {
		return err
	}

	ctxDB := s.db.WithContext(ctx)
	result := ctxDB.Model(&tournament{}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("uuid = ?", tm.UUID).Updates(dbt)

	return result.Error
}

func (s *DBStore) toDBObj(t *entity.Tournament) (*tournament, error) {

	directors, err := json.Marshal(t.Directors)
	if err != nil {
		return nil, err
	}

	for _, division := range t.Divisions {
		dmJSON, err := json.Marshal(division.DivisionManager)
		if err != nil {
			return nil, err
		}

		division.DivisionRawMessage = dmJSON
	}

	divisions, err := json.Marshal(t.Divisions)
	if err != nil {
		return nil, err
	}

	dbt := &tournament{
		UUID:        t.UUID,
		Name:        t.Name,
		Description: t.Description,
		AliasOf:     t.AliasOf,
		Directors:   directors,
		IsStarted:   t.IsStarted,
		IsFinished:  t.IsFinished,
		Divisions:   divisions,
		Type:        string(t.Type),
		Parent:      t.ParentID,
		Slug:        t.Slug,
	}
	return dbt, nil
}

// Main

func main() {
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	pool, err := common.OpenDB(cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
	if err != nil {
		panic(err)
	}

	userStore, err := user.NewDBStore(pool)
	if err != nil {
		panic(err)
	}

	tmpGameStore, err := game.NewDBStore(cfg, userStore, pool)
	if err != nil {
		panic(err)
	}
	gameStore := game.NewCache(tmpGameStore)

	tournamentStore, err := NewDBStore(cfg, gameStore)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	ids, err := tournamentStore.ListAllIDs(ctx)
	if err != nil {
		panic(err)
	}
	log.Info().Interface("ids", ids).Msg("listed-tournament-ids")

	tournamentStore.db.Transaction(func(tx *gorm.DB) error {
		for _, tid := range ids {
			oldTournament, err := oldDatabaseObjectToEntity(ctx, tournamentStore, tid)
			if err != nil {
				return err
			}

			// Do some conversions here

			newDirectors := &ipc.TournamentPersons{Persons: []*ipc.TournamentPerson{}}
			for oldDirector, key := range oldTournament.Directors.Persons {
				newDirectors.Persons = append(newDirectors.Persons, &ipc.TournamentPerson{Id: oldDirector, Rating: key, Suspended: false})
			}

			newDivisions := make(map[string]*entity.TournamentDivision)
			for name, oldDivision := range oldTournament.Divisions {
				newDivision := &entity.TournamentDivision{}
				newDivision.ManagerType = entity.TournamentType(oldDivision.ManagerType)

				newClassicDivision := pkgtournament.NewClassicDivision(oldTournament.Name, name)
				newClassicDivision.Matrix = oldDivision.DivisionManager.Matrix
				newClassicDivision.PlayerIndexMap = oldDivision.DivisionManager.PlayerIndexMap
				newClassicDivision.RoundControls = oldDivision.DivisionManager.RoundControls
				newClassicDivision.DivisionControls.GameRequest = oldDivision.Controls.GameRequest
				newClassicDivision.DivisionControls.SuspendedResult = ipc.TournamentGameResult_FORFEIT_LOSS
				newClassicDivision.DivisionControls.SuspendedSpread = -50
				newClassicDivision.DivisionControls.AutoStart = oldDivision.Controls.AutoStart

				for _, player := range oldDivision.DivisionManager.Players {
					playerIndex := newClassicDivision.PlayerIndexMap[player]
					newClassicDivision.Players.Persons =
						append(newClassicDivision.Players.Persons,
							&ipc.TournamentPerson{
								Id:        player,
								Rating:    oldDivision.DivisionManager.PlayersProperties[playerIndex].Rating,
								Suspended: oldDivision.DivisionManager.PlayersProperties[playerIndex].Removed,
							})
				}

				for round, pairings := range oldDivision.DivisionManager.Matrix {
					for _, pairingKey := range pairings {
						oldPairing, ok := oldDivision.DivisionManager.PairingMap[pairingKey]
						if ok {
							newClassicDivision.PairingMap[pairingKey] = &ipc.Pairing{
								Players: []int32{
									newClassicDivision.PlayerIndexMap[oldPairing.Players[0]],
									newClassicDivision.PlayerIndexMap[oldPairing.Players[1]],
								},
								Games:       oldPairing.Games,
								Outcomes:    oldPairing.Outcomes,
								ReadyStates: oldPairing.ReadyStates,
								Round:       int32(round)}
						}
					}
					newClassicDivision.Standings[int32(round)], _, err = newClassicDivision.GetStandings(round)
					if err != nil {
						return err
					}
				}
				newClassicDivision.CurrentRound = int32(oldDivision.DivisionManager.CurrentRound)

				newDivision.DivisionManager = newClassicDivision
				newDivisions[name] = newDivision
			}
			finished := oldTournament.IsFinished
			if len(newDivisions) > 0 {
				// Finish all "new style" tournaments.
				// (let's also drop the data in the registrants table by hand)
				finished = true
			}
			mt := &entity.Tournament{
				UUID:        oldTournament.UUID,
				Name:        oldTournament.Name,
				Description: oldTournament.Description,
				AliasOf:     oldTournament.AliasOf,
				URL:         oldTournament.URL,
				Directors:   newDirectors,
				IsStarted:   oldTournament.IsStarted,
				IsFinished:  finished,
				Divisions:   newDivisions,
				Type:        oldTournament.Type,
				ParentID:    oldTournament.ParentID,
				Slug:        oldTournament.Slug,
			}

			dbt, err := tournamentStore.toDBObj(mt)
			if err != nil {
				return err
			}

			ctxDB := tx.WithContext(ctx)
			result := ctxDB.Model(&tournament{}).Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("uuid = ?", mt.UUID).Updates(dbt)

			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}
