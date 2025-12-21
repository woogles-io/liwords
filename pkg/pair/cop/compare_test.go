package cop_test

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"regexp"
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

type OldLogData struct {
	TSHCfg      *TSHConfig
	Players     []Player
	TotalRounds int
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

// Parses a .t file to extract player information and total rounds.
func parseTFile(filepath string) ([]Player, int, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var players []Player
	totalRounds := -1
	playerID := int32(0)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		classRegex := regexp.MustCompile(`class\s+(\w+)`)
		classMatches := classRegex.FindStringSubmatch(line)
		if len(classMatches) != 2 {
			panic(fmt.Sprintf("Failed to find player class: %s\n", line))
		}
		playerClass := int32(rune(strings.ToUpper(classMatches[1])[0]) - 'A')
		parts := strings.Split(line, ";")
		// Define a regex to match the name, rating, and opponent IDs
		oppRegex := regexp.MustCompile(`^([\w, ']+?)\s+(\d+)((?:\s+\d+)+)$`)

		// Apply the oppRegex to the input string
		oppMatches := oppRegex.FindStringSubmatch(parts[0])

		if len(oppMatches) != 4 {
			panic(fmt.Sprintf("Failed to parse the input string: %s\n", parts[0]))
		}

		// Extract the name, rating, and opponent IDs
		name := strings.TrimSpace(oppMatches[1])
		// Ignore rating for now
		_, err := strconv.Atoi(oppMatches[2])
		if err != nil {
			panic(fmt.Sprintf("Error converting rating to integer: %s\n", err.Error()))
		}

		opponentIDs := strings.Fields(oppMatches[3]) // Split IDs into a slice of strings

		opponents := make([]int32, len(opponentIDs))
		for i, idStr := range opponentIDs {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				panic(fmt.Sprintf("Error converting opponent ID to integer: %s\n", err.Error()))
			}
			opponents[i] = int32(id - 1)
		}

		scoresStr := strings.Fields(parts[1])
		var scores []int32
		for _, score := range scoresStr {
			scoreNum, err := strconv.Atoi(score)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid score: %w", err)
			}
			scores = append(scores, int32(scoreNum))
		}

		if totalRounds == -1 {
			totalRounds = len(opponents)
		} else if totalRounds != len(opponents) {
			return nil, 0, fmt.Errorf("mismatched rounds count")
		}

		players = append(players, Player{
			ID:        playerID,
			Name:      name,
			Class:     playerClass,
			Opponents: opponents,
			Scores:    scores,
			Active:    true,
		})
		playerID++
	}

	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading file: %w", err)
	}

	return players, totalRounds, nil
}

func parseConfigFile(filepath string) (*TSHConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &TSHConfig{UseCopAPI: true}
	scanner := bufio.NewScanner(file)
	classPrizesMap := make(map[int32]int32)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "config") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(strings.TrimPrefix(parts[0], "config"))
			value := strings.TrimSpace(parts[1])

			switch key {
			case "simulations":
				if v, err := strconv.Atoi(value); err == nil {
					config.Simulations = int32(v)
				}
			case "always_wins_simulations":
				if v, err := strconv.Atoi(value); err == nil {
					config.AlwaysWinsSimulations = int32(v)
				}
			case "gibson_spread":
				config.GibsonSpread = parseInt32List(value)
			case "control_loss_thresholds":
				config.ControlLossThresholds = parseFloat64List(value)
			case "hopefulness":
				config.Hopefulness = parseFloat64List(value)
			case "control_loss_activation_round":
				if v, err := strconv.Atoi(value); err == nil {
					config.ControlLossActivationRound = int32(v - 1)
				}
			}
		} else {
			prizeRegex := regexp.MustCompile(`prize\s+rank\s+(\d+)\s+a(?:.*class.(\w+))?`)
			matches := prizeRegex.FindStringSubmatch(line)
			numMatches := len(matches)
			if numMatches >= 2 {
				prizeRank, err := strconv.Atoi(matches[1])
				if err != nil {
					return nil, fmt.Errorf("invalid prize rank: %w", err)
				}
				if numMatches >= 3 && matches[2] != "" {
					class := int32(rune(strings.ToUpper(matches[2])[0]) - 'A')
					if class > 4 {
						return nil, fmt.Errorf("invalid class: %s", line)
					}
					if class > 0 {
						numPrizesForClass, exists := classPrizesMap[class]
						if !exists || numPrizesForClass < int32(prizeRank) {
							classPrizesMap[class] = int32(prizeRank)
						}
					}
				} else if int32(prizeRank) > config.NumPlacePrizes {
					config.NumPlacePrizes = int32(prizeRank)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	numUnderClasses := len(classPrizesMap)
	config.ClassPrizes = make([]int32, numUnderClasses)
	for i := 0; i < numUnderClasses; i++ {
		config.ClassPrizes[i] = classPrizesMap[int32(i+1)]
	}

	return config, nil
}

func parseInt32List(value string) []int32 {
	value = strings.Trim(value, "[]")
	parts := strings.Split(value, ",")
	var result []int32
	for _, part := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			result = append(result, int32(v))
		}
	}
	return result
}

