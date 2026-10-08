package analyze

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

var (
	zerg   = replay.Player{ID: 0, Race: replay.Zerg, Team: 1, Human: true}
	terran = replay.Player{ID: 1, Race: replay.Terran, Team: 2, Human: true}
)

// tile returns a pixel position for tile coordinates.
func tile(x, y int) *replay.Point {
	return &replay.Point{X: x * 32, Y: y * 32}
}

func build(f replay.Frame, pid byte, name string, pos *replay.Point) replay.ProductionCmd {
	return replay.ProductionCmd{Frame: f, PlayerID: pid, Kind: replay.KindBuild, Name: name, Pos: pos, Effective: true}
}

func prod(f replay.Frame, pid byte, kind replay.ProductionKind, name string) replay.ProductionCmd {
	return replay.ProductionCmd{Frame: f, PlayerID: pid, Kind: kind, Name: name, Effective: true}
}

func ineffective(pc replay.ProductionCmd) replay.ProductionCmd {
	pc.Effective = false
	return pc
}

func step(f replay.Frame, kind StepKind, name string) BuildStep {
	return BuildStep{Frame: f, Kind: kind, Name: name}
}

func TestBuildOrder(t *testing.T) {
	tests := []struct {
		name   string
		player replay.Player
		cmds   []replay.ProductionCmd
		want   []BuildStep
	}{
		{
			name:   "no commands",
			player: zerg,
			want:   nil,
		},
		{
			name:   "kinds map to structure, unit, research",
			player: zerg,
			cmds: []replay.ProductionCmd{
				prod(at(0, 1), 0, replay.KindUnitMorph, "Drone"),
				build(at(1, 30), 0, "Spawning Pool", tile(10, 10)),
				prod(at(3, 0), 0, replay.KindBuildingMorph, "Lair"),
				prod(at(3, 30), 0, replay.KindUpgrade, "Metabolic Boost (Zergling Speed)"),
				prod(at(5, 0), 0, replay.KindTech, "Lurker Aspect"),
			},
			want: []BuildStep{
				step(at(0, 1), Unit, "Drone"),
				step(at(1, 30), Structure, "Spawning Pool"),
				step(at(3, 0), Structure, "Lair"),
				step(at(3, 30), Research, "Metabolic Boost (Zergling Speed)"),
				step(at(5, 0), Research, "Lurker Aspect"),
			},
		},
		{
			name:   "other player's commands ignored",
			player: zerg,
			cmds: []replay.ProductionCmd{
				build(at(1, 0), 1, "Supply Depot", tile(5, 5)),
				build(at(1, 30), 0, "Spawning Pool", tile(10, 10)),
			},
			want: []BuildStep{step(at(1, 30), Structure, "Spawning Pool")},
		},
		{
			name:   "ineffective commands dropped",
			player: terran,
			cmds: []replay.ProductionCmd{
				prod(at(0, 1), 1, replay.KindTrain, "SCV"),
				ineffective(prod(at(0, 2), 1, replay.KindTrain, "SCV")),
				ineffective(build(at(1, 0), 1, "Barracks", tile(5, 5))),
			},
			want: []BuildStep{step(at(0, 1), Unit, "SCV")},
		},
		{
			name:   "all commands ineffective",
			player: terran,
			cmds: []replay.ProductionCmd{
				ineffective(prod(at(0, 1), 1, replay.KindTrain, "SCV")),
				ineffective(build(at(1, 0), 1, "Barracks", tile(5, 5))),
			},
			want: nil,
		},
		{
			name:   "unknown and off-race names dropped",
			player: terran,
			cmds: []replay.ProductionCmd{
				prod(at(0, 1), 1, replay.KindTrain, "Unknown 0xe4"),
				prod(at(0, 2), 1, replay.KindTrain, "Jim Raynor (Marine)"),
				build(at(0, 3), 1, "Spawning Pool", tile(1, 1)),
				prod(at(0, 4), 1, replay.KindTrain, ""),
				prod(at(0, 5), 1, replay.KindTrain, "Marine"),
			},
			want: []BuildStep{step(at(0, 5), Unit, "Marine")},
		},
		{
			name:   "repeated structure at the same position collapses into the first",
			player: terran,
			cmds: []replay.ProductionCmd{
				build(at(2, 0), 1, "Supply Depot", tile(33, 96)),
				build(at(2, 4), 1, "Supply Depot", tile(33, 96)),
				build(at(2, 10), 1, "Supply Depot", tile(33, 96)),
			},
			want: []BuildStep{step(at(2, 0), Structure, "Supply Depot")},
		},
		{
			name:   "a burst longer than the window still collapses",
			player: terran,
			cmds: []replay.ProductionCmd{
				build(at(2, 0), 1, "Supply Depot", tile(33, 96)),
				build(at(2, 8), 1, "Supply Depot", tile(33, 96)),
				build(at(2, 16), 1, "Supply Depot", tile(33, 96)),
			},
			want: []BuildStep{step(at(2, 0), Structure, "Supply Depot")},
		},
		{
			name:   "repeat after the window is kept",
			player: terran,
			cmds: []replay.ProductionCmd{
				build(at(2, 0), 1, "Supply Depot", tile(33, 96)),
				build(at(2, 11), 1, "Supply Depot", tile(33, 96)),
			},
			want: []BuildStep{
				step(at(2, 0), Structure, "Supply Depot"),
				step(at(2, 11), Structure, "Supply Depot"),
			},
		},
		{
			name:   "same structure at another position is kept",
			player: terran,
			cmds: []replay.ProductionCmd{
				build(at(2, 0), 1, "Barracks", tile(30, 90)),
				build(at(2, 2), 1, "Barracks", tile(34, 90)),
			},
			want: []BuildStep{
				step(at(2, 0), Structure, "Barracks"),
				step(at(2, 2), Structure, "Barracks"),
			},
		},
		{
			name:   "changed mind on the same position keeps the later order",
			player: zerg,
			cmds: []replay.ProductionCmd{
				build(at(1, 54), 0, "Evolution Chamber", tile(121, 3)),
				prod(at(1, 55), 0, replay.KindUnitMorph, "Drone"),
				build(at(1, 56), 0, "Spawning Pool", tile(121, 3)),
			},
			want: []BuildStep{
				step(at(1, 55), Unit, "Drone"),
				step(at(1, 56), Structure, "Spawning Pool"),
			},
		},
		{
			name:   "different structure on the same position after the window is kept",
			player: zerg,
			cmds: []replay.ProductionCmd{
				build(at(5, 0), 0, "Creep Colony", tile(20, 20)),
				build(at(9, 0), 0, "Evolution Chamber", tile(20, 20)),
			},
			want: []BuildStep{
				step(at(5, 0), Structure, "Creep Colony"),
				step(at(9, 0), Structure, "Evolution Chamber"),
			},
		},
		{
			name:   "research spam collapses, a later level is kept",
			player: terran,
			cmds: []replay.ProductionCmd{
				prod(at(5, 0), 1, replay.KindUpgrade, "Terran Infantry Weapons"),
				prod(at(5, 3), 1, replay.KindUpgrade, "Terran Infantry Weapons"),
				prod(at(5, 6), 1, replay.KindTech, "Stim Packs"),
				prod(at(5, 7), 1, replay.KindTech, "Stim Packs"),
				prod(at(9, 0), 1, replay.KindUpgrade, "Terran Infantry Weapons"),
			},
			want: []BuildStep{
				step(at(5, 0), Research, "Terran Infantry Weapons"),
				step(at(5, 6), Research, "Stim Packs"),
				step(at(9, 0), Research, "Terran Infantry Weapons"),
			},
		},
		{
			name:   "units are never collapsed",
			player: terran,
			cmds: []replay.ProductionCmd{
				prod(at(3, 0), 1, replay.KindTrain, "Marine"),
				prod(at(3, 0), 1, replay.KindTrain, "Marine"),
				prod(at(3, 1), 1, replay.KindTrain, "Marine"),
			},
			want: []BuildStep{
				step(at(3, 0), Unit, "Marine"),
				step(at(3, 0), Unit, "Marine"),
				step(at(3, 1), Unit, "Marine"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &replay.Replay{Frames: at(20, 0), Production: tt.cmds}
			got := BuildOrder(r, tt.player)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildOrder =\n  %v\nwant\n  %v", got, tt.want)
			}
		})
	}
}

