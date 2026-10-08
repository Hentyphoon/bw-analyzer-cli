package analyze

import (
	"time"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// BuildStep is one entry of a build order: something the player ordered.
// A replay records commands, not game state, so a step may have been
// cancelled or may never have finished.
type BuildStep struct {
	Frame replay.Frame `json:"frame"`
	Kind  StepKind     `json:"kind"`
	Name  string       `json:"name"`
}

// RepeatWindow is how close in time two matching orders must be for the
// later one to count as a repeat of the earlier one. The gap is measured
// from the previous matching order, so a burst of spam clicks collapses into
// its first order however long the burst lasts.
const RepeatWindow = 10 * time.Second

// BuildOrder returns the de-noised build order of player p in r, in command
// order. Production commands are dropped when they are:
//
//   - ineffective, as classified by the parsing library;
//   - not on the allowlist for the player's race;
//   - a structure ordered again on the same position within RepeatWindow
//     (spam clicks, or a worker re-sent to the same site);
//   - a structure replaced by a different structure on the same position
//     within RepeatWindow (a changed mind: only the later order is kept);
//   - research ordered again by name within RepeatWindow (research spam).
//
// Units are never collapsed: training several of the same unit is normal,
// and the library already marks orders beyond a full queue ineffective.
func BuildOrder(r *replay.Replay, p replay.Player) []BuildStep {
	window := replay.FrameAt(RepeatWindow)

	// lastSite tracks the latest structure order on each position, and
	// lastResearch the latest order of each research name.
	type site struct {
		name  string
		frame replay.Frame
		index int // position of the kept step in steps
	}
	lastSite := map[replay.Point]site{}
	lastResearch := map[string]replay.Frame{}

	var steps []BuildStep
	removed := map[int]bool{}
	for _, pc := range r.Production {
		if pc.PlayerID != p.ID || !pc.Effective {
			continue
		}
		kind, ok := kindOf(p.Race, pc.Name)
		if !ok {
			continue
		}

		switch {
		case pc.Kind == replay.KindBuild && pc.Pos != nil:
			prev, seen := lastSite[*pc.Pos]
			if seen && pc.Frame-prev.frame <= window {
				if prev.name == pc.Name {
					// A repeat: keep the first order, extend the burst.
					prev.frame = pc.Frame
					lastSite[*pc.Pos] = prev
					continue
				}
				// A different structure on the same spot: the earlier
				// order was abandoned.
				removed[prev.index] = true
			}
			lastSite[*pc.Pos] = site{name: pc.Name, frame: pc.Frame, index: len(steps)}

		case kind == Research:
			if prev, seen := lastResearch[pc.Name]; seen && pc.Frame-prev <= window {
				lastResearch[pc.Name] = pc.Frame
				continue
			}
			lastResearch[pc.Name] = pc.Frame
		}

		steps = append(steps, BuildStep{Frame: pc.Frame, Kind: kind, Name: pc.Name})
	}

	if len(removed) == 0 {
		return steps
	}
	kept := steps[:0]
	for i, s := range steps {
		if !removed[i] {
			kept = append(kept, s)
		}
	}
	return kept
}

// Until returns the steps ordered before frame f.
func Until(steps []BuildStep, f replay.Frame) []BuildStep {
	for i, s := range steps {
		if s.Frame >= f {
			return steps[:i]
		}
	}
	return steps
}

// StartingWorkers is the number of workers every race starts with.
const StartingWorkers = 4

// WorkerBuildFrames is how long an SCV or Probe takes to build: 300 frames,
// about 12.6 s at Fastest. It comes from Brood War's unit data, not from
// screp, which carries no build times.
const WorkerBuildFrames replay.Frame = 300

// WorkersBefore estimates a player's worker count just before frame f: the
// starting workers plus the workers whose production started before f. It
// is an estimate, not game state: it ignores workers that died and Zerg
// drones that became structures, which matches how players name openings
// ("9 Pool" is a Spawning Pool ordered at 9 workers).
//
// How production starts differs by race:
//
//   - A Drone starts when it is ordered: each one uses up a larva, so drone
//     orders already arrive at the pace they can be made.
//   - SCVs and Probes come from one Command Center or Nexus queue, one at a
//     time. Players queue several at once, and orders they cannot afford
//     are still recorded, so counting orders overcounts badly. Instead each
//     order starts when it arrives or when the previous worker finishes,
//     whichever is later.
//
// The queue model assumes a single town hall. Once an expansion finishes,
// workers are built in parallel and the estimate runs low; openings are
// decided before that.
func WorkersBefore(steps []BuildStep, f replay.Frame) int {
	n := StartingWorkers
	var queueFree replay.Frame // when the town hall can start the next worker
	for _, s := range steps {
		if s.Frame >= f {
			break // steps are in order, and nothing starts before it is ordered
		}
		if s.Kind != Unit || !workerNames[s.Name] {
			continue
		}
		start := s.Frame
		if s.Name != "Drone" {
			start = max(s.Frame, queueFree)
			queueFree = start + WorkerBuildFrames
		}
		if start < f {
			n++
		}
	}
	return n
}
