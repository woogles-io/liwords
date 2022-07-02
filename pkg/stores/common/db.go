package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/glicko"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

type SelectByType int

const (
	SelectByUUID SelectByType = iota
	SelectByID
	SelectByUserID
	SelectByUsername
	SelectByEmail
	SelectByAPIKey
	SelectBySeekerID
	SelectBySeekerConnID
	SelectByReceiverID
	SelectByReceiverConnID
)

type TableType int

const (
	UsersTable TableType = iota
	ProfilesTable
	GamesTable
	PuzzlesTable
	SoughtGamesTable
)

type RowsAffectedType int

const (
	AnyRowsAffected RowsAffectedType = iota
	AtMostOneRowAffected
	ExactlyOneRowAffected
	AtLeastOneRowAffected
)

type CommonDBConfig struct {
	SelectByType     SelectByType
	TableType        TableType
	RowsAffectedType RowsAffectedType
	Value            interface{}
	SetUpdatedAt     bool
	IncludeProfile   bool
}

var SelectByTypeToString = map[SelectByType]string{
	SelectByUUID:           "uuid",
	SelectByID:             "id",
	SelectByUserID:         "user_id",
	SelectByUsername:       "lower(username)",
	SelectByEmail:          "lower(email)",
	SelectByAPIKey:         "api_key",
	SelectBySeekerID:       "seeker",
	SelectBySeekerConnID:   "seeker_conn_id",
	SelectByReceiverID:     "receiver",
	SelectByReceiverConnID: "receiver_conn_id",
}

var TableTypeToString = map[TableType]string{
	UsersTable:       "users",
	ProfilesTable:    "profiles",
	GamesTable:       "games",
	PuzzlesTable:     "puzzles",
	SoughtGamesTable: "soughtgames",
}

var DefaultTxOptions = pgx.TxOptions{
	IsoLevel:       pgx.ReadCommitted,
	AccessMode:     pgx.ReadWrite,
	DeferrableMode: pgx.Deferrable, // not used for this isolevel/access mode
}

var RepeatableReadTxOptions = pgx.TxOptions{
	IsoLevel:       pgx.RepeatableRead,
	AccessMode:     pgx.ReadWrite,
	DeferrableMode: pgx.Deferrable, // not used for this isolevel/access mode
}

var InitialRating = &entity.SingleRating{
	Rating:          float64(glicko.InitialRating),
	RatingDeviation: float64(glicko.InitialRatingDeviation),
	Volatility:      glicko.InitialVolatility,
}

type RowIterator interface {
	Close()
	Next() bool
	Scan(dest ...interface{}) error
}

func GetUserDBIDFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, "SELECT id FROM users WHERE uuid = $1", uuid).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table users", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetUsernameFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (string, error) {
	var username string
	err := tx.QueryRow(ctx, "SELECT username FROM users WHERE uuid = $1", uuid).Scan(&username)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("cannot get username from uuid %s: no rows for table users", uuid)
	}
	if err != nil {
		return "", err
	}
	return username, nil
}

func GetGameDBIDFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, "SELECT id FROM games WHERE uuid = $1", uuid).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table games", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetPuzzleDBIDFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, "SELECT id FROM puzzles WHERE uuid = $1", uuid).Scan(&id)
	if err == pgx.ErrNoRows {
		return 0, fmt.Errorf("cannot get id from uuid %s: no rows for table puzzles", uuid)
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetGameInfo(ctx context.Context, tx pgx.Tx, gameId int) (*macondopb.GameHistory, *ipc.GameRequest, string, error) {
	var uuid string
	var historyBytes []byte
	var requestBytes []byte
	err := tx.QueryRow(ctx, `SELECT uuid, history, request FROM games WHERE id = $1`, gameId).Scan(&uuid, &historyBytes, &requestBytes)
	if err == pgx.ErrNoRows {
		return nil, nil, "", fmt.Errorf("no rows for games table: %d", gameId)
	}
	if err != nil {
		return nil, nil, "", err
	}

	hist := &macondopb.GameHistory{}
	err = proto.Unmarshal(historyBytes, hist)
	if err != nil {
		return nil, nil, "", err
	}

	req := &ipc.GameRequest{}
	err = proto.Unmarshal(requestBytes, req)
	if err != nil {
		return nil, nil, "", err
	}

	return hist, req, uuid, nil
}

func InitializeUserRating(ctx context.Context, tx pgx.Tx, userId int64) error {
	_, err := tx.Exec(ctx, `UPDATE profiles SET ratings = jsonb_set(ratings, '{Data}', jsonb '{}') WHERE user_id = $1 AND NULLIF(ratings->'Data', 'null') IS NULL`, userId)
	return err
}

func InitializeUserStats(ctx context.Context, tx pgx.Tx, userId int64) error {
	_, err := tx.Exec(ctx, `UPDATE profiles SET stats = jsonb_set(stats, '{Data}', jsonb '{}') WHERE user_id = $1 AND NULLIF(stats->'Data', 'null') IS NULL`, userId)
	return err
}

func GetUserRating(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey) (*entity.SingleRating, error) {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	var playerRating *entity.SingleRating
	err = tx.QueryRow(ctx, "SELECT ratings->'Data'->$1 FROM profiles WHERE user_id = $2", ratingKey, userId).Scan(&playerRating)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("ratings not found for user_id: %d", userId)
	}
	if err != nil {
		return nil, err
	}

	if playerRating == nil {
		playerRating = entity.NewDefaultRating(true)
		err = UpdateUserRating(ctx, tx, userId, ratingKey, playerRating)
		if err != nil {
			return nil, err
		}
	}

	return playerRating, nil
}

