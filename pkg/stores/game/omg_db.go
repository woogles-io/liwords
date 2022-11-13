package game

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/stores/common"
	gs "github.com/domino14/liwords/rpc/api/proto/game_service"
	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
	pb "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gaddag"
	macondogame "github.com/domino14/macondo/game"
	macondoipc "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type play struct {
	isBingo             bool
	isUnchallengedPhony bool
	isChallengedPhony   bool
	isChallengedWord    bool
	word                string
	score               int
}

type playerStats struct {
	wordList               []*play
	bingos                 int
	challengedPhonies      int
	unchallengedPhonies    int
	challengedWords        int
	successfulChallenges   int
	unsuccessfulChallenges int
	exchanges              int
	score                  int
	wins                   int
	losses                 int
	draws                  int
	turns                  int
	tilesPlayed            int
	highPlay               int
}

func (s *DBStore) OMGGet(ctx context.Context, uuid string) (*entity.Game, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT
	omgwords.id,
	omgwords.created_at,
	omgwords.timers,
	omgwords.history,
	omgwords.request,
	omgwords.meta_events,
	omgwords.tournament_data,
	omgwords.tournament_id,
	omgwords.started,
	omgwords.game_end_reason,
	omgwords.type,
	omgwords_games_players.player_id,
	omgwords_games_players.won
	omgwords_games_players.player_old_rating,
	omgwords_games_players.player_new_rating,
	users.username,
	users.uuid,
	users.internal_bot,
	profiles.ratings
	FROM omgwords
	INNER JOIN omgwords_histories ON omgwords.id = omgwords_histories.game_id
	INNER JOIN omgwords_games_players ON omgwords.id = omgwords_games_players.game_id
	INNER JOIN users ON users.id = omgwords_games_players.player_id
	INNER JOIN profiles ON users.id = profiles.user_id
	WHERE omgwords.uuid = $1
	ORDER BY omgwords_games_players.first DESC`, uuid)
	if err != nil {
		return nil, err
	}

	var gameDBID uint
	var createdAt time.Time
	var timers entity.Timers
	var history macondoipc.GameHistory
	var request ipc.GameRequest
	var metaEvents entity.MetaEventData
	var tournamentData entity.TournamentData
	var tournamentID string
	var started bool
	var gameEndReason int
	var gameType int

	var p0id uint
	var p0won bool
	var p0OldRating float64
	var p0NewRating float64
	var p0Nickname string
	var p0UserId string
	var p0IsBot bool
	var p0Rating entity.Ratings

	var p1id uint
	var p1won bool
	var p1OldRating float64
	var p1NewRating float64
	var p1Nickname string
	var p1UserId string
	var p1IsBot bool
	var p1Rating entity.Ratings

	idx := 0
	for rows.Next() {
		if idx == 0 {
			if err := rows.Scan(&gameDBID, &createdAt, &timers, &history, &request, &metaEvents, &tournamentData, &tournamentID, &started, &gameEndReason, &gameType,
				&p0id, &p0won, &p0OldRating, &p0NewRating,
				&p0Nickname, &p0UserId, &p0IsBot, &p0Rating); err != nil {
				rows.Close()
				return nil, err
			}
		} else {
			if err := rows.Scan(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				&p1id, &p1won, &p1OldRating, &p1NewRating,
				&p1Nickname, &p1UserId, &p1IsBot, &p1Rating); err != nil {
				rows.Close()
				return nil, err
			}
		}
		idx++
	}
	rows.Close()

	if p0won && p1won {
		return nil, fmt.Errorf("both players won for game %s", uuid)
	}

	winnerIdx := -1
	loserIdx := -1

	if p0won {
		winnerIdx = 0
		loserIdx = 1
	} else if p1won {
		winnerIdx = 1
		loserIdx = 0
	}

	timefmt, variant, err := entity.VariantFromGameReq(&request)
	if err != nil {
		return nil, err
	}
	variantKey := entity.ToVariantKey(request.Lexicon, variant, timefmt)

	qdata := entity.Quickdata{
		OriginalRequestId: request.OriginalRequestId,
		FinalScores:       history.FinalScores,
		OriginalRatings:   []float64{p0OldRating, p1OldRating},
		NewRatings:        []float64{p0NewRating, p1NewRating},
		PlayerInfo: []*pb.PlayerInfo{
			{Nickname: p0Nickname,
				UserId: p0UserId,
				Rating: entity.RelevantRating(p0Rating, variantKey),
				IsBot:  p0IsBot,
				First:  true},
			{Nickname: p1Nickname,
				UserId: p1UserId,
				Rating: entity.RelevantRating(p1Rating, variantKey),
				IsBot:  p1IsBot,
				First:  false},
		},
	}

	stats := entity.Stats{}

	entGame, err := fromStateOMG(timers, &qdata, started, gameEndReason, p0id, p1id, winnerIdx, loserIdx,
		&request, &history, &stats, &metaEvents, s.gameEventChan, s.cfg, createdAt, ipc.GameType(gameType), gameDBID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entGame, nil
}

// For use in pkg/bus/bus.go:381
func (s *DBStore) GetTournamentIdAndPlayerUserIds(ctx context.Context, uuid string) (string, []string, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return "", nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT
	omgwords.tournament_id,
	users.uuid,
	FROM omgwords
	INNER JOIN omgwords_games_players ON omgwords.id = omgwords_games_players.game_id
	INNER JOIN users ON users.id = omgwords_games_players.player_id
	WHERE omgwords.uuid = $1
	ORDER BY omgwords_games_players.first DESC`, uuid)
	if err != nil {
		return "", nil, err
	}

	var tournamentID string
	var p0UserId string
	var p1UserId string

	idx := 0
	for rows.Next() {
		if idx == 0 {
			if err := rows.Scan(&tournamentID, &p0UserId); err != nil {
				rows.Close()
				return "", nil, err
			}
		} else {
			if err := rows.Scan(nil, &p1UserId); err != nil {
				rows.Close()
				return "", nil, err
			}
		}
		idx++
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return "", nil, err
	}

	return tournamentID, []string{p0UserId, p1UserId}, nil
}