func TestWorkersBefore(t *testing.T) {
	steps := []BuildStep{
		step(at(0, 1), Unit, "Drone"),
		step(at(0, 15), Unit, "Drone"),
		step(at(0, 30), Unit, "Overlord"),
		step(at(0, 40), Unit, "Drone"),
		step(at(1, 0), Structure, "Spawning Pool"),
		step(at(1, 10), Unit, "Drone"),
	}
	tests := []struct {
		name string
		f    replay.Frame
		want int
	}{
		{"game start", 0, 4},
		{"at the first drone", at(0, 1), 4},
		{"after two drones", at(0, 20), 6},
		{"overlords are not workers", at(0, 35), 6},
		{"at the pool", at(1, 0), 7},
		{"after everything", at(9, 0), 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorkersBefore(steps, tt.f); got != tt.want {
				t.Errorf("WorkersBefore(%s) = %d, want %d", tt.f, got, tt.want)
			}
		})
	}
	if got := WorkersBefore(nil, at(5, 0)); got != StartingWorkers {
		t.Errorf("WorkersBefore(nil) = %d, want %d", got, StartingWorkers)
	}
}

func TestUntil(t *testing.T) {
	steps := []BuildStep{step(10, Unit, "SCV"), step(20, Unit, "SCV"), step(30, Unit, "SCV")}
	tests := []struct {
		f    replay.Frame
		want int
	}{{0, 0}, {10, 0}, {11, 1}, {30, 2}, {31, 3}}
	for _, tt := range tests {
		if got := len(Until(steps, tt.f)); got != tt.want {
			t.Errorf("len(Until(%d)) = %d, want %d", tt.f, got, tt.want)
		}
	}
}

