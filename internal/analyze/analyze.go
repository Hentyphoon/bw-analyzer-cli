package analyze

import (
	"errors"
	"slices"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// ErrNot1v1 is returned by Analyze for a replay that fails the 1v1 filter.
var ErrNot1v1 = errors.New("replay is not a 1v1 game")

// Report is the analysis of one 1v1 replay.
type Report struct {
	Matchup string         `json:"matchup"`
	Players []PlayerReport `json:"players"` // team order, observers excluded
}

// PlayerReport is the analysis of one player.
type PlayerReport struct {
	Player       replay.Player `json:"player"`
	OpponentRace replay.Race   `json:"opponent_race"`
	Activity     Activity      `json:"activity"`
	BuildOrder   []BuildStep   `json:"build_order"`
}

// Analyze computes the report for a replay that passes the 1v1 filter.
// Filtering first matters: computer players share an ID, so keying by
// player ID is only safe once the game is known to be two humans.
func Analyze(r *replay.Replay) (Report, error) {
	if reason := r.SkipReason(); reason != "" {
		return Report{}, ErrNot1v1
	}
	ps := r.Competitors()
	rep := Report{Matchup: Matchup(ps[0].Race, ps[1].Race)}
	for i, p := range ps {
		rep.Players = append(rep.Players, PlayerReport{
			Player:       p,
			OpponentRace: ps[1-i].Race,
			Activity:     PlayerActivity(r, p.ID),
			BuildOrder:   BuildOrder(r, p),
		})
	}
	return rep, nil
}

// Matchup names a race pairing: the two race letters sorted alphabetically
// and joined with "v", so both players of a game share one matchup (PvT,
// never TvP).
func Matchup(a, b replay.Race) string {
	rs := []string{string(a), string(b)}
	slices.Sort(rs)
	return rs[0] + "v" + rs[1]
}
