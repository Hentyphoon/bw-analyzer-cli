package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite testdata/golden from current output")

const (
	replayDir = "../../testdata/replays"
	goldenDir = "../../testdata/golden"
	sample    = replayDir + "/screp_shieldbattery_zvt.rep"
)

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

func TestParseCommand(t *testing.T) {
	garbage := filepath.Join(t.TempDir(), "garbage.rep")
	if err := os.WriteFile(garbage, []byte("not a replay"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{"text", []string{"parse", sample}, exitOK, "Map:       Eclipse 1.3", ""},
		{"text players", []string{"parse", sample}, exitOK, "   2  T     win      IlIIlIlIlIIllII", ""},
		{"json after file", []string{"parse", sample, "--json"}, exitOK, `"map": "Eclipse 1.3"`, ""},
		{"json before file", []string{"parse", "-json", sample}, exitOK, `"map": "Eclipse 1.3"`, ""},
		{"no file", []string{"parse"}, exitUsage, "", "Usage: bwa parse"},
		{"two files", []string{"parse", sample, sample}, exitUsage, "", "Usage: bwa parse"},
		{"bad flag", []string{"parse", "--nope", sample}, exitUsage, "", "flag provided but not defined"},
		{"help", []string{"parse", "-h"}, exitOK, "", "Usage: bwa parse"},
		{"missing file", []string{"parse", "does-not-exist.rep"}, exitFatal, "", "bwa parse:"},
		{"garbage file", []string{"parse", garbage}, exitFatal, "", "invalid replay"},
	}
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
