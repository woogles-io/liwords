package game

// The guard has to fire. A check that cannot fail is worse than no check,
// because it reads like protection.

import (
	"strings"
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func goodArchive() *macondopb.GameHistory {
	return &macondopb.GameHistory{
		PlayState:      macondopb.PlayState_GAME_OVER,
		FinalScores:    []int32{412, 389},
		Players:        []*macondopb.PlayerInfo{{UserId: "a"}, {UserId: "b"}},
		LastKnownRacks: []string{"ABC", ""},
		Events: []*macondopb.GameEvent{
			{Type: macondopb.GameEvent_TILE_PLACEMENT_MOVE},
			{Type: macondopb.GameEvent_END_RACK_PTS, Rack: "ABC", PlayerIndex: 1},
		},
	}
}

func violations(t *testing.T, h *macondopb.GameHistory, reason pb.GameEndReason) []string {
	t.Helper()
	return archiveContractViolations(h, reason)
}

func TestArchiveContractAcceptsACompleteHistory(t *testing.T) {
	is := is.New(t)
	is.Equal(violations(t, goodArchive(), pb.GameEndReason_STANDARD), nil)
}

func TestArchiveContractCatchesEachOmission(t *testing.T) {
	for _, tc := range []struct {
		name   string
		break_ func(*macondopb.GameHistory)
		reason pb.GameEndReason
		want   string
	}{
		{"the play state the incident destroyed", func(h *macondopb.GameHistory) {
			h.PlayState = macondopb.PlayState_PLAYING
		}, pb.GameEndReason_STANDARD, "play_state"},
		{"final scores", func(h *macondopb.GameHistory) {
			h.FinalScores = nil
		}, pb.GameEndReason_STANDARD, "final_scores"},
		{"the racks a replay cannot derive", func(h *macondopb.GameHistory) {
			h.LastKnownRacks = []string{"ABC"}
		}, pb.GameEndReason_STANDARD, "last_known_racks"},
		{"the rack on an end-rack event", func(h *macondopb.GameHistory) {
			h.Events[1].Rack = ""
		}, pb.GameEndReason_STANDARD, "carries no rack"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			is := is.New(t)
			h := goodArchive()
			tc.break_(h)
			got := violations(t, h, tc.reason)
			is.True(len(got) > 0)
			is.True(strings.Contains(strings.Join(got, "; "), tc.want))
		})
	}
}

// An aborted game never produced a score, so requiring one would cry wolf on
// every abort. 21 of 6,031 finished games in the corpus are like this.
func TestArchiveContractExemptsAbortedGamesFromFinalScores(t *testing.T) {
	is := is.New(t)
	h := goodArchive()
	h.FinalScores = nil
	is.Equal(violations(t, h, pb.GameEndReason_ABORTED), nil)
	is.Equal(violations(t, h, pb.GameEndReason_CANCELLED), nil)
	// ...but a standard game with no score is still worth a line.
	is.True(len(violations(t, h, pb.GameEndReason_STANDARD)) > 0)
}
