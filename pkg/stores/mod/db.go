package mod

import (
	ms "github.com/domino14/liwords/rpc/api/proto/mod_service"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jinzhu/gorm"
)

type NotorietyStore struct {
	db *gorm.DB
}

type ActionHistoryStore struct {
	db *gorm.DB
}

type notoriousgame struct {
	GameID    string `gorm:"index"`
	PlayerID  string `gorm:"index"`
	Type      int    `gorm:"index"`
	Timestamp int64  `gorm:"index"`
}

type action struct {
	UserID        string                 `gorm:"index"`
	Type          int                    `gorm:"index"`
	Duration      int32                  `gorm:"index"`
	StartTime     *timestamppb.Timestamp `gorm:"index"`
	EndTime       *timestamppb.Timestamp `gorm:"index"`
	RemovedTime   *timestamppb.Timestamp `gorm:"index"`
	MessageID     string                 `gorm:"index"`
	ApplierUserID string                 `gorm:"index"`
	RemoverUserID string                 `gorm:"index"`
	ChatText      string                 `gorm:"index"`
	Note          string                 `gorm:"index"`
}

func NewNotorietyStore(dbURL string) (*NotorietyStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&notoriousgame{})
	return &NotorietyStore{db: db}, nil
}

func NewActionHistoryStore(dbURL string) (*ActionHistoryStore, error) {
	db, err := gorm.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	db.AutoMigrate(&action{})
	return &ActionHistoryStore{db: db}, nil
}

func (ns *NotorietyStore) AddNotoriousGame(playerID string, gameID string, gameType int, time int64) error {
	dbi := &notoriousgame{
		GameID:    gameID,
		PlayerID:  playerID,
		Type:      gameType,
		Timestamp: time,
	}
	result := ns.db.Create(dbi)
	return result.Error
}

func (ns *NotorietyStore) GetNotoriousGames(playerID string) ([]*ms.NotoriousGame, error) {
	var games []notoriousgame

	result := ns.db.Table("notoriousgames").
		Select("game_id, type, timestamp").
		Where("player_id = ?", []interface{}{playerID}).
		Order("timestamp").Scan(&games)
	if result.Error != nil {
		return nil, result.Error
	}

	items := make([]*ms.NotoriousGame, len(games))
	for idx, dbgame := range games {

		items[idx] = &ms.NotoriousGame{
			Id:   dbgame.GameID,
			Type: ms.NotoriousGameType(dbgame.Type),
			// Converting from a Unix timestamp to
			// protobuf timestamp is quite expensive.
			// As far as I know, it involves two separate
			// conversions. We will omit this time for now.
			// Suggestions welcome.
			// CreatedAt: dbgame.Timestamp,
		}
	}

	return items, nil
}

func (ns *NotorietyStore) DeleteNotoriousGames(playerID string) error {
	result := ns.db.Table("notoriousgames").Delete(&notoriousgame{PlayerID: playerID})
	return result.Error
}

func (ns *NotorietyStore) Disconnect() {
	ns.db.Close()
}

func (ahs *ActionHistoryStore) AddAction(modAction *ms.ModAction) error {
	result := ahs.db.Create(ahs.toDBObj(modAction))
	return result.Error
}

func (ahs *ActionHistoryStore) GetActions(playerID string) ([]*ms.ModAction, error) {
	var actions []action

	result := ahs.db.Table("actionhistory").
		Select("game_id, type, timestamp").
		Where("player_id = ?", []interface{}{playerID}).
		Order("timestamp").Scan(&actions)
	if result.Error != nil {
		return nil, result.Error
	}

	items := make([]*ms.ModAction, len(actions))
	for idx, dbaction := range actions {
		items[idx] = ahs.toGoStruct(dbaction)
	}

	return items, nil
}

func (ahs *ActionHistoryStore) toGoStruct(action action) *ms.ModAction {
	return &ms.ModAction{UserId: action.UserID,
		Type:          ms.ModActionType(action.Type),
		Duration:      int32(action.Duration),
		StartTime:     action.StartTime,
		EndTime:       action.EndTime,
		RemovedTime:   action.RemovedTime,
		MessageId:     action.MessageID,
		ApplierUserId: action.ApplierUserID,
		RemoverUserId: action.RemoverUserID,
		ChatText:      action.ChatText,
		Note:          action.Note}
}

func (ahs *ActionHistoryStore) toDBObj(modAction *ms.ModAction) *action {
	return &action{UserID: modAction.UserId,
		Type:          int(modAction.Type),
		Duration:      modAction.Duration,
		StartTime:     modAction.StartTime,
		EndTime:       modAction.EndTime,
		RemovedTime:   modAction.RemovedTime,
		MessageID:     modAction.MessageId,
		ApplierUserID: modAction.ApplierUserId,
		RemoverUserID: modAction.RemoverUserId,
		ChatText:      modAction.ChatText,
		Note:          modAction.Note}
}

func (ns *ActionHistoryStore) Disconnect() {
	ns.db.Close()
}
