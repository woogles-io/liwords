package rerate

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/gameplay"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// WriteGamesTSV writes the games that ended in [since, until) for checking a
// replay against another implementation: end time, the rating key's lexicon,
// variant and time control, both players' ids, usernames and bot flags, the
// rating spread, and whether the game ended by a quit. Games whose result
// can't be rated are left out.
func WriteGamesTSV(w io.Writer, d *Data, since, until int64) error {
	bw := bufio.NewWriter(w)
	for i := range d.Games {
		g := &d.Games[i]
		if (since != 0 && g.End < since) || (until != 0 && g.End >= until) {
			continue
		}
		spread, err := gameplay.RatingSpread(g.Reason, int(g.Winner), g.Scores[0], g.Scores[1])
		if err != nil {
			continue
		}
		parts := strings.SplitN(string(d.Keys[g.Key]), ".", 3)
		if len(parts) != 3 {
			continue
		}
		quit := g.Reason == pb.GameEndReason_RESIGNED || g.Reason == pb.GameEndReason_TIME ||
			g.Reason == pb.GameEndReason_FORCE_FORFEIT || g.Reason == pb.GameEndReason_TRIPLE_CHALLENGE
		u0, u1 := d.Users[g.Players[0]], d.Users[g.Players[1]]
		fmt.Fprintf(bw, "%s\t%d\t%s\t%s\t%s\t%d\t%d\t%s\t%s\t%t\t%t\t%d\t%t\n",
			time.Unix(g.End, 0).UTC().Format("2006-01-02T15:04:05"), g.ID, parts[0], parts[1], parts[2],
			g.Players[0], g.Players[1], u0.Username, u1.Username,
			u0.Bot || g.QuickdataBot[0], u1.Bot || g.QuickdataBot[1], spread, quit)
	}
	return bw.Flush()
}

// WriteRatingsTSV writes a replay's ratings: user id, the rating key's
// lexicon, variant and time control, rating, RD, and rated games.
func WriteRatingsTSV(w io.Writer, res *Result) error {
	bw := bufio.NewWriter(w)
	ids := make([]int64, 0, len(res.Ratings))
	for id := range res.Ratings {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		for key, r := range res.Ratings[id] {
			parts := strings.SplitN(string(key), ".", 3)
			if len(parts) != 3 {
				continue
			}
			fmt.Fprintf(bw, "%d\t%s\t%s\t%s\t%.6f\t%.6f\t%d\n", id, parts[0], parts[1], parts[2],
				r.Rating, r.RatingDeviation, res.Games[id][key])
		}
	}
	return bw.Flush()
}
