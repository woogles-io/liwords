package rerate

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// ExtractOptions control how much is read per query.
type ExtractOptions struct {
	BatchSize int64         // game ids per query
	Pause     time.Duration // between queries, to keep the load light
}

// gameRow is one finished game as read.
type gameRow struct {
	id             int64
	end            int64
	reason, winner int32
	request        string
	scores         []int32
	uuid0, uuid1   string
	qdBot0, qdBot1 bool
}

// Extract reads every finished rated game, the users, moderator rating
// resets and current ratings. With prev, it only reads games that ended
// since prev was read (new game ids, and games that were unfinished then).
func Extract(ctx context.Context, pool *pgxpool.Pool, prev *Data, opts ExtractOptions) (*Data, error) {
	if opts.BatchSize <= 0 {
		opts.BatchSize = 20000
	}
	d := &Data{Skipped: map[string]int{}, SkippedIDs: map[string][]int64{}}
	if prev != nil {
		d.Games = prev.Games
		d.Keys = prev.Keys
		for k, v := range prev.Skipped {
			d.Skipped[k] = v
		}
		for k, v := range prev.SkippedIDs {
			d.SkippedIDs[k] = v
		}
	}
	d.ReadAt = time.Now().Unix()

	var err error
	if d.Users, err = readUsers(ctx, pool); err != nil {
		return nil, fmt.Errorf("users: %w", err)
	}
	byUUID := make(map[string]int64, len(d.Users))
	for id, u := range d.Users {
		byUUID[u.UUID] = id
	}
	if d.Resets, err = readResets(ctx, pool); err != nil {
		return nil, fmt.Errorf("resets: %w", err)
	}

	var maxID int64
	if err := pool.QueryRow(ctx, `SELECT coalesce(max(id), 0) FROM games`).Scan(&maxID); err != nil {
		return nil, err
	}
	keyIdx := d.keyIndexMap()
	// Each query returns finished games and the ids of unfinished ones
	// together, so a game that ends meanwhile is in one list or the other.
	add := func(rows []gameRow, unfinished []int64) {
		for _, r := range rows {
			d.addGame(r, byUUID, keyIdx)
		}
		d.Unfinished = append(d.Unfinished, unfinished...)
	}

	// Games that were unfinished last time: read the ones that have ended.
	if prev != nil && len(prev.Unfinished) > 0 {
		rows, unf, err := queryGames(ctx, pool, `id = ANY($1)`, prev.Unfinished)
		if err != nil {
			return nil, fmt.Errorf("unfinished: %w", err)
		}
		add(rows, unf)
	}
	from := int64(1)
	if prev != nil {
		from = prev.MaxID + 1
	}
	start := time.Now()
	for lo := from; lo <= maxID; lo += opts.BatchSize {
		hi := min(lo+opts.BatchSize-1, maxID)
		rows, unf, err := queryGames(ctx, pool, `id BETWEEN $1 AND $2`, lo, hi)
		if err != nil {
			return nil, fmt.Errorf("games %d-%d: %w", lo, hi, err)
		}
		add(rows, unf)
		log.Info().Int64("through", hi).Int64("max", maxID).Int("games", len(d.Games)).
			Dur("elapsed", time.Since(start)).Msg("read-games")
		if opts.Pause > 0 {
			time.Sleep(opts.Pause)
		}
	}
	d.MaxID = maxID

	// Read every time: a penalty can come after the game was read.
	if d.Penalties, err = readPenalties(ctx, pool); err != nil {
		return nil, fmt.Errorf("penalties: %w", err)
	}
	if d.Current, err = readCurrentRatings(ctx, pool); err != nil {
		return nil, fmt.Errorf("current ratings: %w", err)
	}
	return d, nil
}

