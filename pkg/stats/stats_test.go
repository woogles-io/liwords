package stats

import (
	"testing"
	"fmt"
	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/macondo/gcgio"
	//"github.com/matryer/is"
)

func InstantiateNewStatsWithHistory(filename string) (*entity.Stats, error) {
	stats := entity.InstantiateNewStats()
	history, err := gcgio.ParseGCG(filename)
	if err != nil {
		return nil, err
	}
	stats.AddGameToStats(history, filename)
	return stats, nil
}

func TestStats(t *testing.T) {
	//is := is.New(t)

	stats, _      := InstantiateNewStatsWithHistory("./testdata/doug_vs_emely.gcg")
	otherStats, _ := InstantiateNewStatsWithHistory("./testdata/noah_vs_mishu.gcg")
	stats.AddStatsToStats(otherStats)

	fmt.Println(stats.ToString())
}