func parseFloat64List(value string) []float64 {
	value = strings.Trim(value, "[]")
	parts := strings.Split(value, ",")
	var result []float64
	for _, part := range parts {
		if v, err := strconv.ParseFloat(strings.TrimSpace(part), 32); err == nil {
			result = append(result, float64(v))
		}
	}
	return result
}

// getRoundInfo parses the log file and extracts information about the round and remaining rounds.
func getRoundInfo(oldLogFile string) (int, int, int) {
	// Read the file content
	content, err := os.ReadFile(oldLogFile)
	if err != nil {
		panic(fmt.Sprintf("Error reading file: %v", err))
	}

	// Define the regex patterns for extracting numPairings, numResults, and roundsRemaining
	roundInfoPattern := regexp.MustCompile(`round_(\d+)_based_on_(\d+)`)
	roundsRemainingPattern := regexp.MustCompile(`Rounds Remaining:\s+(\d+)`)

	// Find matches for round info (numPairings and numResults)
	roundInfoMatches := roundInfoPattern.FindStringSubmatch(string(content))
	if len(roundInfoMatches) < 3 {
		panic("Error: Could not find round info in the log file")
	}

	// Extract and calculate numPairings and numResults
	numPairings, err := strconv.Atoi(roundInfoMatches[1])
	if err != nil {
		panic(fmt.Sprintf("Error converting numPairings to int: %v", err))
	}
	numPairings -= 1

	numResults, err := strconv.Atoi(roundInfoMatches[2])
	if err != nil {
		panic(fmt.Sprintf("Error converting numResults to int: %v", err))
	}

	// Find matches for roundsRemaining
	roundsRemainingMatches := roundsRemainingPattern.FindStringSubmatch(string(content))
	if len(roundsRemainingMatches) < 2 {
		panic("Error: Could not find rounds remaining in the log file")
	}

	// Extract roundsRemaining
	roundsRemaining, err := strconv.Atoi(roundsRemainingMatches[1])
	if err != nil {
		panic(fmt.Sprintf("Error converting roundsRemaining to int: %v", err))
	}

	// Return the parsed values
	return numPairings, numResults, roundsRemaining
}

func createPairRequest(players []Player, totalRounds int, config *TSHConfig, oldLogFile string) *pb.PairRequest {
	var playerNames []string
	var playerClasses []int32
	var removedPlayers []int32

	// Determine class prizes and map TSH class to API class
	for _, player := range players {
		playerNames = append(playerNames, player.Name)
		playerClasses = append(playerClasses, player.Class)
	}

	numPairings, numResults, roundsRemaining := getRoundInfo(oldLogFile)

	// Division pairings and results
	var divisionResults []*pb.RoundResults
	for round := 0; round < numResults; round++ {
		var roundResults []int32
		for _, player := range players {
			roundResults = append(roundResults, player.Scores[round])
		}
		divisionResults = append(divisionResults, &pb.RoundResults{Results: roundResults})
	}

	var divisionPairings []*pb.RoundPairings
	for round := 0; round < numPairings; round++ {
		var roundPairings []int32
		for _, player := range players {
			roundPairings = append(roundPairings, player.Opponents[round])
		}
		divisionPairings = append(divisionPairings, &pb.RoundPairings{Pairings: roundPairings})
	}

	// Threshold logic
	apiControlLossThreshold := config.ControlLossThresholds[len(config.ControlLossThresholds)-1]
	apiHopefulness := config.Hopefulness[len(config.Hopefulness)-1]
	apiGibsonSpread := config.GibsonSpread[len(config.GibsonSpread)-1]

	if roundsRemaining < len(config.ControlLossThresholds) {
		apiControlLossThreshold = config.ControlLossThresholds[roundsRemaining-1]
	}
	if roundsRemaining < len(config.Hopefulness) {
		apiHopefulness = config.Hopefulness[roundsRemaining-1]
		if apiHopefulness == 0 {
			apiHopefulness = 0.1
		}
	}
	if roundsRemaining < len(config.GibsonSpread) {
		apiGibsonSpread = config.GibsonSpread[roundsRemaining-1]
	}

	return &pb.PairRequest{
		PairMethod:                 pb.PairMethod_COP,
		PlayerNames:                playerNames,
		PlayerClasses:              playerClasses,
		DivisionPairings:           divisionPairings,
		DivisionResults:            divisionResults,
		ClassPrizes:                config.ClassPrizes,
		GibsonSpread:               apiGibsonSpread,
		ControlLossThreshold:       apiControlLossThreshold,
		HopefulnessThreshold:       apiHopefulness,
		AllPlayers:                 int32(len(players)),
		ValidPlayers:               int32(len(players)) - int32(len(removedPlayers)),
		Rounds:                     int32(totalRounds),
		PlacePrizes:                config.NumPlacePrizes,
		DivisionSims:               config.Simulations,
		ControlLossSims:            config.AlwaysWinsSimulations,
		ControlLossActivationRound: config.ControlLossActivationRound,
		AllowRepeatByes:            false, // Example default
		RemovedPlayers:             removedPlayers,
		Seed:                       0,
	}
}

