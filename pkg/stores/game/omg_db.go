package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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

	var gameDBID uint
	var createdAt time.Time
	var timers entity.Timers
	var requestBytes []byte
	var metaEvents entity.MetaEventData
	var tournamentData entity.TournamentData
	var tournamentID string
	var started bool
	var gameEndReason int
	var gameType int

	err = tx.QueryRow(ctx, `SELECT id, create_at, timers, request, meta_events,
	tournament_data, tournament_id, started, game_end_reason,
	type FROM omgwords WHERE uuid = $1`, uuid).Scan(
		&gameDBID, &createdAt, &timers, &requestBytes, &metaEvents, &tournamentData, &tournamentID, &started, &gameEndReason, &gameType)
	if err != nil {
		return nil, err
	}

	var historyBytes []byte

	err = tx.QueryRow(ctx, `SELECT history FROM omgwords_histories WHERE game_id = $1`, gameDBID).Scan(&historyBytes)
	if err != nil {
		return nil, err
	}

	var history macondoipc.GameHistory
	err = json.Unmarshal(historyBytes, &history)
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `SELECT player_id, first, won, player_old_rating, player_new_rating FROM omgwords_games WHERE game_id = $1`, gameDBID)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("no rows found for omgwords id: %d", gameDBID)
	} else if err != nil {
		return nil, err
	}

	var p0id uint
	var p0won bool
	var p0OldRating float64
	var p0NewRating float64
	var p1id uint
	var p1won bool
	var p1OldRating float64
	var p1NewRating float64

	for rows.Next() {
		var playerID uint
		var first bool
		var won bool
		var playerOldRating float64
		var playerNewRating float64
		if err := rows.Scan(&playerID, &first, &won, &playerOldRating, &playerNewRating); err != nil {
			rows.Close()
			return nil, err
		}
		if first {
			p0id = playerID
			p0won = won
			p0OldRating = playerOldRating
			p0NewRating = playerNewRating
		} else {
			p1id = playerID
			p1won = won
			p1OldRating = playerOldRating
			p1NewRating = playerNewRating
		}
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

	var req ipc.GameRequest
	err = json.Unmarshal(requestBytes, &req)
	if err != nil {
		return nil, err
	}

	qdata := &entity.Quickdata{}

	stats := &entity.Stats{}

	entGame, err := fromState(timers, qdata, started, gameEndReason, p0id, p1id, winnerIdx, loserIdx,
		requestBytes, historyBytes, stats, &metaEvents, s.gameEventChan, s.cfg, createdAt, ipc.GameType(gameType), gameDBID)
	if err != nil {
		return nil, err
	}

	// In the future we will probably get rid of quickdata
	// but for now we populate it here so that the underlying
	// DB store for a game is consistent across the migration.
	entGame.Quickdata.OriginalRequestId = entGame.GameReq.OriginalRequestId
	entGame.Quickdata.FinalScores = entGame.History().FinalScores
	entGame.Quickdata.OriginalRatings = []float64{p0OldRating, p1OldRating}
	entGame.Quickdata.NewRatings = []float64{p0NewRating, p1NewRating}

	playerDBIDs := []uint{p0id, p1id}
	playerInfos := make([]*pb.PlayerInfo, 2)

	for idx, playerDBID := range playerDBIDs {
		user, err := common.GetUserBy(ctx, tx,
			&common.CommonDBConfig{TableType: common.UsersTable,
				SelectByType:   common.SelectByID,
				Value:          playerDBID,
				IncludeProfile: false})
		if err != nil {
			return nil, err
		}
		ratingKey, err := entGame.RatingKey()
		if err != nil {
			return nil, err
		}
		playerInfos = append(playerInfos, &pb.PlayerInfo{
			Nickname: user.Username,
			UserId:   user.UUID,
			Rating:   user.GetRelevantRating(ratingKey),
			IsBot:    user.IsBot,
			First:    idx == 0,
		})
	}

	entGame.Quickdata.PlayerInfo = playerInfos

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entGame, nil
}

func (s *DBStore) OMGGetMetadata(ctx context.Context, uuid string) (*ipc.GameInfoResponse, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	info, err := getGameInfoResponseFromUUID(ctx, tx, uuid)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return info, nil
}

