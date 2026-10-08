# Progress

Read this first each session. Update it whenever a step finishes.

## Current state

- **Current milestone:** M2 Analyze, awaiting owner review
- **Last updated:** 2026-10-07
- **Next step:** owner reviews M2, including the worker-estimate question below. After that, M3 (store and ingest). Don't start M3 unprompted.

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
| M1 Parse | done | `16a4a04`, `345eb0c`. Domain types, frame helpers, content hash, 1v1 filter, screp adapter, `bwa parse <file> [--json]`, golden test. Findings for all nine unconfirmed library items are in `docs/architecture.md`. Two remain open with safe fallbacks (Random race, and observers in non-melee games). |
| M2 Analyze | awaiting review | `271d27a`, `4caa697`, `485eb7b`, plus the docs commit. APM, EAPM, redundancy, per-minute curves, de-noised build orders with per-race allowlists, worker estimate, matchup, and `Analyze` for one replay. `bwa parse` prints APM, EAPM, redundancy, and build orders (`--minutes`, default 5). Tests cover the section 12 edge cases; computed APM, EAPM, and redundancy agree with screp on all 11 samples. |
| M3 Store and ingest | not started | |
| M4 Benchmarks | not started | |
| M5 Classifier | not started | |
| M6 API | not started | |
| M7 Package | not started | |

## Deviations from the plan

Anything built differently from `PLAN.md`, with the reason. One line each, prefixed with the milestone.

- M2: `replay.ProductionCmd` carries an `Effective` flag and `replay.Player` carries `Human` and `Observer` (added in M1), which the plan's domain model does not list. De-noising and the 1v1 filter need them.
- M2: de-noising goes beyond section 7's starting point, based on the samples. The 10 s window is measured from the previous repeat, not the first order. A different structure ordered on the same position within 10 s replaces the earlier one. Research repeated within 10 s collapses too.
- M2: the result (`win`/`loss`/`unknown`) is computed in the parser adapter, not in `internal/analyze`, because it needs only the library's winner team and each player's team.
- M2: `bwa parse` has a `--minutes` flag (not in section 5) that limits the build order, default 5. It keeps the golden files readable.
- M1: `go.mod` requires Go 1.25 instead of 1.24, because screp v1.13.4 requires it.
- M1: owner replays and their golden files are git-ignored (size, and the golden files carry player names). Only screp's public ShieldBattery ZvT sample (Apache-2.0) is tracked, so CI tests run on that one file.
- M0: the Makefile has a `RACE ?= -race` override (`make test RACE=`) for machines without cgo and a C compiler, such as this Windows box.

## Surprises in the replay data

Anything unexpected found in real replays. These feed `docs/architecture.md` and the owner's interview answers.

- M1: build positions are in tiles, but start locations are in pixels. The adapter multiplies build positions by 32.
- M1: ShieldBattery records 1v1 games as `Top vs Bottom`, so game type can't be used as a 1v1 signal.
- M1: screp's `ineffective` flag catches only some repeated build commands. 50 same-structure, same-tile repeats within 10 seconds were still marked effective, so M2 needs its own collapse step.
- M1: the map name in the header is truncated to 26 bytes. `MapData.Name` has the full name, with color control characters that have to be stripped.
- M2: research is spammed far more than structures. 127 effective repeats of the same research within 10 s, such as Psionic Storm clicked 0.2 s apart.
- M2: the worker estimate is only meaningful for Zerg. Terran and Protoss players queue workers, and orders they cannot afford are still recorded. The tracked sample's Terran has 17 effective SCV orders by 0:39 (850 minerals' worth, from a 50-mineral start).
- M1: screp only fills `Computed` (winner, teams, observers, ineffective flags) after an explicit `Compute()` call.

## Questions for the owner

Open questions that block or affect upcoming work. Remove each one when answered and record the answer in `docs/decisions.md`.

- Worker estimate (affects M5): the plan's rule (4 plus worker orders issued) works for Zerg but badly overcounts Terran and Protoss, because queued and unaffordable worker orders are still recorded. Section 8 only uses worker counts in Zerg rules, so M2 implements the rule as written. Options for M5: keep it Zerg-only (no change), or, if a Terran or Protoss rule ever needs worker counts, add a per-race correction such as ignoring worker orders while one is already pending. Recommendation: keep it Zerg-only.

- A replay with a Random pick, and one with an observer in a `One on One` or `Top vs Bottom` game, would settle the two open findings in `docs/architecture.md`.
