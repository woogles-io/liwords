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

	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
)

var RenderMutex sync.Mutex

type MementoService struct {
	userStore user.Store
	gameStore gameplay.GameStore
}

func NewMementoService(u user.Store, gs gameplay.GameStore) *MementoService {
	return &MementoService{u, gs}
}

type WhichFile struct {
	GameId          string
	HasNextEventNum bool
	NextEventNum    int
	FileType        string // "png" or "animated-gif"
	WhichColor      int    // 0, 1, or -1
}

var errInvalidFilename = fmt.Errorf("invalid filename")

func determineWhichFile(s string) (WhichFile, error) {
	var err error
	hasNextEventNum := false
	nextEventNum := -1
	fileType := ""
	if strings.HasSuffix(s, ".png") {
		// "gameid.png"
		// "gameid-num.png"
		fileType = "png"
		s = s[:len(s)-4]
		if v := strings.LastIndexByte(s, '-'); v >= 0 {
			nextEventNum, err = strconv.Atoi(s[v+1:])
			if err != nil || s[v+1:] != strconv.Itoa(nextEventNum) {
				// Fail because there's leading zero.
				return WhichFile{}, errInvalidFilename
			}
			hasNextEventNum = true
			s = s[:v]
		}
	} else if strings.HasSuffix(s, ".gif") {
		// "gameid-a.gif"
		// "gameid-a-num.gif"
		fileType = "animated-gif"
		s = s[:len(s)-4]
		if strings.HasSuffix(s, "-a") {
			s = s[:len(s)-2]
		} else if v := strings.LastIndexByte(s, '-'); v >= 0 {
			nextEventNum, err = strconv.Atoi(s[v+1:])
			if err != nil || s[v+1:] != strconv.Itoa(nextEventNum) {
				// Fail because there's leading zero.
				return WhichFile{}, errInvalidFilename
			}
			hasNextEventNum = true
			s = s[:v]
			if strings.HasSuffix(s, "-a") {
				s = s[:len(s)-2]
			} else {
				return WhichFile{}, errInvalidFilename
			}
		} else {
			return WhichFile{}, errInvalidFilename
		}
	} else {
		return WhichFile{}, errInvalidFilename
	}

	if len(s) == 0 || strings.IndexFunc(s, func(c rune) bool {
		return !strings.ContainsRune(shortuuid.DefaultAlphabet, c)
	}) != -1 {
		return WhichFile{}, errInvalidFilename
	}

	return WhichFile{
		GameId:          s,
		HasNextEventNum: hasNextEventNum,
		NextEventNum:    nextEventNum,
		FileType:        fileType,
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
