package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lithammer/shortuuid/v4"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/common"
	"github.com/woogles-io/liwords/pkg/stores/models"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	ms "github.com/woogles-io/liwords/rpc/api/proto/mod_service"
	pb "github.com/woogles-io/liwords/rpc/api/proto/user_service"
)

var (
	tracer = otel.Tracer("user-db")
)

var botNames = map[macondopb.BotRequest_BotCode]string{
	macondopb.BotRequest_HASTY_BOT: "HastyBot",
	// These are temporary names!
	macondopb.BotRequest_LEVEL1_PROBABILISTIC: "BeginnerBot",
	macondopb.BotRequest_LEVEL2_PROBABILISTIC: "BasicBot",
	macondopb.BotRequest_LEVEL3_PROBABILISTIC: "BetterBot",
	macondopb.BotRequest_LEVEL4_PROBABILISTIC: "STEEBot",

	// For dictionaries with common-word. Reuse level1-3 names from above.
	macondopb.BotRequest_LEVEL1_COMMON_WORD_BOT: "BeginnerBot",
	macondopb.BotRequest_LEVEL2_COMMON_WORD_BOT: "BasicBot",
	macondopb.BotRequest_LEVEL4_COMMON_WORD_BOT: "BetterBot",

	macondopb.BotRequest_SIMMING_BOT: "BestBot",
}

var userActionsSQLSelection = "user_actions.id, user_actions.user_id, user_actions.action_type, user_actions.start_time, user_actions.end_time, user_actions.removed_time, user_actions.message_id, user_actions.applier_id, user_actions.remover_id, user_actions.note, user_actions.removal_note, user_actions.chat_text, user_actions.email_type"

type DBUniqueValues struct {
	actionDBID  int64
	removalNote string
	userDBID    int64
	applierDBID pgtype.Int8
	removerDBID pgtype.Int8
}

// DBStore is a postgres-backed store for users.
type DBStore struct {
	dbPool  *pgxpool.Pool
	queries *models.Queries
}

func NewDBStore(p *pgxpool.Pool) (*DBStore, error) {
	q := models.New(p)
	return &DBStore{dbPool: p, queries: q}, nil
}

func (s *DBStore) Disconnect() {
	s.dbPool.Close()
}

// Get gets a user by username.
func (s *DBStore) Get(ctx context.Context, username string) (*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	entu, err := common.GetUserBy(ctx, tx, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUsername, Value: username, IncludeProfile: true})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entu, nil
}

func (s *DBStore) SetNotoriety(ctx context.Context, uuid string, notoriety int) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = common.Update(ctx, tx, []string{"notoriety"}, []interface{}{notoriety}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: uuid})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetByEmail gets the user by email. It does not try to get the profile.
// We don't get the profile here because GetByEmail is only used for things
// like password resets and there is no need.
func (s *DBStore) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	entu, err := common.GetUserBy(ctx, tx, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByEmail, Value: strings.ToLower(email)})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entu, nil
}

// GetByUUID gets user by UUID
func (s *DBStore) GetByUUID(ctx context.Context, uuid string) (*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	entu, err := common.GetUserBy(ctx, tx, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: uuid, IncludeProfile: true})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entu, nil
}

// GetByAPIKey gets a user by api key. It does not try to fetch the profile. We only
// call this for API functions where we care about access levels, etc.
func (s *DBStore) GetByAPIKey(ctx context.Context, apikey string) (*entity.User, error) {
	if apikey == "" {
		return nil, errors.New("api-key is blank")
	}
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	entu, err := common.GetUserBy(ctx, tx, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByAPIKey, Value: apikey})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entu, nil
}

// GetByVerificationToken gets a user by their verification token
func (s *DBStore) GetByVerificationToken(ctx context.Context, token string) (*entity.User, error) {
	if token == "" {
		return nil, errors.New("verification token is blank")
	}
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	entu, err := common.GetUserBy(ctx, tx, &common.CommonDBConfig{
		TableType:      common.UsersTable,
		SelectByType:   common.SelectByVerificationToken,
		Value:          token,
		IncludeProfile: true,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entu, nil
}

// SetEmailVerified marks a user's email as verified or unverified
func (s *DBStore) SetEmailVerified(ctx context.Context, uuid string, verified bool) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Set verified status (keep verification token for idempotency - it will be cleaned up by maintenance job)
	if verified {
		err = common.Update(ctx, tx, []string{"verified"},
			[]interface{}{true},
			&common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: uuid})
	} else {
		err = common.Update(ctx, tx, []string{"verified"},
			[]interface{}{false},
			&common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: uuid})
	}

	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// UpdateVerificationToken updates a user's verification token and expiration
