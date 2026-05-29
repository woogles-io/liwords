package gameplay

import (
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"

	"github.com/matryer/is"
	"github.com/woogles-io/liwords/pkg/config"
)

func TestCalculateReturnedTiles(t *testing.T) {
	is := is.New(t)
	cfg := config.DefaultConfig()

	testcases := []struct {
		letdist          string
		playerRack       string
		lastEventRack    string
		lastEventTiles   string
		expectedReturned string
	}{
		{"english", "ABCCDEF", "CCDJMUY", "JUM.Y", "ABEF"},
		{"english", "ZY??YVA", "VYAYNKE", "KA.VYEN", "??AVYZ"},
		{"english", "BEFITAR", "DANGLES", "GEN..DL.A....S", "ABEFIRT"},
	}

	for _, tc := range testcases {
		tiles, err := calculateReturnedTiles(cfg, tc.letdist, tc.playerRack, tc.lastEventRack, tc.lastEventTiles)
		is.NoErr(err)
		is.Equal(tiles, tc.expectedReturned)
	}

}

func TestSanitizeGCGNicknames(t *testing.T) {
	is := is.New(t)
	hist := &macondopb.GameHistory{
		Players: []*macondopb.PlayerInfo{
			{Nickname: "Richards, Nigel"},
			{Nickname: "Iamudom, Charnwit"},
		},
	}
	sanitizeGCGNicknames(hist)
	is.Equal(hist.Players[0].Nickname, "RichardsNigel")
	is.Equal(hist.Players[1].Nickname, "IamudomCharnwit")
}
