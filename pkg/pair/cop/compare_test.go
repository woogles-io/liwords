package cop_test

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	"github.com/woogles-io/liwords/pkg/pair/standings"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/encoding/protojson"
)

type TSHConfig struct {
	UseCopAPI                  bool
	Simulations                int32
	AlwaysWinsSimulations      int32
	GibsonSpread               []int32
	ControlLossThresholds      []float64
	Hopefulness                []float64
	ControlLossActivationRound int32
	ClassPrizes                []int32
	NumPlacePrizes             int32
}

type Player struct {
	ID        int32
	Name      string
	Class     int32
	Opponents []int32
	Scores    []int32
	Active    bool
}

// WriteStringToFile writes a string to a specified file.
func writeStringToFile(filename, content string) error {
	// Create or open the file for writing
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Write the string content to the file
	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	return nil
}

func downloadFile(url, filepath string, strict bool) bool {
	// Attempt to download the file
	resp, err := http.Get(url)
	if err != nil {
		if strict {
			panic(fmt.Sprintf("Failed to download file: %v", err))
		}
		return false
	}
	defer resp.Body.Close()

	// Check HTTP response status code
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		if strict {
			panic(err)
		}
		return false
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		panic(fmt.Sprintf("Failed to create file: %v", err))
	}
	defer out.Close()

	// Write the response body to the file
	_, err = bufio.NewReader(resp.Body).WriteTo(out)
	if err != nil {
		panic(fmt.Sprintf("Failed to write to file: %v", err))
	}

	return true
}

func getLogFilenames() {

}

func getReqFromLogFile(is *is.I, tourneyName string, round int, useRandomScores bool, previousRoundWasCOP bool) (*pb.PairRequest, string) {
	oldLogURL := fmt.Sprintf("https://scrabbleplayers.org/directors/AA003954/%s/html/A%d_cop.log", tourneyName, round+1)
	logTourneyAndRound := fmt.Sprintf("%s-%d", tourneyName, round+1)
	oldLogFile := fmt.Sprintf("%s-old.log", logTourneyAndRound)
	downloadSuccess := downloadFile(oldLogURL, oldLogFile, false)
	if !downloadSuccess {
		if !previousRoundWasCOP {
			fmt.Printf("No COP log found for %s in round %d\n", tourneyName, round+1)
		}
		return nil, ""
	}
	fmt.Printf("Running %s round %d\n", tourneyName, round+1)

	oldLogFileContent, err := os.ReadFile(oldLogFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to read old log file %s: %s", oldLogFile, err.Error()))
	}

	// The data extracted from the log is a text/JSON representation of the PairRequest
	reqJSON := strings.TrimPrefix(strings.Split(string(oldLogFileContent), "\nPair request:\n")[1], "\n")

	// Delete the file using os.Remove()
	if err := os.Remove(oldLogFile); err != nil {
		panic(fmt.Sprintf("Error removing file: %v", err))
	}

	var req pb.PairRequest
	unmarshaler := protojson.UnmarshalOptions{
		// Allow string values to be unmarshaled into numeric fields (like "12345" -> 12345)
		AllowPartial:   true,
		DiscardUnknown: true,
	}

	err = unmarshaler.Unmarshal([]byte(reqJSON), &req)
	is.NoErr(err)

	var newLogFilename string
	if useRandomScores {
		newLogFilename = fmt.Sprintf("%s-new-random.log", logTourneyAndRound)
	} else {
		newLogFilename = fmt.Sprintf("%s-new.log", logTourneyAndRound)
	}

	return &req, newLogFilename
}

func TestCompare(t *testing.T) {
	if os.Getenv("COP_CMP") == "" {
		t.Skip("Skipping COP comparison test. Use 'COP_CMP=1 go test -run Compare' to run it.")
	}

	var err error
	useRandomScores := false
	startRound := -1
	if os.Getenv("COP_SR") != "" {
		startRound, err = strconv.Atoi(os.Getenv("COP_SR"))
		if err != nil {
			panic(err)
		}
		startRound = startRound - 1
		useRandomScores = true
	} else {
		startRound = 0
	}

	is := is.New(t)

	tourneyName := "2025-12-12-Belleville-NWL-ME"
	// tourneyName := "2025-07-03-Albany-CSW-ME"

	previousRoundWasCOP := false
	var randomDivResults []*pb.RoundResults
	var randomDivPairings []*pb.RoundPairings
	var numResults int
	spreadsDist := standings.GetScoreDifferences()
	spreadsDistSize := len(spreadsDist)
	for round := startRound; ; round++ {
		req, newLogFile := getReqFromLogFile(is, tourneyName, round, useRandomScores, previousRoundWasCOP)
		if req == nil {
			if previousRoundWasCOP {
				break
			}
			continue
		}
		if useRandomScores {
			if !previousRoundWasCOP {
				randomDivResults = req.DivisionResults
				randomDivPairings = req.DivisionPairings
				numResults = len(randomDivResults[0].Results)
			} else {
				if len(randomDivPairings) != len(req.DivisionPairings) {
					panic(fmt.Sprintf("randomDivPairings (%d) and req.DivisionPairings (%d) should be same length", len(randomDivPairings), len(req.DivisionPairings)))
				}
				req.DivisionPairings = randomDivPairings
				if len(randomDivResults) < len(req.DivisionResults) {
					panic(fmt.Sprintf("randomDivResults (%d) should be >= req.DivisionResults (%d) should be same length", len(randomDivResults), len(req.DivisionResults)))
				}
				// Truncate the randomDivResults to the same length as req.DivisionResults
				roundResultsLength := len(req.DivisionResults)
				req.DivisionResults = randomDivResults[:roundResultsLength]
			}
		}
		resp := cop.COPPair(req)
		writeStringToFile(newLogFile, resp.Log)
		is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)

		if useRandomScores {
			randomRoundPairings := &pb.RoundPairings{
				Pairings: resp.Pairings,
			}
			randomDivPairings = append(randomDivPairings, randomRoundPairings)

			for len(randomDivResults) < len(randomDivPairings) {
				// add some random scores
				playedPairings := randomDivPairings[len(randomDivResults)].Pairings
				randomRoundResults := &pb.RoundResults{
					Results: make([]int32, numResults),
				}
				for i := 0; i < numResults; i++ {
					randomRoundResults.Results[i] = -1
				}
				for i := 0; i < numResults; i++ {
					opp := playedPairings[i]
					if opp == -1 {
						panic("Opponent index should not be -1")
					}
					if int32(i) == opp {
						randomRoundResults.Results[i] = 50
					} else if randomRoundResults.Results[i] == -1 {
						// Get a random value from spreadsDist
						spread := spreadsDist[rand.Intn(spreadsDistSize)]
						// Flip a coin
						if rand.Intn(2) == 0 {
							spread = -spread
						}
						randomRoundResults.Results[i] = 400 + int32(spread)
						randomRoundResults.Results[opp] = 400
					}
				}
				randomDivResults = append(randomDivResults, randomRoundResults)
			}
		}
		previousRoundWasCOP = true
	}
}