func (s *DBStore) UpdateVerificationToken(ctx context.Context, uuid string, token string, expiresAt time.Time) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = common.Update(ctx, tx, []string{"verification_token", "verification_expires_at"},
		[]interface{}{token, expiresAt},
		&common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: uuid})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// DeleteUnverifiedUsers deletes users who haven't verified their email after the specified duration
func (s *DBStore) DeleteUnverifiedUsers(ctx context.Context, olderThan time.Duration) (int, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	cutoffTime := time.Now().Add(-olderThan)

	// Delete profiles first (foreign key constraint)
	_, err = tx.Exec(ctx, `
		DELETE FROM profiles
		WHERE user_id IN (
			SELECT id FROM users
			WHERE verified = false AND created_at < $1
		)
	`, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete unverified user profiles: %w", err)
	}

	// Delete users
	result, err := tx.Exec(ctx, `
		DELETE FROM users
		WHERE verified = false AND created_at < $1
	`, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete unverified users: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	rowsAffected := int(result.RowsAffected())
	if rowsAffected > 0 {
		log.Info().Int("count", rowsAffected).Dur("older_than", olderThan).Msg("deleted-unverified-users")
	}

	return rowsAffected, nil
}

// New creates a new user in the DB.
func (s *DBStore) New(ctx context.Context, u *entity.User) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if u.UUID == "" {
		u.UUID = shortuuid.New()
	}

	var userId uint
	err = tx.QueryRow(ctx, `INSERT INTO users (username, uuid, email, password, internal_bot, notoriety, verified, verification_token, verification_expires_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()) RETURNING id`,
		u.Username, u.UUID, u.Email, u.Password, u.IsBot, u.Notoriety, u.Verified, u.VerificationToken, u.VerificationExpiresAt).Scan(&userId)
	if err != nil {
		return err
	}

	prof := u.Profile
	if prof == nil {
		prof = &entity.Profile{}
	}

	_, err = tx.Exec(ctx, `INSERT INTO profiles (user_id, first_name, last_name, country_code, title, about, ratings, stats, avatar_url, birth_date, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())`,
		userId, prof.FirstName, prof.LastName, prof.CountryCode, prof.Title, prof.About, prof.Ratings, prof.Stats, prof.AvatarUrl, prof.BirthDate)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// SetPassword sets the password for the user. The password is already hashed.
func (s *DBStore) SetPassword(ctx context.Context, uuid string, hashpass string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = common.Update(ctx, tx, []string{"password"}, []interface{}{hashpass}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: uuid})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// SetAvatarUrl sets the avatar_url (profile field) for the user.
func (s *DBStore) SetAvatarUrl(ctx context.Context, uuid string, avatarUrl string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := common.GetUserDBIDFromUUID(ctx, tx, uuid)
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"avatar_url"}, []interface{}{avatarUrl}, &common.CommonDBConfig{TableType: common.ProfilesTable, SelectByType: common.SelectByUserID, Value: id})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) GetBriefProfiles(ctx context.Context, uuids []string) (map[string]*pb.BriefProfile, error) {
	ctx, span := tracer.Start(ctx, "backing.GetBriefProfiles")
	defer span.End()

	profiles, err := s.queries.GetBriefProfiles(ctx, uuids)
	if err != nil {
		return nil, err
	}

	response := make(map[string]*pb.BriefProfile)
	now := time.Now()

	for pi := range profiles {
		username := profiles[pi].Username.String
		avatarUrl := profiles[pi].AvatarUrl.String
		if avatarUrl == "" && profiles[pi].InternalBot.Bool {
			// see entity/user.go
			log.Debug().Str("username", username).Msg("using-default-bot-avatar")
			avatarUrl = "https://woogles-prod-assets.s3.amazonaws.com/macondog.png"
		}
		subjectIsAdult := entity.IsAdult(profiles[pi].BirthDate.String, now)
		censoredAvatarUrl := ""
		censoredFullName := ""
		if subjectIsAdult {
			censoredAvatarUrl = avatarUrl
			// see entity/user.go RealName()
			if profiles[pi].FirstName.String != "" {
				if profiles[pi].LastName.String != "" {
					censoredFullName = profiles[pi].FirstName.String + " " + profiles[pi].LastName.String
				} else {
					censoredFullName = profiles[pi].FirstName.String
				}
			} else {
				censoredFullName = profiles[pi].LastName.String
			}
		}
		response[profiles[pi].Uuid.String] = &pb.BriefProfile{
			Username:    username,
			CountryCode: profiles[pi].CountryCode.String,
			AvatarUrl:   censoredAvatarUrl,
			FullName:    censoredFullName,
			BadgeCodes:  profiles[pi].BadgeCodes,
		}
	}

	return response, nil
}