func (s *DBStore) OMGGetMetadata(ctx context.Context, uuid string) (*ipc.GameInfoResponse, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT
	omgwords.request,
	omgwords.tournament_id,
	omgwords.game_end_reason,
	omgwords.created_at,
	omgwords_games_players.won
	omgwords_games_players.player_old_rating,
	omgwords_games_players.player_new_rating,
	users.username,
	users.uuid,
	users.internal_bot
	FROM omgwords
	INNER JOIN omgwords_games_players ON omgwords.id = omgwords_games_players.game_id
	INNER JOIN users ON users.id = omgwords_games_players.player_id
	WHERE omgwords.uuid = $1
	ORDER BY omgwords_games_players.first DESC`, uuid)
	if err != nil {
		return nil, err
	}

	var request ipc.GameRequest
	var tournamentID string
	var gameEndReason int
	var createdAt time.Time

	var p0won bool
	var p0Nickname string
	var p0UserId string
	var p0IsBot bool

	var p1won bool
	var p1Nickname string
	var p1UserId string
	var p1IsBot bool

	idx := 0
	for rows.Next() {
		if idx == 0 {
			if err := rows.Scan(&request, &tournamentID, &gameEndReason, &createdAt,
				&p0won, &p0Nickname, &p0UserId, &p0IsBot); err != nil {
				rows.Close()
				return nil, err
			}
		} else {
			if err := rows.Scan(nil, nil, nil, nil,
				&p1won, &p1Nickname, &p1UserId, &p1IsBot); err != nil {
				rows.Close()
				return nil, err
			}
		}
		idx++
	}
	rows.Close()

	if p0won && p1won {
		return nil, fmt.Errorf("both players won for game %s", uuid)
	}

	playerInfo :=
		[]*pb.PlayerInfo{
			{Nickname: p0Nickname,
				UserId: p0UserId,
				IsBot:  p0IsBot,
				First:  true},
			{Nickname: p1Nickname,
				UserId: p1UserId,
				IsBot:  p1IsBot,
				First:  false},
		}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &pb.GameInfoResponse{
		Players:       playerInfo,
		GameEndReason: pb.GameEndReason(gameEndReason),
		CreatedAt:     timestamppb.New(createdAt),
		TournamentId:  tournamentID,
		GameRequest:   &request}, nil
}

func (s *DBStore) OMGGetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT
	omgwords.uuid AS game_uuid,
	omgwords_games_players.won,
	users.username,
	users.uuid AS player_uuid
	FROM omgwords 
	INNER JOIN omgwords_games_players ON omgwords.id = omgwords_games_players.game_id
	INNER JOIN users ON omgwords_games_players.player_id = users.id
	WHERE request->>'original_request_id' = $1 AND game_end_reason not in ($2, $3, $4) ORDER BY omgwords.created_at, username DESC`,
		originalRequestId, pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED)
	if err == pgx.ErrNoRows {
		return &gs.StreakInfoResponse{Streak: make([]*gs.StreakInfoResponse_SingleGameInfo, 0)}, nil
	}
	if err != nil {
		return nil, err
	}

	resp := &gs.StreakInfoResponse{
		Streak:      []*gs.StreakInfoResponse_SingleGameInfo{},
		PlayersInfo: []*gs.StreakInfoResponse_PlayerInfo{},
	}

	playerWonArr := [2]bool{false, false}

	idx := 0
	for rows.Next() {
		var gameUUID string
		var playerWon bool
		var playerUsername string
		var playerUUID string
		if idx < 2 {
			if err = rows.Scan(&gameUUID, &playerWon, &playerUsername, &playerUUID); err != nil {
				rows.Close()
				return nil, err
			}
			resp.PlayersInfo = append(resp.PlayersInfo, &gs.StreakInfoResponse_PlayerInfo{
				Nickname: playerUsername,
				Uuid:     playerUUID,
			})
		} else {
			if err = rows.Scan(&gameUUID, &playerWon, nil, nil, nil); err != nil {
				rows.Close()
				return nil, err
			}
		}

		playerWonArr[idx%2] = playerWon

		if idx%2 == 1 {
			if playerWonArr[0] && playerWonArr[1] {
				return nil, fmt.Errorf("both players won in game %s for rematch streak %s", gameUUID, originalRequestId)
			}
			// Assume a tie
			winner := -1
			if playerWonArr[0] {
				// First player in the game pairs won
				winner = 0
			} else if playerWonArr[1] {
				// Second player in the game pairs won
				winner = 1
			}
			resp.Streak = append(resp.Streak, &gs.StreakInfoResponse_SingleGameInfo{
				GameId: gameUUID,
				Winner: int32(winner),
			})
		}
		idx++
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *DBStore) OMGGetRecentGames(ctx context.Context, username string, numGames int, offset int) (*ipc.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, fmt.Errorf("too many games requested: %d", numGames)
	}
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT
	omgwords.uuid,
	omgwords.created_at,
	omgwords.request,
	omgwords.tournament_id,
	omgwords.game_end_reason,
	omgwords_games_players.won
	omgwords_games_players.score
	users.username,
	users.uuid,
	users.internal_bot,
	profiles.ratings
	FROM omgwords
	INNER JOIN omgwords_games_players ON omgwords.id = omgwords_games_players.game_id
	INNER JOIN users ON users.id = omgwords_games_players.player_id
	INNER JOIN profiles ON users.id = profiles.user_id
	WHERE users.username = $1
	ORDER BY omgwords.created_at, omgwords_games_players.first DESC limit $2 offset $3`, username, numGames, offset)
	if err != nil {
		return nil, err
	}

	var gameUUID string
	var createdAt time.Time
	var request ipc.GameRequest
	var tournamentID string
	var gameEndReason int

	var p0Won bool
	var p0Score int
	var p0Username string
	var p0UUID string
	var p0IsBot bool
	var p0Ratings entity.Ratings

	var p1Won bool
	var p1Score int
	var p1Username string
	var p1UUID string
	var p1IsBot bool
	var p1Ratings entity.Ratings

	responses := []*pb.GameInfoResponse{}

	idx := 0
	for rows.Next() {
		if idx%2 == 0 {
			if err := rows.Scan(&gameUUID, &createdAt, &request, &tournamentID, &gameEndReason,
				&p0Won, &p0Score, &p0Username, &p0UUID, &p0IsBot, &p0Ratings); err != nil {
				rows.Close()
				return nil, err
			}
		} else {
			if err := rows.Scan(nil, nil, nil, nil, nil,
				&p1Won, &p1Score, &p1Username, &p1UUID, &p1IsBot, &p1Ratings); err != nil {
				rows.Close()
				return nil, err
			}
		}
		if p0Won && p1Won {
			return nil, fmt.Errorf("both players won for game %s", gameUUID)
		}

		winnerIdx := -1

		if p0Won {
			winnerIdx = 0
		} else if p1Won {
			winnerIdx = 1
		}

		timefmt, variant, err := entity.VariantFromGameReq(&request)
		if err != nil {
			return nil, err
		}
		variantKey := entity.ToVariantKey(request.Lexicon, variant, timefmt)

		playerInfo := []*pb.PlayerInfo{
			{Nickname: p0Username,
				UserId: p0UUID,
				Rating: entity.RelevantRating(p0Ratings, variantKey),
				IsBot:  p0IsBot,
				First:  true},
			{Nickname: p1Username,
				UserId: p1UUID,
				Rating: entity.RelevantRating(p1Ratings, variantKey),
				IsBot:  p1IsBot,
				First:  false},
		}

		responses = append(responses, &pb.GameInfoResponse{
			Players:         playerInfo,
			TimeControlName: string(timefmt),
			TournamentId:    tournamentID,
			GameEndReason:   pb.GameEndReason(gameEndReason),
			CreatedAt:       timestamppb.New(createdAt),
			Winner:          int32(winnerIdx),
			Scores:          []int32{int32(p0Score), int32(p1Score)},
			GameId:          gameUUID,
			GameRequest:     &request})
		idx++
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

func (s *DBStore) OMGGetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*ipc.GameInfoResponses, error) {
	if numGames > MaxRecentGames {
		return nil, fmt.Errorf("too many games requested: %d", numGames)
	}
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `SELECT
	omgwords.uuid,
	omgwords.updated_at,
	omgwords.request,
	omgwords.tournament_id,
	omgwords.game_end_reason,
	omgwords.tournament_data,
	omgwords_games_players.won
	omgwords_games_players.score
	users.username,
	users.uuid,
	users.internal_bot,
	profiles.ratings
	FROM omgwords
	INNER JOIN omgwords_games_players ON omgwords.id = omgwords_games_players.game_id
	INNER JOIN users ON users.id = omgwords_games_players.player_id
	WHERE omgwords.tournament_id = $1
	ORDER BY omgwords.updated_at, omgwords_games_players.first DESC limit $2 offset $3`, tourneyID, numGames, offset)
	if err != nil {
		return nil, err
	}

	var gameUUID string
	var updatedAt time.Time
	var request ipc.GameRequest
	var tournamentID string
	var gameEndReason int
	var tournamentData entity.TournamentData

	var p0Won bool
	var p0Score int
	var p0Username string
	var p0UUID string
	var p0IsBot bool

	var p1Won bool
	var p1Score int
	var p1Username string
	var p1UUID string
	var p1IsBot bool

	responses := []*pb.GameInfoResponse{}

	idx := 0
	for rows.Next() {
		if idx%2 == 0 {
			if err := rows.Scan(&gameUUID, &updatedAt, &request, &tournamentID, &gameEndReason, &tournamentData,
				&p0Won, &p0Score, &p0Username, &p0UUID, &p0IsBot); err != nil {
				rows.Close()
				return nil, err
			}
		} else {
			if err := rows.Scan(nil, nil, nil, nil, nil,
				&p1Won, &p1Score, &p1Username, &p1UUID, &p1IsBot); err != nil {
				rows.Close()
				return nil, err
			}
		}
		if p0Won && p1Won {
			return nil, fmt.Errorf("both players won for game %s", gameUUID)
		}

		winnerIdx := -1

		if p0Won {
			winnerIdx = 0
		} else if p1Won {
			winnerIdx = 1
		}

		timefmt, _, err := entity.VariantFromGameReq(&request)
		if err != nil {
			return nil, err
		}

		playerInfo := []*pb.PlayerInfo{
			{Nickname: p0Username,
				UserId: p0UUID,
				IsBot:  p0IsBot,
				First:  true},
			{Nickname: p1Username,
				UserId: p1UUID,
				IsBot:  p1IsBot,
				First:  false},
		}

		responses = append(responses, &pb.GameInfoResponse{
			Players:             playerInfo,
			TimeControlName:     string(timefmt),
			TournamentId:        tournamentID,
			GameEndReason:       pb.GameEndReason(gameEndReason),
			LastUpdate:          timestamppb.New(updatedAt),
			Winner:              int32(winnerIdx),
			Scores:              []int32{int32(p0Score), int32(p1Score)},
			GameId:              gameUUID,
			TournamentDivision:  tournamentData.Division,
			TournamentRound:     int32(tournamentData.Round),
			TournamentGameIndex: int32(tournamentData.GameIndex),
			GameRequest:         &request})
		idx++
	}
	rows.Close()

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &pb.GameInfoResponses{GameInfo: responses}, nil
}

func (s *DBStore) OMGSet(ctx context.Context, g *entity.Game) error {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	req, err := json.Marshal(g.GameReq)
	if err != nil {
		return err
	}

	tourneyID := ""
	if g.TournamentData != nil {
		tourneyID = g.TournamentData.Id
	}

	gameDBID, err := common.GetOmgwordsDBIDFromUUID(ctx, tx, g.GameID())
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"timers", "request", "meta_events", "tournament_data", "tournament_id", "started", "game_end_reason"},
		[]interface{}{g.Timers, req, g.MetaEvents, g.TournamentData, tourneyID, g.Started, int(g.GameEndReason)},
		&common.CommonDBConfig{TableType: common.OmgwordsTable, SelectByType: common.SelectByUUID, Value: gameDBID, SetUpdatedAt: true})

	if err != nil {
		return err
	}

	hist, err := json.Marshal(g.History())
	if err != nil {
		return err
	}

	err = common.Update(ctx, tx, []string{"history"}, []interface{}{hist}, &common.CommonDBConfig{TableType: common.OmgwordsHistoryTable, SelectByType: common.SelectByGameID, Value: g.GameID()})

	if err != nil {
		return err
	}

	if g.GameEndReason != ipc.GameEndReason_NONE {
		winner := ""
		if g.WinnerIdx != -1 {
			winner = g.History().Players[g.WinnerIdx].UserId
		}
		first := g.History().Players[0].UserId

		for playerIdx, playerInfo := range g.History().Players {
			err = common.Update(ctx, tx, []string{"player_score", "player_old_rating", "player_new_rating", "won", "first"},
				[]interface{}{g.History().FinalScores[playerIdx], g.Quickdata.OriginalRatings[playerIdx], g.Quickdata.NewRatings[playerIdx], playerInfo.UserId == winner, playerInfo.UserId == first},
				&common.CommonDBConfig{TableType: common.OmgwordsGamesTable, SelectByType: common.SelectByGameID, Value: g.GameID()})

			if err != nil {
				return err
			}
		}

		if g.RatingMode() == ipc.RatingMode_RATED {
			// The game has ended set the stats
			// Check if game stats exist, skip if they already exist
			var statsExist bool
			err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM omgwords_stats WHERE game_id = $1)`, gameDBID).Scan(&statsExist)
			if err != nil {
				return err
			}
			if !statsExist {
				// Get stats
				pStats, err := setStats(ctx, tx, g)
				if err != nil {
					return err
				}
				variant, timeControl, err := entity.VariantFromGameReq(g.GameReq)
				if err != nil {
					return err
				}
				lexicon := g.History().Lexicon
				for playerIdx, ps := range pStats {
					playerUUID := g.History().Players[playerIdx].UserId
					playerDBID, err := common.GetUserDBIDFromUUID(ctx, tx, playerUUID)
					if err != nil {
						return err
					}

					var numRows int
					err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM omgwords_player_stats WHERE variant = $1 AND lexicon = $2 AND time_control = $3`, variant, lexicon, timeControl).Scan(&numRows)
					if err != nil {
						return err
					}

					if numRows > 1 {
						return fmt.Errorf("multiple omgwords_player_stats rows (%d) for player %d: %s, %s, %s", numRows, playerDBID, variant, lexicon, timeControl)
					}

					if numRows == 0 {
						oppps := pStats[1-playerIdx]
						_, err = tx.Exec(ctx, `INSERT INTO omgwords_player_stats
					(player_id, variant, lexicon, time_control,
						games, bingos, exchanges,
						challenged_phonies, opp_challenged_phonies,
						unchallenged_phonies, opp_unchallenged_phonies,
						challenged_words, opp_unchallenged_words,
						score, wins, losses, draws, turns, tiles_played,
						high_game_score, high_game_id, high_play_score, high_play_game_id
						VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
							playerDBID, variant, lexicon, timeControl,
							1, ps.bingos, ps.exchanges,
							ps.challengedPhonies, oppps.challengedPhonies,
							ps.unchallengedPhonies, oppps.unchallengedPhonies,
							ps.challengedWords, oppps.challengedWords,
							ps.score, ps.wins, ps.losses, ps.draws, ps.turns, ps.tilesPlayed,
							ps.score, gameDBID, ps.highPlay, gameDBID)
						if err != nil {
							return err
						}
					} else if numRows == 1 {
						oppps := pStats[1-playerIdx]
						_, err = tx.Exec(ctx, `UPDATE omgwords_player_stats SET
					   (games = games + 1,
						bingos = bingos + $1,
						exchanges = exchanges + $2,
						challenged_phonies = challenged_phonies + $3,
						opp_challenged_phonies = opp_challenged_phonies + $4,
						unchallenged_phonies = unchallenged_phonies + $5,
						opp_unchallenged_phonies = opp_unchallenged_phonies + $6,
						challenged_words = challenged_words + $7,
						opp_unchallenged_words = opp_unchallenged_words + $8,
						score = score + $9,
						wins = wins + $10,
						losses = losses + $11,
						draws = draws + $12,
						turns = turns + $13,
						tiles_played = tiles_played + $14,
						high_game_score = CASE WHEN high_game_score < $15 THEN $15 ELSE high_game_score,
						high_game_id = CASE WHEN high_game_score < $15 THEN $16 ELSE high_game_id,
						high_play_score = CASE WHEN high_play_score < $17 THEN $17 ELSE high_play_score,
						high_play_game_id = CASE WHEN high_play_score < $17 THEN $16 ELSE high_play_game_id,
						WHERE player_id = $18 AND variant = $19 AND lexicon = $20 AND time_control = $21`,
							ps.bingos, ps.exchanges,
							ps.challengedPhonies, oppps.challengedPhonies,
							ps.unchallengedPhonies, oppps.unchallengedPhonies,
							ps.challengedWords, oppps.challengedWords,
							ps.score, ps.wins, ps.losses, ps.draws, ps.turns, ps.tilesPlayed,
							ps.score, gameDBID, ps.highPlay,
							playerDBID, variant, lexicon, timeControl)
						if err != nil {
							return err
						}
					} else {
						return fmt.Errorf("invalid number of omgwords_player_stats rows (%d) for player %d: %s, %s, %s", numRows, playerDBID, variant, lexicon, timeControl)
					}

					_, err = tx.Exec(ctx, `INSERT INTO omgwords_stats
				(game_id, player_id, bingo, exchanges, challenged_phonies,
					unchallenged_phonies, challenged_words, successful_challenges,
					unsuccessful_challenges, score, wins, losses,
					draws, turns, tiles_played) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
						gameDBID, playerDBID, ps.bingos, ps.exchanges, ps.challengedPhonies, ps.unchallengedPhonies,
						ps.challengedWords, ps.successfulChallenges, ps.unsuccessfulChallenges, ps.score, ps.wins, ps.losses,
						ps.draws, ps.turns, ps.tilesPlayed)
					if err != nil {
						return err
					}

					for _, wp := range ps.wordList {
						_, err = tx.Exec(ctx, `INSERT INTO omgwords_word_stats
					(game_id, player_id, is_bingo, is_unchallenged_phony,
						is_challenged_phony, is_challenged_word, word, score) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
							gameDBID, playerDBID, wp.isBingo, wp.isUnchallengedPhony, wp.isChallengedPhony,
							wp.isChallengedWord, wp.word, wp.score)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *DBStore) OMGCreate(ctx context.Context, g *entity.Game) error {
	return nil
}

