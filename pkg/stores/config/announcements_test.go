package config

import (
	"testing"

	"github.com/matryer/is"

	pb "github.com/woogles-io/liwords/rpc/api/proto/config_service"
)

const blog = "https://blog.woogles.io"

func TestReplacementSlotSkipsPinned(t *testing.T) {
	is := is.New(t)
	annos := []*pb.Announcement{
		{Title: "Board", Link: blog + "/posts/board/", Body: "b", Pinned: true},
		{Title: "Recap", Link: blog + "/posts/recap/", Body: "r"},
		{Title: "Donate", Link: "https://woogles.io/donate", Body: "d"},
	}
	idx, replace, err := replacementSlot(annos, blog,
		&pb.Announcement{Title: "New", Link: blog + "/posts/new/", Body: "n"})
	is.NoErr(err)
	is.True(replace)
	is.Equal(idx, 1)
}

func TestReplacementSlotUnchanged(t *testing.T) {
	is := is.New(t)
	annos := []*pb.Announcement{
		{Title: "Recap", Link: blog + "/posts/recap/", Body: "r"},
	}
	_, replace, err := replacementSlot(annos, blog,
		&pb.Announcement{Title: "Recap", Link: blog + "/posts/recap/", Body: "r"})
	is.NoErr(err)
	is.True(!replace)
}

func TestReplacementSlotNewestPostIsPinned(t *testing.T) {
	is := is.New(t)
	annos := []*pb.Announcement{
		{Title: "Board", Link: blog + "/posts/board/", Body: "b", Pinned: true},
		{Title: "Recap", Link: blog + "/posts/recap/", Body: "r"},
	}
	// The feed's newest post is the pinned one: don't show it twice.
	_, replace, err := replacementSlot(annos, blog,
		&pb.Announcement{Title: "Board!", Link: blog + "/posts/board/", Body: "x"})
	is.NoErr(err)
	is.True(!replace)
}

func TestReplacementSlotOnlyPinned(t *testing.T) {
	is := is.New(t)
	annos := []*pb.Announcement{
		{Title: "Board", Link: blog + "/posts/board/", Body: "b", Pinned: true},
	}
	_, _, err := replacementSlot(annos, blog,
		&pb.Announcement{Title: "New", Link: blog + "/posts/new/", Body: "n"})
	is.True(err != nil)
}