func (s *DBStore) SetPersonalInfo(ctx context.Context, uuid string, email string, firstName string, lastName string, birthDate string, countryCode string, about string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := common.GetUserDBIDFromUUID(ctx, tx, uuid)
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"first_name", "last_name", "birth_date", "country_code", "about"}, []interface{}{firstName, lastName, birthDate, countryCode, about}, &common.CommonDBConfig{TableType: common.ProfilesTable, SelectByType: common.SelectByUserID, Value: id})
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"email"}, []interface{}{email}, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUUID, Value: uuid})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// SetRatings set the specific ratings for the given variant in a transaction.
func (s *DBStore) SetRatings(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
	p0Rating *entity.SingleRating, p1Rating *entity.SingleRating) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	p0id, err := common.GetUserDBIDFromUUID(ctx, tx, p0uuid)
	if err != nil {
		return err
	}

	p1id, err := common.GetUserDBIDFromUUID(ctx, tx, p1uuid)
	if err != nil {
		return err
	}

	err = common.UpdateUserRating(ctx, tx, p0id, variant, p0Rating)
	if err != nil {
		return err
	}

	err = common.UpdateUserRating(ctx, tx, p1id, variant, p1Rating)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) SetStats(ctx context.Context, p0uuid string, p1uuid string, variant entity.VariantKey,
	p0Stats *entity.Stats, p1Stats *entity.Stats) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	p0id, err := common.GetUserDBIDFromUUID(ctx, tx, p0uuid)
	if err != nil {
		return err
	}

	p1id, err := common.GetUserDBIDFromUUID(ctx, tx, p1uuid)
	if err != nil {
		return err
	}

	err = common.UpdateUserStats(ctx, tx, p0id, variant, p0Stats)
	if err != nil {
		return err
	}

	err = common.UpdateUserStats(ctx, tx, p1id, variant, p1Stats)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) GetBot(ctx context.Context, botType macondopb.BotRequest_BotCode) (*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	username := botNames[botType]

	var botUsername string
	err = tx.QueryRow(ctx, `SELECT username FROM users WHERE internal_bot IS TRUE AND lower(username) = lower($1)`, username).Scan(&botUsername)
	if err == pgx.ErrNoRows {
		// Just pick any random bot. This should not be done on prod.
		log.Warn().Msg("picking-random-bot")
		err = tx.QueryRow(ctx, `SELECT username FROM users WHERE internal_bot IS TRUE ORDER BY RANDOM()`).Scan(&botUsername)
		if err == pgx.ErrNoRows {
			return nil, errors.New("no bots found")
		}
	}

	entu, err := common.GetUserBy(ctx, tx, &common.CommonDBConfig{TableType: common.UsersTable, SelectByType: common.SelectByUsername, Value: strings.ToLower(botUsername), IncludeProfile: true})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entu, nil
}