func (s *DBStore) OMGCreateRaw(ctx context.Context, g *entity.Game, gt ipc.GameType) error {
	return nil
}

func (s *DBStore) OMGExists(ctx context.Context, id string) (bool, error) {
	return false, nil
}

func (s *DBStore) OMGListActive(ctx context.Context, tourneyID string, bust bool) (*ipc.GameInfoResponses, error) {
	return nil, nil
}

func (s *DBStore) OMGCount(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *DBStore) OMGCachedCount(ctx context.Context) int {
	return 0
}

func (s *DBStore) OMGGameEventChan() chan<- *entity.EventWrapper {
	return nil
}

func (s *DBStore) OMGSetGameEventChan(c chan<- *entity.EventWrapper) {

}

func (s *DBStore) OMGUnload(ctx context.Context, uuid string) {

}

func (s *DBStore) OMGSetReady(ctx context.Context, gid string, pidx int) (int, error) {
	return 0, nil
}

func (s *DBStore) OMGGetHistory(ctx context.Context, id string) (*macondoipc.GameHistory, error) {
	return nil, nil
}

// fromState returns an entity.Game from a DB State.
func fromStateOMG(timers entity.Timers, qdata *entity.Quickdata, Started bool,
	GameEndReason int, p0id, p1id uint, WinnerIdx, LoserIdx int, req *ipc.GameRequest, hist *macondoipc.GameHistory,
	stats *entity.Stats, mdata *entity.MetaEventData,
	gameEventChan chan<- *entity.EventWrapper, cfg *config.Config, createdAt time.Time, gameType pb.GameType, DBID uint) (*entity.Game, error) {

	g := &entity.Game{
		Started:       Started,
		Timers:        timers,
		GameReq:       req,
		GameEndReason: pb.GameEndReason(GameEndReason),
		WinnerIdx:     WinnerIdx,
		LoserIdx:      LoserIdx,
		ChangeHook:    gameEventChan,
		PlayerDBIDs:   [2]uint{p0id, p1id},
		Stats:         stats,
		MetaEvents:    mdata,
		Quickdata:     qdata,
		CreatedAt:     createdAt,
		Type:          gameType,
		DBID:          DBID,
	}
	g.SetTimerModule(&entity.GameTimer{})

	g.GameReq = req

	lexicon := hist.Lexicon
	if lexicon == "" {
		// This can happen for some early games where we didn't migrate this.
		lexicon = req.Lexicon
	}

	rules, err := macondogame.NewBasicGameRules(
		&cfg.MacondoConfig, lexicon, req.Rules.BoardLayoutName,
		req.Rules.LetterDistributionName, macondogame.CrossScoreOnly,
		macondogame.Variant(req.Rules.VariantName))
	if err != nil {
		return nil, err
	}

	// There's a chance the game is over, so we want to get that state before
	// the following function modifies it.
	histPlayState := hist.GetPlayState()
	log.Debug().Interface("old-play-state", histPlayState).Msg("play-state-loading-hist")
	// This function modifies the history. (XXX it probably shouldn't)
	// It modifies the play state as it plays the game from the beginning.
	mcg, err := macondogame.NewFromHistory(hist, rules, len(hist.Events))
	if err != nil {
		return nil, err
	}
	// XXX: We should probably move this to `NewFromHistory`:
	mcg.SetBackupMode(macondogame.InteractiveGameplayMode)
	// Note: we don't need to set the stack length here, as NewFromHistory
	// above does it.

	g.Game = *mcg
	log.Debug().Interface("history", g.History()).Msg("from-state")
	// Finally, restore the play state from the passed-in history. This
	// might immediately end the game (for example, the game could have timed
	// out, but the NewFromHistory function doesn't actually handle that).
	// We could consider changing NewFromHistory, but we want it to be as
	// flexible as possible for things like analysis mode.
	g.SetPlaying(histPlayState)
	g.History().PlayState = histPlayState
	return g, nil
}

