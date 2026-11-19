package sockets

import (
	"time"

	"github.com/rs/zerolog/log"
)

// When a socket server is being replaced, the load balancer stops sending it
// new connections and, once its deregistration delay is up, cuts the ones it
// has, all at once. Every browser then reconnects within a second or two, and
// the new server and the API take the whole wave together.
//
// Drain avoids that: it closes this server's sockets a few at a time over a
// period shorter than the deregistration delay, so browsers reconnect, to the
// other server, at a steady trickle. Once the last one is closed the load
// balancer finishes deregistering at once, without waiting out the delay.

// drainStepPeriod is how often a draining server closes its next sockets.
const drainStepPeriod = time.Second

// Drain closes every socket on this server, evenly over the given period.
// Only the first call counts.
func (h *Hub) Drain(over time.Duration) {
	select {
	case h.drainRequests <- over:
	default: // already draining or asked to
	}
}

type drainState struct {
	until   time.Time
	drained bool
}

// startDrain begins draining. Call it only from Run.
func (h *Hub) startDrain(now time.Time, over time.Duration) {
	if h.draining != nil {
		return
	}
	h.draining = &drainState{until: now.Add(over)}
	log.Info().Int("sockets", len(h.clients)).Dur("over", over).Msg("draining")
}

// drainStep closes the share of the remaining sockets due by now, so that the
// last ones close when the drain period ends. Call it only from Run.
func (h *Hub) drainStep(now time.Time) {
	if h.draining == nil {
		return
	}
	// The steps left, counting this one; the last falls when the period ends.
	steps := int(h.draining.until.Sub(now)/drainStepPeriod) + 1
	n := len(h.clients)
	if steps > 1 {
		n = (n + steps - 1) / steps
	}
	for c := range h.clients {
		if n == 0 {
			break
		}
		// Its write pump sends a close frame; the browser reconnects.
		h.removeClient(c)
		n--
	}
	if len(h.clients) == 0 && !h.draining.drained {
		h.draining.drained = true
		log.Info().Msg("drained")
	}
}
