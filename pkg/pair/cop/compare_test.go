package cop_test

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/pair/cop"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
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

func TestCompare(t *testing.T) {
	if os.Getenv("COP_CMP") == "" {
		t.Skip("Skipping COP comparison test. Use 'COP_CMP=1 go test -run Compare' to run it.")
	}

	is := is.New(t)

	tourneyName := "2025-07-03-Albany-CSW-ME"

	previousRoundWasCOP := false
	for round := 0; true; round++ {
		oldLogURL := fmt.Sprintf("https://scrabbleplayers.org/directors/AA003954/%s/html/A%d_cop.log", tourneyName, round+1)
		logTourneyAndRound := fmt.Sprintf("%s-%d", tourneyName, round+1)
		oldLogFile := fmt.Sprintf("%s-old.log", logTourneyAndRound)
		newLogFile := fmt.Sprintf("%s-new.log", logTourneyAndRound)
		downloadSuccess := downloadFile(oldLogURL, oldLogFile, false)
		if !downloadSuccess {
			fmt.Printf("No COP log found for %s in round %d\n", tourneyName, round+1)
			if previousRoundWasCOP {
				break
			}
			continue
		} else {
			fmt.Printf("Running %s round %d\n", tourneyName, round+1)
			previousRoundWasCOP = true
		}

		oldLogFileContent, err := os.ReadFile(oldLogFile)
		if err != nil {
			t.Fatalf("Failed to read old log file %s: %s", oldLogFile, err)
		}

		// The data extracted from the log is a text/JSON representation of the PairRequest
		reqJSON := strings.TrimPrefix(strings.Split(string(oldLogFileContent), "\nPair request:\n")[1], "\n")

		var req pb.PairRequest
		unmarshaler := protojson.UnmarshalOptions{
			// Allow string values to be unmarshaled into numeric fields (like "12345" -> 12345)
			AllowPartial:   true,
			DiscardUnknown: true,
		}

		err = unmarshaler.Unmarshal([]byte(reqJSON), &req)
		is.NoErr(err)

		resp := cop.COPPair(&req)
		writeStringToFile(newLogFile, resp.Log)
		is.Equal(resp.ErrorCode, pb.PairError_SUCCESS)
	}
}
