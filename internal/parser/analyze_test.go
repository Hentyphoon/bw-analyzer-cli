package parser

// These tests check internal/analyze against screp. They live here because
// this is the only package allowed to import screp.

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/icza/screp/rep/repcmd"
	"github.com/icza/screp/repparser"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/analyze"
	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// TestActivityMatchesLibrary checks that APM and EAPM computed by
// internal/analyze round to the values screp reports.
func TestActivityMatchesLibrary(t *testing.T) {
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

			for _, pd := range lib.Computed.PlayerDescs {
				a := analyze.PlayerActivity(r, pd.PlayerID)
				// screp rounds half up to an integer.
				if got := int32(a.APM + 0.5); got != pd.APM {
					t.Errorf("player %d: APM %.2f rounds to %d, library says %d", pd.PlayerID, a.APM, got, pd.APM)
				}
				if got := int32(a.EAPM + 0.5); got != pd.EAPM {
					t.Errorf("player %d: EAPM %.2f rounds to %d, library says %d", pd.PlayerID, a.EAPM, got, pd.EAPM)
				}
				if pd.CmdCount == 0 {
					continue
				}
				if want := pd.RedundancyFloat() / 100; math.Abs(a.Redundancy-want) > 1e-9 {
					t.Errorf("player %d: redundancy %v, library says %v", pd.PlayerID, a.Redundancy, want)
				}
			}
		})
	}
}

// TestWorkerEstimateOnSample checks the worker estimate against the screp
// sample's Terran, who opens with the standard 9 Depot and 11 Barracks. The
// plain count of SCV orders would put the Depot at 21 workers.
func TestWorkerEstimateOnSample(t *testing.T) {
	r, err := Screp{}.Parse(readSample(t))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := analyze.Analyze(r)
	if err != nil {
		t.Fatal(err)
	}
	var terran *analyze.PlayerReport
	for i := range rep.Players {
		if rep.Players[i].Player.Race == replay.Terran {
			terran = &rep.Players[i]
		}
	}
	if terran == nil {
		t.Fatal("sample has no Terran")
	}
	want := map[string]int{"Supply Depot": 9, "Barracks": 11}
	for _, s := range terran.BuildOrder {
		w, ok := want[s.Name]
		if !ok {
			continue
		}
		if got := analyze.WorkersBefore(terran.BuildOrder, s.Frame); got != w {
			t.Errorf("first %s at %s: %d workers, want %d", s.Name, s.Frame, got, w)
		}
		delete(want, s.Name) // only the first of each
	}
	if len(want) > 0 {
		t.Errorf("structures not found in the build order: %v", want)
	}
}

// TestAllowlistNamesExistInLibrary guards against a typo in an allowlist,
// which would silently drop that unit or research from every build order.
func TestAllowlistNamesExistInLibrary(t *testing.T) {
	known := map[string]bool{}
	for _, u := range repcmd.Units {
		known[u.Name] = true
	}
	for _, x := range repcmd.Techs {
		known[x.Name] = true
	}
	for _, x := range repcmd.Upgrades {
		known[x.Name] = true
	}
	for _, race := range []replay.Race{replay.Terran, replay.Protoss, replay.Zerg} {
		names := analyze.AllowedNames(race)
		if len(names) == 0 {
			t.Errorf("race %s has an empty allowlist", race)
		}
		for _, n := range names {
			if !known[n] {
				t.Errorf("race %s: %q is not a screp unit, tech, or upgrade name", race, n)
			}
		}
	}
}
