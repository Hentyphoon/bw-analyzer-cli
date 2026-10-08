// Package parser turns raw replay bytes into replay.Replay values. It defines
// the Parser interface and the adapter over github.com/icza/screp. No screp
// type escapes this package.
package parser

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode"

	"github.com/icza/screp/rep"
	"github.com/icza/screp/rep/repcmd"
	"github.com/icza/screp/rep/repcore"
	"github.com/icza/screp/repparser"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// Parser decodes one replay file. Implementations must be safe for
// concurrent use.
type Parser interface {
	Parse(data []byte) (*replay.Replay, error)
}

// ErrInvalid is wrapped by every error returned for input that is not a
// readable replay, as opposed to a bug in this package.
var ErrInvalid = errors.New("invalid replay")

// Screp is the Parser backed by github.com/icza/screp. The zero value is
// ready to use. screp documents itself as safe for concurrent use, and Screp
// holds no state, so one value can be shared by all ingest workers.
type Screp struct{}

// screpConfig parses the minimum the analysis needs: commands for APM, build
// orders, teams, and winners; map data for start locations. Map graphics and
// debug data are never needed.
//
// The logger swallows the library's parse warnings, which it would otherwise
// print through the global log package; lost commands are still reported
// through Replay.ParseErrors. A fresh config per call keeps the package free
// of shared mutable state, and costs one small allocation.
func screpConfig() repparser.Config {
	return repparser.Config{
		Commands: true,
		MapData:  true,
		Logger:   log.New(io.Discard, "", 0),
	}
}

// Parse implements Parser. The library already converts its own panics into
// repparser.ErrParsing; the recover here is a second line of defense that
// also covers this adapter's conversion code.
func (Screp) Parse(data []byte) (r *replay.Replay, err error) {
	defer func() {
		if p := recover(); p != nil {
			r, err = nil, fmt.Errorf("%w: panic while parsing: %v", ErrInvalid, p)
		}
	}()

	sr, err := repparser.ParseConfig(data, screpConfig())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	if sr.Header == nil {
		return nil, fmt.Errorf("%w: missing header", ErrInvalid)
	}
	// Parsing does not fill Computed; teams, observers, winners, start
	// locations, and each command's IneffKind are only set by Compute.
	sr.Compute()
	return convert(sr), nil
}

func convert(sr *rep.Replay) *replay.Replay {
	h := sr.Header
	r := &replay.Replay{
		MapName:   mapName(sr),
		Version:   h.Version,
		Frames:    replay.Frame(h.Frames),
		StartTime: h.StartTime.UTC(),
	}
	if h.Type != nil {
		r.GameType = h.Type.Name
	}

	// Header.Players and Computed.PlayerDescs are index-aligned, in team
	// order; Compute keeps them aligned when it reassigns teams.
	pids := make(map[byte]bool, len(h.Players))
	for i, p := range h.Players {
		rp := replay.Player{
			ID:       p.ID,
			Name:     p.Name,
			Race:     race(p.Race),
			Team:     int(p.Team),
			Human:    p.Type != nil && p.Type.ID == repcore.PlayerTypeHuman.ID,
			Observer: p.Observer,
			Result:   result(sr.Computed.WinnerTeam, p),
		}
		if pd := sr.Computed.PlayerDescs[i]; pd.StartLocation != nil {
			rp.Start = &replay.Point{X: int(pd.StartLocation.X), Y: int(pd.StartLocation.Y)}
		}
		r.Players = append(r.Players, rp)
		pids[p.ID] = true
	}

	if sr.Commands != nil {
		r.ParseErrors = len(sr.Commands.ParseErrCmds)
		r.Actions = make([]replay.Action, 0, len(sr.Commands.Cmds))
		for _, cmd := range sr.Commands.Cmds {
			base := cmd.BaseCmd()
			// Commands from IDs outside the header (observers chatting use
			// 128 and up) are not player actions; the library skips them too.
			if !pids[base.PlayerID] {
				continue
			}
			effective := base.IneffKind.Effective()
			r.Actions = append(r.Actions, replay.Action{
				Frame:     replay.Frame(base.Frame),
				PlayerID:  base.PlayerID,
				Effective: effective,
			})
			if pc, ok := production(cmd); ok {
				pc.Frame = replay.Frame(base.Frame)
				pc.PlayerID = base.PlayerID
				pc.Effective = effective
				r.Production = append(r.Production, pc)
			}
		}
	}
	return r
}

// production converts the command types that order something. The library
// decodes both Train and Unit Morph into *repcmd.TrainCmd and tells them
// apart by command type.
func production(cmd repcmd.Cmd) (replay.ProductionCmd, bool) {
	switch c := cmd.(type) {
	case *repcmd.BuildCmd:
		// Build command positions are tile coordinates, while map start
		// locations are pixels. Convert to pixels so both share one unit.
		const tile = 32
		return replay.ProductionCmd{
			Kind: replay.KindBuild,
			Name: unitName(c.Unit),
			Pos:  &replay.Point{X: int(c.Pos.X) * tile, Y: int(c.Pos.Y) * tile},
		}, true
	case *repcmd.TrainCmd:
		kind := replay.KindTrain
		if c.Type != nil && c.Type.ID == repcmd.TypeIDUnitMorph {
			kind = replay.KindUnitMorph
		}
		return replay.ProductionCmd{Kind: kind, Name: unitName(c.Unit)}, true
	case *repcmd.BuildingMorphCmd:
		return replay.ProductionCmd{Kind: replay.KindBuildingMorph, Name: unitName(c.Unit)}, true
	case *repcmd.TechCmd:
		name := ""
		if c.Tech != nil {
			name = c.Tech.Name
		}
		return replay.ProductionCmd{Kind: replay.KindTech, Name: name}, true
	case *repcmd.UpgradeCmd:
		name := ""
		if c.Upgrade != nil {
			name = c.Upgrade.Name
		}
		return replay.ProductionCmd{Kind: replay.KindUpgrade, Name: name}, true
	}
	return replay.ProductionCmd{}, false
}

func unitName(u *repcmd.Unit) string {
	if u == nil {
		return ""
	}
	return u.Name
}

func race(r *repcore.Race) replay.Race {
	if r == nil {
		return replay.RaceUnknown
	}
	switch r.Letter {
	case 'T':
		return replay.Terran
	case 'P':
		return replay.Protoss
	case 'Z':
		return replay.Zerg
	}
	return replay.RaceUnknown
}

// result maps the library's winning team onto one player. WinnerTeam 0 means
// the library could not tell; observers never win or lose.
func result(winnerTeam byte, p *rep.Player) replay.Result {
	switch {
	case winnerTeam == 0 || p.Observer:
		return replay.ResultUnknown
	case p.Team == winnerTeam:
		return replay.Win
	default:
		return replay.Loss
	}
}

// mapName prefers the map data's name, which is not truncated like the
// header's 26-byte copy, and strips the color and formatting control
// characters map makers embed.
func mapName(sr *rep.Replay) string {
	name := sr.Header.Map
	if sr.MapData != nil && strings.TrimSpace(sr.MapData.Name) != "" {
		name = sr.MapData.Name
	}
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	return strings.TrimSpace(name)
}