// AddFollower creates a follower -> target follow.
func (s *DBStore) AddFollower(ctx context.Context, targetUser, follower uint) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO followings (user_id, follower_id) VALUES ($1, $2)`, targetUser, follower)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// RemoveFollower removes a follower -> target follow.
func (s *DBStore) RemoveFollower(ctx context.Context, targetUser, follower uint) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM followings WHERE user_id = $1 AND follower_id = $2`, targetUser, follower)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetFollows gets all the users that the passed-in user DB ID is following.
func (s *DBStore) GetFollows(ctx context.Context, uid uint) ([]*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT u0.uuid, u0.username FROM followings JOIN users AS u0 ON u0.id = user_id WHERE follower_id = $1`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var UUID string
	var username string
	entUsers := []*entity.User{}
	for rows.Next() {
		if err := rows.Scan(&UUID, &username); err != nil {
			return nil, err
		}
		entUsers = append(entUsers, &entity.User{UUID: UUID, Username: username})
	}
	return entUsers, nil
}

// GetFollowedBy gets all the users that are following the passed-in user DB ID.
func (s *DBStore) GetFollowedBy(ctx context.Context, uid uint) ([]*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT u0.uuid, u0.username FROM followings JOIN users AS u0 ON u0.id = follower_id WHERE user_id = $1`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var UUID string
	var username string
	entUsers := []*entity.User{}
	for rows.Next() {
		if err := rows.Scan(&UUID, &username); err != nil {
			return nil, err
		}
		entUsers = append(entUsers, &entity.User{UUID: UUID, Username: username})
	}
	return entUsers, nil
}

// IsFollowing checks if followerID is following userID.
func (s *DBStore) IsFollowing(ctx context.Context, followerID, userID uint) (bool, error) {
	var exists bool
	err := s.dbPool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM followings WHERE follower_id = $1 AND user_id = $2)`,
		followerID, userID).Scan(&exists)
	return exists, err
}

func (s *DBStore) AddBlock(ctx context.Context, targetUser, blocker uint) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO blockings (user_id, blocker_id) VALUES ($1, $2)`, targetUser, blocker)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) RemoveBlock(ctx context.Context, targetUser, blocker uint) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx, `DELETE FROM blockings WHERE user_id = $1 AND blocker_id = $2`, targetUser, blocker)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("block does not exist")
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetBlocks gets all the users that the passed-in user DB ID is blocking.
func (s *DBStore) GetBlocks(ctx context.Context, uid uint) ([]*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT u0.uuid, u0.username FROM blockings JOIN users AS u0 ON u0.id = user_id WHERE blocker_id = $1`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var UUID string
	var username string
	entUsers := []*entity.User{}
	for rows.Next() {
		if err := rows.Scan(&UUID, &username); err != nil {
			return nil, err
		}
		entUsers = append(entUsers, &entity.User{UUID: UUID, Username: username})
	}
	return entUsers, nil
}

// GetBlockedBy gets all the users that are blocking the passed-in user DB ID.
func (s *DBStore) GetBlockedBy(ctx context.Context, uid uint) ([]*entity.User, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT u0.uuid, u0.username FROM blockings JOIN users AS u0 ON u0.id = blocker_id WHERE user_id = $1`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var UUID string
	var username string
	entUsers := []*entity.User{}
	for rows.Next() {
		if err := rows.Scan(&UUID, &username); err != nil {
			return nil, err
		}
		entUsers = append(entUsers, &entity.User{UUID: UUID, Username: username})
	}
	return entUsers, nil
}

// GetFullBlocks gets users uid is blocking AND users blocking uid
func (s *DBStore) GetFullBlocks(ctx context.Context, uid uint) ([]*entity.User, error) {
	// There's probably a way to do this with one db query but eh.
	players := map[string]*entity.User{}

	blocks, err := s.GetBlocks(ctx, uid)
	if err != nil {
		return nil, err
	}
	blockedby, err := s.GetBlockedBy(ctx, uid)
	if err != nil {
		return nil, err
	}

	for _, u := range blocks {
		players[u.UUID] = u
	}
	for _, u := range blockedby {
		players[u.UUID] = u
	}

	plist := make([]*entity.User, len(players))
	idx := 0
	for _, v := range players {
		plist[idx] = v
		idx++
	}
	return plist, nil
}

