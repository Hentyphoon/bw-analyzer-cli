# Progress

Read this first each session. Update it whenever a step finishes.

## Current state

- **Current milestone:** M1 Parse, awaiting owner review
- **Last updated:** 2026-10-06
- **Next step:** owner reviews M1. After that, M2 (analyze: EAPM buckets, de-noised build orders, classifier inputs). Don't start M2 unprompted.

## Owner prerequisites

The owner ticks these. Stop and ask if one is needed and still unticked.

- [x] 5 to 10 sample 1v1 replays in `testdata/replays/`, all three races, at least one pre-1.18 and one Remastered (needed for M1). 10 owner replays from TL.net (git-ignored) plus the tracked screp sample; see `docs/architecture.md`.
- [ ] Large benchmark corpus available locally, target 10,000 replays (needed for M4)
- [ ] Label set in `PLAN.md` section 8 reviewed (needed for M5)
- [ ] `labels.csv` with about 100 hand-labeled replays (needed for M5)

## Milestones

Status is one of: not started, in progress, awaiting review, done.

| Milestone | Status | Notes |
|---|---|---|
| M0 Scaffold | done | `4191db8`. Module layout, flag-based subcommand dispatch with stubs, Makefile, Postgres 16 compose, GitHub Actions CI (build, vet, golangci-lint, `go test -race`). Module renamed in `5df7203`. |
| M1 Parse | awaiting review | `16a4a04`, `345eb0c`. Domain types, frame helpers, content hash, 1v1 filter, screp adapter, `bwa parse <file> [--json]`, golden test. Findings for all nine unconfirmed library items are in `docs/architecture.md`. Two remain open with safe fallbacks (Random race, and observers in non-melee games). |
| M2 Analyze | not started | |
| M3 Store and ingest | not started | |
| M4 Benchmarks | not started | |
| M5 Classifier | not started | |
| M6 API | not started | |
| M7 Package | not started | |

## Deviations from the plan

Anything built differently from `PLAN.md`, with the reason. One line each, prefixed with the milestone.

- M1: `go.mod` requires Go 1.25 instead of 1.24, because screp v1.13.4 requires it.
- M1: owner replays and their golden files are git-ignored (size, and the golden files carry player names). Only screp's public ShieldBattery ZvT sample (Apache-2.0) is tracked, so CI tests run on that one file.
- M0: the Makefile has a `RACE ?= -race` override (`make test RACE=`) for machines without cgo and a C compiler, such as this Windows box.

## Surprises in the replay data

Anything unexpected found in real replays. These feed `docs/architecture.md` and the owner's interview answers.

- M1: build positions are in tiles, but start locations are in pixels. The adapter multiplies build positions by 32.
- M1: ShieldBattery records 1v1 games as `Top vs Bottom`, so game type can't be used as a 1v1 signal.
- M1: screp's `ineffective` flag catches only some repeated build commands. 50 same-structure, same-tile repeats within 10 seconds were still marked effective, so M2 needs its own collapse step.
- M1: the map name in the header is truncated to 26 bytes. `MapData.Name` has the full name, with color control characters that have to be stripped.
- M1: screp only fills `Computed` (winner, teams, observers, ineffective flags) after an explicit `Compute()` call.

## Questions for the owner

Open questions that block or affect upcoming work. Remove each one when answered and record the answer in `docs/decisions.md`.

- A replay with a Random pick, and one with an observer in a `One on One` or `Top vs Bottom` game, would settle the two open findings in `docs/architecture.md`.
