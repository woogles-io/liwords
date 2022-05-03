package tournament

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/domino14/liwords/pkg/entity"
	"github.com/domino14/liwords/pkg/user"
	ipc "github.com/domino14/liwords/rpc/api/proto/ipc"
	"github.com/rs/zerolog/log"
)

func exportTournament(ctx context.Context, t *entity.Tournament, us user.Store, format string) (string, error) {
	switch format {
	case "tsh":
		return exportToTSH(ctx, t, us)
	case "standingsonly":
		return exportStandings(ctx, t)
	default:
		return "", fmt.Errorf("the format %s is not supported", format)
	}
}

func exportToTSH(ctx context.Context, t *entity.Tournament, us user.Store) (string, error) {
	// http://scrabbleplayers.org/w/Submitting_tournament_results#No_software
	var sb strings.Builder
	for dname, division := range t.Divisions {
		sb.WriteString("#division ")
		sb.WriteString(strings.ReplaceAll(dname, " ", ""))
		sb.WriteString("\n")
		sb.WriteString("#ratingcheck off\n")
		if division.DivisionManager == nil {
			return "", errors.New("nil division manager")
		}
		xhr, err := division.DivisionManager.GetXHRResponse()
		if err != nil {
			return "", err
		}

		// pre-parse pairing map to allow for faster lookup
		// create a map of user indexes to pairings with potential duplicates
		biggerMap := map[string]*ipc.Pairing{}
		for _, pairing := range xhr.PairingMap {
			for _, p := range pairing.Players {
				key := fmt.Sprintf("%d-%d", p, pairing.Round)
				biggerMap[key] = pairing
			}
		}

		for pidx, p := range xhr.Players.Persons {
			split := strings.Split(p.Id, ":")
			if len(split) != 2 {
				return "", fmt.Errorf("unexpected badly formatted player id %s", p.Id)
			}
			u, err := us.GetByUUID(ctx, split[0])
			if err != nil {
				return "", err
			}
			realName := u.RealName()
			if realName == "" {
				realName = u.Username
			}
			// try to split
			split = strings.SplitN(realName, " ", 2)
			if len(split) == 2 {
				realName = split[1] + ", " + split[0]
			}
			sb.WriteString(realName)
			sb.WriteString("\t")
			sb.WriteString(strconv.Itoa(int(p.Rating)))
			scores := make([]int, xhr.CurrentRound+1)
			// Write all pairings and then scores.
			for rd := int32(0); rd <= xhr.CurrentRound; rd++ {
				var score int
				key := fmt.Sprintf("%d-%d", pidx, rd)
				pairing := biggerMap[key]
				if pairing == nil {
					log.Info().Int32("rd", rd).Int("p", pidx).Msg("nil-pairing")
					continue
				}
				if pairing.Players[0] == pairing.Players[1] {
					// It's a bye; write a 0.
					sb.WriteString(" 0")
					switch pairing.Outcomes[0] {
					case ipc.TournamentGameResult_BYE, ipc.TournamentGameResult_FORFEIT_WIN:
						score = 50
					case ipc.TournamentGameResult_FORFEIT_LOSS:
						score = -50
					case ipc.TournamentGameResult_VOID:
						score = 0
					default:
						return "", fmt.Errorf("unexpected tournament game result; rd %v outcome %v pidx %v", rd, pairing.Outcomes[0], pidx)
					}
				} else {
					for idx, opp := range pairing.Players {
						if int(opp) != pidx {
							// This is the opponent.
							// Player-indexes are 1-indexed:
							sb.WriteString(" ")
							sb.WriteString(strconv.Itoa(int(opp + 1)))
							// This assumes 2 players per game but we've already made our bed:
							score = int(pairing.Games[0].Scores[1-idx])
						}
					}
				}
				scores[rd] = score
			}
			sb.WriteString(";")
			for _, score := range scores {
				sb.WriteString(" ")
				sb.WriteString(strconv.Itoa(score))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}

func exportStandings(ctx context.Context, t *entity.Tournament) (string, error) {
	var sb strings.Builder
	sb.WriteString("division,rank,username,wins,losses,draws,winpts,spread\n")
	for dname, division := range t.Divisions {

		if division.DivisionManager == nil {
			return "", errors.New("nil division manager")
		}
		xhr, err := division.DivisionManager.GetXHRResponse()
		if err != nil {
			return "", err
		}
		rdStandings := xhr.Standings[xhr.CurrentRound]
		if rdStandings == nil {
			return "", errors.New("round standings are nil?")
		}
		for idx, std := range rdStandings.Standings {
			p := std.PlayerId
			split := strings.Split(p, ":")
			if len(split) != 2 {
				return "", fmt.Errorf("unexpected badly formatted player id %s", p)
			}
			username := split[1]
			sb.WriteString(dname)
			sb.WriteString(",")
			sb.WriteString(strconv.Itoa(idx + 1))
			sb.WriteString(",")
			sb.WriteString(username)
			sb.WriteString(",")
			sb.WriteString(strconv.Itoa(int(std.Wins)))
			sb.WriteString(",")
			sb.WriteString(strconv.Itoa(int(std.Losses)))
			sb.WriteString(",")
			sb.WriteString(strconv.Itoa(int(std.Draws)))
			sb.WriteString(",")
			winpts := float32(std.Wins) + 0.5*float32(std.Draws)
			sb.WriteString(strconv.FormatFloat(float64(winpts), 'f', 1, 64))
			sb.WriteString(",")
			sb.WriteString(strconv.Itoa(int(std.Spread)))
			sb.WriteString("\n")
		}
	}
	return sb.String(), nil
}