func (s *DBStore) OMGGetRematchStreak(ctx context.Context, originalRequestId string) (*gs.StreakInfoResponse, error) {
	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// This query enforces a player_id DESC order for each
	// game pair. The arbitrary but consistent ordering
	// will allow us to order by username later.
	rows, err := tx.Query(ctx, `SELECT omgwords.uuid AS game_uuid, won, player_id
	FROM omgwords 
	INNER JOIN omgwords_games ON omgwords.id = omgwords_games.game_id
	WHERE request->>'original_request_id' = $1 AND game_end_reason not in ($2, $3, $4) ORDER BY omgwords.created_at, player_id DESC`,
		originalRequestId, pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED)
	if err == pgx.ErrNoRows {
		return &gs.StreakInfoResponse{Streak: make([]*gs.StreakInfoResponse_SingleGameInfo, 0)}, nil
	}
	if err != nil {
		return nil, err
	}

	idx := 0
	resp := &gs.StreakInfoResponse{Streak: []*gs.StreakInfoResponse_SingleGameInfo{}}
	playerWonArr := []bool{false, false}
	playerDBIDArr := []int{-1, -1}
	var playerDBIDArrCheck []int
	var playerDBIDSAreAlphabeticallyOrdered bool
	for rows.Next() {
		var gameUUID string
		var playerWon bool
		var playerDBID int
		if err = rows.Scan(gameUUID, playerWon, playerDBID); err != nil {
			rows.Close()
			return nil, err
		}
		playerWonArr[idx%2] = playerWon
		playerDBIDArr[idx%2] = playerDBID
		if idx == 1 {
			var playersInfo []*gs.StreakInfoResponse_PlayerInfo
			playersInfo, playerDBIDSAreAlphabeticallyOrdered, err = createPlayersInfo(ctx, tx, playerDBIDArr[0], playerDBIDArr[1])
			if err != nil {
				return nil, err
			}
			resp.PlayersInfo = playersInfo
			playerDBIDArrCheck = []int{playerDBIDArr[0], playerDBIDArr[1]}
		} else if idx > 1 {
			if resp.PlayersInfo == nil {
				return nil, fmt.Errorf("players info nil for original request id %s", originalRequestId)
			}
			if playerDBID != playerDBIDArrCheck[idx%2] {
				return nil, fmt.Errorf("player id %d is not consistent for streak with original request id %s", playerDBID, originalRequestId)
			}
		}
		if idx%2 == 1 {
			// Assume a tie
			winner := -1
			if playerWonArr[idx-1] && !playerWonArr[idx] {
				// First player in the game pairs won
				winner = 0
			} else if !playerWonArr[idx-1] && playerWonArr[idx] {
				// Second player in the game pairs won
				winner = 1
			}
			if winner != -1 && !playerDBIDSAreAlphabeticallyOrdered {
				winner = winner - 1
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
		return nil, errors.New("too many games")
	}

	tx, err := s.dbPool.BeginTx(ctx, common.DefaultTxOptions)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	userDBID, _, err := common.GetUUIDAndDBIDFromUsername(ctx, tx, username)

	rows, err := tx.Query(ctx, `SELECT uuid
	FROM omgwords_games INNER JOIN omgwords ON omgwords_games.game_id = omgwords.id
	WHERE player_id = $1
	AND game_end_reason NOT IN ($2, $3, $4) ORDER BY created_at DESC LIMIT $5 OFFSET $6`,
		userDBID,
		pb.GameEndReason_NONE, pb.GameEndReason_ABORTED, pb.GameEndReason_CANCELLED, numGames, offset)

	for rows.Next() {
		var gameUUID string
		var playerWon bool
		var playerDBID int
		if err = rows.Scan(gameUUID, playerWon, playerDBID); err != nil {
			rows.Close()
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *DBStore) OMGGetRecentTourneyGames(ctx context.Context, tourneyID string, numGames int, offset int) (*ipc.GameInfoResponses, error) {
	return nil, nil
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

	if g.GameEndReason != ipc.GameEndReason_NONE && g.RatingMode() == ipc.RatingMode_RATED {
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
