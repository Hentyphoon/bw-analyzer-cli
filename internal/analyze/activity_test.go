package analyze

import (
	"math"
	"testing"
	"time"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// at returns the frame at a game time given as minutes and seconds.
func at(min, sec int) replay.Frame {
	return replay.FrameAt(time.Duration(min)*time.Minute + time.Duration(sec)*time.Second)
}

func act(f replay.Frame, pid byte, effective bool) replay.Action {
	return replay.Action{Frame: f, PlayerID: pid, Effective: effective}
}

func TestPlayerActivity(t *testing.T) {
	tests := []struct {
		name        string
		frames      replay.Frame
		actions     []replay.Action
		wantActions int
		wantEff     int
		wantAPM     float64
		wantEAPM    float64
		wantRed     float64
		wantMinutes []Minute
	}{
		{
			name:        "no commands",
			frames:      at(3, 0),
			wantMinutes: []Minute{{0, 0, 0}, {1, 0, 0}, {2, 0, 0}, {3, 0, 0}},
		},
		{
			name:   "game shorter than a minute",
			frames: at(0, 40),
			actions: []replay.Action{
				act(at(0, 10), 0, true), act(at(0, 20), 0, true), act(at(0, 30), 0, false),
			},
			wantActions: 3, wantEff: 2,
			// Divided by the 30 s up to the last command, not the 40 s game.
			wantAPM: 3 / at(0, 30).Minutes(), wantEAPM: 2 / at(0, 30).Minutes(),
			wantRed:     1.0 / 3,
			wantMinutes: []Minute{{0, 3, 2}},
		},
		{
			name:   "all ineffective",
			frames: at(2, 0),
			actions: []replay.Action{
				act(at(0, 30), 0, false), act(at(1, 0), 0, false),
			},
			wantActions: 2, wantEff: 0,
			wantAPM: 2 / at(1, 0).Minutes(), wantEAPM: 0,
			wantRed:     1,
			wantMinutes: []Minute{{0, 1, 0}, {1, 1, 0}, {2, 0, 0}},
		},
		{
			name:   "other players and minute buckets",
			frames: at(2, 30),
			actions: []replay.Action{
				act(at(0, 5), 0, true), act(at(0, 6), 1, true),
				act(at(1, 0), 0, true), act(at(1, 59), 0, false),
				act(at(2, 0), 0, true), act(at(2, 1), 1, false),
			},
			wantActions: 4, wantEff: 3,
			wantAPM: 4 / at(2, 0).Minutes(), wantEAPM: 3 / at(2, 0).Minutes(),
			wantRed:     0.25,
			wantMinutes: []Minute{{0, 1, 1}, {1, 2, 1}, {2, 1, 1}},
		},
		{
			name:   "frame past the end of the game",
			frames: at(1, 30),
			actions: []replay.Action{
				act(at(1, 0), 0, true), act(at(9, 0), 0, true),
			},
			// Counted, but kept out of the curve and the last-command frame,
			// as the library does.
			wantActions: 2, wantEff: 2,
			wantAPM: 2 / at(1, 0).Minutes(), wantEAPM: 2 / at(1, 0).Minutes(),
			wantMinutes: []Minute{{0, 0, 0}, {1, 1, 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &replay.Replay{Frames: tt.frames, Actions: tt.actions}
			got := PlayerActivity(r, 0)
			if got.Actions != tt.wantActions || got.EffectiveActions != tt.wantEff {
				t.Errorf("actions = %d/%d, want %d/%d", got.Actions, got.EffectiveActions, tt.wantActions, tt.wantEff)
			}
			for _, c := range []struct {
				name      string
				got, want float64
			}{
				{"APM", got.APM, tt.wantAPM},
				{"EAPM", got.EAPM, tt.wantEAPM},
				{"Redundancy", got.Redundancy, tt.wantRed},
			} {
				if math.Abs(c.got-c.want) > 1e-9 {
					t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
				}
			}
			if len(got.Minutes) != len(tt.wantMinutes) {
				t.Fatalf("Minutes = %v, want %v", got.Minutes, tt.wantMinutes)
			}
			for i := range got.Minutes {
				if got.Minutes[i] != tt.wantMinutes[i] {
					t.Errorf("Minutes[%d] = %+v, want %+v", i, got.Minutes[i], tt.wantMinutes[i])
				}
			}
		})
	}
}
