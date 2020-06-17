package glicko

import (
	"fmt"
	"testing"
	"github.com/matryer/is"
	"math/rand"
)

func CreatePlayers() [10][6]float64 {
	players := [10][6]float64{
		{500, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{460, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{440, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{410, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{380, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{370, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{360, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{340, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{310, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{220, 30, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
	}
	return players
}

func TestRatingGain(t *testing.T) {

	is := is.New(t)

	rating, deviation, volatility :=
	  Rate(InitialVolatility,
		   float64(InitialRating),
		   float64(InitialRatingDeviation),
		   float64(InitialRating - 100),
		   float64(MinimumRatingDeviation),
		   100,
		   RatingPeriodinSeconds)

	is.True(rating > float64(InitialRating))
	is.True(deviation < float64(InitialRatingDeviation))
	is.True(volatility < InitialVolatility)
}

func TestRatingLoss(t *testing.T) {

	is := is.New(t)

	rating, deviation, volatility :=
	  Rate(InitialVolatility,
		   float64(InitialRating),
		   float64(InitialRatingDeviation),
		   float64(InitialRating - 100),
		   float64(MinimumRatingDeviation),
		   -100,
		   RatingPeriodinSeconds)

	is.True(rating < float64(InitialRating))
	is.True(deviation < float64(InitialRatingDeviation))
	is.True(volatility < InitialVolatility)
}

func TestVolatility(t *testing.T) {

	is := is.New(t)

	rating := float64(InitialRating)
	deviation := float64(InitialRatingDeviation)
	volatility := InitialVolatility

	for i := 0; i < 1000; i++ {
	rating, deviation, volatility =
	  Rate(volatility,
		   rating,
		   deviation,
		   float64(InitialRating),
		   float64(MinimumRatingDeviation),
		   0,
		   RatingPeriodinSeconds / 1000)

	}

	is.True(int(rating) == InitialRating)
	is.True(int(deviation) == MinimumRatingDeviation)
	is.True(volatility < InitialVolatility)

	reducedVolatility := volatility

	rating, deviation, volatility =
	  Rate(volatility,
		   rating,
		   deviation,
		   float64(InitialRating + 1000),
		   float64(MinimumRatingDeviation),
		   200,
		   RatingPeriodinSeconds)

	is.True(volatility > reducedVolatility)
}

func TestVolatilityMaximum(t *testing.T) {

	is := is.New(t)

	rating := float64(InitialRating)
	deviation := float64(InitialRatingDeviation)
	volatility := InitialVolatility

	for i := 0; i < 10000; i++ {
	rating, deviation, volatility =
	  Rate(volatility,
		   rating,
		   deviation,
		   float64(InitialRating) + float64( (i % 2) * 2 - 1)*float64(InitialRating),
		   float64(MinimumRatingDeviation),
		   200 * ( (i % 2) * 2 - 1),
		   RatingPeriodinSeconds / 100)

	}

	is.True(int(deviation) == MinimumRatingDeviation)
	is.True(volatility == MaximumVolatility)
}

func TestRatingDeviationMaximum(t *testing.T) {

	is := is.New(t)

	rating := float64(InitialRating)
	deviation := float64(InitialRatingDeviation)
	volatility := InitialVolatility

	rating, deviation, volatility =
	  Rate(volatility,
		   rating,
		   deviation,
		   float64(InitialRating),
		   float64(MinimumRatingDeviation),
		   0,
		   RatingPeriodinSeconds * 100000000)

	is.True(int(deviation) == MaximumRatingDeviation)
}


func TestSpread(t *testing.T) {

	is := is.New(t)

	rating1 := float64(InitialRating)
	deviation1 := float64(InitialRatingDeviation)
	volatility1 := InitialVolatility

	rating1, deviation1, volatility1 =
	  Rate(volatility1,
		   rating1,
		   deviation1,
		   float64(InitialRating),
		   float64(MinimumRatingDeviation),
		   SpreadScaling - 50,
		   RatingPeriodinSeconds)

	rating2 := float64(InitialRating)
	deviation2 := float64(InitialRatingDeviation)
	volatility2 := InitialVolatility

	rating2, deviation2, volatility2 =
	  Rate(volatility2,
		   rating2,
		   deviation2,
		   float64(InitialRating),
		   float64(MinimumRatingDeviation),
		   SpreadScaling,
		   RatingPeriodinSeconds)

	is.True(rating1 < rating2)
}

func TestSpreadMax(t *testing.T) {

	is := is.New(t)

	rating1 := float64(InitialRating)
	deviation1 := float64(InitialRatingDeviation)
	volatility1 := InitialVolatility

	rating1, deviation1, volatility1 =
	  Rate(volatility1,
		   rating1,
		   deviation1,
		   float64(InitialRating),
		   float64(MinimumRatingDeviation),
		   SpreadScaling,
		   RatingPeriodinSeconds)

	rating2 := float64(InitialRating)
	deviation2 := float64(InitialRatingDeviation)
	volatility2 := InitialVolatility

	rating2, deviation2, volatility2 =
	  Rate(volatility2,
		   rating2,
		   deviation2,
		   float64(InitialRating),
		   float64(MinimumRatingDeviation),
		   SpreadScaling + 1,
		   RatingPeriodinSeconds)

	is.True(rating1 == rating2)
}

func TestBehaviorUniformPairings(t *testing.T) {

	players := CreatePlayers()

	for i := 0; i < 10000; i++ {
		for j := 0; j < len(players); j++ {
			for k := j + 1; k < len(players); k++ {

				playerScore    := int(rand.NormFloat64() * players[j][1]  + players[j][0])
				oppponentScore := int(rand.NormFloat64() * players[k][1]  + players[k][0])
				spread := playerScore - oppponentScore
				playerOriginalRating          := players[j][3]
				playerOriginalRatingDeviation := players[j][4]

				players[j][3], players[j][4], players[j][2] =
				  Rate(players[j][2],
					   playerOriginalRating,
					   playerOriginalRatingDeviation,
					   players[k][3],
					   players[k][4],
					   spread,
					   RatingPeriodinSeconds / 12)

				players[k][3], players[k][4], players[k][2] =
				  Rate(players[k][2],
				  	   players[k][3],
					   players[k][4],
					   playerOriginalRating,
					   playerOriginalRatingDeviation,
					   -spread,
					   RatingPeriodinSeconds / 12)
			}
		}
	}
	fmt.Println("Uniform pairings")
	for i := range players {
	    fmt.Println(players[i])
	}
}

func TestBehaviorStratifiedPairings(t *testing.T) {

	players := CreatePlayers()

	for i := 0; i < 10000; i++ {
		for j := 0; j < len(players); j++ {
			for k := j + 1; k < len(players); k++ {
				if (j - k > 1) {
					continue
				}
				playerScore    := int(rand.NormFloat64() * players[j][1]  + players[j][0])
				oppponentScore := int(rand.NormFloat64() * players[k][1]  + players[k][0])
				spread := playerScore - oppponentScore
				playerOriginalRating          := players[j][3]
				playerOriginalRatingDeviation := players[j][4]

				players[j][3], players[j][4], players[j][2] =
				  Rate(players[j][2],
					   playerOriginalRating,
					   playerOriginalRatingDeviation,
					   players[k][3],
					   players[k][4],
					   spread,
					   RatingPeriodinSeconds / 12)

				players[k][3], players[k][4], players[k][2] =
				  Rate(players[k][2],
				  	   players[k][3],
					   players[k][4],
					   playerOriginalRating,
					   playerOriginalRatingDeviation,
					   -spread,
					   RatingPeriodinSeconds / 12)
			}
		}
	}
	fmt.Println("Stratified pairings")
	for i := range players {
	    fmt.Println(players[i])
	}
}
