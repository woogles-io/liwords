package sockets

// A followIndex records, for each user connected to this socket server, whom
// they follow, so that a presence change can be delivered to the followers
// connected here without asking the API who they are.
//
// Every socket server receives every presence change and delivers it to its
// own connections; together they reach every connected follower.
//
// It is owned by the hub's Run loop and isn't safe for concurrent use.
type followIndex struct {
	// follows[u] is the set of users that u, a connected user, follows.
	follows map[string]map[string]struct{}
	// followers[f] is the set of connected users that follow f.
	followers map[string]map[string]struct{}
}

func newFollowIndex() *followIndex {
	return &followIndex{
		follows:   map[string]map[string]struct{}{},
		followers: map[string]map[string]struct{}{},
	}
}

// set records that user follows exactly the given users, replacing whatever
// was recorded for them before.
func (fi *followIndex) set(user string, follows []string) {
	fi.remove(user)
	set := make(map[string]struct{}, len(follows))
	for _, f := range follows {
		set[f] = struct{}{}
		fi.link(user, f)
	}
	fi.follows[user] = set
}

// remove forgets user, who is no longer connected.
func (fi *followIndex) remove(user string) {
	for f := range fi.follows[user] {
		fi.unlink(user, f)
	}
	delete(fi.follows, user)
}

// follow records that user now follows followed, if user is recorded.
func (fi *followIndex) follow(user, followed string) {
	set, ok := fi.follows[user]
	if !ok {
		return
	}
	set[followed] = struct{}{}
	fi.link(user, followed)
}

// unfollow records that user no longer follows followed.
func (fi *followIndex) unfollow(user, followed string) {
	set, ok := fi.follows[user]
	if !ok {
		return
	}
	delete(set, followed)
	fi.unlink(user, followed)
}

// followersOf returns the recorded users who follow user. The caller must not
// modify it.
func (fi *followIndex) followersOf(user string) map[string]struct{} {
	return fi.followers[user]
}

// size returns the number of users recorded and the number of follows among
// them.
func (fi *followIndex) size() (users, follows int) {
	for _, set := range fi.follows {
		follows += len(set)
	}
	return len(fi.follows), follows
}

func (fi *followIndex) link(user, followed string) {
	set := fi.followers[followed]
	if set == nil {
		set = map[string]struct{}{}
		fi.followers[followed] = set
	}
	set[user] = struct{}{}
}

func (fi *followIndex) unlink(user, followed string) {
	set := fi.followers[followed]
	delete(set, user)
	if len(set) == 0 {
		delete(fi.followers, followed)
	}
}
