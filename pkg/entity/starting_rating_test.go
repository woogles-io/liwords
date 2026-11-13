package entity

import (
	"testing"

	"github.com/matryer/is"
)

func TestRelevantRatingValue(t *testing.T) {
	is := is.New(t)
	key := VariantKey("NWL23.classic.blitz")
	ratings := Ratings{Data: map[VariantKey]SingleRating{
		key: {Rating: 1649.6, RatingDeviation: RatingDeviationConfidence},
		"other.classic.blitz": {Rating: 1820.2, RatingDeviation: RatingDeviationConfidence + 1},
	}}

	for _, tc := range []struct {
		name    string
		ratings Ratings
		key     VariantKey
		want    StartingRating
		display string
	}{
		{"confident", ratings, key, StartingRating{1649.6, false}, "1650"},
		{"provisional", ratings, "other.classic.blitz", StartingRating{1820.2, true}, "1820?"},
		{"no rating in the variant", ratings, "CSW24.classic.blitz", StartingRating{1500, true}, "1500?"},
		{"no ratings at all", Ratings{}, key, StartingRating{1500, true}, "1500?"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			got := RelevantRatingValue(tc.ratings, tc.key)
			is.Equal(got, tc.want)
			// The display string is the one RelevantRating always gave.
			is.Equal(RelevantRating(tc.ratings, tc.key), tc.display)
			// And it reads back as the rating rounded.
			back := ParseDisplayRating(tc.display)
			is.True(back != nil)
			is.Equal(back.Provisional, tc.want.Provisional)
			is.Equal(back.String(), tc.display)
		})
	}

	// A user with no profile has no starting rating, as they have no
	// displayed one.
	is.Equal((&User{}).GetStartingRating(key), (*StartingRating)(nil))
	is.Equal((&User{}).GetRelevantRating(key), "UnratedAnon")
	for _, s := range []string{"UnratedAnon", "", "?", "abc?"} {
		is.Equal(ParseDisplayRating(s), (*StartingRating)(nil)) // s
	}
}
