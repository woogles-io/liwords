package memento

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/domino14/liwords/pkg/gameplay"
	"github.com/domino14/liwords/pkg/mod"
	"github.com/domino14/liwords/pkg/user"
	macondopb "github.com/domino14/macondo/gen/api/proto/macondo"
	"github.com/lithammer/shortuuid"
	"github.com/rs/zerolog/log"
)

type MementoService struct {
	userStore user.Store
	gameStore gameplay.GameStore
}

func NewMementoService(u user.Store, gs gameplay.GameStore) *MementoService {
	return &MementoService{u, gs}
}

type whichFile struct {
	gameId          string
	hasNextEventNum bool
	nextEventNum    int
	fileType        string // "png" or "animated-gif"
}

var errInvalidFilename = fmt.Errorf("invalid filename")

func determineWhichFile(s string) (whichFile, error) {
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
				return whichFile{}, errInvalidFilename
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
				return whichFile{}, errInvalidFilename
			}
			hasNextEventNum = true
			s = s[:v]
			if strings.HasSuffix(s, "-a") {
				s = s[:len(s)-2]
			} else {
				return whichFile{}, errInvalidFilename
			}
		} else {
			return whichFile{}, errInvalidFilename
		}
	} else {
		return whichFile{}, errInvalidFilename
	}

	if len(s) == 0 || strings.IndexFunc(s, func(c rune) bool {
		return !strings.ContainsRune(shortuuid.DefaultAlphabet, c)
	}) != -1 {
		return whichFile{}, errInvalidFilename
	}

	return whichFile{
		gameId:          s,
		hasNextEventNum: hasNextEventNum,
		nextEventNum:    nextEventNum,
		fileType:        fileType,
	}, nil
}

func (ms *MementoService) loadAndRender(name string) ([]byte, error) {
	wf, err := determineWhichFile(name)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	hist, err := ms.gameStore.GetHistory(ctx, wf.gameId)
	if err != nil {
		return nil, err
	}

	// Just following GameService.GetGameHistory although it doesn't matter.
	hist = mod.CensorHistory(ctx, ms.userStore, hist)
	if hist.PlayState != macondopb.PlayState_GAME_OVER && !wf.hasNextEventNum {
		return nil, fmt.Errorf("game is not over")
	}

	if wf.hasNextEventNum && (wf.nextEventNum <= 0 || wf.nextEventNum > len(hist.Events)+1) {
		return nil, fmt.Errorf("game only has %d events", len(hist.Events))
	}

	return renderImage(hist, wf)
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
