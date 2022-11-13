package game

import "github.com/jackc/pgx/v4/pgxpool"

type AnnotatedDBStore struct {
	dbPool *pgxpool.Pool
}

func NewAnnotatedDBStore(p *pgxpool.Pool) (*AnnotatedDBStore, error) {
	return &AnnotatedDBStore{dbPool: p}, nil
}

func (s *AnnotatedDBStore) Disconnect() {
	s.dbPool.Close()
}

func (s *AnnotatedDBStore) CreateRelationship(gid string, uid string) error {

	return nil
}

func (s *AnnotatedDBStore) GetGamesForUser(uid string, limit, offset int) error {

	return nil
}

func (s *AnnotatedDBStore) GetUnfinishedGamesForUser(uid string) ([]string, error) {

	return nil, nil
}

func (s *AnnotatedDBStore) AddEditor(gid string, uid string) error {

	return nil
}

func (s *AnnotatedDBStore) RemoveEditor(gid string, uid string) error {
	return nil
}

// DeleteGame should delete the game from the original game table, plus
// the relationship with any editors. It should be rejected if
// the game is already finished.
func (s *AnnotatedDBStore) DeleteGame(gid, uid string) error {

	return nil
}
