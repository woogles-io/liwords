package memento

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/pkg/config"
	"github.com/woogles-io/liwords/pkg/entity/utilities"
	"github.com/woogles-io/liwords/pkg/gameplay"
	"github.com/woogles-io/liwords/pkg/mod"
	"github.com/woogles-io/liwords/pkg/omgwords/stores"
	"github.com/woogles-io/liwords/pkg/user"
)

var RenderMutex sync.Mutex

type MementoService struct {
	userStore user.Store
	gameStore gameplay.GameStore
	// New stores. These will replace the game store eventually. Or something.
	gameDocumentStore *stores.GameDocumentStore
	cfg               *config.Config
}

func NewMementoService(u user.Store, gs gameplay.GameStore, gds *stores.GameDocumentStore,
	cfg *config.Config) *MementoService {
	return &MementoService{u, gs, gds, cfg}
}

type WhichFile struct {
	GameId          string
	HasNextEventNum bool
	NextEventNum    int
	FileType        string // "png", "gif", "animated-gif", "animated-gif-b", "animated-gif-c"
	WhichColor      int    // 0, 1, or -1
	Version         int
}

var errInvalidFilename = fmt.Errorf("invalid filename")

func determineWhichFile(s string) (WhichFile, error) {
	var err error
	fileType := ""
	hasNextEventNum := false
	nextEventNum := -1
	ver := 0

	// GAMEID, optional "-vVERSION", optional "-a"/"-b"/"-c" for gif, optional "-NEXTEVENTNUM", ".png"/".gif"

	v := strings.LastIndexByte(s, '.')
	if v < 0 {
		return WhichFile{}, errInvalidFilename
	} else if s[v:] == ".png" {
		fileType = "png"
	} else if s[v:] == ".gif" {
		fileType = "gif"
	} else {
		return WhichFile{}, errInvalidFilename
	}
	s = s[:v]

	nextToken := func() string {
		v := strings.IndexByte(s, '-')
		if v < 0 {
			ret := s
			s = s[len(s):]
			return ret
		} else {
			ret := s[:v]
			s = s[v+1:]
			return ret
		}
	}

	gameId := nextToken()
	if strings.IndexFunc(gameId, func(c rune) bool {
		return !strings.ContainsRune(shortuuid.DefaultAlphabet, c)
	}) != -1 {
		return WhichFile{}, errInvalidFilename
	}

	if strings.HasPrefix(s, "v") {
		tok := nextToken()[1:]
		ver, err = strconv.Atoi(tok)
		// Only -v2 and -v3 supported. Default (v0) should not be specified.
		if err != nil || tok != strconv.Itoa(ver) || (ver != 2 && ver != 3) {
			// Fail because there's leading zero.
			return WhichFile{}, errInvalidFilename
		}
	}

	if strings.HasPrefix(s, "a") {
		if fileType == "gif" && nextToken() == "a" {
			fileType = "animated-gif"
		} else {
			return WhichFile{}, errInvalidFilename
		}
	} else if strings.HasPrefix(s, "b") {
		if fileType == "gif" && nextToken() == "b" {
			fileType = "animated-gif-b"
		} else {
			return WhichFile{}, errInvalidFilename
		}
	} else if strings.HasPrefix(s, "c") {
		if fileType == "gif" && nextToken() == "c" {
			fileType = "animated-gif-c"
		} else {
			return WhichFile{}, errInvalidFilename
		}
	}

	if len(s) > 0 {
		nextEventNum, err = strconv.Atoi(s)
		if err != nil || s != strconv.Itoa(nextEventNum) {
			// Fail because there's leading zero.
			return WhichFile{}, errInvalidFilename
		}
		hasNextEventNum = true
	}

	return WhichFile{
		GameId:          gameId,
		HasNextEventNum: hasNextEventNum,
		NextEventNum:    nextEventNum,
		FileType:        fileType,
		Version:         ver,
		WhichColor:      -1,
	}, nil
}

func (ms *MementoService) loadAndRender(name string) ([]byte, error) {
	wf, err := determineWhichFile(name)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	hist, err := ms.gameStore.GetHistory(ctx, wf.GameId)
	if err != nil {
		return nil, err
	}
	if hist.Version == 0 {
		// A shortcut for a blank history. Look in the game document store.
		gdoc, err := ms.gameDocumentStore.GetDocument(ctx, wf.GameId, false)
		if err != nil {
			return nil, err
		}
		hist, err = utilities.ToGameHistory(gdoc.GameDocument, ms.cfg)
		if err != nil {
			return nil, err
		}
	}

	// Just following GameService.GetGameHistory although it doesn't matter.
	hist = mod.CensorHistory(ctx, ms.userStore, hist)
	if hist.PlayState != macondopb.PlayState_GAME_OVER && !wf.HasNextEventNum {
		return nil, fmt.Errorf("game is not over")
	}

	if wf.HasNextEventNum && (wf.NextEventNum <= 0 || wf.NextEventNum > len(hist.Events)+1) {
		return nil, fmt.Errorf("game only has %d events", len(hist.Events))
	}
	RenderMutex.Lock()
	defer RenderMutex.Unlock()

	return RenderImage(hist, wf)
}

func (ms *MementoService) GameimgEndpoint(w http.ResponseWriter, r *http.Request, name string) {
	b, err := ms.loadAndRender(name)
	if err != nil {
		log.Err(err).Str("name", name).Msg("memento-action-gameimg")
	} else {
		// If we can return the true modtime, it can be useful for caches.
		// usernames may be censored after the fact, but we are not rendering them.
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(b))
		return
	}

	http.NotFound(w, r)
}

// must end with /
const GameimgPrefix = "/gameimg/"

// impl http.Handler
func (ms *MementoService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, GameimgPrefix) {
		ms.GameimgEndpoint(w, r, strings.TrimPrefix(r.URL.Path, GameimgPrefix))
	} else {
		http.NotFound(w, r)
	}
}
