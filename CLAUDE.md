# Brood War Replay Analyzer

Go CLI (`bwa`) and HTTP service that ingests StarCraft: Brood War replays into PostgreSQL and serves opponent scouting reports.

## Start here

- `PLAN.md` is the spec. Read section 14 (milestones) and the sections the current milestone touches. Don't load the whole file every session.
- `docs/progress.md` records the current milestone, what is done, and any deviations from the plan. Read it first each session and update it whenever a step finishes. It also holds the owner's prerequisite checklist.
- `docs/benchmarks.md` is a template with empty cells. Fill it in place from real runs and keep its sections.
- `docs/goals.md` says why the project exists and which claims it has to back with evidence. Read it once at the start, and again before benchmark, evaluation, or documentation work.
- `docs/decisions.md` records why the plan is the way it is. Read it before changing an approach, and append to it when you make a new decision.
- If this file and `PLAN.md` disagree, follow `PLAN.md` and tell the owner.
- This file was written from `PLAN.md` before the code existed. A line that doesn't match the current code (a command, a path, a flag) may describe a later step. Leave it as written; don't "fix" it to match today's state. The exception is a rule the owner has since changed in conversation, which is recorded in `docs/decisions.md`.

## Workflow

- Work on one milestone at a time. When its "Done when" checks pass, summarize what was built and what deviated, update `docs/progress.md`, and stop for review. Don't start the next milestone unprompted.
- Run `make test lint` before calling any step done.
- Commit after each coherent step with the milestone in the message, for example `M2: add EAPM per-minute buckets`. Don't push.
- Keep every `.md` file current as you go, not only at milestone end: `README.md`, `docs/*.md`, and this file when a convention changes. `PLAN.md` is the spec; don't edit it, record deviations in `docs/progress.md` instead.
- Stop and ask when `testdata/replays/` is empty, when a new dependency seems necessary, or when real replay data contradicts the plan.

## Commands

The Makefile is created in M0. Keep this list in sync with it.

- `make build`: build the `bwa` binary
- `make test`: `go test -race ./...`
- `make lint`: `go vet` and `golangci-lint run`
- `make db-up`: start Postgres with docker compose
- `make migrate`: apply migrations
- `make bench CORPUS=<dir>`: run the ingest benchmark matrix
- Single test: `go test -race -run TestName ./internal/analyze/`
- Golden files: `go test ./internal/parser/ -update`, only when an output change is intended. Review the diff before committing.

Integration tests skip silently unless `TEST_DATABASE_URL` is set. Run `make db-up` and set it before claiming store, pipeline, or API work is tested.

## Hard rules

- **Dependencies.** Only `github.com/icza/screp`, `github.com/jackc/pgx/v5`, and `golang.org/x/sync`. Everything else is standard library: `net/http`, `flag`, `log/slog`, `testing`. Ask before adding a module.
- **Measurements.** Never write a number into `docs/benchmarks.md` or the README that did not come from a run in this repo. Record the command and hardware beside it. The owner cites these numbers in interviews, so an invented one is worse than a missing one.
- **Library API.** Confirm every `screp` name with `go doc github.com/icza/screp/...` or the source in the module cache before using it. `PLAN.md` section 6 separates what is confirmed from what must be checked against real replays.
- **Package boundaries.** `screp` types stay inside `internal/parser`. `pgx` types stay inside `internal/store`. `internal/analyze` is pure functions over `internal/replay` types with no I/O. `internal/replay` imports nothing internal.
- **Explainable code.** The owner will be interviewed on this codebase. Prefer plain, idiomatic Go to clever abstractions, and comment concurrency decisions with the reason, not the mechanics.

## Go conventions

- `context.Context` is the first parameter of anything that does I/O. Never store one in a struct.
- Wrap errors with `fmt.Errorf("doing x: %w", err)` and compare sentinels with `errors.Is`.
- No package-level mutable state. Pass dependencies through constructors.
- Every goroutine has an owner that waits for it, normally an `errgroup`. No bare `go` statement without a shutdown path.
- Tests are table-driven with `t.Run` subtests.
- Log with `slog` and attach the replay file name or hash as attributes. Only `cmd/bwa` prints to stdout.

## Domain gotchas

- Time is counted in frames (1 frame = 42 ms). Convert only through the helper in `internal/replay`.
- All computer players share player ID 255. Apply the 1v1 filter before keying anything by player ID.
- The winner can be unknown. Win rates exclude unknown results and report both total games and games with a known result.
- A build order is a list of commands the player issued, not of things that finished. Name and document it that way.
- A player's identity is their name, compared case-insensitively. Replays carry no account ID.
- Assume replays are 1v1 games (owner decision, M2). Don't add handling for observers that screp misses in `One on One` or `Top vs Bottom` games. A player who picked Random is recorded under the race they played.
- Assume replays are valid Brood War replay files (owner decision, M2). Enforce the 8 MB size cap (`replay.MaxFileSize`) and pass on the errors the library returns, but don't add defensive code for corrupt or adversarial data, such as implausible header values. Keep the parser's existing `recover()` and M3's per-replay failure isolation, which the plan requires.
- Never commit anything under `testdata/` (owner decision, M2). Sample replays, golden files, and the benchmark corpus are git-ignored and stay local, including screp's public sample. Tests that need a replay skip when it is absent, so CI runs without real replays. How to set up the samples locally is in `README.md`.
