package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/domino14/liwords/pkg/config"
	"github.com/domino14/liwords/pkg/cwgame"
	"github.com/domino14/liwords/pkg/cwgame/board"
	"github.com/domino14/liwords/pkg/cwgame/runemapping"
	"github.com/domino14/liwords/pkg/cwgame/tiles"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/samber/lo"
	"google.golang.org/protobuf/encoding/protojson"
)

var RemoteServer = "https://woogles.io"

var DataDir = os.Getenv("DATA_PATH")
var DefaultConfig = &config.Config{DataPath: DataDir}

// a script to display the state of a game document
func main() {
	gameid := flag.String("gameid", "", "the game id you wish to obtain")
	server := flag.String("server", "prod", "the server you wish to read from: local or prod")
	file := flag.String("file", "", "the filename you wish to read from")
	flag.Parse()

	if *gameid == "" && *file == "" {
		panic("Need either a game ID or a file")
	}
	var serverURL string

	if *server == "prod" {
		serverURL = RemoteServer
	} else if *server == "local" {
		serverURL = "http://liwords.localhost"
	} else if *server == "" {
		serverURL = ""
	} else {
		panic("if specifying server, values can only be prod or local")
	}
	var body []byte
	var err error

	if *gameid != "" {

		url := serverURL + "/twirp/omgwords_service.GameEventService/GetGameDocument"
		reader := strings.NewReader(`{"gameId": "` + os.Args[1] + `"}`)
		resp, err := http.Post(url, "application/json", reader)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
	} else if *file != "" {
		body, err = os.ReadFile(*file)
		if err != nil {
			panic(err)
		}
	}

	gdoc := &ipc.GameDocument{}
	err = protojson.Unmarshal(body, gdoc)
	if err != nil {
		panic(err)
	}

	dist, err := tiles.GetDistribution(DefaultConfig, gdoc.LetterDistribution)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%-50s%-50s\n", gdoc.Players[0].RealName, gdoc.Players[1].RealName)
	fmt.Printf("%-50s%-50s\n", strings.Repeat("-", 20), strings.Repeat("-", 20))
	fullLine := false
	s1 := ""
	s2 := ""
	for _, evt := range gdoc.Events {
		if evt.PlayerIndex == 0 {
			if s1 != "" {
				s1 += "\n"
			}
			s1 += cwgame.EventDescription(evt, dist.RuneMapping())
		} else {
			if s2 != "" {
				s2 += "\n"
			}
			s2 += cwgame.EventDescription(evt, dist.RuneMapping())
			fullLine = true
		}
		if fullLine {
			fmt.Printf("%-50s%-50s\n", s1, s2)
			fullLine = false
			s1 = ""
			s2 = ""
		}
	}
	fmt.Println()
	s, err := board.ToUserVisibleString(gdoc.Board, gdoc.BoardLayout, dist.RuneMapping())
	if err != nil {
		panic(err)
	}
	fmt.Println(s)
	fmt.Println(lo.Map(gdoc.Players, func(p *ipc.GameDocument_MinimalPlayerInfo, idx int) string {
		pname := p.RealName
		pscore := gdoc.CurrentScores[idx]
		prack := runemapping.FromByteArr(gdoc.Racks[idx]).UserVisible(dist.RuneMapping())
		onturn := ""
		if int(gdoc.PlayerOnTurn) == idx {
			onturn = "*"
		}
		return fmt.Sprintf("<%s (rack [%s], score [%d])>%s", pname, prack, pscore, onturn)
	}))
	fmt.Printf("Bag: %v (%d)\n", runemapping.FromByteArr(gdoc.Bag.Tiles).UserVisible(dist.RuneMapping()), len(gdoc.Bag.Tiles))
}
