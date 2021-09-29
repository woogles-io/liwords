package entity

import (
	"strconv"
	"strings"

	"github.com/domino14/liwords/rpc/api/proto/realtime"
)

type WooglesError struct {
	code realtime.WooglesError
	data []string
}

const WooglesErrorDelimiter = ";"

func NewWooglesError(code realtime.WooglesError, data []string) *WooglesError {
	return &WooglesError{
		code: code,
		data: data,
	}
}

func (w *WooglesError) Error() string {
	return WooglesErrorDelimiter + strings.Join(append([]string{strconv.Itoa(int(w.code))}, w.data...), WooglesErrorDelimiter)
}
