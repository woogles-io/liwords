package config

import (
	"errors"
	"strings"

	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
)

// replacementSlot picks which announcement SetAnnouncement should overwrite:
// the first unpinned one whose link contains linkSearchString. replace is
// false when nothing needs to change -- the slot already holds this
// announcement, or the incoming link is already shown as a pinned
// announcement (so it would otherwise appear twice).
func replacementSlot(annos []*pb.Announcement, linkSearchString string,
	announcement *pb.Announcement) (idx int, replace bool, err error) {

	for _, a := range annos {
		if a.Pinned && a.Link == announcement.Link {
			return 0, false, nil
		}
	}
	idx = -1
	for i, a := range annos {
		if !a.Pinned && strings.Contains(a.Link, linkSearchString) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return 0, false, errors.New("link search string not found in unpinned announcements")
	}
	cur := annos[idx]
	if cur.Body == announcement.Body && cur.Link == announcement.Link &&
		cur.Title == announcement.Title {
		return idx, false, nil
	}
	return idx, true, nil
}