func (d *Data) addGame(r gameRow, byUUID map[string]int64, keyIdx map[entity.VariantKey]uint16) {
	req := &pb.GameRequest{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal([]byte(r.request), req); err != nil {
		d.skip("unreadable request", r.id)
		return
	}
	if req.RatingMode != pb.RatingMode_RATED {
		return
	}
	timefmt, variant, err := entity.VariantFromGameReq(req)
	if err != nil {
		d.skip("no rating key", r.id)
		return
	}
	p0, ok0 := byUUID[r.uuid0]
	p1, ok1 := byUUID[r.uuid1]
	if !ok0 || !ok1 || len(r.scores) != 2 {
		d.skip("players or scores missing", r.id)
		return
	}
	d.Games = append(d.Games, Game{
		ID:           r.id,
		End:          r.end,
		Key:          d.keyIndex(entity.ToVariantKey(req.Lexicon, variant, timefmt), keyIdx),
		Players:      [2]int64{p0, p1},
		QuickdataBot: [2]bool{r.qdBot0, r.qdBot1},
		Reason:       pb.GameEndReason(r.reason),
		Winner:       int8(r.winner),
		Scores:       [2]int32{r.scores[0], r.scores[1]},
	})
}

// queryGames reads the games matching where: finished ones (not aborted or
// cancelled), and the ids of those not finished yet.
func queryGames(ctx context.Context, pool *pgxpool.Pool, where string, args ...any) ([]gameRow, []int64, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, coalesce(extract(epoch FROM coalesce(updated_at, created_at))::bigint, 0), game_end_reason,
		       coalesce(winner_idx, -1), game_request::text,
		       coalesce(quickdata->'s', 'null'::jsonb)::text,
		       coalesce(quickdata->'pi'->0->>'user_id', ''),
		       coalesce(quickdata->'pi'->1->>'user_id', ''),
		       coalesce((quickdata->'pi'->0->>'is_bot')::bool, false),
		       coalesce((quickdata->'pi'->1->>'is_bot')::bool, false)
		FROM games
		WHERE `+where+` AND game_end_reason NOT IN (5, 7) AND game_request IS NOT NULL`, args...)
	if err != nil {
		return nil, nil, err
	}
	var out []gameRow
	var unfinished []int64
	for rows.Next() {
		var r gameRow
		var scores string
		if err := rows.Scan(&r.id, &r.end, &r.reason, &r.winner, &r.request, &scores,
			&r.uuid0, &r.uuid1, &r.qdBot0, &r.qdBot1); err != nil {
			return nil, nil, err
		}
		if r.reason == 0 {
			unfinished = append(unfinished, r.id)
			continue
		}
		_ = json.Unmarshal([]byte(scores), &r.scores)
		out = append(out, r)
	}
	return out, unfinished, rows.Err()
}

func (d *Data) skip(reason string, id int64) {
	d.Skipped[reason]++
	d.SkippedIDs[reason] = append(d.SkippedIDs[reason], id)
}

// readPenalties finds league games whose result was cancelled for cheating:
// the cheater's score in game_players no longer matches the game's own
// record, which the penalty leaves alone.
func readPenalties(ctx context.Context, pool *pgxpool.Pool) (map[int64]Penalty, error) {
	rows, err := pool.Query(ctx, `
		SELECT g.id, gp.player_index, gp.score, gp.opponent_score,
		       coalesce((g.timers->>'lu')::bigint / 1000, 0)
		FROM game_players gp JOIN games g ON g.uuid = gp.game_uuid
		WHERE gp.league_season_id IS NOT NULL AND gp.won = false
		  AND g.game_end_reason NOT IN (0, 5, 7)
		  AND gp.score <> (g.quickdata->'s'->>(gp.player_index::int))::int`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]Penalty{}
	for rows.Next() {
		var id, end int64
		var idx int16
		var score, opp int32
		if err := rows.Scan(&id, &idx, &score, &opp, &end); err != nil {
			return nil, err
		}
		if idx != 0 && idx != 1 {
			continue
		}
		p := Penalty{Loser: int8(idx), End: end}
		p.Scores[idx], p.Scores[1-idx] = score, opp
		out[id] = p
	}
	return out, rows.Err()
}

func readUsers(ctx context.Context, pool *pgxpool.Pool) (map[int64]User, error) {
	rows, err := pool.Query(ctx, `SELECT id, uuid, username, coalesce(internal_bot, false) FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]User{}
	for rows.Next() {
		var id int64
		var u User
		if err := rows.Scan(&id, &u.UUID, &u.Username, &u.Bot); err != nil {
			return nil, err
		}
		out[id] = u
	}
	return out, rows.Err()
}

// readResets reads moderator actions that cleared a user's ratings
// (RESET_RATINGS and RESET_STATS_AND_RATINGS).
func readResets(ctx context.Context, pool *pgxpool.Pool) ([]Reset, error) {
	rows, err := pool.Query(ctx, `
		SELECT user_id, extract(epoch FROM start_time)::bigint FROM user_actions
		WHERE action_type IN (4, 6) ORDER BY start_time`)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Reset, error) {
		var x Reset
		return x, r.Scan(&x.UserID, &x.Time)
	})
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time < out[j].Time })
	return out, err
}

func readCurrentRatings(ctx context.Context, pool *pgxpool.Pool) (map[int64]map[entity.VariantKey]entity.SingleRating, error) {
	rows, err := pool.Query(ctx, `
		SELECT user_id, ratings FROM profiles
		WHERE ratings->'Data' IS NOT NULL AND ratings->'Data' <> 'null'::jsonb`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]map[entity.VariantKey]entity.SingleRating{}
	for rows.Next() {
		var id int64
		var r entity.Ratings
		if err := rows.Scan(&id, &r); err != nil {
			return nil, err
		}
		if len(r.Data) > 0 {
			out[id] = r.Data
		}
	}
	return out, rows.Err()
}