func TestCompare(t *testing.T) {
	tourneyName := os.Getenv("COP_TOURNEY")
	if tourneyName == "" {
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

	// tourneyName = "2025-12-12-Belleville-NWL-ME"
	// tourneyName = "2025-07-03-Albany-CSW-ME"

	previousRoundWasCOP := false
	var randomDivResults []*pb.RoundResults
	var randomDivPairings []*pb.RoundPairings
	var numResults int
	spreadsDist := standings.GetScoreDifferences()
	spreadsDistSize := len(spreadsDist)
	tourneyBaseURL := fmt.Sprintf("https://scrabbleplayers.org/directors/AA003954/%s", tourneyName)
	var oldLogData *OldLogData
	for round := startRound; ; round++ {
		oldLogURL := fmt.Sprintf("%s/html/A%d_cop.log", tourneyBaseURL, round+1)
		logTourneyAndRound := fmt.Sprintf("%s-%d", tourneyName, round+1)
		oldLogFile := fmt.Sprintf("%s-old.log", logTourneyAndRound)
		downloadSuccess := downloadFile(oldLogURL, oldLogFile, false)
		if !downloadSuccess {
			if !previousRoundWasCOP {
				fmt.Printf("No COP log found for %s in round %d\n", tourneyName, round+1)
			} else {
				break
			}
			continue
		}
		fmt.Printf("Running %s round %d\n", tourneyName, round+1)

		oldLogFileContent, err := os.ReadFile(oldLogFile)
		if err != nil {
			panic(fmt.Sprintf("Failed to read old log file %s: %s", oldLogFile, err.Error()))
		}
		var req *pb.PairRequest

		// The data extracted from the log is a text/JSON representation of the PairRequest
		splitOldLogFileContent := strings.Split(string(oldLogFileContent), "\nPair request:\n")
		if len(splitOldLogFileContent) != 2 {
			// assume this is the old log file
			if oldLogData == nil {
				// URLs and filepaths
				tfileURL := fmt.Sprintf("%s/a.t", tourneyBaseURL)
				tFile := "a.t"
				configURL := fmt.Sprintf("%s/config.tsh", tourneyBaseURL)
				configFile := "config.tsh"

				downloadSuccess := downloadFile(tfileURL, tFile, true)
				is.True(downloadSuccess)
				downloadSuccess = downloadFile(configURL, configFile, true)
				is.True(downloadSuccess)
				players, totalRounds, err := parseTFile(tFile)
				is.NoErr(err)

				config, err := parseConfigFile(configFile)
				is.NoErr(err)

				oldLogData = &OldLogData{
					TSHCfg:      config,
					Players:     players,
					TotalRounds: totalRounds,
				}
			}
			req = createPairRequest(oldLogData.Players, oldLogData.TotalRounds, oldLogData.TSHCfg, oldLogFile)
		} else {
			reqJSON := strings.TrimPrefix(splitOldLogFileContent[1], "\n")
			unmarshaler := protojson.UnmarshalOptions{
				// Allow string values to be unmarshaled into numeric fields (like "12345" -> 12345)
				AllowPartial:   true,
				DiscardUnknown: true,
			}
			req = &pb.PairRequest{}
			err = unmarshaler.Unmarshal([]byte(reqJSON), req)
			is.NoErr(err)
		}

		// Delete the file using os.Remove()
		if err := os.Remove(oldLogFile); err != nil {
			panic(fmt.Sprintf("Error removing file: %v", err))
		}

		var newLogFile string
		if useRandomScores {
			newLogFile = fmt.Sprintf("%s-new-random.log", logTourneyAndRound)
		} else {
			newLogFile = fmt.Sprintf("%s-new.log", logTourneyAndRound)
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