func getGameInfoResponseFromUUID(ctx context.Context, tx pgx.Tx, uuid string) (*pb.GameInfoResponse, error) {

	// Write a more efficient way to do this

	// timefmt, _, err := entity.VariantFromGameReq(game.GameReq)
	// if err != nil {
	// 	return nil, err
	// }

	// var updatedAt time.Time
	// err = tx.QueryRow(ctx, `SELECT updated_at FROM omgwords WHERE uuid = $1`, game.Uid()).Scan(&updatedAt)
	// if err != nil {
	// 	return nil, err
	// }

	// return &pb.GameInfoResponse{
	// 	Players:             game.Quickdata.PlayerInfo,
	// 	GameEndReason:       pb.GameEndReason(game.GameEndReason),
	// 	Scores:              game.Quickdata.FinalScores,
	// 	Winner:              int32(game.WinnerIdx),
	// 	TimeControlName:     string(timefmt),
	// 	CreatedAt:           timestamppb.New(game.CreatedAt),
	// 	LastUpdate:          timestamppb.New(updatedAt),
	// 	GameId:              game.GameID(),
	// 	TournamentId:        game.TournamentData.Id,
	// 	GameRequest:         game.GameReq,
	// 	TournamentDivision:  game.TournamentData.Division,
	// 	TournamentRound:     int32(game.TournamentData.Round),
	// 	TournamentGameIndex: int32(game.TournamentData.GameIndex),
	// 	Type:                game.Type,
	// }, nil
	return nil, nil
}

