package game

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// The active-game listings, each served from ongoing_games when
// config.ReadsOngoing says so and from games otherwise.
//
// The FromOngoing queries return the same columns as their games twins, so
// sqlc generates structs with identical fields and each row converts with a
// plain type conversion. If the two ever drift apart, these conversions stop
// compiling, which is the point: the listings must not differ by flag.
//
// They take a *models.Queries rather than hanging off DBStore because the
// mod and gameplay services call the queries directly.

func convertRows[N, O any](rows []N, conv func(N) O) []O {
	out := make([]O, len(rows))
	for i, r := range rows {
		out[i] = conv(r)
	}
	return out
}

// ActiveGames lists every live game that is not correspondence.
func ActiveGames(ctx context.Context, q *models.Queries) ([]models.ListActiveGamesRow, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveGames(ctx)
	}
	rows, err := q.ListActiveGamesFromOngoing(ctx)
	if err != nil {
		return nil, err
	}
	return convertRows(rows, func(r models.ListActiveGamesFromOngoingRow) models.ListActiveGamesRow {
		return models.ListActiveGamesRow(r)
	}), nil
}

// ActiveTournamentGames lists a tournament's live games that are not
// correspondence.
func ActiveTournamentGames(ctx context.Context, q *models.Queries, tourneyID string) ([]models.ListActiveTournamentGamesRow, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveTournamentGames(ctx, tourneyID)
	}
	rows, err := q.ListActiveTournamentGamesFromOngoing(ctx, tourneyID)
	if err != nil {
		return nil, err
	}
	return convertRows(rows, func(r models.ListActiveTournamentGamesFromOngoingRow) models.ListActiveTournamentGamesRow {
		return models.ListActiveTournamentGamesRow(r)
	}), nil
}

// ActiveRealTimeGamesForAdjudication lists the columns the adjudicator needs
// for every live game that is not correspondence.
func ActiveRealTimeGamesForAdjudication(ctx context.Context, q *models.Queries) ([]models.ListActiveRealTimeGamesForAdjudicationRow, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveRealTimeGamesForAdjudication(ctx)
	}
	rows, err := q.ListActiveRealTimeGamesForAdjudicationFromOngoing(ctx)
	if err != nil {
		return nil, err
	}
	return convertRows(rows, func(r models.ListActiveRealTimeGamesForAdjudicationFromOngoingRow) models.ListActiveRealTimeGamesForAdjudicationRow {
		return models.ListActiveRealTimeGamesForAdjudicationRow(r)
	}), nil
}

// ActiveCorrespondenceGames lists every live correspondence game.
func ActiveCorrespondenceGames(ctx context.Context, q *models.Queries) ([]models.ListActiveCorrespondenceGamesRow, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveCorrespondenceGames(ctx)
	}
	rows, err := q.ListActiveCorrespondenceGamesFromOngoing(ctx)
	if err != nil {
		return nil, err
	}
	return convertRows(rows, func(r models.ListActiveCorrespondenceGamesFromOngoingRow) models.ListActiveCorrespondenceGamesRow {
		return models.ListActiveCorrespondenceGamesRow(r)
	}), nil
}

// ActiveCorrespondenceGamesForUser lists one user's live correspondence games.
func ActiveCorrespondenceGamesForUser(ctx context.Context, q *models.Queries, userUUID string) ([]models.ListActiveCorrespondenceGamesForUserRow, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveCorrespondenceGamesForUser(ctx, userUUID)
	}
	rows, err := q.ListActiveCorrespondenceGamesForUserFromOngoing(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	return convertRows(rows, func(r models.ListActiveCorrespondenceGamesForUserFromOngoingRow) models.ListActiveCorrespondenceGamesForUserRow {
		return models.ListActiveCorrespondenceGamesForUserRow(r)
	}), nil
}

// ActiveCorrespondenceGamesForUserAndLeague lists one user's live
// correspondence games in one league.
func ActiveCorrespondenceGamesForUserAndLeague(ctx context.Context, q *models.Queries, arg models.ListActiveCorrespondenceGamesForUserAndLeagueParams) ([]models.ListActiveCorrespondenceGamesForUserAndLeagueRow, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveCorrespondenceGamesForUserAndLeague(ctx, arg)
	}
	rows, err := q.ListActiveCorrespondenceGamesForUserAndLeagueFromOngoing(ctx,
		models.ListActiveCorrespondenceGamesForUserAndLeagueFromOngoingParams(arg))
	if err != nil {
		return nil, err
	}
	return convertRows(rows, func(r models.ListActiveCorrespondenceGamesForUserAndLeagueFromOngoingRow) models.ListActiveCorrespondenceGamesForUserAndLeagueRow {
		return models.ListActiveCorrespondenceGamesForUserAndLeagueRow(r)
	}), nil
}

// CountActiveCorrespondenceGames counts the live correspondence games.
func CountActiveCorrespondenceGames(ctx context.Context, q *models.Queries) (int32, error) {
	if !config.ReadsOngoing(ctx) {
		return q.CountActiveCorrespondenceGames(ctx)
	}
	return q.CountActiveCorrespondenceGamesFromOngoing(ctx)
}

// ActiveCorrespondenceGamesWithBotOnTurn lists the live correspondence games
// waiting on an internal bot (other than BestBot).
func ActiveCorrespondenceGamesWithBotOnTurn(ctx context.Context, q *models.Queries) ([]pgtype.Text, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveCorrespondenceGamesWithBotOnTurn(ctx)
	}
	return q.ListActiveCorrespondenceGamesWithBotOnTurnFromOngoing(ctx)
}

// ActiveRealtimeGamesWithBotOnTurn lists the live games that are not
// correspondence and are waiting on an internal bot (other than BestBot).
func ActiveRealtimeGamesWithBotOnTurn(ctx context.Context, q *models.Queries) ([]pgtype.Text, error) {
	if !config.ReadsOngoing(ctx) {
		return q.ListActiveRealtimeGamesWithBotOnTurn(ctx)
	}
	return q.ListActiveRealtimeGamesWithBotOnTurnFromOngoing(ctx)
}
