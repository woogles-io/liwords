package gameplay

import (
	"testing"

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
