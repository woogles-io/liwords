package game

import (
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores/models"
)

// startingRatingParams is a starting rating as the two nullable columns that
// store it. A player with none gets NULL in both.
func startingRatingParams(r *entity.StartingRating) (pgtype.Float4, pgtype.Bool) {
	if r == nil {
		return pgtype.Float4{}, pgtype.Bool{}
	}
	return pgtype.Float4{Float32: float32(r.Rating), Valid: true},
		pgtype.Bool{Bool: r.Provisional, Valid: true}
}

// startingRatingFromColumns reads back what startingRatingParams wrote.
func startingRatingFromColumns(rating pgtype.Float4, provisional pgtype.Bool) *entity.StartingRating {
	if !rating.Valid {
		return nil
	}
	return &entity.StartingRating{Rating: float64(rating.Float32), Provisional: provisional.Bool}
}

// loadedStartingRatings is a loaded game's starting ratings: from its
// ongoing_games row, or, for a game whose row has none -- created before the
// columns were, or with no row at all -- from its quickdata, rounded as it was
// displayed. The next save writes those into the row.
//
// The quickdata fallback goes when quickdata does. By then every game created
// before these columns has been saved since, or has ended.
func loadedStartingRatings(row models.GetGameWithTurnsRow, qd *entity.Quickdata) [2]*entity.StartingRating {
	ratings := [2]*entity.StartingRating{
		startingRatingFromColumns(row.Player0StartingRating, row.Player0StartingRatingProvisional),
		startingRatingFromColumns(row.Player1StartingRating, row.Player1StartingRatingProvisional),
	}
	for i := range ratings {
		if ratings[i] == nil && qd != nil && i < len(qd.PlayerInfo) && qd.PlayerInfo[i] != nil {
			ratings[i] = entity.ParseDisplayRating(qd.PlayerInfo[i].Rating)
		}
	}
	return ratings
}
