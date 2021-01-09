package glicko

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"

	"github.com/matryer/is"
)

var file = flag.String("file", "", "File to use in the last test")

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

const epsilon = 1e-4

func withinEpsilon(a, b float64) bool {
	return math.Abs(float64(a-b)) < float64(epsilon)
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

	spreads := []int{SpreadScaling, SpreadScaling/2, SpreadScaling/4, 10, 1}

	for i := 0; i < len(spreads); i++ {
		spread := spreads[i]
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

func TestGamesToMinimumDeviation(t *testing.T) {

	rating := float64(InitialRating)
	deviation := float64(InitialRatingDeviation)
	volatility := InitialVolatility

	games_to_min := 0
	spread := 0
	opponent_rating := float64(InitialRating)

	for i := 0; i < 1000; i++ {
		games_to_min++
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
		if deviation < 1 + float64(MinimumRatingDeviation) {
			break
		}
	}

	fmt.Printf("Starting at a rating of %d and a deviation of %d,\n"+
		"it took %d games winning by %d points each against\n"+
		"a %.2f rated oppoent to get to a deviation of %.2f\n\n", InitialRating, InitialRatingDeviation, games_to_min, spread, opponent_rating, deviation)
}

/*func TestRatingManually(t *testing.T) {
	rating1 := 1600.0
	deviation1 := 350.0
	volatility1 := 0.06

	rating2 := 1300.0
	deviation2 := 80.0
	volatility2 := 0.06

	spread := 100

	// u1 = 0.57564624926 = (1600 - 1500) / 173.7178
	// u2 = -1.15129249852 = (1300 - 1500) / 173.7178
	// RD1 = 2.01476187242 = 350 / 173.7178
	// RD2 = 0.4605169994 = 80 / 173.7178
	// g1 = 0.96924739925 = 1 / sqrt(1 + 3*(0.4605169994)^2/(pi^2))
	// g2 = 0.66906941258 = 1 / sqrt(1 + 3*(2.01476187242)^2/(pi^2))
	// E1 = 0.84208591013 = 1 / (1 + exp(-1 * 0.96924739925 * (0.57564624926 - -1.15129249852)))
	// E2 = 0.23949650317 = 1 / (1 + exp(-1 * 0.66906941258 * (-1.15129249852 - 0.57564624926)))
	// var1 = 8.00485425177 = 1 / (0.96924739925^2 * 0.84208591013*(1-0.84208591013))
	// var2 = 12.2647092165 = 1 / (0.66906941258^2 * 0.23949650317*(1-0.23949650317s))

	// K = 400 = 4 * .25 * * 200 / (1 - 2*0.25)
	// r1 = 0.875 = 100 / (2 * 200 + 400) + 0.25 + 0.5
	// r2 = 0.125 = -100 / (2 * 200 + 400) - 0.25 + 0.5
	// d1 = 0.25537002787 = 8.00485425177*0.96924739925*(0.875 - 0.84208591013)
	// d2 = -0.93955164028 = 12.2647092165*0.66906941258*(0.125 - 0.23949650317)

	// RD1' = 1.64116641184 = 1 / sqrt( (1 / 2.01476187242^2) + (1 / 8.00485425177) )
	// RD2' = 0.45658637379 = 1 / sqrt( (1 / 0.4605169994^2) + (1 / 12.2647092165) )

	// u1' = 0.66157168341 = 0.57564624926 + 1.64116641184'^2 * 0.96924739925 * (0.875 - 0.84208591013)
	// u2' = -1.16726265943 = -1.15129249852 +  0.45658637379^2 * 0.66906941258 * (0.125 - 0.23949650317)

	// new rating1 = 1614.92677738 = 0.66157168341 * 173.7178 + 1500
	// new rating2 = 1297.22569878 = -1.16726265943 * 173.7178 + 1500

	// new rd1 = 285.099818499 = 1.64116641184 * 173.7178
	// new rd2 = 80
	newrating1, newdeviation1, newvolatility1 :=
		Rate(
			rating1,
			deviation1,
			volatility1,
			rating2,
			deviation2,
			spread,
			0)

	newrating2, newdeviation2, newvolatility2 :=
		Rate(
			rating2,
			deviation2,
			volatility2,
			rating1,
			deviation1,
			-spread,
			0)

	fmt.Printf("Manual ratings test:\n\n")
	fmt.Printf("Result: Player 1 wins by %d\n", spread)
	fmt.Printf("Player 1 (r, d, v): %f, %f, %f\n", newrating1, newdeviation1, newvolatility1)
	fmt.Printf("Player 2 (r, d, v): %f, %f, %f\n", newrating2, newdeviation2, newvolatility2)
	is := is.New(t)
	is.True(withinEpsilon(newrating1, 1614.926777))
	is.True(withinEpsilon(newrating2, 1297.225699))
	is.True(withinEpsilon(newdeviation1, 285.099818))
	is.True(withinEpsilon(newdeviation2, 80))
}*/

func TestRatingWeirdness(t *testing.T) {
	// New ratings were 1679
	// and 1476

	rating1 := 1800.0
	// deviation1 := 220.69 + 20.0
	deviation1 := float64(MinimumRatingDeviation)
	volatility1 := float64(InitialVolatility)

	rating2 := 1000.0
	// deviation2 := 138.1 + 20.0
	deviation2 := float64(MaximumRatingDeviation)
	volatility2 := float64(InitialVolatility)

	spread := SpreadScaling - 15

	newrating1, newdeviation1, newvolatility1 :=
		Rate(
			rating1,
			deviation1,
			volatility1,
			rating2,
			deviation2,
			spread,
			60*60*24)

	newrating2, newdeviation2, newvolatility2 :=
		Rate(
			rating2,
			deviation2,
			volatility2,
			rating1,
			deviation1,
			-spread,
			60*60*24)

	fmt.Println("\n\nProving that in some cases, both players may gain or lose rating.")
	fmt.Printf("\nPlayer 1 (r, d, v): %f, %f, %f\n", rating1, deviation1, volatility1)
	fmt.Printf("Player 2 (r, d, v): %f, %f, %f\n", rating2, deviation2, volatility2)
	fmt.Printf("Result: Player 1 wins by %d\n", spread)
	fmt.Printf("Player 1 (r, d, v): %f, %f, %f\n", newrating1, newdeviation1, newvolatility1)
	fmt.Printf("Player 2 (r, d, v): %f, %f, %f\n", newrating2, newdeviation2, newvolatility2)
	fmt.Printf("Differences: %f, %f\n\n\n", newrating1-rating1, newrating2-rating2)
}

func TestGamesToMinimumDeviationNewPlayers(t *testing.T) {

	rating1 := float64(InitialRating)
	deviation1 := float64(InitialRatingDeviation)
	volatility1 := InitialVolatility

	rating2 := float64(InitialRating)
	deviation2 := float64(InitialRatingDeviation)
	volatility2 := InitialVolatility

	games_to_min := 0
	spread := 0

	for i := 0; i < 1000; i++ {
		games_to_min++

		new_rating1, new_deviation1, new_volatility1 :=
			Rate(
				rating1,
				deviation1,
				volatility1,
				rating2,
				deviation2,
				spread,
				RatingPeriodinSeconds/12)
		rating2, deviation2, volatility2 =
			Rate(
				rating2,
				deviation2,
				volatility2,
				rating1,
				deviation1,
				spread,
				RatingPeriodinSeconds/12)
		rating1 = new_rating1
		deviation1 = new_deviation1
		volatility1 = new_volatility1
		// fmt.Printf("%.2f %.2f %.2f\n", rating, deviation, volatility)
		if deviation1 < 1 + float64(MinimumRatingDeviation) && deviation2 < 1 + float64(MinimumRatingDeviation) {
			break
		}
	}

	fmt.Printf("Starting at a rating of %d and a deviation of %d,\n"+
		"it took %d games winning by %d points each against\n"+
		"a %.2f rated oppoent with the maximum rating deviation\n"+
		"to get to a deviation of %.2f\n\n", InitialRating, InitialRatingDeviation, games_to_min, spread, rating2, deviation1)
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

func TestTradeoffs(t *testing.T) {
	winSpread := 0.0
	loseSpread := 0.0
	winResult := 0.0
	loseResult := 0.0

	winSpreadCap := 13
	loseSpreadCap := 13

	fmt.Print("      ")
	for i := 0; i < loseSpreadCap; i++ {
		loseSpread = float64(10 * (i + 1))
		fmt.Printf("%4d      |", int(loseSpread))
	}
	fmt.Println()
	for i := 0; i < winSpreadCap; i++ {
		winSpread = float64(10 * (i + 1))
		fmt.Printf("%4d |", int(winSpread))
		for j := 0; j < loseSpreadCap; j++ {
			winSpread = float64(10 * (i + 1))
			loseSpread = float64(10 * (j + 1))
			winResult  = 0.5 + WinBoost + ((0.5 - WinBoost) * math.Min(1.0, winSpread  / float64(SpreadScaling)))
			loseResult = 0.5 - WinBoost - ((0.5 - WinBoost) * math.Min(1.0, loseSpread / float64(SpreadScaling)))
			zeroEV := loseResult / (winResult - loseResult)
			fmt.Printf(" %7.4f%% |", zeroEV * 100)
		}
		fmt.Println()
	}
}

func TestRealData(t *testing.T) {

	players := make(map[int]*Player)

	datafile := "crosstables_game_results.csv"

	if len(*file) > 0 {
		datafile = *file
	}

	data := readCsvFile(datafile)
	predictions := 0
	correctPredictions := 0
	maxPlayerId := 2000000
	var spreads []float64

	outer: for _, game := range data {
		for i := 0; i < 2; i++ {
			playerid, _ := strconv.Atoi(game[2+i])
			if playerid > maxPlayerId {
				continue outer
			}
			playername := game[6+i]
			if _, ok := players[playerid]; !ok {
				players[playerid] = &Player{id: playerid,
					name:            playername,
					rating:          float64(InitialRating),
					ratingDeviation: float64(InitialRatingDeviation),
					volatility:      InitialVolatility}
			}
		}

		game_id, _ := strconv.Atoi(game[0])
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

		// Start predictin in 2018
		if game_id > 1865903 {
			predictions++
			if (playerOneRating > playerTwoRating && spread > 0) ||
				(playerOneRating < playerTwoRating && spread < 0) {
				correctPredictions++
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
	fmt.Printf("The system correctly predicted %d results out of %d (%.4f)\n\n", correctPredictions, predictions, float64(correctPredictions)/float64(predictions))

	for i := 0; i < len(playersArray); i++ {
		if i < 20 || i > len(playersArray)-20 {
			fmt.Printf("%-5s %-24s %.2f (rd: %.2f, v: %.2f)\n", strconv.Itoa(i+1)+":", playersArray[i].name+":", playersArray[i].rating, playersArray[i].ratingDeviation, playersArray[i].volatility)
		}
	}

}