func (s *DBStore) UsersByPrefix(ctx context.Context, prefix string) ([]*pb.BasicUser, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT username, uuid FROM users
	WHERE substr(lower(users.username), 1, length($1)) = $1
	AND users.internal_bot IS FALSE
	AND NOT EXISTS(
		SELECT 1 FROM user_actions
		WHERE user_actions.user_id = users.id AND
		user_actions.removed_time IS NULL AND
		user_actions.end_time IS NULL AND
		user_actions.action_type = $2
	)`, strings.ToLower(prefix), ms.ModActionType_SUSPEND_ACCOUNT)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var UUID string
	var username string
	users := []*pb.BasicUser{}
	for rows.Next() {
		if err := rows.Scan(&username, &UUID); err != nil {
			return nil, err
		}
		users = append(users, &pb.BasicUser{Uuid: UUID, Username: username})
	}
	return users, nil
}

// List all user IDs.
func (s *DBStore) ListAllIDs(ctx context.Context) ([]string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT uuid FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []string{}
	var UUID string
	for rows.Next() {
		if err := rows.Scan(&UUID); err != nil {
			return nil, err
		}
		users = append(users, UUID)
	}
	return users, nil
}

func (s *DBStore) ResetStats(ctx context.Context, uuid string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := common.GetUserDBIDFromUUID(ctx, tx, uuid)
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"stats"}, []interface{}{&entity.Stats{}}, &common.CommonDBConfig{TableType: common.ProfilesTable, SelectByType: common.SelectByUserID, Value: id})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) ResetRatings(ctx context.Context, uuid string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := common.GetUserDBIDFromUUID(ctx, tx, uuid)
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"ratings"}, []interface{}{&entity.Ratings{}}, &common.CommonDBConfig{TableType: common.ProfilesTable, SelectByType: common.SelectByUserID, Value: id})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) ResetStatsAndRatings(ctx context.Context, uuid string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := common.GetUserDBIDFromUUID(ctx, tx, uuid)
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"stats", "ratings"}, []interface{}{&entity.Stats{}, &entity.Ratings{}}, &common.CommonDBConfig{TableType: common.ProfilesTable, SelectByType: common.SelectByUserID, Value: id})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) ResetPersonalInfo(ctx context.Context, uuid string) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	id, err := common.GetUserDBIDFromUUID(ctx, tx, uuid)
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"first_name", "last_name", "about", "title", "avatar_url", "country_code"}, []interface{}{"", "", "", "", "", ""}, &common.CommonDBConfig{TableType: common.ProfilesTable, SelectByType: common.SelectByUserID, Value: id})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) ResetProfile(ctx context.Context, uid string) error {
	err := s.ResetStatsAndRatings(ctx, uid)
	if err != nil {
		return err
	}
	return s.ResetPersonalInfo(ctx, uid)
}

func (s *DBStore) Count(ctx context.Context) (int64, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var count int64
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return count, nil
}

func (s *DBStore) CachedCount(ctx context.Context) int {
	return 0
}

func (s *DBStore) Username(ctx context.Context, uuid string) (string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var username string
	err = tx.QueryRow(ctx, "SELECT username FROM users WHERE uuid = $1", uuid).Scan(&username)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return username, nil
}

func (s *DBStore) GetAPIKey(ctx context.Context, uuid string) (string, error) {
	var apikey string
	err := s.dbPool.QueryRow(ctx, `SELECT COALESCE((SELECT api_key FROM users where uuid = $1), '')`, uuid).Scan(
		&apikey)
	if err != nil {
		// Ignore errors, but just return a blank API key.
		return "", nil
	}
	return apikey, err
}

func (s *DBStore) ResetAPIKey(ctx context.Context, uuid string) (string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	apikey := shortuuid.New() + shortuuid.New()

	result, err := tx.Exec(ctx, `
		UPDATE users SET api_key = $1 WHERE uuid = $2`, apikey, uuid)
	if result.RowsAffected() != 1 {
		return "", fmt.Errorf("unable to reset api key")
	}
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return apikey, nil
}

func scanRowsIntoModActions(ctx context.Context, tx pgx.Tx, rows pgx.Rows) ([]*ms.ModAction, []*DBUniqueValues, error) {
	defer rows.Close()
	actions := []*ms.ModAction{}
	dbUniqueValues := []*DBUniqueValues{}
	for rows.Next() {
		var action_dbid int64
		var user_dbid int64
		var action_type int
		var start_time pgtype.Timestamptz
		var end_time pgtype.Timestamptz
		var removed_time pgtype.Timestamptz
		var message_id pgtype.Text
		var applier_dbid pgtype.Int8
		var remover_dbid pgtype.Int8
		var chat_text pgtype.Text
		var note pgtype.Text
		var removal_note pgtype.Text
		var email_type int

		if err := rows.Scan(&action_dbid, &user_dbid, &action_type, &start_time,
			&end_time, &removed_time, &message_id, &applier_dbid,
			&remover_dbid, &note, &removal_note, &chat_text, &email_type); err != nil {
			return nil, nil, err
		}

		var startTime *timestamppb.Timestamp = nil
		if start_time.Valid {
			startTime = timestamppb.New(start_time.Time)
		}

		var endTime *timestamppb.Timestamp = nil
		if end_time.Valid {
			endTime = timestamppb.New(end_time.Time)
		}

		var removedTime *timestamppb.Timestamp = nil
		if removed_time.Valid {
			removedTime = timestamppb.New(removed_time.Time)
		}

		var duration int32 = 0
		if start_time.Valid && end_time.Valid {
			duration = int32(endTime.Seconds) - int32(startTime.Seconds)
		}

		modAction := &ms.ModAction{
			Type:        ms.ModActionType(action_type),
			Duration:    duration,
			StartTime:   startTime,
			EndTime:     endTime,
			RemovedTime: removedTime,
			MessageId:   message_id.String,
			ChatText:    chat_text.String,
			Note:        note.String,
			EmailType:   ms.EmailType(email_type),
		}
		dbUnique := &DBUniqueValues{
			actionDBID:  action_dbid,
			userDBID:    user_dbid,
			removalNote: removal_note.String,
			applierDBID: applier_dbid,
			removerDBID: remover_dbid,
		}

		actions = append(actions, modAction)
		dbUniqueValues = append(dbUniqueValues, dbUnique)
	}
	return actions, dbUniqueValues, nil
}

func getSingleActionDB(ctx context.Context, tx pgx.Tx, userUUID string, actionType ms.ModActionType) (*ms.ModAction, *DBUniqueValues, error) {

	query := fmt.Sprintf(`SELECT %s FROM users JOIN user_actions ON users.id = user_actions.user_id
	WHERE users.uuid = $1 AND user_actions.action_type = $2
	  AND user_actions.removed_time IS NULL AND (user_actions.end_time IS NULL OR user_actions.end_time > NOW())
	ORDER BY start_time DESC LIMIT 1`, userActionsSQLSelection)
	rows, err := tx.Query(ctx, query, userUUID, actionType)
	if err != nil {
		return nil, nil, err
	}

	modActions, dbUniqueValues, err := scanRowsIntoModActions(ctx, tx, rows)

	if len(modActions) != len(dbUniqueValues) {
		return nil, nil, fmt.Errorf("lengths different for user %s, action %s, %d != %d", userUUID, actionType.String(), len(modActions), len(dbUniqueValues))
	}

	if len(modActions) == 0 {
		return nil, nil, nil
	}

	if len(modActions) > 1 {
		return nil, nil, fmt.Errorf("not exactly one action %s found for user %s, found %d", actionType.String(), userUUID, len(modActions))
	}

	err = addUserUUIDsToActions(ctx, tx, modActions, dbUniqueValues)
	if err != nil {
		return nil, nil, err
	}
	return modActions[0], dbUniqueValues[0], nil
}

func getActionsDB(ctx context.Context, tx pgx.Tx, userUUID string) (map[string]*ms.ModAction, map[string]*DBUniqueValues, error) {
	// Only get current actions that are in effect.
	query := fmt.Sprintf(`SELECT DISTINCT ON (action_type) %s
	FROM users JOIN user_actions ON users.id = user_actions.user_id
	WHERE users.uuid = $1 AND user_actions.removed_time IS NULL AND (user_actions.end_time IS NULL OR user_actions.end_time > NOW())
	ORDER BY action_type, start_time DESC`, userActionsSQLSelection)
	rows, err := tx.Query(ctx, query, userUUID)
	if err != nil {
		return nil, nil, err
	}

	modActions, dbUniqueValues, err := scanRowsIntoModActions(ctx, tx, rows)
	if err != nil {
		return nil, nil, err
	}

	err = addUserUUIDsToActions(ctx, tx, modActions, dbUniqueValues)
	if err != nil {
		return nil, nil, err
	}

	modActionsMap := map[string]*ms.ModAction{}
	dbUniqueValuesMap := map[string]*DBUniqueValues{}

	for idx, modAction := range modActions {
		_, exists := modActionsMap[modAction.Type.String()]
		if exists {
			return nil, nil, fmt.Errorf("mod action %s already exists for user %s", modAction.Type.String(), userUUID)
		}
		modActionsMap[modAction.Type.String()] = modAction
		dbUniqueValuesMap[modAction.Type.String()] = dbUniqueValues[idx]
	}
	return modActionsMap, dbUniqueValuesMap, nil
}

func ApplySingleActionDB(ctx context.Context, tx pgx.Tx, userDBID int64, applierDBID pgtype.Int8, removerDBID pgtype.Int8, action *ms.ModAction) error {
	var endTime pgtype.Timestamptz
	endTime.Valid = false
	if action.EndTime != nil {
		endTime.Time = action.EndTime.AsTime()
		endTime.Valid = true
	}

	_, err := tx.Exec(ctx, `INSERT INTO user_actions
	(user_id, action_type, start_time, end_time, message_id, applier_id, remover_id, chat_text, note, email_type)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		userDBID, action.Type, action.StartTime.AsTime(), endTime,
		action.MessageId, applierDBID, removerDBID, action.ChatText, action.Note, action.EmailType)
	return err
}

