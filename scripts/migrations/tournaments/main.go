package main

import (
	"context"
	"os"
	"fmt"
	"encoding/json"
	"sync"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/driver/postgres"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"


	macondo "github.com/domino14/macondo/gen/api/proto/macondo"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/stores/user"
	"github.com/domino14/liwords/pkg/stores/game"
)

// Legacy realtime types that have probably since changed are
// copied here so we don't have to think about conflicts
type OldGameMode int32

type OldRatingMode int32

type OldGameRules struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BoardLayoutName        string `protobuf:"bytes,1,opt,name=board_layout_name,json=boardLayoutName,proto3" json:"board_layout_name,omitempty"`
	LetterDistributionName string `protobuf:"bytes,2,opt,name=letter_distribution_name,json=letterDistributionName,proto3" json:"letter_distribution_name,omitempty"`
	// If blank, variant is classic, otherwise it could be some other game
	// (a is worth 100, dogworms, etc.)
	VariantName string `protobuf:"bytes,3,opt,name=variant_name,json=variantName,proto3" json:"variant_name,omitempty"`
}

type OldGameRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Lexicon            string                `protobuf:"bytes,1,opt,name=lexicon,proto3" json:"lexicon,omitempty"`
	Rules              *OldGameRules            `protobuf:"bytes,2,opt,name=rules,proto3" json:"rules,omitempty"`
	InitialTimeSeconds int32                 `protobuf:"varint,3,opt,name=initial_time_seconds,json=initialTimeSeconds,proto3" json:"initial_time_seconds,omitempty"`
	IncrementSeconds   int32                 `protobuf:"varint,4,opt,name=increment_seconds,json=incrementSeconds,proto3" json:"increment_seconds,omitempty"`
	ChallengeRule      macondo.ChallengeRule `protobuf:"varint,5,opt,name=challenge_rule,json=challengeRule,proto3,enum=macondo.ChallengeRule" json:"challenge_rule,omitempty"`
	GameMode           OldGameMode              `protobuf:"varint,6,opt,name=game_mode,json=gameMode,proto3,enum=liwords.GameMode" json:"game_mode,omitempty"`
	RatingMode         OldRatingMode            `protobuf:"varint,7,opt,name=rating_mode,json=ratingMode,proto3,enum=liwords.RatingMode" json:"rating_mode,omitempty"`
	RequestId          string                `protobuf:"bytes,8,opt,name=request_id,json=requestId,proto3" json:"request_id,omitempty"`
	MaxOvertimeMinutes int32                 `protobuf:"varint,9,opt,name=max_overtime_minutes,json=maxOvertimeMinutes,proto3" json:"max_overtime_minutes,omitempty"`
	PlayerVsBot        bool                  `protobuf:"varint,10,opt,name=player_vs_bot,json=playerVsBot,proto3" json:"player_vs_bot,omitempty"`
	OriginalRequestId  string                `protobuf:"bytes,11,opt,name=original_request_id,json=originalRequestId,proto3" json:"original_request_id,omitempty"`
}


type OldTournamentGameResult int32

type OldGameEndReason int32

type OldPairingMethod int32

type OldFirstMethod int32

type OldTournamentGame struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Scores        []int32                `protobuf:"varint,1,rep,packed,name=scores,proto3" json:"scores,omitempty"`
	Results       []OldTournamentGameResult `protobuf:"varint,2,rep,packed,name=results,proto3,enum=liwords.TournamentGameResult" json:"results,omitempty"`
	GameEndReason OldGameEndReason          `protobuf:"varint,3,opt,name=game_end_reason,json=gameEndReason,proto3,enum=liwords.GameEndReason" json:"game_end_reason,omitempty"`
	Id            string                 `protobuf:"bytes,4,opt,name=id,proto3" json:"id,omitempty"`
}

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

	Players     []string               `protobuf:"bytes,1,rep,name=players,proto3" json:"players,omitempty"`
	Games       []*OldTournamentGame      `protobuf:"bytes,2,rep,name=games,proto3" json:"games,omitempty"` // can be a list, for elimination tourneys
	Outcomes    []OldTournamentGameResult `protobuf:"varint,3,rep,packed,name=outcomes,proto3,enum=liwords.TournamentGameResult" json:"outcomes,omitempty"`
	ReadyStates []string               `protobuf:"bytes,4,rep,name=ready_states,json=readyStates,proto3" json:"ready_states,omitempty"`
}

