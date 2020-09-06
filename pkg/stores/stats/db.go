package stats

import (
	"github.com/domino14/liwords/pkg/entity"
)

type ListStatStore struct {
}

func NewListStatStore(dbURL string) *ListStatStore {
	return &ListStatStore{}
}

func (l *ListStatStore) AddListItem(gameId string, playerId string, statType int, time int64, item interface{}) error {
	return nil
}

func (l *ListStatStore) GetListItems(statType int, gameIds []string, playerId string) ([]*entity.ListItem, error) {
	return nil, nil
}
