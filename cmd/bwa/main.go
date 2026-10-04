// Command bwa ingests StarCraft: Brood War replays, extracts per-player
// metrics, stores them in PostgreSQL, and serves them over HTTP.
//
// This file only dispatches subcommands; the work lives in internal/.
package main

import (
	"fmt"
	"io"
	"os"
)

// Exit codes. Individual replay failures are never fatal; only bad usage and
// fatal conditions (such as an unreachable database) produce a non-zero exit.
const (
	exitOK    = 0
	exitFatal = 1
	exitUsage = 2
)

// command is one bwa subcommand. Each run function parses its own flags
// from args (which excludes the subcommand name) and returns an exit code.
type command struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

var commands = []command{
	{"parse", "print one replay's summary and build orders", notImplemented("parse")},
	{"ingest", "concurrently ingest a directory of replays into Postgres", notImplemented("ingest")},
	{"eval", "score the opening classifier against hand labels", notImplemented("eval")},
	{"serve", "serve the REST API", notImplemented("serve")},
	{"migrate", "apply database migrations", notImplemented("migrate")},
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		usage(stdout)
		return exitOK
	}
	for _, c := range commands {
		if c.name == args[0] {
			return c.run(args[1:], stdout, stderr)
		}
	}
	printf(stderr, "bwa: unknown command %q\n\n", args[0])
	usage(stderr)
	return exitUsage
}

func usage(w io.Writer) {
	printf(w, "Usage: bwa <command> [flags]\n\nCommands:\n")
	for _, c := range commands {
		printf(w, "  %-8s %s\n", c.name, c.summary)
	}
	printf(w, "\nRun 'bwa <command> -h' for the flags of a command.\n")
}

// notImplemented is a placeholder until the command's milestone lands.
func notImplemented(name string) func([]string, io.Writer, io.Writer) int {
	return func(_ []string, _, stderr io.Writer) int {
		printf(stderr, "bwa %s: not implemented yet\n", name)
		return exitFatal
	}
}

// printf writes to a terminal stream. There is nowhere to report a failed
// write to stdout or stderr, so the error is deliberately discarded.
func printf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