func updateActionForRemoval(ctx context.Context, tx pgx.Tx, removedActionDBID int64, removerDBID pgtype.Int8, removalNote string) error {
	result, err := tx.Exec(ctx, `UPDATE user_actions SET removed_time = NOW(), remover_id = $1, removal_note = $2
	WHERE id = $3`, removerDBID, removalNote, removedActionDBID)
	if result.RowsAffected() != 1 {
		return fmt.Errorf("failed to update action %d", removedActionDBID)
	}
	return err
}

func addUserUUID(ctx context.Context, tx pgx.Tx, userUUID string, userUUIDtoDBID map[string]int64) error {
	_, exists := userUUIDtoDBID[userUUID]
	if exists {
		return nil
	}
	userDBID, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return err
	}
	userUUIDtoDBID[userUUID] = userDBID
	return nil
}

func addUserDBID(ctx context.Context, tx pgx.Tx, userDBID int64, userDBIDtoUUID map[int64]string) error {
	_, exists := userDBIDtoUUID[userDBID]
	if exists {
		return nil
	}
	userUUID, err := common.GetUserUUIDFromDBID(ctx, tx, userDBID)
	if err != nil {
		return err
	}
	userDBIDtoUUID[userDBID] = userUUID
	return nil
}

func getUserDBIDsFromActions(ctx context.Context, tx pgx.Tx, actions []*ms.ModAction) (map[string]int64, error) {
	userUUIDtoDBID := map[string]int64{}
	for _, action := range actions {
		if action.UserId == "" {
			return nil, fmt.Errorf("user id missing for action %s", action.Type.String())
		}
		err := addUserUUID(ctx, tx, action.UserId, userUUIDtoDBID)
		if err != nil {
			return nil, err
		}
		if action.ApplierUserId != "" {
			err = addUserUUID(ctx, tx, action.ApplierUserId, userUUIDtoDBID)
			if err != nil {
				return nil, err
			}
		}
	}
	return userUUIDtoDBID, nil
}

