package words

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"connectrpc.com/connect"
	macondoconfig "github.com/domino14/macondo/config"
	"github.com/woogles-io/liwords/pkg/apiserver"
	"github.com/woogles-io/liwords/pkg/config"

	"github.com/domino14/word-golib/kwg"
	"github.com/domino14/word-golib/tilemapping"
	"github.com/rs/zerolog/log"
	pb "github.com/woogles-io/liwords/rpc/api/proto/word_service"
)

type WordService struct {
	cfg               *config.Config
	definitionSources map[string]*defSource
}

// NewWordService creates a WordService
func NewWordService(cfg *config.Config) *WordService {

	lexPath := filepath.Join(cfg.MacondoConfig().GetString(macondoconfig.ConfigDataPath), "lexica")
	kwgPath := filepath.Join(lexPath, "gaddag")
	pp := cfg.MacondoConfig().GetString(macondoconfig.ConfigKWGPathPrefix)
	if pp != "" {
		kwgPath = filepath.Join(kwgPath, pp)
	}
	kwgDir, err := os.Open(kwgPath)

	var filenames []string
	if err != nil {
		log.Warn().Err(err).Msgf("cannot open directory %s", kwgPath)
	} else {
		filenames, err = kwgDir.Readdirnames(-1)
		if err != nil {
			log.Warn().Err(err).Msgf("cannot readdir %s", kwgPath)
		}
		kwgDir.Close()
	}

	definitionSources := make(map[string]*defSource)
	dictionaryPath := filepath.Join(lexPath, "words")
	for _, filename := range filenames {
		lexicon := strings.TrimSuffix(filename, ".kwg")
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

func (ws *WordService) DefineWords(ctx context.Context, req *connect.Request[pb.DefineWordsRequest],
) (*connect.Response[pb.DefineWordsResponse], error) {
	gd, err := kwg.Get(ws.cfg.WGLConfig(), req.Msg.Lexicon)
	if err != nil {
		return nil, apiserver.InvalidArg(err.Error())
	}

	alph := gd.GetAlphabet()
	definer, hasDefiner := ws.definitionSources[req.Msg.Lexicon]

	var wordsToDefine []string
	results := make(map[string]*pb.DefineWordsResult)
	var anagrams map[string][]string
	if req.Msg.Anagrams {
		anagrams = make(map[string][]string)
		da := daPool.Get().(*kwg.KWGAnagrammer)
		defer daPool.Put(da)
		for _, query := range req.Msg.Words {
			if _, found := anagrams[query]; found {
				continue
			}

			if strings.IndexByte(query, tilemapping.BlankToken) >= 0 {
				return nil, apiserver.InvalidArg("word cannot have blanks")
			}
			if err = da.InitForString(gd, query); err != nil {
				return nil, apiserver.InvalidArg(err.Error())
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
					if req.Msg.Definitions {
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
		for _, word := range req.Msg.Words {
			machineWord, err := tilemapping.ToMachineWord(word, alph)
			if err != nil {
				return nil, apiserver.InvalidArg(err.Error())
			}

			if _, found := results[word]; found {
				continue
			}

			if kwg.FindMachineWord(gd, machineWord) {
				definition := ""
				if req.Msg.Definitions {
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
			log.Warn().Err(err).Msgf("cannot read %s definition", req.Msg.Lexicon)
		} else {
			for _, word := range wordsToDefine {
				if definition, ok := definitions[word]; ok && definition != "" {
					results[word].D = definition
				}
			}
		}
	}

	if req.Msg.Anagrams {
		originalResults := results
		results = make(map[string]*pb.DefineWordsResult)
		for _, query := range req.Msg.Words {
			if _, found := results[query]; found {
				continue
			}

			if words, found := anagrams[query]; found && len(words) > 0 {
				definitions := ""
				if req.Msg.Definitions {
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

	return connect.NewResponse(&pb.DefineWordsResponse{Results: results}), nil
}
