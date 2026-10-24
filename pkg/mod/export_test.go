package mod

import "time"

// UserGameIDs exposes userGameIDs to the mod_test package.
var UserGameIDs = (*ModService).userGameIDs

// SetChatReviewNow replaces the clock behind the recent-game window and
// returns a function that restores it.
func SetChatReviewNow(now func() time.Time) func() {
	old := chatReviewNow
	chatReviewNow = now
	return func() { chatReviewNow = old }
}

// ChannelContext exposes channelContext to the mod_test package.
var ChannelContext = (*ModService).channelContext
