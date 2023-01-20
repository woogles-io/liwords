package omgwords

import (
	"testing"

	"github.com/domino14/liwords/pkg/omgwords/stores"
	"github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/matryer/is"
	"google.golang.org/protobuf/proto"
)

func TestMergeDocs(t *testing.T) {
	is := is.New(t)
	doc := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", RealName: "César Del Solar", UserId: "abc"},
			{Nickname: "mina", RealName: "Mina Bun", UserId: "mina"},
		},
		Description: "",
	}
	ld := &stores.MaybeLockedDocument{doc, ""}

	patchDoc := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", RealName: "César Del Solar", UserId: "abc"},
			{Nickname: "mina", RealName: "Mina Le", UserId: "mina"},
		},
		Description: "Vegas Worlds round 42",
	}

	MergeGameDocuments(ld.GameDocument, patchDoc)
	is.True(proto.Equal(ld.GameDocument, &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", RealName: "César Del Solar", UserId: "abc"},
			{Nickname: "mina", RealName: "Mina Le", UserId: "mina"},
		},
		Description: "Vegas Worlds round 42",
	}))
}

func TestMergeMoreComplexDocs(t *testing.T) {
	doc := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", RealName: "César Del Solar", UserId: "abc"},
			{Nickname: "mina", RealName: "Mina Bun", UserId: "mina"},
		},
		Lexicon:     "CSW21",
		Description: "",
		Racks: [][]byte{
			{0, 1, 2},
			{2, 4, 5},
		},
		Bag: &ipc.Bag{
			Tiles: []byte{0, 4, 10, 20},
		},
		Board: &ipc.GameBoard{
			Tiles:   []byte{3, 10, 11, 22},
			NumRows: 2,
			NumCols: 2,
			IsEmpty: false,
		},
		Timers: &ipc.Timers{
			TimeOfLastUpdate: 123,
			TimeRemaining:    []int64{100, 90},
		},
	}
	patchDoc := &ipc.GameDocument{
		Lexicon:     "NWL20",
		Description: "FOO",
		Racks: [][]byte{
			{},
			{2, 4, 5},
		},
		Bag: &ipc.Bag{
			Tiles: []byte{10, 15, 20},
		},
		Board: &ipc.GameBoard{
			Tiles:   []byte{2, 10, 10, 10},
			NumRows: 2,
			NumCols: 2,
			IsEmpty: false,
		},
		Timers: &ipc.Timers{
			TimeOfLastUpdate: 2546,
			TimeRemaining:    []int64{10, 80},
		},
	}
	err := MergeGameDocuments(doc, patchDoc)
	is := is.New(t)
	is.NoErr(err)

	expected := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", RealName: "César Del Solar", UserId: "abc"},
			{Nickname: "mina", RealName: "Mina Bun", UserId: "mina"},
		},
		Lexicon:     "NWL20",
		Description: "FOO",
		Racks: [][]byte{
			{},
			{2, 4, 5},
		},
		Bag: &ipc.Bag{
			Tiles: []byte{10, 15, 20},
		},
		Board: &ipc.GameBoard{
			Tiles:   []byte{2, 10, 10, 10},
			NumRows: 2,
			NumCols: 2,
			IsEmpty: false,
		},
		Timers: &ipc.Timers{
			TimeOfLastUpdate: 2546,
			TimeRemaining:    []int64{10, 80},
		},
	}

	is.True(proto.Equal(doc, expected))
}

// TestMergeKeepComplexValues tests that the source document "keeps" its
// other complex fields intact if not specified in the patch. (i.e.
// the proto message fields like Board, Bag, etc)
func TestMergeKeepComplexValues(t *testing.T) {
	doc := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", RealName: "César Del Solar", UserId: "abc"},
			{Nickname: "mina", RealName: "Mina Bun", UserId: "mina"},
		},
		Lexicon:     "CSW21",
		Description: "",
		Racks: [][]byte{
			{0, 1, 2},
			{2, 4, 5},
		},
		Bag: &ipc.Bag{
			Tiles: []byte{0, 4, 10, 20},
		},
		Board: &ipc.GameBoard{
			Tiles:   []byte{3, 10, 11, 22},
			NumRows: 2,
			NumCols: 2,
			IsEmpty: false,
		},
		Timers: &ipc.Timers{
			TimeOfLastUpdate: 123,
			TimeRemaining:    []int64{100, 90},
		},
	}
	patchDoc := &ipc.GameDocument{
		Lexicon:     "NWL20",
		Description: "FOO",
	}
	err := MergeGameDocuments(doc, patchDoc)
	is := is.New(t)
	is.NoErr(err)

	expected := &ipc.GameDocument{
		Players: []*ipc.GameDocument_MinimalPlayerInfo{
			{Nickname: "cesar", RealName: "César Del Solar", UserId: "abc"},
			{Nickname: "mina", RealName: "Mina Bun", UserId: "mina"},
		},
		Lexicon:     "NWL20",
		Description: "FOO",
		Racks: [][]byte{
			{0, 1, 2},
			{2, 4, 5},
		},
		Bag: &ipc.Bag{
			Tiles: []byte{0, 4, 10, 20},
		},
		Board: &ipc.GameBoard{
			Tiles:   []byte{3, 10, 11, 22},
			NumRows: 2,
			NumCols: 2,
			IsEmpty: false,
		},
		Timers: &ipc.Timers{
			TimeOfLastUpdate: 123,
			TimeRemaining:    []int64{100, 90},
		},
	}

	is.True(proto.Equal(doc, expected))
}
