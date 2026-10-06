package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Hentyphoon/bw-analyzer-cli/internal/parser"
	"github.com/Hentyphoon/bw-analyzer-cli/internal/replay"
)

// parseOutput is the --json shape of bwa parse. It is also the golden file
// format, so changing it means regenerating testdata/golden.
type parseOutput struct {
	File            string         `json:"file"`
	Hash            string         `json:"hash"`
	Map             string         `json:"map"`
	Version         string         `json:"version"`
	GameType        string         `json:"game_type"`
	PlayedAt        time.Time      `json:"played_at"`
	Frames          replay.Frame   `json:"frames"`
	DurationSeconds float64        `json:"duration_seconds"`
	SkipReason      string         `json:"skip_reason,omitempty"`
	ParseErrors     int            `json:"parse_errors"`
	Players         []playerOutput `json:"players"`
}

type playerOutput struct {
	replay.Player
	Actions          int `json:"actions"`
	EffectiveActions int `json:"effective_actions"`
	ProductionCmds   int `json:"production_cmds"`
}

func runParse(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print JSON instead of text")
	fs.Usage = func() {
		printf(stderr, "Usage: bwa parse <file> [--json]\n\nFlags:\n")
		fs.PrintDefaults()
	}
	files, err := parseInterspersed(fs, args)
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	if err != nil {
		return exitUsage
	}
	if len(files) != 1 {
		fs.Usage()
		return exitUsage
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		printf(stderr, "bwa parse: %v\n", err)
		return exitFatal
	}
	r, err := parser.Screp{}.Parse(data)
	if err != nil {
		printf(stderr, "bwa parse: %s: %v\n", files[0], err)
		return exitFatal
	}
	r.Hash = replay.ContentHash(data)
	r.FileName = filepath.Base(files[0])

	out := summarize(r)
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			printf(stderr, "bwa parse: %v\n", err)
			return exitFatal
		}
		return exitOK
	}
	printText(stdout, out)
	return exitOK
}

func summarize(r *replay.Replay) parseOutput {
	out := parseOutput{
		File:            r.FileName,
		Hash:            r.Hash,
		Map:             r.MapName,
		Version:         r.Version,
		GameType:        r.GameType,
		PlayedAt:        r.StartTime,
		Frames:          r.Frames,
		DurationSeconds: r.Frames.Seconds(),
		SkipReason:      r.SkipReason(),
		ParseErrors:     r.ParseErrors,
	}
	idx := make(map[byte]int, len(r.Players))
	for _, p := range r.Players {
		idx[p.ID] = len(out.Players)
		out.Players = append(out.Players, playerOutput{Player: p})
	}
	for _, a := range r.Actions {
		if i, ok := idx[a.PlayerID]; ok {
			out.Players[i].Actions++
			if a.Effective {
				out.Players[i].EffectiveActions++
			}
		}
	}
	for _, pc := range r.Production {
		if i, ok := idx[pc.PlayerID]; ok {
			out.Players[i].ProductionCmds++
		}
	}
	return out
}

func printText(w io.Writer, out parseOutput) {
	printf(w, "File:      %s\n", out.File)
	printf(w, "Map:       %s\n", out.Map)
	printf(w, "Version:   %s (%s)\n", out.Version, out.GameType)
	printf(w, "Played at: %s\n", out.PlayedAt.Format("2006-01-02 15:04:05 MST"))
	printf(w, "Duration:  %s\n", out.Frames)
	if out.SkipReason != "" {
		printf(w, "Skipped:   %s\n", out.SkipReason)
	}
	if out.ParseErrors > 0 {
		printf(w, "Warning:   %d commands failed to parse\n", out.ParseErrors)
	}
	printf(w, "\nTeam  Race  Result   Player\n")
	for _, p := range out.Players {
		res, note := string(p.Result), ""
		if p.Observer {
			res, note = "-", " (observer)"
		} else if !p.Human {
			note = " (computer)"
		}
		printf(w, "%4d  %-4s  %-7s  %s%s\n", p.Team, p.Race, res, p.Name, note)
	}
}

// parseInterspersed parses fs from args, allowing flags after positional
// arguments ("bwa parse x.rep --json"), which the flag package alone stops at.
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return positional, nil
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
}
