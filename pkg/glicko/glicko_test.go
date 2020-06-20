package glicko

import (
	"encoding/csv"
	"fmt"
	"github.com/matryer/is"
	"log"
	"sort"
	"math"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

type Player struct {
	id              int
	rating          float64
	ratingDeviation float64
	volatility      float64
}

type ByRating []*Player

func (a ByRating) Len() int           { return len(a) }
func (a ByRating) Less(i, j int) bool { return a[i].rating < a[j].rating }
func (a ByRating) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }


func CreatePlayers() [][]float64 {
	players := [][]float64{
		{460, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{440, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{410, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{380, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{370, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{360, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{340, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{310, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
		{220, 60, InitialVolatility, float64(InitialRating), float64(InitialRatingDeviation)},
	}
	return players
}

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
			RatingPeriodinSeconds*100000000)

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

func TestBehaviorUniformPairings(t *testing.T) {

	players := CreatePlayers()

	for i := 0; i < 10000; i++ {
		for j := 0; j < len(players); j++ {
			for k := j + 1; k < len(players); k++ {

				playerScore := int(rand.NormFloat64()*players[j][1] + players[j][0])
				oppponentScore := int(rand.NormFloat64()*players[k][1] + players[k][0])
				spread := playerScore - oppponentScore
				playerOriginalRating := players[j][3]
				playerOriginalRatingDeviation := players[j][4]

				players[j][3], players[j][4], players[j][2] =
					Rate(
						playerOriginalRating,
						playerOriginalRatingDeviation,
						players[j][2],
						players[k][3],
						players[k][4],
						spread,
						RatingPeriodinSeconds/12)

				players[k][3], players[k][4], players[k][2] =
					Rate(
						players[k][3],
						players[k][4],
						players[k][2],
						playerOriginalRating,
						playerOriginalRatingDeviation,
						-spread,
						RatingPeriodinSeconds/12)
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
				if j-k > 1 {
					continue
				}
				playerScore := int(rand.NormFloat64()*players[j][1] + players[j][0])
				oppponentScore := int(rand.NormFloat64()*players[k][1] + players[k][0])
				spread := playerScore - oppponentScore
				playerOriginalRating := players[j][3]
				playerOriginalRatingDeviation := players[j][4]

				players[j][3], players[j][4], players[j][2] =
					Rate(
						playerOriginalRating,
						playerOriginalRatingDeviation,
						players[j][2],
						players[k][3],
						players[k][4],
						spread,
						RatingPeriodinSeconds/12)

				players[k][3], players[k][4], players[k][2] =
					Rate(
						players[k][3],
						players[k][4],
						players[k][2],
						playerOriginalRating,
						playerOriginalRatingDeviation,
						-spread,
						RatingPeriodinSeconds/12)
			}
		}
	}
	fmt.Println("Stratified pairings")
	for i := range players {
		fmt.Println(players[i])
	}
}

func TestRealData(t *testing.T) {

	players := make(map[int]*Player)
	data := readCsvFile("crosstables_game_results.csv")

	for _, game := range data {
		for i := 0; i < 2; i++ {
			playerid, _ := strconv.Atoi(game[2+i])
			if _, ok := players[playerid]; !ok {
				players[playerid] = &Player{id: playerid,
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

	for i := 0; i < len(playersArray); i++ {
		fmt.Println(playersArray[i])
	}
	fmt.Println(len(playersArray))

}
