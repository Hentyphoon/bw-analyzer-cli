package replay

import (
	"fmt"
	"time"
)

// MinDuration is the shortest game kept by the 1v1 filter.
const MinDuration = 2 * time.Minute

// SkipReason applies the 1v1 filter. It returns "" when the replay is a
// game between exactly two human, non-observer players on different teams
// with known races, lasting at least MinDuration. Otherwise it returns a
// short reason for storing alongside the skipped replay.
func (r *Replay) SkipReason() string {
	ps := r.Competitors()
	if len(ps) != 2 {
		return fmt.Sprintf("not 1v1: %d non-observer players", len(ps))
	}
	for _, p := range ps {
		if !p.Human {
			return "not 1v1: computer player"
		}
		if p.Race == RaceUnknown {
			return "unknown race"
		}
	}
	if ps[0].Team == ps[1].Team {
		return "not 1v1: both players on the same team"
	}
	if r.Frames.Duration() < MinDuration {
		return fmt.Sprintf("too short: %s", r.Frames)
	}
	return ""
}

// Competitors returns the non-observer players in team order.
func (r *Replay) Competitors() []Player {
	var ps []Player
	for _, p := range r.Players {
		if !p.Observer {
			ps = append(ps, p)
		}
	}
	return ps
}
