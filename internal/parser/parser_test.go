package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/icza/screp/rep"
	"github.com/icza/screp/rep/repcmd"
	"github.com/icza/screp/rep/repcore"
	"github.com/icza/screp/repparser"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// sampleFile is screp's public ShieldBattery ZvT test replay, copied
// locally. Nothing under testdata/ is committed, so tests that need it skip
// when it is absent (as in CI).
const sampleFile = "../../testdata/replays/screp_shieldbattery_zvt.rep"

func readSample(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(sampleFile)
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("%s not present; see README.md for how to add it", sampleFile)
	}
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	return data
}

func TestParseInvalidInput(t *testing.T) {
	// truncate cuts the sample replay; it skips the case when the sample is
	// absent, while the other cases still run.
	truncate := func(t *testing.T, n func(int) int) []byte {
		sample := readSample(t)
		return sample[:n(len(sample))]
	}
	tests := []struct {
		name string
		data func(t *testing.T) []byte
	}{
		{"empty", func(*testing.T) []byte { return nil }},
		{"garbage", func(*testing.T) []byte { return []byte("this is not a replay file at all, just some text") }},
		{"zeros", func(*testing.T) []byte { return make([]byte, 4096) }},
		{"replay id only", func(t *testing.T) []byte { return truncate(t, func(int) int { return 16 }) }},
		{"truncated header", func(t *testing.T) []byte { return truncate(t, func(int) int { return 300 }) }},
		{"truncated commands", func(t *testing.T) []byte { return truncate(t, func(n int) int { return n / 3 }) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Screp{}.Parse(tt.data(t))
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
			lib, err := repparser.ParseConfig(data, screpConfig())
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

func TestRaceFromBuilds(t *testing.T) {
	build := func(pid byte, unitID uint16) repcmd.Cmd {
		return &repcmd.BuildCmd{Base: &repcmd.Base{PlayerID: pid, Type: repcmd.TypeBuild}, Unit: repcmd.UnitByID(unitID)}
	}
	train := func(pid byte, unitID uint16) repcmd.Cmd {
		return &repcmd.TrainCmd{Base: &repcmd.Base{PlayerID: pid, Type: repcmd.TypeTrain}, Unit: repcmd.UnitByID(unitID)}
	}
	const marine = 0x00 // a unit, which RaceOfUnitID does not know
	tests := []struct {
		name string
		cmds []repcmd.Cmd
		want replay.Race
	}{
		{"no commands", nil, replay.RaceUnknown},
		{"first structure decides", []repcmd.Cmd{build(0, repcmd.UnitIDSupplyDepot), build(0, repcmd.UnitIDPylon)}, replay.Terran},
		{"zerg", []repcmd.Cmd{build(0, repcmd.UnitIDSpawningPool)}, replay.Zerg},
		{"protoss", []repcmd.Cmd{build(0, repcmd.UnitIDPylon)}, replay.Protoss},
		{"other player's structures ignored", []repcmd.Cmd{build(1, repcmd.UnitIDHatchery), build(0, repcmd.UnitIDPylon)}, replay.Protoss},
		{"units are skipped", []repcmd.Cmd{train(0, marine), build(0, repcmd.UnitIDHatchery)}, replay.Zerg},
		{"only units", []repcmd.Cmd{train(0, marine)}, replay.RaceUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := raceFromBuilds(tt.cmds, 0); got != tt.want {
				t.Errorf("raceFromBuilds = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRandomRaceRecordedAsPlayed rewrites the screp sample's Terran to
// header race 6 (Random) and checks the adapter still reports Terran.
func TestRandomRaceRecordedAsPlayed(t *testing.T) {
	sr, err := repparser.ParseConfig(readSample(t), screpConfig())
	if err != nil {
		t.Fatal(err)
	}
	sr.Compute()
	const terranID = 1
	sr.Header.PIDPlayers[terranID].Race = repcore.RaceByID(6)

	r := convert(sr)
	for _, p := range r.Players {
		if p.ID == terranID && p.Race != replay.Terran {
			t.Errorf("player with Random header race = %q, want %q", p.Race, replay.Terran)
		}
	}
	if reason := r.SkipReason(); reason != "" {
		t.Errorf("SkipReason = %q, want the game kept", reason)
	}
}

func TestMarkNonBuildersAsObservers(t *testing.T) {
	human := func(id byte, team int) replay.Player {
		return replay.Player{ID: id, Race: replay.Terran, Team: team, Human: true, Result: replay.Win}
	}
	builds := func(ids ...byte) []repcmd.Cmd {
		var cmds []repcmd.Cmd
		for _, id := range ids {
			cmds = append(cmds, &repcmd.BuildCmd{Base: &repcmd.Base{PlayerID: id, Type: repcmd.TypeBuild}, Unit: repcmd.UnitByID(repcmd.UnitIDSupplyDepot)})
		}
		return cmds
	}
	cpu := replay.Player{ID: 255, Race: replay.Zerg, Team: 2}
	tests := []struct {
		name    string
		players []replay.Player
		cmds    []repcmd.Cmd
		wantObs []bool
	}{
		{"1v1, both build", []replay.Player{human(0, 1), human(1, 2)}, builds(0, 1), []bool{false, false}},
		{"1v1 plus an idle human", []replay.Player{human(0, 1), human(1, 2), human(2, 3)}, builds(0, 1), []bool{false, false, true}},
		{"would leave one player", []replay.Player{human(0, 1), human(1, 2)}, builds(0), []bool{false, false}},
		{"nobody builds", []replay.Player{human(0, 1), human(1, 2), human(2, 3)}, nil, []bool{false, false, false}},
		{"computer left alone", []replay.Player{human(0, 1), cpu}, builds(0), []bool{false, false}},
		{"2v2 with one idle player", []replay.Player{human(0, 1), human(1, 1), human(2, 2), human(3, 2)}, builds(0, 1, 3), []bool{false, false, true, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markNonBuildersAsObservers(tt.players, tt.cmds)
			for i, p := range tt.players {
				if p.Observer != tt.wantObs[i] {
					t.Errorf("player %d observer = %v, want %v", i, p.Observer, tt.wantObs[i])
				}
				if p.Observer && p.Result != replay.ResultUnknown {
					t.Errorf("observer %d result = %q, want %q", i, p.Result, replay.ResultUnknown)
				}
			}
		})
	}
}

// TestObserverInTopVsBottomGame adds an idle human to the screp sample, a
// ShieldBattery "Top vs Bottom" game where screp does not look for
// observers, and checks the game is still kept as a 1v1.
func TestObserverInTopVsBottomGame(t *testing.T) {
	sr, err := repparser.ParseConfig(readSample(t), screpConfig())
	if err != nil {
		t.Fatal(err)
	}
	sr.Compute()
	if got := sr.Header.Type.Name; got != "Top vs Bottom" {
		t.Fatalf("sample game type = %q, want Top vs Bottom", got)
	}
	obs := &rep.Player{SlotID: 2, ID: 2, Type: repcore.PlayerTypeHuman, Race: repcore.RaceProtoss, Team: 3, Name: "watcher"}
	sr.Header.Players = append(sr.Header.Players, obs)
	sr.Header.PIDPlayers[obs.ID] = obs
	sr.Computed.PlayerDescs = append(sr.Computed.PlayerDescs, &rep.PlayerDesc{PlayerID: obs.ID})

	r := convert(sr)
	if reason := r.SkipReason(); reason != "" {
		t.Errorf("SkipReason = %q, want the game kept", reason)
	}
	for _, p := range r.Players {
		if p.ID == obs.ID && !p.Observer {
			t.Errorf("idle player not marked as observer: %+v", p)
		}
	}
}