func createPlayersInfo(ctx context.Context, tx pgx.Tx, p0DBID int, p1DBID int) ([]*gs.StreakInfoResponse_PlayerInfo, bool, error) {
	playersInfo := make([]*gs.StreakInfoResponse_PlayerInfo, 2)
	p0Username, p0UUID, err := common.GetUsernameAndUUIDFromDBID(ctx, tx, p0DBID)
	if err != nil {
		return nil, false, err
	}
	playersInfo[0] = &gs.StreakInfoResponse_PlayerInfo{
		Nickname: p0Username,
		Uuid:     p0UUID,
	}
	p1Username, p1UUID, err := common.GetUsernameAndUUIDFromDBID(ctx, tx, p1DBID)
	if err != nil {
		return nil, false, err
	}
	playersInfo[1] = &gs.StreakInfoResponse_PlayerInfo{
		Nickname: p1Username,
		Uuid:     p1UUID,
	}
	var playerDBIDSAreAlphabeticallyOrdered = p0Username < p1Username
	if !playerDBIDSAreAlphabeticallyOrdered {
		playersInfo[0], playersInfo[1] = playersInfo[1], playersInfo[0]
	}
	return playersInfo, playerDBIDSAreAlphabeticallyOrdered, nil
}

func setStats(ctx context.Context, tx pgx.Tx, g *entity.Game) ([]*playerStats, error) {
	ps := []*playerStats{{wordList: []*play{}}, {wordList: []*play{}}}
	history := g.History()
	events := history.Events
	for evtIdx, evt := range events {
		var prevEvent *macondoipc.GameEvent
		if evtIdx > 0 {
			prevEvent = events[evtIdx-1]
		}

		var succEvent *macondoipc.GameEvent
		if evtIdx+1 < len(events) {
			succEvent = events[evtIdx+1]
		}
		turn := 0
		if evt.Type == macondoipc.GameEvent_TILE_PLACEMENT_MOVE {
			turn = 1
			unchallenged := (succEvent == nil || succEvent.Type != macondoipc.GameEvent_PHONY_TILES_RETURNED)
			var playStat *play
			if unchallenged {
				playStat = makePlayFromEvent(evt)
				if evt.IsBingo {
					playStat.isBingo = true
					ps[evt.PlayerIndex].wordList = append(ps[evt.PlayerIndex].wordList, playStat)
					ps[evt.PlayerIndex].bingos++
				}
				ps[evt.PlayerIndex].tilesPlayed += countTilesPlayed(evt)
				if playStat.score > ps[evt.PlayerIndex].highPlay {
					ps[evt.PlayerIndex].highPlay = playStat.score
				}
			}
			isUnchallengedPhony, err := isUnchallengedPhonyEvent(evt, history, nil)
			if err != nil {
				return nil, err
			}
			if isUnchallengedPhony {
				playStat.isUnchallengedPhony = true
				ps[evt.PlayerIndex].unchallengedPhonies++
			}
		} else if evt.Type == macondoipc.GameEvent_PHONY_TILES_RETURNED {
			playStat := makePlayFromEvent(prevEvent)
			playStat.isChallengedPhony = true
			ps[evt.PlayerIndex].wordList = append(ps[evt.PlayerIndex].wordList, playStat)
			ps[evt.PlayerIndex].challengedPhonies++
		} else if evt.Type == macondoipc.GameEvent_CHALLENGE_BONUS {
			playStat := makePlayFromEvent(prevEvent)
			playStat.isChallengedWord = true
			ps[evt.PlayerIndex].wordList = append(ps[evt.PlayerIndex].wordList, playStat)
			ps[evt.PlayerIndex].challengedWords++
			ps[1-evt.PlayerIndex].unsuccessfulChallenges++
		} else if evt.Type == macondoipc.GameEvent_UNSUCCESSFUL_CHALLENGE_TURN_LOSS {
			playStat := makePlayFromEvent(prevEvent)
			playStat.isChallengedWord = true
			ps[1-evt.PlayerIndex].wordList = append(ps[1-evt.PlayerIndex].wordList, playStat)
			ps[1-evt.PlayerIndex].challengedWords++
			ps[evt.PlayerIndex].unsuccessfulChallenges++
		} else if evt.Type == macondoipc.GameEvent_EXCHANGE {
			turn = 1
			ps[evt.PlayerIndex].exchanges++
		} else if evt.Type == macondoipc.GameEvent_PASS {
			turn = 1
		}
		ps[evt.PlayerIndex].turns += turn
	}
	ps[0].score += int(g.History().FinalScores[0])
	ps[1].score += int(g.History().FinalScores[1])
	winnerIdx := g.History().Winner
	if winnerIdx == 0 {
		ps[0].wins++
		ps[1].losses++
	} else if winnerIdx == 1 {
		ps[0].losses++
		ps[1].wins++
	} else {
		ps[0].draws++
		ps[1].draws++
	}

	return ps, nil
}

