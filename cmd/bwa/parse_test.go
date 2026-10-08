package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

var update = flag.Bool("update", false, "rewrite testdata/golden from current output")

const (
	replayDir = "../../testdata/replays"
	goldenDir = "../../testdata/golden"
	// sample is screp's public ShieldBattery ZvT test replay, copied
	// locally. Nothing under testdata/ is committed, so tests that read it
	// skip when it is absent (as in CI).
	sample = replayDir + "/screp_shieldbattery_zvt.rep"
)

func requireSample(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(sample); errors.Is(err, os.ErrNotExist) {
		t.Skipf("%s not present; see README.md for how to add it", sample)
	}
}

// TestParseGolden compares bwa parse --json for every sample replay with its
// golden file. Run with -update to regenerate after an intended change.
func TestParseGolden(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(replayDir, "*.rep"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("no sample replays in testdata/replays")
	}
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run([]string{"parse", f, "--json"}, &stdout, &stderr); code != exitOK {
				t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
			}
			golden := filepath.Join(goldenDir, name+".json")
			if *update {
				if err := os.MkdirAll(goldenDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(golden, stdout.Bytes(), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("%v (run with -update to create it)", err)
			}
			if !bytes.Equal(stdout.Bytes(), want) {
				t.Errorf("output differs from %s (run with -update if the change is intended)\ngot:\n%s", golden, stdout.String())
			}
		})
	}
}

func TestParseBuildOrderMinutes(t *testing.T) {
	requireSample(t)
	steps := func(minutes string) []stepOutput {
		t.Helper()
		var stdout, stderr bytes.Buffer
		if code := run([]string{"parse", sample, "--json", "--minutes", minutes}, &stdout, &stderr); code != exitOK {
			t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
		}
		var out parseOutput
		if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		var all []stepOutput
		for _, p := range out.Players {
			if p.Analysis == nil {
				t.Fatalf("player %s has no analysis", p.Name)
			}
			all = append(all, p.Analysis.BuildOrder...)
		}
		return all
	}

	five, whole := steps("5"), steps("0")
	cutoff := replay.FrameAt(5 * time.Minute)
	for _, s := range five {
		if s.Frame >= cutoff {
			t.Errorf("step %s %s is past the 5 minute cutoff", s.Time, s.Name)
		}
	}
	if len(whole) <= len(five) {
		t.Errorf("whole game has %d steps, first 5 minutes %d", len(whole), len(five))
	}
}

// cliCase is one bwa invocation and what its output must contain.
type cliCase struct {
	name       string
	args       []string
	wantCode   int
	wantStdout string
	wantStderr string
}

func runCLICases(t *testing.T, tests []cliCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.wantCode, stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

// TestParseSampleOutput checks bwa parse output on the screp sample.
func TestParseSampleOutput(t *testing.T) {
	requireSample(t)
	runCLICases(t, []cliCase{
		{"text", []string{"parse", sample}, exitOK, "Map:       Eclipse 1.3", ""},
		{"text players", []string{"parse", sample}, exitOK, "   2  T     win       322   224         31%  IlIIlIlIlIIllII", ""},
		{"text matchup", []string{"parse", sample}, exitOK, "Matchup:   TvZ", ""},
		{"text build order", []string{"parse", sample}, exitOK, "   1:55  Spawning Pool\n", ""},
		{"build order cut at 5 minutes", []string{"parse", sample}, exitOK, "   4:48  Barracks\n", ""},
		{"whole build order", []string{"parse", sample, "--minutes", "0"}, exitOK, "Build order: [z]home (Z)\n", ""},
		{"json after file", []string{"parse", sample, "--json"}, exitOK, `"map": "Eclipse 1.3"`, ""},
		{"json before file", []string{"parse", "-json", sample}, exitOK, `"map": "Eclipse 1.3"`, ""},
	})
}

// TestParseCommand covers usage and file errors. None of these cases reads
// a real replay, so they run everywhere.
func TestParseCommand(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, size int) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, bytes.Repeat([]byte("x"), size), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	garbage := write("garbage.rep", 12)
	atCap := write("at-cap.rep", replay.MaxFileSize)
	overCap := write("over-cap.rep", replay.MaxFileSize+1)
	runCLICases(t, []cliCase{
		// Usage errors are caught before the file is opened.
		{"no file", []string{"parse"}, exitUsage, "", "Usage: bwa parse"},
		{"two files", []string{"parse", garbage, garbage}, exitUsage, "", "Usage: bwa parse"},
		{"bad flag", []string{"parse", "--nope", garbage}, exitUsage, "", "flag provided but not defined"},
		{"negative minutes", []string{"parse", garbage, "--minutes", "-1"}, exitUsage, "", "Usage: bwa parse"},
		{"help", []string{"parse", "-h"}, exitOK, "", "Usage: bwa parse"},
		{"missing file", []string{"parse", "does-not-exist.rep"}, exitFatal, "", "bwa parse:"},
		{"garbage file", []string{"parse", garbage}, exitFatal, "", "invalid replay"},
		// At the cap the file is read and then rejected by the parser;
		// one byte over, it is refused before parsing.
		{"file at the size cap", []string{"parse", atCap}, exitFatal, "", "invalid replay"},
		{"file over the size cap", []string{"parse", overCap}, exitFatal, "", "replay file too large (over 8 MB)"},
	})
}
