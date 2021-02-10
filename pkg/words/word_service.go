package words

import (
	"context"

	pb "github.com/domino14/liwords/rpc/api/proto/word_service"
	"github.com/domino14/macondo/alphabet"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/gaddag"
	"github.com/twitchtv/twirp"
)

type WordService struct {
	cfg *macondoconfig.Config
}

// NewWordService creates a Twirp WordService
func NewWordService(cfg *macondoconfig.Config) *WordService {
	return &WordService{cfg}
}

func (ws *WordService) DefineWords(ctx context.Context, req *pb.DefineWordsRequest) (*pb.DefineWordsResponse, error) {
	gd, err := gaddag.GetDawg(ws.cfg, req.Lexicon)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	alph := gd.GetAlphabet()

	results := make(map[string]*pb.DefineWordsResult)
	for _, word := range req.Words {
		machineWord, err := alphabet.ToMachineWord(word, alph)
		if err != nil {
			return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
		}

		if gaddag.FindMachineWord(gd, machineWord) {
			results[word] = &pb.DefineWordsResult{W: word}
		}
	}

	return &pb.DefineWordsResponse{Results: results}, nil
}
