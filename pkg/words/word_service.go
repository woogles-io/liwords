package words

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	pb "github.com/domino14/liwords/rpc/api/proto/word_service"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/domino14/macondo/kwg"
	"github.com/domino14/macondo/tilemapping"
	"github.com/rs/zerolog/log"
	"github.com/twitchtv/twirp"
)

type WordService struct {
	cfg               *macondoconfig.Config
	definitionSources map[string]*defSource
}

// NewWordService creates a Twirp WordService
func NewWordService(cfg *macondoconfig.Config) *WordService {
	dawgPath := filepath.Join(cfg.LexiconPath, "dawg")
	dawgDir, err := os.Open(dawgPath)
	var filenames []string
	if err != nil {
		log.Warn().Err(err).Msgf("cannot open directory %s", dawgPath)
	} else {
		filenames, err = dawgDir.Readdirnames(-1)
		if err != nil {
			log.Warn().Err(err).Msgf("cannot readdir %s", dawgPath)
		}
		dawgDir.Close()
	}

	definitionSources := make(map[string]*defSource)
	dictionaryPath := filepath.Join(cfg.LexiconPath, "words")
	for _, filename := range filenames {
		lexicon := strings.TrimSuffix(filename, ".dawg")
		if len(lexicon) == len(filename) {
			continue
		}
		definitionSource, err := loadDefinitionSource(filepath.Join(dictionaryPath, lexicon+".txt"))
		if err != nil {
			log.Warn().Err(err).Msgf("bad definition source for %s", lexicon)
		} else {
			definitionSources[lexicon] = definitionSource
			log.Info().Msgf("found definition source for %s", lexicon)
		}
	}

	return &WordService{cfg, definitionSources}
}

var daPool = sync.Pool{
	New: func() interface{} {
		return &kwg.KWGAnagrammer{}
	},
}

func (ws *WordService) DefineWords(ctx context.Context, req *pb.DefineWordsRequest) (*pb.DefineWordsResponse, error) {
	gd, err := kwg.Get(ws.cfg, req.Lexicon)
	if err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	alph := gd.GetAlphabet()
	definer, hasDefiner := ws.definitionSources[req.Lexicon]

	var wordsToDefine []string
	results := make(map[string]*pb.DefineWordsResult)
	var anagrams map[string][]string
	if req.Anagrams {
		anagrams = make(map[string][]string)
		da := daPool.Get().(*kwg.KWGAnagrammer)
		defer daPool.Put(da)
		for _, query := range req.Words {
			if _, found := anagrams[query]; found {
				continue
			}

			if strings.IndexByte(query, tilemapping.BlankToken) >= 0 {
				return nil, twirp.NewError(twirp.InvalidArgument, "word cannot have blanks")
			}
			if err = da.InitForString(gd, query); err != nil {
				return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
			}

			var words []string
			da.Anagram(gd, func(word tilemapping.MachineWord) error {
				words = append(words, word.UserVisible(alph))
				return nil
			})

			anagrams[query] = words
			if len(words) > 0 {
				for _, word := range words {
					if _, found := results[word]; found {
						continue
					}

					definition := ""
					if req.Definitions {
						// IMPORTANT: "" will make frontend do infinite loop
						definition = word // lame
						if hasDefiner {
							wordsToDefine = append(wordsToDefine, word)
						}
					}
					results[word] = &pb.DefineWordsResult{D: definition, V: true}
				}
			}
		}
	} else {
		for _, word := range req.Words {
			machineWord, err := tilemapping.ToMachineWord(word, alph)
			if err != nil {
				return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
			}

			if _, found := results[word]; found {
				continue
			}

			if kwg.FindMachineWord(gd, machineWord) {
				definition := ""
				if req.Definitions {
					// IMPORTANT: "" will make frontend do infinite loop
					definition = word // lame
					if hasDefiner {
						wordsToDefine = append(wordsToDefine, word)
					}
				}
				results[word] = &pb.DefineWordsResult{D: definition, V: true}
			} else {
				results[word] = &pb.DefineWordsResult{D: "", V: false}
			}
		}
	}

	if len(wordsToDefine) > 0 {
		// this must be sort|uniq and matching definitions file (all-caps)
		sort.Strings(wordsToDefine)
		definitions, err := definer.bulkDefine(wordsToDefine)
		if err != nil {
			log.Warn().Err(err).Msgf("cannot read %s definition", req.Lexicon)
		} else {
			for _, word := range wordsToDefine {
				if definition, ok := definitions[word]; ok && definition != "" {
					results[word].D = definition
				}
			}
		}
	}

	if req.Anagrams {
		originalResults := results
		results = make(map[string]*pb.DefineWordsResult)
		for _, query := range req.Words {
			if _, found := results[query]; found {
				continue
			}

			if words, found := anagrams[query]; found && len(words) > 0 {
				definitions := ""
				if req.Definitions {
					var definitionBytes []byte
					for _, word := range words {
						if len(definitionBytes) > 0 {
							definitionBytes = append(definitionBytes, '\n')
						}
						definitionBytes = append(definitionBytes, word...)
						definitionBytes = append(definitionBytes, " - "...)
						definitionBytes = append(definitionBytes, originalResults[word].D...)
					}
					definitions = string(definitionBytes)
				}
				results[query] = &pb.DefineWordsResult{D: definitions, V: true}
			} else {
				results[query] = &pb.DefineWordsResult{D: "", V: false}
			}
		}
	}

	return &pb.DefineWordsResponse{Results: results}, nil
}
