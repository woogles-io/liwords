package entity

import (
	"errors"
	"fmt"
	"testing"

	"github.com/matryer/is"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
)

func TestIsSeekGone(t *testing.T) {
	is := is.New(t)
	is.True(IsSeekGone(ErrSeekGone))
	is.True(IsSeekGone(fmt.Errorf("claim: %w", ErrSeekGone)))
	is.True(IsSeekGone(NewWooglesError(ipc.WooglesError_GAME_NO_LONGER_AVAILABLE)))
	is.True(!IsSeekGone(NewWooglesError(ipc.WooglesError_DEFAULT)))
	is.True(!IsSeekGone(errors.New("connection refused")))
	is.True(!IsSeekGone(nil))
}
