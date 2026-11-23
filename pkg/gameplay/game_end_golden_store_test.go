package gameplay_test

import (
	"context"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
)

// The golden test's two store calls. The code that wrote the golden file
// made them with SetRatings and the profile's Stats field, both gone since.

func seedRatings(ctx context.Context, st *stores.Stores, p0, p1 string, variant entity.VariantKey,
	r0, r1 *entity.SingleRating) error {
	return st.UserStore.UpdateRatings(ctx, p0, p1, variant,
		func(_, _ *entity.SingleRating) (*entity.SingleRating, *entity.SingleRating, error) {
			return r0, r1, nil
		})
}

func profileStats(ctx context.Context, st *stores.Stores, uuid string) (entity.ProfileStats, error) {
	return st.UserStore.GetProfileStats(ctx, uuid)
}
