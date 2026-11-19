package gameplay

import (
	"context"
	"strconv"
	"time"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/woogles-io/liwords/pkg/entity"
	"github.com/woogles-io/liwords/pkg/stores"
)

// A bot move request is published once and forgotten. If it is lost (NATS
// restarting, a bot task dying mid-move) or the bot's answer is, nothing asks
// again and the game sits with the bot on turn until an admin runs
// UnfreezeBot. RunBotWatchdog asks again.
//
// Asking again means a bot may answer the same position twice, or answer
// after the game has moved on. Each request therefore carries an ID, the
// number of events in the game when it was sent, which the bot echoes in its
// response; a response whose ID no longer matches the game is dropped
// (StaleBotResponse).

const (
	// BotWatchdogInterval is how often RunBotWatchdog looks for stuck games.
	BotWatchdogInterval = 30 * time.Second
	// BotTurnStaleAfter is how long a bot may be on turn, with nothing saved,
	// before its request is sent again. Bots other than BestBot answer in
	// seconds; a request resent early only costs a duplicate move that is
	// then dropped.
	BotTurnStaleAfter = 2 * time.Minute
)

// botRequestIDField is the field number of request_id in macondo's
// BotRequest and BotResponse.
//
// TODO: liwords is on a macondo release from before the field existed
// (domino14/macondo#584), so it is written and read as an unknown field.
// Use the generated RequestId once go.mod is on a release that has it.
const botRequestIDField protowire.Number = 5

// BotRequestID identifies the position a bot is asked to move in.
func BotRequestID(g *entity.Game) string {
	return strconv.Itoa(len(g.History().Events))
}

func setBotRequestID(req *macondopb.BotRequest, id string) {
	b := protowire.AppendTag(nil, botRequestIDField, protowire.BytesType)
	b = protowire.AppendString(b, id)
	req.ProtoReflect().SetUnknown(protoreflect.RawFields(b))
}

// BotResponseRequestID is the request ID the bot echoed, or "" for a bot
// that doesn't echo one.
func BotResponseRequestID(resp *macondopb.BotResponse) string {
	b := resp.ProtoReflect().GetUnknown()
	id := ""
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return id
		}
		b = b[n:]
		if num == botRequestIDField && typ == protowire.BytesType {
			v, m := protowire.ConsumeString(b)
			if m < 0 {
				return id
			}
			id = v // the last occurrence wins, as for any proto3 scalar
			b = b[m:]
			continue
		}
		m := protowire.ConsumeFieldValue(num, typ, b)
		if m < 0 {
			return id
		}
		b = b[m:]
	}
	return id
}

// StaleBotResponse says why a bot's move shouldn't be applied to g, or ""
// if it should. botOnTurn is whether the player on turn in g is a bot.
func StaleBotResponse(resp *macondopb.BotResponse, g *entity.Game, botOnTurn bool) string {
	if !botOnTurn {
		return "bot-not-on-turn"
	}
	if id := BotResponseRequestID(resp); id != "" && id != BotRequestID(g) {
		return "position-changed"
	}
	return ""
}

// RunBotWatchdog resends the move request of every game whose bot has been
// on turn for longer than staleAfter, every interval, until ctx is done. It
// is safe to run on every API server: ClaimStaleBotTurns hands each stuck
// game to one of them.
func RunBotWatchdog(ctx context.Context, s *stores.Stores, interval, staleAfter time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ResendStaleBotRequests(ctx, s, staleAfter)
		}
	}
}

// ResendStaleBotRequests is one pass of RunBotWatchdog. It returns the IDs
// of the games it claimed.
func ResendStaleBotRequests(ctx context.Context, s *stores.Stores, staleAfter time.Duration) []string {
	gameIDs, err := s.Queries.ClaimStaleBotTurns(ctx, staleAfter.Seconds())
	if err != nil {
		log.Err(err).Msg("bot-watchdog-claim")
		return nil
	}
	for _, gid := range gameIDs {
		if err := resendBotRequest(ctx, s, gid); err != nil {
			log.Err(err).Str("gid", gid).Msg("bot-watchdog-resend")
		}
	}
	return gameIDs
}

func resendBotRequest(ctx context.Context, s *stores.Stores, gid string) error {
	// Under the lock, so the request is for the position the game is in and
	// not one a bot move is about to replace.
	lock, err := s.GameStore.LockGame(ctx, gid)
	if err != nil {
		return err
	}
	defer lock.Unlock(ctx)
	g, err := s.GameStore.Get(ctx, gid)
	if err != nil {
		return err
	}
	log.Warn().Str("gid", gid).Str("request-id", BotRequestID(g)).Msg("bot-watchdog-resending-move-request")
	return PotentiallySendBotMoveRequest(ctx, s.UserStore, g)
}
