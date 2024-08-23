package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/woogles-io/liwords/pkg/memento"
	pb "github.com/woogles-io/liwords/rpc/api/proto/game_service"
	"github.com/woogles-io/liwords/rpc/api/proto/game_service/game_serviceconnect"
)

func GetGameHistory(url string, id string) (*macondopb.GameHistory, error) {

	client := game_serviceconnect.NewGameMetadataServiceClient(http.DefaultClient, url+"/api")
	history, err := client.GetGameHistory(context.Background(),
		connect.NewRequest(&pb.GameHistoryRequest{GameId: id}))

	if err != nil {
		return &macondopb.GameHistory{}, err
	}
	return history.Msg.History, nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `usage of %s:

  params: gameId [n]
    fetch game from woogles.io or a compatible site
    example: hBQhT94n
    example: XgTRffsq 8
      n = next event number (so 1-based, matches ?turn= examiner param)

params can be prefixed with these flags:
`, os.Args[0])
		flag.PrintDefaults()
	}

	badUsage := func(err error) {
		flag.Usage()
		panic(err)
	}

	var colorFlag = flag.String("color", "", "0 = use player 0's colors, 1 = use player 1's colors")
	var gifFlag = flag.Bool("gif", false, "generate static gif")
	var agifFlag = flag.Bool("agif", false, "generate animated gif (version A)")
	var bgifFlag = flag.Bool("bgif", false, "generate animated gif (version B)")
	var cgifFlag = flag.Bool("cgif", false, "generate animated gif (version C)")
	var verFlag = flag.Int("ver", 0, "specify version")
	var urlFlag = flag.String("url", "https://woogles.io", "specify url, -url local for http://localhost")
	flag.Parse()
	args := flag.Args()

	if *urlFlag == "local" {
		*urlFlag = "http://localhost" // compatible with docker compose
	}

	var whichColor int
	switch *colorFlag {
	case "0":
		whichColor = 0
	case "1":
		whichColor = 1
	case "":
		whichColor = -1
	default:
		badUsage(fmt.Errorf("-color can only be 0 or 1 or \"\""))
	}

	if len(args) < 1 {
		badUsage(fmt.Errorf("not enough params"))
	}

	wf := memento.WhichFile{}
	wf.GameId = args[0]
	wf.NextEventNum = -1
	wf.Version = *verFlag
	wf.WhichColor = whichColor
	outputFilename := wf.GameId

	var outputFilenameSuffix string
	if len(args) > 1 {
		numEvents, err := strconv.Atoi(args[1])
		if err != nil {
			panic(err)
		}
		wf.HasNextEventNum = true
		wf.NextEventNum = numEvents
		outputFilenameSuffix += fmt.Sprintf("-%v", wf.NextEventNum)
	}
	if wf.Version != 0 {
		outputFilename += fmt.Sprintf("-v%v", wf.Version)
	}
	if *agifFlag {
		wf.FileType = "animated-gif"
		outputFilename += "-a"
		outputFilenameSuffix += ".gif"
	} else if *bgifFlag {
		wf.FileType = "animated-gif-b"
		outputFilename += "-b"
		outputFilenameSuffix += ".gif"
	} else if *cgifFlag {
		wf.FileType = "animated-gif-c"
		outputFilename += "-c"
		outputFilenameSuffix += ".gif"
	} else if *gifFlag {
		wf.FileType = "gif"
		outputFilenameSuffix += ".gif"
	} else {
		wf.FileType = "png"
		outputFilenameSuffix += ".png"
	}
	outputFilename += outputFilenameSuffix

	t0 := time.Now()

	hist, err := GetGameHistory(*urlFlag, wf.GameId)
	if err != nil {
		panic(err)
	}

	// Omit censoring.

	// Just following GameService.GetGameHistory although it doesn't matter.
	if hist.PlayState != macondopb.PlayState_GAME_OVER && !wf.HasNextEventNum {
		// This check is useless, GetGameHistory already checks for GAME_OVER.
		panic(fmt.Errorf("game is not over"))
	}

	if wf.HasNextEventNum && (wf.NextEventNum <= 0 || wf.NextEventNum > len(hist.Events)+1) {
		panic(fmt.Errorf("game only has %d events", len(hist.Events)))
	}

	t1 := time.Now()

	outputBytes, err := memento.RenderImage(hist, wf)
	if err != nil {
		panic(err)
	}

	t2 := time.Now()

	fmt.Printf("writing %d bytes\n", len(outputBytes))
	fmt.Println("downloading history", t1.Sub(t0))
	fmt.Println("rendering image", t2.Sub(t1))

	err = os.WriteFile(outputFilename, outputBytes, 0644)
	if err != nil {
		panic(err)
	}
}
