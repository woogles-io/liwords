package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/domino14/liwords/pkg/config"
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
	if os.Getenv("DEBUG") == "1" {
		RemoteServer = "http://liwords.localhost"
	}
	if len(os.Args) != 2 {
		panic("need a game id argument")
	}
	url := RemoteServer + "/twirp/omgwords_service.GameEventService/GetGameDocument"
	reader := strings.NewReader(`{"gameId": "` + os.Args[1] + `"}`)
	resp, err := http.Post(url, "application/json", reader)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
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
	fmt.Println("Bag: ", runemapping.FromByteArr(gdoc.Bag.Tiles).UserVisible(dist.RuneMapping()))
}
