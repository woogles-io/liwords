package league

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/stores/models"
	ipc "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestCrosstableGame(t *testing.T) {
	is := is.New(t)

	finished := crosstableGame(models.GetDivisionCrosstableGamesRow{
		GameUuid:      pgtype.Text{String: "f", Valid: true},
		GameEndReason: pgtype.Int4{Int32: int32(ipc.GameEndReason_STANDARD), Valid: true},
		WinnerIdx:     pgtype.Int4{Int32: 1, Valid: true},
		PlayerOnTurn:  pgtype.Int4{Int32: 0, Valid: true},
		Player0Score:  350,
		Player1Score:  420,
	})
	is.Equal(finished.WinnerIndex, int32(1))
	is.Equal(finished.PlayerOnTurn, int32(-1))
	is.Equal(finished.Player1Score, int32(420))

	draw := crosstableGame(models.GetDivisionCrosstableGamesRow{
		GameEndReason: pgtype.Int4{Int32: int32(ipc.GameEndReason_STANDARD), Valid: true},
		WinnerIdx:     pgtype.Int4{Int32: -1, Valid: true},
	})
	is.Equal(draw.WinnerIndex, int32(-1))

	history, err := proto.Marshal(&macondopb.GameHistory{Events: []*macondopb.GameEvent{
		{PlayerIndex: 0, Cumulative: 30},
		{PlayerIndex: 1, Cumulative: 24},
		{PlayerIndex: 0, Cumulative: 71},
	}})
	is.NoErr(err)
	live := crosstableGame(models.GetDivisionCrosstableGamesRow{
		GameEndReason: pgtype.Int4{Int32: 0, Valid: true},
		PlayerOnTurn:  pgtype.Int4{Int32: 1, Valid: true},
		History:       history,
	})
	is.Equal(live.Player0Score, int32(71))
	is.Equal(live.Player1Score, int32(24))
	is.Equal(live.WinnerIndex, int32(-1))
	is.Equal(live.PlayerOnTurn, int32(1))
}
