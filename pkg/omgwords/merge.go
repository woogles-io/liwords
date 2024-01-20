package omgwords

import (
	"errors"

	"github.com/woogles-io/liwords/rpc/api/proto/ipc"
	"google.golang.org/protobuf/proto"
)

// MergeGameDocuments merges the src document into the dst document.
// We don't use proto.Merge directly since we want to replace most arrays if
// they're specified in both documents.
func MergeGameDocuments(dst *ipc.GameDocument, src *ipc.GameDocument) error {
	if len(dst.Players) > 0 && len(src.Players) > 0 {
		if len(src.Players) != len(dst.Players) {
			return errors.New("must have same number of players if specified")
		}

		dst.Players = nil

	}
	if len(dst.Events) > 0 && len(src.Events) > 0 {
		// # of events don't need to match here. If both specify it, do a complete
		// replacement. This allows us to overwrite history if the annotator
		// makes a mistake and wants to fix it.
		dst.Events = nil
	}
	if len(dst.Racks) > 0 && len(src.Racks) > 0 {
		if len(src.Racks) != len(dst.Racks) {
			return errors.New("must have same number of racks if specified")
		}

		dst.Racks = nil
	}
	if len(dst.CurrentScores) > 0 && len(src.CurrentScores) > 0 {
		if len(src.CurrentScores) != len(dst.CurrentScores) {
			return errors.New("must have same number of scores if specified")
		}
		dst.CurrentScores = nil
	}
	// MetaEventData -- ignore for now; we might not have to deal with it.

	// Note that a "bytes" field is not considered an array; it is a scalar
	// type and thus properly replaced.
	if dst.Timers != nil && src.Timers != nil {
		if len(dst.Timers.TimeRemaining) > 0 && len(src.Timers.TimeRemaining) > 0 {
			if len(dst.Timers.TimeRemaining) != len(src.Timers.TimeRemaining) {
				return errors.New("must have same time remaining length if specified")
			}
			dst.Timers.TimeRemaining = nil
		}
	}

	proto.Merge(dst, src)

	return nil
}