func isUnchallengedPhonyEvent(event *macondoipc.GameEvent,
	history *macondoipc.GameHistory,
	cfg *macondoconfig.Config) (bool, error) {
	phony := false
	if event.Type == macondoipc.GameEvent_TILE_PLACEMENT_MOVE {
		dawg, err := gaddag.GetDawg(cfg, history.Lexicon)
		if err != nil {
			return phony, err
		}
		for _, word := range event.WordsFormed {
			phony, err := isPhony(dawg, word, history.Variant)
			if err != nil {
				return false, err
			}
			if phony {
				return phony, nil
			}
		}

	}
	return false, nil
}

func isPhony(gd gaddag.GenericDawg, word, variant string) (bool, error) {
	lex := gaddag.Lexicon{GenericDawg: gd}
	machineWord, err := alphabet.ToMachineWord(word, lex.GetAlphabet())
	if err != nil {
		return false, err
	}
	var valid bool
	switch string(variant) {
	case string(macondogame.VarWordSmog):
		valid = lex.HasAnagram(machineWord)
	default:
		valid = lex.HasWord(machineWord)
	}
	return !valid, nil
}

func countTilesPlayed(event *macondoipc.GameEvent) int {
	sum := 0
	for _, char := range event.PlayedTiles {
		if char != alphabet.ASCIIPlayedThrough {
			sum++
		}
	}
	return sum
}

func makePlayFromEvent(event *macondoipc.GameEvent) *play {
	word := event.PlayedTiles
	if len(event.WordsFormed) > 0 {
		// We can have played tiles with words formed being empty
		// for phony_tiles_returned events, for example.
		word = event.WordsFormed[0]
	}
	return &play{word: word, score: int(event.Score)}
}