func addUserUUIDsToActions(ctx context.Context, tx pgx.Tx, actions []*ms.ModAction, dbVals []*DBUniqueValues) error {
	userDBIDtoUUID := map[int64]string{}
	for idx, action := range actions {
		err := addUserDBID(ctx, tx, dbVals[idx].userDBID, userDBIDtoUUID)
		if err != nil {
			return err
		}

		action.UserId = userDBIDtoUUID[dbVals[idx].userDBID]

		if dbVals[idx].applierDBID.Valid {
			err = addUserDBID(ctx, tx, dbVals[idx].applierDBID.Int64, userDBIDtoUUID)
			if err != nil {
				return err
			}
			action.ApplierUserId = userDBIDtoUUID[dbVals[idx].applierDBID.Int64]
		}

		if dbVals[idx].removerDBID.Valid {
			err = addUserDBID(ctx, tx, dbVals[idx].removerDBID.Int64, userDBIDtoUUID)
			if err != nil {
				return err
			}
			action.RemoverUserId = userDBIDtoUUID[dbVals[idx].removerDBID.Int64]
		}
	}
	return nil
}

func (s *DBStore) GetActions(ctx context.Context, userUUID string) (map[string]*ms.ModAction, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	actions, _, err := getActionsDB(ctx, tx, userUUID)

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return actions, nil
}

