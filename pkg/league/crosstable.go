package league

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/woogles-io/liwords/pkg/apiserver"
	gamestore "github.com/woogles-io/liwords/pkg/stores/game"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	pb "github.com/woogles-io/liwords/rpc/api/proto/league_service"
)

// crosstableGame converts one division game row for the cross-table: final
// scores for a finished game, running scores (from its history) for one in
// progress.
func crosstableGame(row models.GetDivisionCrosstableGamesRow) *pb.CrosstableGame {
	g := &pb.CrosstableGame{
		GameId:        row.GameUuid.String,
		Player0UserId: row.Player0Uuid,
		Player1UserId: row.Player1Uuid,
		Player0Score:  row.Player0Score,
		Player1Score:  row.Player1Score,
		GameEndReason: ipc.GameEndReason(row.GameEndReason.Int32),
		WinnerIndex:   -1,
		PlayerOnTurn:  -1,
	}
	if g.GameEndReason == ipc.GameEndReason_NONE {
		scores := gamestore.ScoresFromHistory(row.History)
		g.Player0Score, g.Player1Score = scores[0], scores[1]
		if row.PlayerOnTurn.Valid {
			g.PlayerOnTurn = row.PlayerOnTurn.Int32
		}
	} else if row.WinnerIdx.Valid && (row.WinnerIdx.Int32 == 0 || row.WinnerIdx.Int32 == 1) {
		g.WinnerIndex = row.WinnerIdx.Int32
	}
	return g
}

func (ls *LeagueService) GetDivisionCrosstable(
	ctx context.Context,
	req *connect.Request[pb.DivisionRequest],
) (*connect.Response[pb.DivisionCrosstableResponse], error) {
	divisionID, err := uuid.Parse(req.Msg.DivisionId)
	if err != nil {
		return nil, apiserver.InvalidArg("invalid division_id")
	}
	rows, err := ls.store.GetDivisionCrosstableGames(ctx, divisionID)
	if err != nil {
		return nil, apiserver.InternalErr(fmt.Errorf("failed to get division games: %w", err))
	}
	games := make([]*pb.CrosstableGame, len(rows))
	for i, row := range rows {
		games[i] = crosstableGame(row)
	}
	return connect.NewResponse(&pb.DivisionCrosstableResponse{Games: games}), nil
}
