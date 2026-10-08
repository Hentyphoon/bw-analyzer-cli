package analyze

import "github.com/Hentyphoon/bw-analyzer-cli/internal/replay"

// Activity is a player's command rate over one game.
//
// The definitions follow screp so that totals can be checked against it:
// APM counts every command the player issued, EAPM only the effective ones,
// and both divide by the minutes up to the player's last command rather
// than the game length, so a player who leaves early is not diluted.
type Activity struct {
	Actions          int     `json:"actions"`
	EffectiveActions int     `json:"effective_actions"`
	APM              float64 `json:"apm"`
	EAPM             float64 `json:"eapm"`
	// Redundancy is the share of actions that were ineffective, from 0 to 1.
	Redundancy float64 `json:"redundancy"`
	// Minutes holds one bucket per whole game minute, starting at minute 0,
	// including minutes with no actions.
	Minutes []Minute `json:"minutes"`
}

// Minute is the action count of one whole game minute.
type Minute struct {
	Minute           int `json:"minute"`
	Actions          int `json:"actions"`
	EffectiveActions int `json:"effective_actions"`
}

// PlayerActivity computes the activity of one player in r.
func PlayerActivity(r *replay.Replay, playerID byte) Activity {
	var a Activity
	// Cover the whole game even for a player who left early, so both
	// players' curves line up minute for minute.
	a.Minutes = make([]Minute, r.Frames.Minute()+1)
	for i := range a.Minutes {
		a.Minutes[i].Minute = i
	}

	var last replay.Frame
	for _, act := range r.Actions {
		if act.PlayerID != playerID {
			continue
		}
		a.Actions++
		if act.Effective {
			a.EffectiveActions++
		}
		// A frame outside the game means a corrupt command. The library
		// skips such frames when finding the last command; so do we, and
		// they stay out of the curve.
		if act.Frame < 0 || act.Frame > r.Frames {
			continue
		}
		last = max(last, act.Frame)
		m := &a.Minutes[act.Frame.Minute()]
		m.Actions++
		if act.Effective {
			m.EffectiveActions++
		}
	}

	if a.Actions > 0 {
		a.Redundancy = float64(a.Actions-a.EffectiveActions) / float64(a.Actions)
	}
	if last > 0 {
		mins := last.Minutes()
		a.APM = float64(a.Actions) / mins
		a.EAPM = float64(a.EffectiveActions) / mins
	}
	return a
}