// GetActionsBatch fetches actions for multiple users in a single query
// Returns map[userUUID]map[actionType]*ModAction
func (s *DBStore) GetActionsBatch(ctx context.Context, userUUIDs []string) (map[string]map[string]*ms.ModAction, error) {
	if len(userUUIDs) == 0 {
		return make(map[string]map[string]*ms.ModAction), nil
	}

	rows, err := s.queries.GetActionsBatch(ctx, userUUIDs)
	if err != nil {
		return nil, err
	}

	// Group actions by user UUID, then by action type
	result := make(map[string]map[string]*ms.ModAction)

	for _, row := range rows {
		// Convert pgtype.Text to string
		if !row.UserUuid.Valid {
			continue
		}
		userUUID := row.UserUuid.String

		// Initialize map for this user if needed
		if _, exists := result[userUUID]; !exists {
			result[userUUID] = make(map[string]*ms.ModAction)
		}

		// Convert row to ModAction
		action := &ms.ModAction{
			Type: ms.ModActionType(row.ActionType),
		}

		if row.StartTime.Valid {
			action.StartTime = timestamppb.New(row.StartTime.Time)
		}
		if row.EndTime.Valid {
			action.EndTime = timestamppb.New(row.EndTime.Time)
		}
		if row.Note.Valid {
			action.Note = row.Note.String
		}
		if row.ChatText.Valid {
			action.ChatText = row.ChatText.String
		}
		// EmailType is not nullable in the query, use it directly
		action.EmailType = ms.EmailType(row.EmailType)

		result[userUUID][action.Type.String()] = action
	}

	return result, nil
}

func (s *DBStore) GetActionHistory(ctx context.Context, userUUID string) ([]*ms.ModAction, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	userDBID, err := common.GetUserDBIDFromUUID(ctx, tx, userUUID)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT %s	FROM user_actions
	WHERE user_actions.user_id = $1 AND (user_actions.removed_time IS NOT NULL OR (user_actions.end_time IS NOT NULL AND user_actions.end_time < NOW()))
	ORDER BY start_time ASC`, userActionsSQLSelection)
	rows, err := tx.Query(ctx, query, userDBID)
	// query := fmt.Sprintf(`SELECT %s	FROM user_actions`, userActionsSQLSelection)
	// rows, err := tx.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	modActions, dbUniqueValues, err := scanRowsIntoModActions(ctx, tx, rows)

	err = addUserUUIDsToActions(ctx, tx, modActions, dbUniqueValues)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return modActions, nil
}

func applyOrRemoveActionsDB(ctx context.Context, s *DBStore, actions []*ms.ModAction, apply bool) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	userDBIDs, err := getUserDBIDsFromActions(ctx, tx, actions)
	if err != nil {
		return err
	}

	for _, action := range actions {
		currentAction, dbUniqueValues, err := getSingleActionDB(ctx, tx, action.UserId, action.Type)
		if err != nil {
			return err
		}

		userDBID, exists := userDBIDs[action.UserId]
		if !exists {
			return fmt.Errorf("DBID not found for user: %s", action.UserId)
		}

		var nullOrAppliererDBID pgtype.Int8
		nullOrAppliererDBID.Valid = false
		if action.ApplierUserId != "" {
			removerDBID, exists := userDBIDs[action.ApplierUserId]
			if !exists {
				return fmt.Errorf("DBID not found for applier user: %s", action.UserId)
			}
			nullOrAppliererDBID.Int64 = removerDBID
			nullOrAppliererDBID.Valid = true
		}

		if currentAction != nil {
			note := action.Note
			if apply {
				// This action is being supplanted by a more recent action
				// so the note is only relevant to the new action. Create
				// an automatic removal note in that case
				note = "REMOVED BY NEW ACTION: " + note
			}

			err = updateActionForRemoval(ctx, tx, dbUniqueValues.actionDBID, nullOrAppliererDBID, note)
			if err != nil {
				return err
			}
		}

		if apply {
			err := ApplySingleActionDB(ctx, tx, userDBID, nullOrAppliererDBID, pgtype.Int8{Valid: false}, action)
			if err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *DBStore) ApplyActions(ctx context.Context, actions []*ms.ModAction) error {
	return applyOrRemoveActionsDB(ctx, s, actions, true)
}

func (s *DBStore) RemoveActions(ctx context.Context, actions []*ms.ModAction) error {
	return applyOrRemoveActionsDB(ctx, s, actions, false)
}
