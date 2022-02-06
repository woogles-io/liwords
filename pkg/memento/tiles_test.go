package memento

import (
	"encoding/json"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"
)

const gh = `
{
  "events": [
    {
      "nickname": "doug",
      "rack": "DINNVWY",
      "cumulative": 32,
      "row": 7,
      "column": 3,
      "position": "8D",
      "played_tiles": "WINDY",
      "score": 32,
      "words_formed": ["WINDY"]
    },
    {
      "nickname": "emely",
      "rack": "ADEEGIL",
      "cumulative": 16,
      "row": 6,
      "column": 2,
      "position": "7C",
      "played_tiles": "GALE",
      "score": 16,
      "words_formed": ["GALE", "AW", "LI", "EN"]
    },
    {
      "nickname": "doug",
      "rack": "AEJNOSV",
      "cumulative": 66,
      "row": 2,
      "column": 4,
      "direction": 1,
      "position": "E3",
      "played_tiles": "JAVE..N",
      "score": 34,
      "words_formed": ["JAVELIN"]
    },
    {
      "nickname": "emely",
      "rack": "DEILOVX",
      "cumulative": 55,
      "row": 1,
      "column": 5,
      "direction": 1,
      "position": "F2",
      "played_tiles": "VOX",
      "score": 39,
      "words_formed": ["VOX", "JO", "AX"]
    },
    {
      "nickname": "doug",
      "rack": "ADENOST",
      "cumulative": 148,
      "row": 9,
      "column": 1,
      "position": "10B",
      "played_tiles": "DONATES",
      "score": 82,
      "is_bingo": true,
      "words_formed": ["DONATES", "JAVELINA"]
    },
    {
      "nickname": "emely",
      "rack": "DEIILTZ",
      "cumulative": 79,
      "row": 3,
      "column": 1,
      "position": "4B",
      "played_tiles": "TIL..",
      "score": 24,
      "words_formed": ["TILAX"]
    },
    {
      "nickname": "emely",
      "rack": "DEIILTZ",
      "type": 1,
      "cumulative": 55,
      "played_tiles": "TIL..",
      "lost_score": 24
    },
    {
      "nickname": "doug",
      "rack": "AAEINRU",
      "cumulative": 164,
      "row": 8,
      "column": 6,
      "position": "9G",
      "played_tiles": "EAU",
      "score": 16,
      "words_formed": ["EAU", "DEE", "YAS"]
    },
    {
      "nickname": "emely",
      "rack": "DEIILTZ",
      "cumulative": 93,
      "row": 3,
      "column": 3,
      "position": "4D",
      "played_tiles": "Z..",
      "score": 38,
      "words_formed": ["ZAX"]
    },
    {
      "nickname": "doug",
      "rack": "AILNORT",
      "cumulative": 191,
      "row": 4,
      "direction": 1,
      "position": "A5",
      "played_tiles": "LATINO",
      "score": 27,
      "words_formed": ["LATINO", "ODONATES"]
    },
    {
      "nickname": "emely",
      "rack": "DEEIILT",
      "cumulative": 122,
      "row": 1,
      "column": 1,
      "direction": 1,
      "position": "B2",
      "played_tiles": "TEIID",
      "score": 29,
      "words_formed": ["TEIID", "LI", "AD"]
    },
    {
      "nickname": "doug",
      "rack": "?BDERUW",
      "cumulative": 221,
      "direction": 1,
      "position": "A1",
      "played_tiles": "WEB",
      "score": 30,
      "words_formed": ["WEB", "ET", "BE"]
    },
    {
      "nickname": "emely",
      "rack": "AELLNST",
      "cumulative": 173,
      "row": 10,
      "column": 4,
      "position": "11E",
      "played_tiles": "SAT",
      "score": 51,
      "words_formed": ["SAT", "JAVELINAS", "TA", "DEET"]
    },
    {
      "nickname": "doug",
      "rack": "?DINRRU",
      "cumulative": 243,
      "row": 5,
      "column": 3,
      "position": "6D",
      "played_tiles": "R.D",
      "score": 22,
      "words_formed": ["RED", "RAW", "DEN"]
    },
    {
      "nickname": "emely",
      "rack": "ACELLMN",
      "cumulative": 196,
      "column": 2,
      "direction": 1,
      "position": "C1",
      "played_tiles": "CAN",
      "score": 23,
      "words_formed": ["CAN", "ETA", "BEN"]
    },
    {
      "nickname": "doug",
      "rack": "?EFINRU",
      "cumulative": 257,
      "row": 7,
      "column": 1,
      "direction": 1,
      "position": "B8",
      "played_tiles": "FU.",
      "score": 14,
      "words_formed": ["FUD", "IF", "NU"]
    },
    {
      "nickname": "emely",
      "rack": "EILLMRR",
      "cumulative": 208,
      "row": 6,
      "column": 7,
      "position": "7H",
      "played_tiles": "RILL",
      "score": 12,
      "words_formed": ["RILL", "RYAS"]
    },
    {
      "nickname": "doug",
      "rack": "?EIINOR",
      "cumulative": 335,
      "row": 4,
      "column": 10,
      "direction": 1,
      "position": "K5",
      "played_tiles": "RE.IgION",
      "score": 78,
      "is_bingo": true,
      "words_formed": ["RELIGION"]
    },
    {
      "nickname": "emely",
      "rack": "EKMORRU",
      "cumulative": 236,
      "row": 10,
      "column": 11,
      "direction": 1,
      "position": "L11",
      "played_tiles": "MURK",
      "score": 28,
      "words_formed": ["MURK", "OM", "NU"]
    },
    {
      "nickname": "doug",
      "rack": "AEIOORS",
      "cumulative": 368,
      "row": 14,
      "column": 7,
      "position": "15H",
      "played_tiles": "ARIOSE",
      "score": 33,
      "words_formed": ["ARIOSE", "MURKS"]
    },
    {
      "nickname": "emely",
      "rack": "?CEORUY",
      "cumulative": 255,
      "row": 13,
      "column": 5,
      "position": "14F",
      "played_tiles": "COY",
      "score": 19,
      "words_formed": ["COY", "YA"]
    },
    {
      "nickname": "doug",
      "rack": "EGHMOPT",
      "cumulative": 380,
      "row": 3,
      "column": 11,
      "direction": 1,
      "position": "L4",
      "played_tiles": "GET",
      "score": 12,
      "words_formed": ["GET", "RE", "ET"]
    },
    {
      "nickname": "emely",
      "rack": "?BERSTU",
      "cumulative": 264,
      "row": 1,
      "column": 5,
      "position": "2F",
      "played_tiles": ".ERB",
      "score": 9,
      "words_formed": ["VERB"]
    },
    {
      "nickname": "doug",
      "rack": "AEHIMOP",
      "cumulative": 409,
      "column": 6,
      "position": "1G",
      "played_tiles": "PEA",
      "score": 29,
      "words_formed": ["PEA", "PE", "ER", "AB"]
    },
    {
      "nickname": "emely",
      "rack": "?HOQSTU",
      "cumulative": 310,
      "row": 1,
      "column": 12,
      "direction": 1,
      "position": "M2",
      "played_tiles": "QUOTH",
      "score": 46,
      "words_formed": ["QUOTH", "GO", "RET", "ETH"]
    },
    {
      "nickname": "doug",
      "rack": "EGHIMOP",
      "cumulative": 451,
      "column": 13,
      "direction": 1,
      "position": "N1",
      "played_tiles": "HIM",
      "score": 42,
      "words_formed": ["HIM", "QI", "UM"]
    },
    {
      "nickname": "emely",
      "rack": "?FS",
      "cumulative": 331,
      "row": 13,
      "column": 11,
      "position": "14L",
      "played_tiles": ".aFS",
      "score": 21,
      "words_formed": ["KAFS", "AE"]
    },
    {
      "nickname": "emely",
      "rack": "OPEG",
      "type": 5,
      "cumulative": 345,
      "end_rack_points": 14
    }
  ],
  "players": [
    { "nickname": "doug", "real_name": "doug" },
    { "nickname": "emely", "real_name": "emely" }
  ],
  "version": 1,
  "original_gcg": "#player1 doug doug\n#player2 emely emely\n\u003edoug: DINNVWY 8D WINDY +32 32\n\u003eemely: ADEEGIL 7C GALE +16 16\n\u003edoug: AEJNOSV E3 JAVE..N +34 66\n\u003eemely: DEILOVX F2 VOX +39 55\n\u003edoug: ADENOST 10B DONATES +82 148\n\u003eemely: DEIILTZ 4B TIL.. +24 79\n\u003eemely: DEIILTZ --  -24 55\n\u003edoug: AAEINRU 9G EAU +16 164\n\u003eemely: DEIILTZ 4D Z.. +38 93\n\u003edoug: AILNORT A5 LATINO +27 191\n\u003eemely: DEEIILT B2 TEIID +29 122\n\u003edoug: ?BDERUW A1 WEB +30 221\n\u003eemely: AELLNST 11E SAT +51 173\n\u003edoug: ?DINRRU 6D R.D +22 243\n\u003eemely: ACELLMN C1 CAN +23 196\n\u003edoug: ?EFINRU B8 FU. +14 257\n\u003eemely: EILLMRR 7H RILL +12 208\n\u003edoug: ?EIINOR K5 RE.IgION +78 335\n\u003eemely: EKMORRU L11 MURK +28 236\n\u003edoug: AEIOORS 15H ARIOSE +33 368\n\u003eemely: ?CEORUY 14F COY +19 255\n\u003edoug: EGHMOPT L4 GET +12 380\n\u003eemely: ?BERSTU 2F .ERB +9 264\n\u003edoug: AEHIMOP 1G PEA +29 409\n\u003eemely: ?HOQSTU M2 QUOTH +46 310\n\u003edoug: EGHIMOP N1 HIM +42 451\n\u003eemely: ?FS 14L .aFS +21 331\n\u003eemely:  (OPEG) +14 345",
  "lexicon": "NWL20",
  "play_state": 2,
  "final_scores": [451, 345]
}
`

func BenchmarkRenderAGif(b *testing.B) {
	is := is.New(b)
	hist := &macondopb.GameHistory{}
	err := json.Unmarshal([]byte(gh), hist)
	is.NoErr(err)
	wf := whichFile{
		fileType:        "animated-gif",
		hasNextEventNum: false,
	}
	// benchmark runs around 250ms per render on my M1 Mac but it's significantly
	// slower when run within Docker for Mac. why?
	for i := 0; i < b.N; i++ {
		_, err := renderImage(hist, wf)
		is.NoErr(err)
	}
}

func BenchmarkRenderPNG(b *testing.B) {
	is := is.New(b)
	hist := &macondopb.GameHistory{}
	err := json.Unmarshal([]byte(gh), hist)
	is.NoErr(err)
	wf := whichFile{
		fileType:        "png",
		hasNextEventNum: false,
	}
	// benchmark runs around 109 ms
	for i := 0; i < b.N; i++ {
		_, err := renderImage(hist, wf)
		is.NoErr(err)
	}
}