type OldPlayerProperties struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Removed   bool  `protobuf:"varint,1,opt,name=removed,proto3" json:"removed,omitempty"`
	Rating    int32 `protobuf:"varint,2,opt,name=rating,proto3" json:"rating,omitempty"`
	CheckedIn bool  `protobuf:"varint,3,opt,name=checked_in,json=checkedIn,proto3" json:"checked_in,omitempty"`
}

type OldRoundControl struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	PairingMethod               OldPairingMethod `protobuf:"varint,1,opt,name=pairing_method,json=pairingMethod,proto3,enum=liwords.PairingMethod" json:"pairing_method,omitempty"`
	FirstMethod                 OldFirstMethod   `protobuf:"varint,2,opt,name=first_method,json=firstMethod,proto3,enum=liwords.FirstMethod" json:"first_method,omitempty"`
	GamesPerRound               int32         `protobuf:"varint,3,opt,name=games_per_round,json=gamesPerRound,proto3" json:"games_per_round,omitempty"`
	Round                       int32         `protobuf:"varint,4,opt,name=round,proto3" json:"round,omitempty"`
	Factor                      int32         `protobuf:"varint,5,opt,name=factor,proto3" json:"factor,omitempty"`
	InitialFontes               int32         `protobuf:"varint,6,opt,name=initial_fontes,json=initialFontes,proto3" json:"initial_fontes,omitempty"`
	MaxRepeats                  int32         `protobuf:"varint,7,opt,name=max_repeats,json=maxRepeats,proto3" json:"max_repeats,omitempty"`
	AllowOverMaxRepeats         bool          `protobuf:"varint,8,opt,name=allow_over_max_repeats,json=allowOverMaxRepeats,proto3" json:"allow_over_max_repeats,omitempty"`
	RepeatRelativeWeight        int32         `protobuf:"varint,9,opt,name=repeat_relative_weight,json=repeatRelativeWeight,proto3" json:"repeat_relative_weight,omitempty"`
	WinDifferenceRelativeWeight int32         `protobuf:"varint,10,opt,name=win_difference_relative_weight,json=winDifferenceRelativeWeight,proto3" json:"win_difference_relative_weight,omitempty"`
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
	PlayerIndexMap    map[string]int32            `protobuf:"bytes,6,rep,name=player_index_map,json=playerIndexMap,proto3" json:"player_index_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	PairingMap        map[string]*OldPlayerRoundInfo `protobuf:"bytes,7,rep,name=pairing_map,json=pairingMap,proto3" json:"pairing_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	CurrentRound      int32                       `protobuf:"varint,8,opt,name=current_round,json=currentRound,proto3" json:"current_round,omitempty"`
	Standings         map[int32]*OldRoundStandings   `protobuf:"bytes,9,rep,name=standings,proto3" json:"standings,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	PlayersProperties []*OldPlayerProperties         `protobuf:"bytes,10,rep,name=players_properties,json=playersProperties,proto3" json:"players_properties,omitempty"`
	Finished          bool                        `protobuf:"varint,11,opt,name=finished,proto3" json:"finished,omitempty"`
}

type OldTournamentControls struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Division      string                 `protobuf:"bytes,2,opt,name=division,proto3" json:"division,omitempty"`
	GameRequest   *OldGameRequest           `protobuf:"bytes,3,opt,name=game_request,json=gameRequest,proto3" json:"game_request,omitempty"`
	RoundControls []*OldRoundControl        `protobuf:"bytes,4,rep,name=round_controls,json=roundControls,proto3" json:"round_controls,omitempty"`
	Type          int32                  `protobuf:"varint,5,opt,name=type,proto3" json:"type,omitempty"`
	StartTime     *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	AutoStart     bool                   `protobuf:"varint,7,opt,name=auto_start,json=autoStart,proto3" json:"auto_start,omitempty"`
}



// Legacy types that need to be migrated

type OldClassicDivision struct {
	Matrix     [][]string                           `json:"matrix"`
	PairingMap map[string]OldPlayerRoundInfo `json:"pairingMap"`
	// By convention, players should look like userUUID:username
	Players           []string                                 `json:"players"`
	PlayersProperties []OldPlayerProperties             `json:"playerProperties"`
	PlayerIndexMap    map[string]int32                         `json:"pidxMap"`
	RoundControls     []OldRoundControl                 `json:"roundCtrls"`
	CurrentRound      int                                      `json:"currentRound"`
	AutoStart         bool                                     `json:"autoStart"`
	LastStarted       OldTournamentRoundStarted         `json:"lastStarted"`
	Response          OldTournamentDivisionDataResponse `json:"response"`
}

type OldCompetitionType string

type OldTournamentType int

const (
	OldClassicTournamentType OldTournamentType = iota
	// It's gonna be lit:
	OldArenaTournamentType
)

type OldTournamentDivision struct {
	Players            OldTournamentPersons  `json:"players"`
	Controls           OldTournamentControls `json:"controls"`
	ManagerType        OldTournamentType               `json:"mgrType"`
	DivisionRawMessage json.RawMessage              `json:"json"`
	DivisionManager    OldClassicDivision              `json:"-"`
}

type OldTournament struct {
	sync.RWMutex
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	AliasOf string `json:"aliasOf"`
	URL     string `json:"url"`
	// XXX: Investigate above.
	ExecutiveDirector string                         `json:"execDirector"`
	Directors         OldTournamentPersons    `json:"directors"`
	IsStarted         bool                           `json:"started"`
	IsFinished        bool                           `json:"finished"`
	Divisions         map[string]*OldTournamentDivision `json:"divs"`
	DefaultSettings   OldGameRequest          `json:"settings"`
	Type              OldCompetitionType                `json:"type"`
	ParentID          string                         `json:"parent"`
	Slug              string                         `json:"slug"`
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
	// DefaultSettings are mostly used for clubs. It's the default settings for
	// games in that club. It can be used for non-clubs as well, in perhaps
	// an advertisement or other such tournament page. (But in tournaments,
	// each division has their own settings).
	DefaultSettings datatypes.JSON
	// Type is tournament, club, session, and maybe other things.
	Type string
	// Parent is a tournament parent ID.
	Parent string `gorm:"index"`
}

type oldregistrant struct {
	UserID       string `gorm:"uniqueIndex:idx_registrant;index:idx_user"`
	TournamentID string `gorm:"uniqueIndex:idx_registrant"`
	DivisionID   string `gorm:"uniqueIndex:idx_registrant"`
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
		if division.ManagerType == OldClassicTournamentType {
			log.Debug().Interface("division", division).Msg("unmarshalling")
			var classicDivision OldClassicDivision
			err = json.Unmarshal(division.DivisionRawMessage, classicDivision)
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
	err = json.Unmarshal(tm.Directors, directors)
	if err != nil {
		return nil, err
	}

	var defaultSettings OldGameRequest
	err = json.Unmarshal(tm.DefaultSettings, defaultSettings)
	if err != nil {
		// it's ok, don't error out; this tournament has no default settings
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
		DefaultSettings:   defaultSettings,
		Type:              OldCompetitionType(tm.Type),
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
	// DefaultSettings are mostly used for clubs. It's the default settings for
	// games in that club. It can be used for non-clubs as well, in perhaps
	// an advertisement or other such tournament page. (But in tournaments,
	// each division has their own settings).
	DefaultSettings datatypes.JSON
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
	db, err := gorm.Open(postgres.Open(config.DBConnString), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&tournament{})
	db.AutoMigrate(&registrant{})
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

	defaultSettings, err := json.Marshal(t.DefaultSettings)
	if err != nil {
		// for now
		defaultSettings = []byte("{}")
	}

	dbt := &tournament{
		UUID:              t.UUID,
		Name:              t.Name,
		Description:       t.Description,
		AliasOf:           t.AliasOf,
		Directors:         directors,
		ExecutiveDirector: t.ExecutiveDirector,
		IsStarted:         t.IsStarted,
		IsFinished:        t.IsFinished,
		Divisions:         divisions,
		DefaultSettings:   defaultSettings,
		Type:              string(t.Type),
		Parent:            t.ParentID,
		Slug:              t.Slug,
	}
	return dbt, nil
}

// Main

func main() {
	// Populate every game with its quickdata
	cfg := &config.Config{}
	cfg.Load(os.Args[1:])
	log.Info().Msgf("Loaded config: %v", cfg)

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	
	userStore, err := user.NewDBStore(cfg.DBConnString)
	if err != nil {
		panic(err)
	}

	tmpGameStore, err := game.NewDBStore(cfg, userStore)
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
	log.Info().Interface("ids", ids).Msg("listed-game-ids")

	for _, tid := range ids {
		_, err := oldDatabaseObjectToEntity(ctx, tournamentStore, tid)
		if err != nil {
			panic(err)
		}

		// Do some conversions here

		err = tournamentStore.Set(ctx, &entity.Tournament{})
		if err != nil {
			panic(err)
		}
	}
}