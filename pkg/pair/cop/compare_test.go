package cop_test

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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
		oppRegex := regexp.MustCompile(`^([\w, ':]+?)\s+(\d+)((?:\s+\d+)+)$`)

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
			if id == 0 {
				opponents[i] = playerID
			} else {
				opponents[i] = int32(id - 1)
			}
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
func getRoundInfo(oldLogFile string) (int, int, int, int64) {
	// Read the file content
	content, err := os.ReadFile(oldLogFile)
	if err != nil {
		panic(fmt.Sprintf("Error reading file: %v", err))
	}

	// Count the number of times the pattern `"pairings": [` appears in content
	numPairingsPattern := regexp.MustCompile(`"pairings":\s*\[`)
	numPairingsMatches := numPairingsPattern.FindAllStringSubmatch(string(content), -1)
	numPairings := len(numPairingsMatches)

	// Count the number of times the pattern `"results": [` appears in content
	numResultsPattern := regexp.MustCompile(`"results":\s*\[`)
	numResultsMatches := numResultsPattern.FindAllStringSubmatch(string(content), -1)
	numResults := len(numResultsMatches)

	// Find rounds remaining
	roundsRemainingPattern := regexp.MustCompile(`Rounds Remaining:\s(\d+)`)

	roundsRemainingMatches := roundsRemainingPattern.FindStringSubmatch(string(content))
	if len(roundsRemainingMatches) < 2 {
		panic("Error: Could not find rounds remaining in the log file")
	}

	roundsRemaining, err := strconv.Atoi(roundsRemainingMatches[1])
	if err != nil {
		panic(fmt.Sprintf("Error converting roundsRemaining to int: %v", err))
	}

	// Capture the seed pattern: "seed":  "\S+"
	seedPattern := regexp.MustCompile(`"seed":\s*"(\S+)"`)
	seedMatches := seedPattern.FindStringSubmatch(string(content))
	if len(seedMatches) < 2 {
		panic("Error: Could not find seed in the log file")
	}
	seedStr := seedMatches[1]

	seed, err := strconv.ParseInt(seedStr, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Error converting seed to int64: %v", err))
	}

	return numPairings, numResults, roundsRemaining, seed
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

	numPairings, numResults, roundsRemaining, seed := getRoundInfo(oldLogFile)

	// Division pairings and results
	var divisionResults []*pb.RoundResults
	for round := range numResults {
		var roundResults []int32
		for _, player := range players {
			roundResults = append(roundResults, player.Scores[round])
		}
		divisionResults = append(divisionResults, &pb.RoundResults{Results: roundResults})
	}

	var divisionPairings []*pb.RoundPairings
	for round := range numPairings {
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
		Seed:                       seed,
	}
}

// Parses the pairings from the old log file.
func parseOldPairings(filepath string) (map[int32]int32, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	pairings := make(map[int32]int32)
	scanner := bufio.NewScanner(file)
	inPairingsSection := false
	playerIDsRegex := regexp.MustCompile(`#(\d+)`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Pairings:") {
			inPairingsSection = true
			continue
		}
		if inPairingsSection {
			if strings.TrimSpace(line) == "" {
				continue
			}
			matches := playerIDsRegex.FindAllStringSubmatch(line, -1)

			if len(matches) != 2 {
				return nil, fmt.Errorf("error: expected 2 captures but found %d in line: %s", len(matches), line)
			}
			p0ID, err := strconv.Atoi(matches[0][1])
			if err != nil {
				return nil, err
			}
			p1ID, err := strconv.Atoi(matches[1][1])
			if err != nil {
				return nil, err
			}
			p0ID--
			p1ID--
			pairings[int32(p0ID)] = int32(p1ID)
			pairings[int32(p1ID)] = int32(p0ID)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return pairings, nil
}

func TestCompare(t *testing.T) {
	if os.Getenv("COP_CMP") == "" {
		t.Skip("Skipping COP comparison test. Use 'COP_CMP=1 go test -run Compare' to run it.")
	}

	is := is.New(t)

	tourneyName := flag.Arg(0)
	flag.Parse()

	// URLs and filepaths
	tfileURL := fmt.Sprintf("https://scrabbleplayers.org/directors/AA003954/%s/a.t", tourneyName)
	tFile := "a.t"
	configURL := fmt.Sprintf("https://scrabbleplayers.org/directors/AA003954/%s/config.tsh", tourneyName)
	configFile := "config.tsh"

	downloadSuccess := downloadFile(tfileURL, tFile, true)
	is.True(downloadSuccess)
	downloadSuccess = downloadFile(configURL, configFile, true)
	is.True(downloadSuccess)

	// Parse the .t and config.tsh files
	players, totalRounds, err := parseTFile(tFile)
	is.NoErr(err)

	// for _, player := range players {
	// 	fmt.Printf("Player ID: %d, Name: %s, Class: %d, Active: %v\n", player.ID, player.Name, player.Class, player.Active)
	// 	fmt.Printf("  Opponents: %v\n", player.Opponents)
	// 	fmt.Printf("  Scores: %v\n", player.Scores)
	// }

	config, err := parseConfigFile(configFile)
	is.NoErr(err)

	// fmt.Println("TSHConfig:")
	// fmt.Printf("UseCopAPI: %v\n", config.UseCopAPI)
	// fmt.Printf("Simulations: %d\n", config.Simulations)
	// fmt.Printf("AlwaysWinsSimulations: %d\n", config.AlwaysWinsSimulations)
	// fmt.Printf("GibsonSpread: %v\n", config.GibsonSpread)
	// fmt.Printf("ControlLossThresholds: %v\n", config.ControlLossThresholds)
	// fmt.Printf("Hopefulness: %v\n", config.Hopefulness)
	// fmt.Printf("ControlLossActivationRound: %d\n", config.ControlLossActivationRound)
	// fmt.Printf("ClassPrizes: %v\n", config.ClassPrizes)
	// fmt.Printf("NumPlacePrizes: %d\n", config.NumPlacePrizes)

	for round := range totalRounds {
		oldLogURL := fmt.Sprintf("https://scrabbleplayers.org/directors/AA003954/%s/html/A%d_cop.log", tourneyName, round+1)
		logTourneyAndRound := fmt.Sprintf("%s-%d", tourneyName, round+1)
		oldLogFile := fmt.Sprintf("%s-old.log", logTourneyAndRound)
		newLogFile := fmt.Sprintf("%s-new.log", logTourneyAndRound)
		downloadSuccess = downloadFile(oldLogURL, oldLogFile, false)
		if !downloadSuccess {
			fmt.Printf("No COP log found for %s in round %d\n", tourneyName, round+1)
			continue
		} else {
			fmt.Printf("Running %s round %d\n", tourneyName, round+1)
		}

		req := createPairRequest(players, totalRounds, config, oldLogFile)
		resp := cop.COPPair(req)
		writeStringToFile(newLogFile, resp.Log)
	}

}
