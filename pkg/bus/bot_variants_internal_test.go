package bus

import (
	"testing"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/matryer/is"

	"github.com/woogles-io/liwords/pkg/entity"
	pb "github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

// Bots play the super variants in every lexicon whose tiles have a super set,
// but BestBot still turns down ZOMGWords in each of them.
func TestBestBotDeclinesClassicSuper(t *testing.T) {
	is := is.New(t)
	ctx, b, st, users := readyTestSetup(t)
	is.NoErr(st.UserStore.New(ctx, &entity.User{Username: "variantbot", Email: "variantbot@example.com",
		UUID: "vbotaaaaaaaaaaaaaaaaaa", IsBot: true}))

	for _, lexicon := range []string{"CSW24", "DISC2", "RD29"} {
		err := b.newBotGame(ctx, &pb.SeekRequest{
			User: &pb.MatchUser{UserId: users[0].UUID},
			GameRequest: &pb.GameRequest{
				Lexicon:            lexicon,
				Rules:              &pb.GameRules{VariantName: "classic_super", BoardLayoutName: entity.CrosswordGame},
				BotType:            macondopb.BotRequest_SIMMING_BOT,
				InitialTimeSeconds: 600,
			},
		}, "")
		is.True(err != nil)
		is.Equal(err.Error(), "that variant is not supported for BestBot yet")
	}
}
