package user

import "context"

// RedisPresenceStore implements a Redis store for user presence.
type RedisPresenceStore struct {
}

// SetPresence sets the user's channel. If blank, this means the user is offline.
func (s *RedisPresenceStore) SetPresence(ctx context.Context, uuid, channel string) error {
	// We try to map channels closely to the pubsub NATS channels (and realms),
	// with some exceptions.
	// If the user is online in two different tabs, we go in priority order,
	// as we only want to show them in one place.
	// Priority (from lowest to highest):
	// 	- lobby - The "base" channel.
	//  - usertv.<user_id> - Following a user's games
	//  - gametv.<game_id> - Watching a game
	//  - game.<game_id> - Playing in a game
	return nil
}
