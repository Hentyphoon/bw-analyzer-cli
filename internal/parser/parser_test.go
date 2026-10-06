package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/icza/screp/repparser"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

const sampleFile = "../../testdata/replays/screp_shieldbattery_zvt.rep"

func readSample(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(sampleFile)
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	return data
}

func TestParseInvalidInput(t *testing.T) {
	sample := readSample(t)
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"garbage", []byte("this is not a replay file at all, just some text")},
		{"zeros", make([]byte, 4096)},
		{"replay id only", sample[:16]},
		{"truncated header", sample[:300]},
		{"truncated commands", sample[:len(sample)/3]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Screp{}.Parse(tt.data)
			if err == nil {
				t.Fatalf("Parse returned no error (players: %d)", len(r.Players))
			}
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("error %q does not wrap ErrInvalid", err)
			}
		})
	}
}

// TestParseAllSampleReplays makes sure every owner-supplied replay parses.
func TestParseAllSampleReplays(t *testing.T) {
	files, err := filepath.Glob("../../testdata/replays/*.rep")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := (Screp{}).Parse(data); err != nil {
				t.Fatalf("Parse: %v", err)
			}
		})
	}
}

// TestActionCountsMatchLibrary checks that the adapter keeps exactly the
// commands the library counts toward APM and EAPM.
func TestActionCountsMatchLibrary(t *testing.T) {
	files, err := filepath.Glob("../../testdata/replays/*.rep")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			data, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			r, err := Screp{}.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			lib, err := repparser.ParseConfig(data, config)
			if err != nil {
				t.Fatal(err)
			}
			lib.Compute()

			cmds := map[byte]uint32{}
			effective := map[byte]uint32{}
			for _, a := range r.Actions {
				cmds[a.PlayerID]++
				if a.Effective {
					effective[a.PlayerID]++
				}
			}
			for _, pd := range lib.Computed.PlayerDescs {
				if cmds[pd.PlayerID] != pd.CmdCount {
					t.Errorf("player %d: %d actions, library counts %d", pd.PlayerID, cmds[pd.PlayerID], pd.CmdCount)
				}
				if effective[pd.PlayerID] != pd.EffectiveCmdCount {
					t.Errorf("player %d: %d effective actions, library counts %d", pd.PlayerID, effective[pd.PlayerID], pd.EffectiveCmdCount)
				}
			}
		})
	}
}

// TestParseSample pins facts about the bundled screp sample that were
// checked by hand against the library's own dump of it.
func TestParseSample(t *testing.T) {
	r, err := Screp{}.Parse(readSample(t))
	if err != nil {
		t.Fatal(err)
	}
	if r.MapName != "Eclipse 1.3" || r.Version != "1.21+" || r.Frames != 30979 {
		t.Errorf("got map %q version %q frames %d", r.MapName, r.Version, r.Frames)
	}
	if reason := r.SkipReason(); reason != "" {
		t.Errorf("SkipReason = %q, want a kept 1v1", reason)
	}
	want := []replay.Player{
		{ID: 0, Name: "[z]home", Race: replay.Zerg, Team: 1, Human: true, Result: replay.Loss, Start: &replay.Point{X: 3776, Y: 272}},
		{ID: 1, Name: "IlIIlIlIlIIllII", Race: replay.Terran, Team: 2, Human: true, Result: replay.Win, Start: &replay.Point{X: 320, Y: 3248}},
	}
	if len(r.Players) != len(want) {
		t.Fatalf("got %d players, want %d", len(r.Players), len(want))
	}
	for i, w := range want {
		g := r.Players[i]
		if g.ID != w.ID || g.Name != w.Name || g.Race != w.Race || g.Team != w.Team ||
			g.Human != w.Human || g.Observer != w.Observer || g.Result != w.Result ||
			g.Start == nil || *g.Start != *w.Start {
			t.Errorf("player %d = %+v (start %v), want %+v (start %v)", i, g, g.Start, w, w.Start)
		}
	}

	// The Terran's first structure: a Supply Depot at tile (32, 94),
	// converted to pixels.
	var first *replay.ProductionCmd
	for i := range r.Production {
		if r.Production[i].PlayerID == 1 && r.Production[i].Kind == replay.KindBuild {
			first = &r.Production[i]
			break
		}
	}
	if first == nil || first.Name != "Supply Depot" || first.Frame != 1285 ||
		first.Pos == nil || *first.Pos != (replay.Point{X: 32 * 32, Y: 94 * 32}) {
		t.Errorf("first Terran build = %+v", first)
	}

	kinds := map[replay.ProductionKind]bool{}
	for _, pc := range r.Production {
		kinds[pc.Kind] = true
	}
	for _, k := range []replay.ProductionKind{replay.KindBuild, replay.KindTrain, replay.KindUnitMorph,
		replay.KindBuildingMorph, replay.KindTech, replay.KindUpgrade} {
		if !kinds[k] {
			t.Errorf("no production command of kind %q", k)
		}
	}
}
