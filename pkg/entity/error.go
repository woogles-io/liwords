package entity

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/domino14/liwords/rpc/api/proto/realtime"
)

type WooglesError struct {
	code realtime.WooglesError
	data []string
}

const WooglesErrorDelimiter = ";"

func NewWooglesError(code realtime.WooglesError, data ...string) *WooglesError {
	return &WooglesError{
		code: code,
		data: data,
	}
}

func (w *WooglesError) Error() string {
	var errb strings.Builder
	fmt.Fprintf(&errb, "%s%s%s", WooglesErrorDelimiter, strconv.Itoa(int(w.code)), WooglesErrorDelimiter)
	dataLength := len(w.data)
	delim := WooglesErrorDelimiter
	for i, d := range w.data {
		if i == dataLength-1 {
			delim = ""
		}
		fmt.Fprintf(&errb, "%s%s", d, delim)
	}
	return errb.String()
}