func TestMatchup(t *testing.T) {
	tests := []struct {
		a, b replay.Race
		want string
	}{
		{replay.Terran, replay.Zerg, "TvZ"},
		{replay.Zerg, replay.Terran, "TvZ"},
		{replay.Terran, replay.Protoss, "PvT"},
		{replay.Zerg, replay.Protoss, "PvZ"},
		{replay.Protoss, replay.Protoss, "PvP"},
		{replay.Terran, replay.Terran, "TvT"},
		{replay.Zerg, replay.Zerg, "ZvZ"},
	}
	for _, tt := range tests {
		if got := Matchup(tt.a, tt.b); got != tt.want {
			t.Errorf("Matchup(%s, %s) = %s, want %s", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAnalyze(t *testing.T) {
	obs := replay.Player{ID: 2, Race: replay.Protoss, Team: 3, Human: true, Observer: true}
	tests := []struct {
		name        string
		players     []replay.Player
		wantErr     error
		wantMatchup string
	}{
		{"1v1", []replay.Player{zerg, terran}, nil, "TvZ"},
		{"1v1 with observer", []replay.Player{zerg, terran, obs}, nil, "TvZ"},
		{"not 1v1", []replay.Player{zerg}, ErrNot1v1, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &replay.Replay{
				Frames:  at(10, 0),
				Players: tt.players,
				Actions: []replay.Action{act(at(1, 0), 0, true), act(at(1, 0), 1, true)},
				Production: []replay.ProductionCmd{
					build(at(1, 0), 0, "Spawning Pool", tile(1, 1)),
					build(at(1, 0), 1, "Barracks", tile(9, 9)),
				},
			}
			rep, err := Analyze(r)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if rep.Matchup != tt.wantMatchup || len(rep.Players) != 2 {
				t.Fatalf("matchup %q with %d players", rep.Matchup, len(rep.Players))
			}
			z, tr := rep.Players[0], rep.Players[1]
			if z.OpponentRace != replay.Terran || tr.OpponentRace != replay.Zerg {
				t.Errorf("opponent races = %s, %s", z.OpponentRace, tr.OpponentRace)
			}
			if len(z.BuildOrder) != 1 || z.BuildOrder[0].Name != "Spawning Pool" ||
				len(tr.BuildOrder) != 1 || tr.BuildOrder[0].Name != "Barracks" {
				t.Errorf("build orders = %v, %v", z.BuildOrder, tr.BuildOrder)
			}
			if z.Activity.Actions != 1 || tr.Activity.Actions != 1 {
				t.Errorf("actions = %d, %d", z.Activity.Actions, tr.Activity.Actions)
			}
		})
	}
}
