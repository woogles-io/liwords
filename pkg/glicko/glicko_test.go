package glicko

import (
	"encoding/csv"
	"fmt"
	"github.com/matryer/is"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"
)

type Player struct {
	name            string
	id              int
	rating          float64
	ratingDeviation float64
	volatility      float64
}

type ByRating []*Player

func (a ByRating) Len() int           { return len(a) }
func (a ByRating) Less(i, j int) bool { return a[i].rating > a[j].rating }
func (a ByRating) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}

func MeanStdDev(floats []float64) (float64, float64) {

	sum := 0.0
	for _, float := range floats {
		sum += float
	}
	length := float64(len(floats))
	mean := sum / length
	stdsum := 0.0
	for _, float := range floats {
		stdsum += math.Pow(float-mean, 2)
	}

	return mean, math.Sqrt(stdsum / length)
}

func TestRatingGain(t *testing.T) {

	is := is.New(t)

	rating, deviation, volatility :=
		Rate(
			float64(InitialRating),
			float64(InitialRatingDeviation),
			InitialVolatility,
			float64(InitialRating-100),
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
		Rate(
			float64(InitialRating),
			float64(InitialRatingDeviation),
			InitialVolatility,
			float64(InitialRating-100),
			float64(MinimumRatingDeviation),
			-100,
			RatingPeriodinSeconds)

	is.True(rating < float64(InitialRating))
	is.True(deviation < float64(InitialRatingDeviation))
	is.True(volatility < InitialVolatility)
}

func TestAverageRatingChange(t *testing.T) {

	spread := 80

	rating, _, _ :=
		Rate(
			float64(InitialRating),
			float64(MinimumRatingDeviation),
			InitialVolatility,
			float64(InitialRating),
			float64(MinimumRatingDeviation),
			spread,
			RatingPeriodinSeconds/12)

	fmt.Printf("A typical rating change for two players with equal ratings\n"+
		"and minimum rating deviations in a game where the final\n"+
		"spread is %d points is %.4f points\n\n", spread, rating-float64(InitialRating))
}

func TestVolatility(t *testing.T) {

	is := is.New(t)

	rating := float64(InitialRating)
	deviation := float64(InitialRatingDeviation)
	volatility := InitialVolatility

	for i := 0; i < 1000; i++ {
		rating, deviation, volatility =
			Rate(
				rating,
				deviation,
				volatility,
				float64(InitialRating),
				float64(MinimumRatingDeviation),
				0,
				RatingPeriodinSeconds/1000)

	}

	is.True(int(rating) == InitialRating)
	is.True(int(deviation) == MinimumRatingDeviation)
	is.True(volatility < InitialVolatility)

	reducedVolatility := volatility

	rating, deviation, volatility =
		Rate(
			rating,
			deviation,
			volatility,
			float64(InitialRating+1000),
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
			Rate(
				rating,
				deviation,
				volatility,
				float64(InitialRating)+float64((i%2)*2-1)*float64(InitialRating),
				float64(MinimumRatingDeviation),
				200*((i%2)*2-1),
				RatingPeriodinSeconds/100)

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
		Rate(
			rating,
			deviation,
			volatility,
			float64(InitialRating),
			float64(MinimumRatingDeviation),
			0,
			RatingPeriodinSeconds*1000000000)

	is.True(int(deviation) == MaximumRatingDeviation)
}

func TestSpread(t *testing.T) {

	is := is.New(t)

	rating1 := float64(InitialRating)
	deviation1 := float64(InitialRatingDeviation)
	volatility1 := InitialVolatility

	rating1, deviation1, volatility1 =
		Rate(
			rating1,
			deviation1,
			volatility1,
			float64(InitialRating),
			float64(MinimumRatingDeviation),
			SpreadScaling-50,
			RatingPeriodinSeconds)

	rating2 := float64(InitialRating)
	deviation2 := float64(InitialRatingDeviation)
	volatility2 := InitialVolatility

	rating2, deviation2, volatility2 =
		Rate(
			rating2,
			deviation2,
			volatility2,
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
		Rate(
			rating1,
			deviation1,
			volatility1,
			float64(InitialRating),
			float64(MinimumRatingDeviation),
			SpreadScaling,
			RatingPeriodinSeconds)

	rating2 := float64(InitialRating)
	deviation2 := float64(InitialRatingDeviation)
	volatility2 := InitialVolatility

	rating2, deviation2, volatility2 =
		Rate(
			rating2,
			deviation2,
			volatility2,
			float64(InitialRating),
			float64(MinimumRatingDeviation),
			SpreadScaling+1,
			RatingPeriodinSeconds)

	is.True(rating1 == rating2)
}

func TestWinBoost(t *testing.T) {

	is := is.New(t)

	additionalRating := 200

	rating1 := float64(InitialRating)
	deviation1 := float64(InitialRatingDeviation)
	volatility1 := InitialVolatility

	rating1, deviation1, volatility1 =
		Rate(
			rating1,
			deviation1,
			volatility1,
			float64(InitialRating+additionalRating),
			float64(MinimumRatingDeviation),
			0,
			RatingPeriodinSeconds)

	rating2 := float64(InitialRating)
	deviation2 := float64(InitialRatingDeviation)
	volatility2 := InitialVolatility

	rating2, deviation2, volatility2 =
		Rate(
			rating2,
			deviation2,
			volatility2,
			float64(InitialRating+additionalRating),
			float64(MinimumRatingDeviation),
			1,
			RatingPeriodinSeconds)

	rating3 := float64(InitialRating)
	deviation3 := float64(InitialRatingDeviation)
	volatility3 := InitialVolatility

	rating3, deviation3, volatility3 =
		Rate(
			rating3,
			deviation3,
			volatility3,
			float64(InitialRating+additionalRating),
			float64(MinimumRatingDeviation),
			2,
			RatingPeriodinSeconds)

	rating4 := float64(InitialRating)
	deviation4 := float64(InitialRatingDeviation)
	volatility4 := InitialVolatility

	rating4, deviation4, volatility4 =
		Rate(
			rating4,
			deviation4,
			volatility4,
			float64(InitialRating+additionalRating),
			float64(MinimumRatingDeviation),
			3,
			RatingPeriodinSeconds)

	// Gosh durned floatin point nonsense
	is.True(math.Abs((rating4-rating3)-(rating3-rating2)) < 0.0000001)
	is.True(rating3-rating2 < rating2-rating1)
}

func TestRatingConvergenceTime(t *testing.T) {

	rating := float64(InitialRating)
	deviation := float64(InitialRatingDeviation)
	volatility := InitialVolatility

	games_to_steady := 0
	spread := 80
	opponent_rating := float64(InitialRating + 600)

	for i := 0; i < 1000; i++ {
		games_to_steady++
		rating, deviation, volatility =
			Rate(
				rating,
				deviation,
				volatility,
				opponent_rating,
				float64(MinimumRatingDeviation),
				spread,
				RatingPeriodinSeconds/12)
		// fmt.Printf("%.2f %.2f %.2f\n", rating, deviation, volatility)
		if float64(InitialRating+600)-rating < float64(MinimumRatingDeviation) {
			break
		}
	}

	fmt.Printf("Starting at a rating of %d and a deviation of %d,\n"+
		"it took %d games winning by %d points each against\n"+
		"a %.2f rated oppoent to get to a rating of %.2f\n\n", InitialRating, InitialRatingDeviation, games_to_steady, spread, opponent_rating, rating)
}

func TestRatingConvergenceTimeAfterSteadyState(t *testing.T) {

	is := is.New(t)

	rating := float64(InitialRating)
	deviation := float64(InitialRatingDeviation)
	volatility := InitialVolatility

	for i := 0; i < 1000; i++ {
		rating, deviation, volatility =
			Rate(
				rating,
				deviation,
				volatility,
				float64(InitialRating),
				float64(MinimumRatingDeviation),
				0,
				RatingPeriodinSeconds/12)

	}

	is.True(int(rating) == InitialRating)
	is.True(int(deviation) == MinimumRatingDeviation)
	is.True(volatility < InitialVolatility)

	games_to_steady := 0
	spread := 50
	opponent_rating := float64(InitialRating + 500)

	for i := 0; i < 1000; i++ {
		games_to_steady++
		rating, deviation, volatility =
			Rate(
				rating,
				deviation,
				volatility,
				float64(InitialRating+500),
				float64(MinimumRatingDeviation),
				spread,
				RatingPeriodinSeconds/12)
		// fmt.Printf("%.2f %.2f %.2f\n", rating, deviation, volatility)
		if float64(InitialRating+500)-rating < float64(MinimumRatingDeviation) {
			break
		}
	}

	fmt.Printf("Starting at a rating of %d and a deviation of %d,\n"+
		"it took %d games winning by %d points each against\n"+
		"a %.2f rated oppoent to get to a rating of %.2f\n\n\n", InitialRating, MinimumRatingDeviation, games_to_steady, spread, opponent_rating, rating)
}

func TestRealData(t *testing.T) {

	players := make(map[int]*Player)
	data := readCsvFile("crosstables_game_results.csv")
	minDeviationGuesses := 0
	correctGuesses := 0
	var spreads []float64

	for _, game := range data {
		for i := 0; i < 2; i++ {
			playerid, _ := strconv.Atoi(game[2+i])
			playername := game[6+i]
			if _, ok := players[playerid]; !ok {
				players[playerid] = &Player{id: playerid,
					name:            playername,
					rating:          float64(InitialRating),
					ratingDeviation: float64(InitialRatingDeviation),
					volatility:      InitialVolatility}
			}
		}

		playerOneID, _ := strconv.Atoi(game[2])
		playerTwoID, _ := strconv.Atoi(game[3])

		playerOneRating := players[playerOneID].rating
		playerOneRatingDeviation := players[playerOneID].ratingDeviation
		playerOneVolatility := players[playerOneID].volatility

		playerTwoRating := players[playerTwoID].rating
		playerTwoRatingDeviation := players[playerTwoID].ratingDeviation
		playerTwoVolatility := players[playerTwoID].volatility

		playerOneScore, _ := strconv.Atoi(game[4])
		playerTwoScore, _ := strconv.Atoi(game[5])

		spread := playerOneScore - playerTwoScore
		spreads = append(spreads, float64(spread))

		if playerOneRatingDeviation == float64(MinimumRatingDeviation) && playerTwoRatingDeviation == float64(MinimumRatingDeviation) {
			minDeviationGuesses++

			if math.Abs(playerOneRating-playerTwoRating) < 4*float64(MinimumRatingDeviation) ||
				(playerOneRating > playerTwoRating && spread > 0) ||
				(playerOneRating < playerTwoRating && spread < 0) {
				correctGuesses++
			}
		}

		players[playerOneID].rating, players[playerOneID].ratingDeviation, players[playerOneID].volatility =
			Rate(playerOneRating,
				playerOneRatingDeviation,
				playerOneVolatility,
				playerTwoRating,
				playerTwoRatingDeviation,
				spread,
				RatingPeriodinSeconds/12)

		players[playerTwoID].rating, players[playerTwoID].ratingDeviation, players[playerTwoID].volatility =
			Rate(playerTwoRating,
				playerTwoRatingDeviation,
				playerTwoVolatility,
				playerOneRating,
				playerOneRatingDeviation,
				-spread,
				RatingPeriodinSeconds/12)
	}

	playersArray := []*Player{}
	for _, v := range players {
		playersArray = append(playersArray, v)
	}

	sort.Sort(ByRating(playersArray))

	fmt.Printf("These are the results of the rating system applied to\n" +
		"every game on cross-tables.com starting on January 1, 2000\n")
	fmt.Printf("The test pool contains %d players\n\n", len(playersArray))
	mean, stdev := MeanStdDev(spreads)
	fmt.Printf("The average victor beat their opponent by %.4f points with a standard deviation of %.4f\n\n", mean, stdev)
	fmt.Printf("Assuming a 95%% confidence interval, the system correctly predicted %d results out of %d (%.4f)\n"+
		"when both players had a minimum rating deviation\n\n", correctGuesses, minDeviationGuesses, float64(correctGuesses)/float64(minDeviationGuesses))

	for i := 0; i < len(playersArray); i++ {
		if i < 20 || i > len(playersArray)-20 {
			fmt.Printf("%-5s %-24s %.2f (rd: %.2f, v: %.2f)\n", strconv.Itoa(i+1)+":", playersArray[i].name+":", playersArray[i].rating, playersArray[i].ratingDeviation, playersArray[i].volatility)
		}
	}

}
