package broadcasts

import (
	"testing"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestComputeGameStat_Basic(t *testing.T) {
	doc := &ipc.GameDocument{
		CurrentScores: []int32{387, 412},
		Winner:        1,
		Events: []*ipc.GameEvent{
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 34, Cumulative: 34, PlayerIndex: 0, IsBingo: false, WordsFormedFriendly: []string{"WORD"}},
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 82, Cumulative: 82, PlayerIndex: 1, IsBingo: true, WordsFormedFriendly: []string{"RETAINS"}},
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 105, Cumulative: 139, PlayerIndex: 0, IsBingo: false, WordsFormedFriendly: []string{"QUIXOTE"}},
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 28, Cumulative: 110, PlayerIndex: 1, IsBingo: false, WordsFormedFriendly: []string{"ZAP"}},
			{Type: ipc.GameEvent_END_RACK_PTS, Score: 4, PlayerIndex: 0},
		},
	}

	s := ComputeGameStat(doc, 1820, 1875)

	if s.Player1Score != 387 {
		t.Errorf("Player1Score: got %d, want 387", s.Player1Score)
	}
	if s.Player2Score != 412 {
		t.Errorf("Player2Score: got %d, want 412", s.Player2Score)
	}
	if s.Winner != 1 {
		t.Errorf("Winner: got %d, want 1", s.Winner)
	}
	if s.Player1Bingos != 0 {
		t.Errorf("Player1Bingos: got %d, want 0", s.Player1Bingos)
	}
	if s.Player2Bingos != 1 {
		t.Errorf("Player2Bingos: got %d, want 1", s.Player2Bingos)
	}
	if s.MaxPlayScore != 105 {
		t.Errorf("MaxPlayScore: got %d, want 105", s.MaxPlayScore)
	}
	if s.MaxPlayWord != "QUIXOTE" {
		t.Errorf("MaxPlayWord: got %q, want QUIXOTE", s.MaxPlayWord)
	}
	if s.MoveCount != 4 {
		t.Errorf("MoveCount: got %d, want 4", s.MoveCount)
	}
	if s.WalkOffBingo {
		t.Error("WalkOffBingo should be false (last tile move was not a bingo)")
	}
	if s.Player1Rating != 1820 {
		t.Errorf("Player1Rating: got %d, want 1820", s.Player1Rating)
	}
}

func TestComputeGameStat_WalkOffBingo(t *testing.T) {
	doc := &ipc.GameDocument{
		CurrentScores: []int32{300, 320},
		Winner:        1,
		Events: []*ipc.GameEvent{
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 20, PlayerIndex: 0},
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 76, PlayerIndex: 1, IsBingo: true, WordsFormedFriendly: []string{"SINGLES"}},
			{Type: ipc.GameEvent_END_RACK_PTS, Score: 8, PlayerIndex: 0},
		},
	}

	s := ComputeGameStat(doc, 0, 0)
	if !s.WalkOffBingo {
		t.Error("WalkOffBingo should be true (last tile move was a bingo followed by END_RACK_PTS)")
	}
	if s.MoveCount != 2 {
		t.Errorf("MoveCount: got %d, want 2", s.MoveCount)
	}
}

func TestComputeGameStat_Exchange(t *testing.T) {
	doc := &ipc.GameDocument{
		CurrentScores: []int32{200, 180},
		Events: []*ipc.GameEvent{
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 30, PlayerIndex: 0},
			{Type: ipc.GameEvent_EXCHANGE, PlayerIndex: 1},
			{Type: ipc.GameEvent_TILE_PLACEMENT_MOVE, Score: 18, PlayerIndex: 0},
		},
	}

	s := ComputeGameStat(doc, 0, 0)
	if s.MoveCount != 3 {
		t.Errorf("MoveCount: got %d, want 3 (exchanges count)", s.MoveCount)
	}
}

func TestMarshalUnmarshalGameStat(t *testing.T) {
	orig := &GameStatJSON{
		Player1Score:  387,
		Player2Score:  412,
		Winner:        1,
		Player1Bingos: 2,
		Player2Bingos: 3,
		MaxPlayScore:  105,
		MaxPlayWord:   "QUIXOTE",
		MoveCount:     28,
		WalkOffBingo:  true,
		Player1Rating: 1820,
		Player2Rating: 1875,
	}

	data, err := MarshalGameStat(orig)
	if err != nil {
		t.Fatal(err)
	}

	got, err := UnmarshalGameStat(data)
	if err != nil {
		t.Fatal(err)
	}

	if got.Player1Score != orig.Player1Score || got.MaxPlayWord != orig.MaxPlayWord || !got.WalkOffBingo {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}