func GetUserStats(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey) (*entity.Stats, error) {
	err := InitializeUserStats(ctx, tx, userId)
	if err != nil {
		return nil, err
	}

	var stats *entity.Stats
	err = tx.QueryRow(ctx, "SELECT stats->'Data'->$1 FROM profiles WHERE user_id = $2", ratingKey, userId).Scan(&stats)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("stats not found for user_id: %d", userId)
	}
	if err != nil {
		return nil, err
	}

	if stats == nil {
		stats = &entity.Stats{}
		err = UpdateUserStats(ctx, tx, userId, ratingKey, stats)
		if err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func Update(ctx context.Context, tx pgx.Tx, columns []string, args []interface{}, cfg *CommonDBConfig) error {
	for i := 0; i < len(columns); i++ {
		columnName := columns[i]
		columns[i] = fmt.Sprintf("%s = $%d", columnName, i+1)
	}

	setUpdatedAtStmt := ""
	if cfg.SetUpdatedAt {
		setUpdatedAtStmt = " updated_at = NOW(), "
	}
	query := fmt.Sprintf("UPDATE %s SET %s %s WHERE %s = $%d", TableTypeToString[cfg.TableType], setUpdatedAtStmt, strings.Join(columns, ","), SelectByTypeToString[cfg.SelectByType], len(columns)+1)
	args = append(args, []interface{}{cfg.Value}...)
	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected != 1 {
		return entity.NewWooglesError(ipc.WooglesError_USER_UPDATE_NOT_FOUND, fmt.Sprintf("%v", cfg.Value))
	}

	return nil
}

func Delete(ctx context.Context, tx pgx.Tx, cfg *CommonDBConfig) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1`, TableTypeToString[cfg.TableType], SelectByTypeToString[cfg.SelectByType])
	result, err := tx.Exec(ctx, query, cfg.Value)
	if err != nil {
		return err
	}

	return checkRowsAffected(int(result.RowsAffected()), cfg)
}

func GetUserBy(ctx context.Context, tx pgx.Tx, cfg *CommonDBConfig) (*entity.User, error) {
	var id uint
	var username string
	var uuid string
	var email sql.NullString
	var password sql.NullString
	var internal_bot sql.NullBool
	var is_admin sql.NullBool
	var is_director sql.NullBool
	var is_mod sql.NullBool
	var notoriety sql.NullInt64
	var actions entity.Actions

	placeholder := "$1"

	if cfg.SelectByType == SelectByEmail || cfg.SelectByType == SelectByUsername {
		placeholder = "lower($1)"
	}

	query := fmt.Sprintf("SELECT id, username, uuid, email, password, internal_bot, is_admin, is_director, is_mod, notoriety, actions FROM users WHERE %s = %s", SelectByTypeToString[cfg.SelectByType], placeholder)
	err := tx.QueryRow(ctx, query, cfg.Value).Scan(&id, &username, &uuid, &email, &password, &internal_bot, &is_admin, &is_director, &is_mod, &notoriety, &actions)
	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	} else if err != nil {
		return nil, err
	}

	entu := &entity.User{
		ID:         id,
		Username:   username,
		UUID:       uuid,
		Email:      email.String,
		Password:   password.String,
		IsBot:      internal_bot.Bool,
		Anonymous:  false,
		IsAdmin:    is_admin.Bool,
		IsDirector: is_director.Bool,
		IsMod:      is_mod.Bool,
		Notoriety:  int(notoriety.Int64),
		Actions:    &actions,
	}

	if cfg.IncludeProfile {
		var firstName sql.NullString
		var lastName sql.NullString
		var birthDate sql.NullString
		var countryCode sql.NullString
		var title sql.NullString
		var about sql.NullString
		var avatar_url sql.NullString
		var rdata entity.Ratings
		var sdata entity.ProfileStats
		var silentMode sql.NullBool

		err = tx.QueryRow(ctx, "SELECT first_name, last_name, birth_date, country_code, title, about, avatar_url, silent_mode, ratings, stats FROM profiles WHERE user_id = $1", id).Scan(&firstName, &lastName, &birthDate, &countryCode, &title, &about, &avatar_url, &silentMode, &rdata, &sdata)
		if err == pgx.ErrNoRows {
			return nil, errors.New("profile not found")
		} else if err != nil {
			return nil, err
		}

		entp := &entity.Profile{
			FirstName:   firstName.String,
			LastName:    lastName.String,
			BirthDate:   birthDate.String,
			CountryCode: countryCode.String,
			Title:       title.String,
			About:       about.String,
			Ratings:     rdata,
			Stats:       sdata,
			AvatarUrl:   avatar_url.String,
			SilentMode:  silentMode.Bool,
		}

		entu.Profile = entp
	}
	return entu, nil
}

func UpdateUserRating(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey, newRating *entity.SingleRating) error {
	err := InitializeUserRating(ctx, tx, userId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE profiles SET ratings = jsonb_set(ratings, array['Data', $1], $2) WHERE user_id = $3", ratingKey, newRating, userId)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserStats(ctx context.Context, tx pgx.Tx, userId int64, ratingKey entity.VariantKey, newStats *entity.Stats) error {
	err := InitializeUserStats(ctx, tx, userId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, "UPDATE profiles SET stats = jsonb_set(stats, array['Data', $1], $2) WHERE user_id = $3", ratingKey, newStats, userId)
	if err != nil {
		return err
	}
	return nil
}

func OpenDB(host, port, name, user, password, sslmode string) (*pgxpool.Pool, error) {
	connStr := PostgresConnUri(host, port, name, user, password, sslmode)
	ctx := context.Background()

	dbPool, err := pgxpool.Connect(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	err = dbPool.Ping(ctx)
	if err != nil {
		return nil, err
	}
	return dbPool, nil
}

func BuildIn(num int, start int) string {
	var stmt strings.Builder
	fmt.Fprintf(&stmt, "$%d", start)
	for i := start + 1; i < start+num; i++ {
		fmt.Fprintf(&stmt, ", $%d", i)
	}
	return stmt.String()
}

func PostgresConnUri(host, port, name, user, password, sslmode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, name, sslmode)
}

// PostgresConnDSN is obsolete and only for Gorm. Remove once we get rid of gorm.
func PostgresConnDSN(host, port, name, user, password, sslmode string) string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		host, port, name, user, password, sslmode)
}

func checkRowsAffected(rowsAffected int, cfg *CommonDBConfig) error {
	if cfg.RowsAffectedType != AnyRowsAffected {
		errType := ""
		if cfg.RowsAffectedType == AtMostOneRowAffected && rowsAffected > 1 {
			errType = "at most"
		} else if cfg.RowsAffectedType == ExactlyOneRowAffected && rowsAffected != 1 {
			errType = "exactly"
		} else if cfg.RowsAffectedType == AtLeastOneRowAffected && rowsAffected < 1 {
			errType = "at least"
		}
		if errType != "" {
			return fmt.Errorf("not %s row with value %v for %v in delete for table %s (%d rows)", errType, cfg.Value, SelectByTypeToString[cfg.SelectByType], TableTypeToString[cfg.TableType], rowsAffected)
		}
	}
	return nil
}
