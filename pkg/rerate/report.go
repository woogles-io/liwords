package rerate

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/glicko"
)

// ReportOptions shape the dry-run report.
type ReportOptions struct {
	MinPlayers int                 // pools with fewer established players are left out
	TopPools   []entity.VariantKey // pools that get a top list and movers
	TopN       int
}

type poolPlayer struct {
	id           int64
	v2, rd       float64
	games        int
	v1           float64
	hasV1        bool
	v1Est        bool
	v1Pct, v2Pct float64
}

// WriteReport compares the replay's ratings with the ratings saved when the
// data was read: per pool distributions, top lists, and who moves most
// relative to the rest of their pool.
func WriteReport(w io.Writer, sys gameplay.RatingSystem, d *Data, extraSkipped map[string]int, res *Result, opts ReportOptions) error {
	if opts.MinPlayers == 0 {
		opts.MinPlayers = 30
	}
	if opts.TopN == 0 {
		opts.TopN = 20
	}
	bw := bufio.NewWriter(w)
	p := func(format string, a ...any) { fmt.Fprintf(bw, format, a...) }

	p("Replay under rating system %d\n", sys.Version)
	p("Data read %s; %d finished rated games; last game replayed %s\n",
		time.Unix(d.ReadAt, 0).UTC().Format(time.RFC3339), len(d.Games),
		time.Unix(res.Last, 0).UTC().Format(time.RFC3339))
	p("Rated: %d\n", res.Rated)
	for _, m := range []map[string]int{d.Skipped, extraSkipped, res.Skipped} {
		for _, k := range sortedKeys(m) {
			p("Skipped, %s: %d\n", k, m[k])
		}
	}
	p("Moderator rating resets applied: %d\n", len(d.Resets))
	p("League results cancelled for cheating, applied: %d games\n", len(d.Penalties))
	below := 0
	for _, rs := range res.Ratings {
		for _, r := range rs {
			if r.Rating < 0 {
				below++
			}
		}
	}
	p("Ratings below 0 (any RD): %d\n\n", below)

	pools := map[entity.VariantKey][]*poolPlayer{}
	for id, rs := range res.Ratings {
		for key, r := range rs {
			if !sys.Established(r) {
				continue
			}
			pp := &poolPlayer{id: id, v2: r.Rating, rd: r.RatingDeviation, games: res.Games[id][key]}
			if cur, ok := d.Current[id][key]; ok {
				pp.v1, pp.hasV1 = cur.Rating, true
				pp.v1Est = cur.RatingDeviation <= glicko.V1.EstablishedRatingDeviation
			}
			pools[key] = append(pools[key], pp)
		}
	}
	keys := make([]entity.VariantKey, 0, len(pools))
	for k, v := range pools {
		if len(v) >= opts.MinPlayers {
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return len(pools[keys[i]]) > len(pools[keys[j]]) })

	p("Established players per pool (v2 RD <= %.0f). Percentiles 5/25/50/75/95.\n", sys.Glicko.EstablishedRatingDeviation)
	p("v1 is the same players' saved ratings.\n\n")
	p("%-28s %6s  %-32s %-32s %6s %6s\n", "pool", "n", "v2", "v1", "v2≥2000", "v1≥2000")
	for _, k := range keys {
		ps := pools[k]
		var v2, v1 []float64
		n2, n1 := 0, 0
		for _, pp := range ps {
			v2 = append(v2, pp.v2)
			if pp.v2 >= 2000 {
				n2++
			}
			if pp.hasV1 {
				v1 = append(v1, pp.v1)
				if pp.v1 >= 2000 {
					n1++
				}
			}
		}
		p("%-28s %6d  %-32s %-32s %6d %6d\n", k, len(ps), pctString(v2), pctString(v1), n2, n1)
	}

	for _, k := range opts.TopPools {
		ps := pools[k]
		if len(ps) == 0 {
			continue
		}
		setPercentiles(ps)
		sort.Slice(ps, func(i, j int) bool { return ps[i].v2 > ps[j].v2 })
		p("\nTop %d, %s\n", opts.TopN, k)
		p("%4s  %-20s %6s %4s %6s %6s\n", "rank", "player", "v2", "rd", "games", "v1")
		for i, pp := range ps[:min(opts.TopN, len(ps))] {
			p("%4d  %-20s %6.0f %4.0f %6d %6s\n", i+1, d.Users[pp.id].Username, pp.v2, pp.rd, pp.games, v1String(pp))
		}
		var moved []*poolPlayer
		for _, pp := range ps {
			if pp.hasV1 && pp.v1Est {
				moved = append(moved, pp)
			}
		}
		if len(moved) == 0 {
			continue
		}
		sort.Slice(moved, func(i, j int) bool { return moved[i].v2Pct-moved[i].v1Pct > moved[j].v2Pct-moved[j].v1Pct })
		over := 0
		for _, pp := range moved {
			if math.Abs(pp.v2Pct-pp.v1Pct) > 20 {
				over++
			}
		}
		p("\nMoves within %s, by percentile in the pool (players established in both): %d of %d move more than 20 points\n", k, over, len(moved))
		p("%-20s %6s %6s %6s %6s %6s\n", "player", "v1", "v1 pct", "v2", "v2 pct", "games")
		show := func(pp *poolPlayer) {
			p("%-20s %6.0f %6.0f %6.0f %6.0f %6d\n", d.Users[pp.id].Username, pp.v1, pp.v1Pct, pp.v2, pp.v2Pct, pp.games)
		}
		n := min(10, len(moved)/2)
		p("Up most:\n")
		for _, pp := range moved[:n] {
			show(pp)
		}
		p("Down most:\n")
		for _, pp := range moved[len(moved)-n:] {
			show(pp)
		}
	}
	return bw.Flush()
}

// setPercentiles sets each player's percentile in the pool under v1 (among
// players with a v1 rating) and v2.
func setPercentiles(ps []*poolPlayer) {
	byV2 := append([]*poolPlayer(nil), ps...)
	sort.Slice(byV2, func(i, j int) bool { return byV2[i].v2 < byV2[j].v2 })
	for i, pp := range byV2 {
		pp.v2Pct = 100 * float64(i) / float64(max(1, len(byV2)-1))
	}
	var byV1 []*poolPlayer
	for _, pp := range ps {
		if pp.hasV1 {
			byV1 = append(byV1, pp)
		}
	}
	sort.Slice(byV1, func(i, j int) bool { return byV1[i].v1 < byV1[j].v1 })
	for i, pp := range byV1 {
		pp.v1Pct = 100 * float64(i) / float64(max(1, len(byV1)-1))
	}
}

func pctString(v []float64) string {
	if len(v) == 0 {
		return "-"
	}
	sort.Float64s(v)
	var parts []string
	for _, q := range []float64{0.05, 0.25, 0.5, 0.75, 0.95} {
		parts = append(parts, fmt.Sprintf("%.0f", v[int(q*float64(len(v)-1))]))
	}
	return strings.Join(parts, "/")
}

func v1String(pp *poolPlayer) string {
	if !pp.hasV1 {
		return "-"
	}
	s := fmt.Sprintf("%.0f", pp.v1)
	if !pp.v1Est {
		s += "?"
	}
	return s
}

func sortedKeys(m map[string]int) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
