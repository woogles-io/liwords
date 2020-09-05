// Package botinterface has the interface with a NATS-based bot.
// See the bot package in macondo.
package botinterface

import "github.com/domino14/macondo/bot"

// Bot is our liwords interface to the Macondo bot client.
type Bot struct {
	client bot.Client
}
