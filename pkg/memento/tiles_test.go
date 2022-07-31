package memento

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
)

func BenchmarkRenderAGif(b *testing.B) {
	is := is.New(b)
	gh, err := ioutil.ReadFile("./testdata/gh1.json")
	is.NoErr(err)
	hist := &macondopb.GameHistory{}
	err = json.Unmarshal([]byte(gh), hist)
	is.NoErr(err)
	wf := WhichFile{
		FileType:        "animated-gif",
		HasNextEventNum: false,
		Version:         2,
	}
	// benchmark runs around 250ms per render on my M1 Mac but it's significantly
	// slower when run within Docker for Mac. why?
	for i := 0; i < b.N; i++ {
		_, err := RenderImage(hist, wf)
		is.NoErr(err)
	}
}

func BenchmarkRenderBigAGif(b *testing.B) {
	is := is.New(b)
	hist := &macondopb.GameHistory{}
	bigGH, err := ioutil.ReadFile("./testdata/gh2.json")
	is.NoErr(err)
	err = json.Unmarshal([]byte(bigGH), hist)
	is.NoErr(err)
	wf := WhichFile{
		FileType:        "animated-gif",
		HasNextEventNum: false,
		Version:         2,
	}
	// Around 550ms on "themonolith" - 12th Gen Intel Linux computer.
	for i := 0; i < b.N; i++ {
		_, err := RenderImage(hist, wf)
		is.NoErr(err)
	}
}

func BenchmarkRenderPNG(b *testing.B) {
	is := is.New(b)
	hist := &macondopb.GameHistory{}
	gh, err := ioutil.ReadFile("./testdata/gh1.json")
	is.NoErr(err)
	err = json.Unmarshal([]byte(gh), hist)
	is.NoErr(err)
	wf := WhichFile{
		FileType:        "png",
		HasNextEventNum: false,
		Version:         2,
	}
	// benchmark runs around 109 ms
	for i := 0; i < b.N; i++ {
		_, err := RenderImage(hist, wf)
		is.NoErr(err)
	}
}
