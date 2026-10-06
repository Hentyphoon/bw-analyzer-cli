# Brood War Replay Analyzer

`bwa` is a Go CLI and HTTP service that ingests StarCraft: Brood War replays,
extracts per-player metrics, stores them in PostgreSQL, and answers one
question well: **what does this opponent open with against my race, and how
often does it win?**

Status: milestone M1 (parse) done. `bwa parse <file> [--json]` works;
the other commands are stubs until their milestones land.

## Development

Requires Go 1.25+ (the minimum set by screp), Docker (for Postgres), and `golangci-lint` v2.

```sh
make build        # bin/bwa
make test         # go test -race ./...  (use RACE= without a C compiler)
make lint         # go vet + golangci-lint
make db-up        # start Postgres 16 via docker compose
```

Configuration: `DATABASE_URL` (defaults to the compose database in the
Makefile); everything else is a flag.
