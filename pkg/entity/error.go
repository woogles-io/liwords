package entity

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

type WooglesError struct {
	code ipc.WooglesError
	data []string
}

const WooglesErrorDelimiter = ";"

func NewWooglesError(code ipc.WooglesError, data ...string) *WooglesError {
	log.Debug().Interface("data", data).Int32("code", int32(code)).Msg("NewWooglesError")
	return &WooglesError{
		code: code,
		data: data,
	}
}

func (w *WooglesError) Error() string {
	var errb strings.Builder
	fmt.Fprintf(&errb, "%s%d", WooglesErrorDelimiter, w.code)
	for _, d := range w.data {
		fmt.Fprintf(&errb, "%s%s", WooglesErrorDelimiter, d)
	}
	return errb.String()
}

// IsSeekGone reports whether err says the seek being accepted is no longer
// there: canceled, or taken by someone else.
func IsSeekGone(err error) bool {
	if errors.Is(err, ErrSeekGone) {
		return true
	}
	var we *WooglesError
	return errors.As(err, &we) && we.code == ipc.WooglesError_GAME_NO_LONGER_AVAILABLE
}
