# Progress

Read this first each session. Update it whenever a step finishes.

## Current state

- **Current milestone:** M2 Analyze, awaiting owner review
- **Last updated:** 2026-10-07
- **Next step:** owner reviews M2. After that, M3 (store and ingest). Don't start M3 unprompted.

## Owner prerequisites

The owner ticks these. Stop and ask if one is needed and still unticked.

- [x] 5 to 10 sample 1v1 replays in `testdata/replays/`, all three races, at least one pre-1.18 and one Remastered (needed for M1). 10 owner replays from TL.net plus screp's public sample, all local only; see `docs/architecture.md`.
- [ ] Large benchmark corpus available locally, target 10,000 replays (needed for M4)
- [ ] Label set in `PLAN.md` section 8 reviewed (needed for M5)
- [ ] `labels.csv` with about 100 hand-labeled replays (needed for M5)

## Milestones

Status is one of: not started, in progress, awaiting review, done.

| Milestone | Status | Notes |
|---|---|---|
| M0 Scaffold | done | `4191db8`. Module layout, flag-based subcommand dispatch with stubs, Makefile, Postgres 16 compose, GitHub Actions CI (build, vet, golangci-lint, `go test -race`). Module renamed in `5df7203`. |
| M1 Parse | done | `16a4a04`, `345eb0c`. Domain types, frame helpers, content hash, 1v1 filter, screp adapter, `bwa parse <file> [--json]`, golden test. Findings for all nine unconfirmed library items are in `docs/architecture.md`. Two remain open with safe fallbacks (Random race, and observers in non-melee games). |
| M2 Analyze | awaiting review | `271d27a`, `4caa697`, `485eb7b`, plus docs, the worker queue model, the 8 MB size cap in `bwa parse`, and review fixes. CI passed on `8ecf9a2` (build, vet, golangci-lint, `go test -race`), with the replay-dependent tests skipped there because `testdata/` is not committed. APM, EAPM, redundancy, per-minute curves, de-noised build orders with per-race allowlists, worker estimate, matchup, and `Analyze` for one replay. `bwa parse` prints APM, EAPM, redundancy, and build orders (`--minutes`, default 5). Tests cover the section 12 edge cases; computed APM, EAPM, and redundancy agree with screp on all 11 samples. |
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
- M2: the worker estimate models SCV and Probe production as a one-at-a-time town hall queue (300 frames per worker) instead of section 7's "four plus worker orders issued", which overcounts Terran and Protoss by 3 to 30. Zerg keeps the plan's rule.
- M2: `bwa parse` has a `--minutes` flag (not in section 5) that limits the build order, default 5. It keeps the golden files readable.
- M1: `go.mod` requires Go 1.25 instead of 1.24, because screp v1.13.4 requires it.
- M2: nothing under `testdata/` is committed, not even screp's public sample (owner decision). Tests that read a replay skip without one, so CI checks the code with hand-built data only: 8 tests skip there, including the golden test, the library-agreement tests, and the `bwa parse` output checks. They run locally.
- M0: the Makefile has a `RACE ?= -race` override (`make test RACE=`) for machines without cgo and a C compiler, such as this Windows box.

## Surprises in the replay data

Anything unexpected found in real replays. These feed `docs/architecture.md` and the owner's interview answers.

- M1: build positions are in tiles, but start locations are in pixels. The adapter multiplies build positions by 32.
- M1: ShieldBattery records 1v1 games as `Top vs Bottom`, so game type can't be used as a 1v1 signal.
- M1: screp's `ineffective` flag catches only some repeated build commands. 50 same-structure, same-tile repeats within 10 seconds were still marked effective, so M2 needs its own collapse step.
- M1: the map name in the header is truncated to 26 bytes. `MapData.Name` has the full name, with color control characters that have to be stripped.
- M2: research is spammed far more than structures. 127 effective repeats of the same research within 10 s, such as Psionic Storm clicked 0.2 s apart.
- M2: counting worker orders badly overcounts Terran and Protoss workers. Players queue workers, and orders they cannot afford are still recorded. The screp sample's Terran has 17 effective SCV orders by 0:39 (850 minerals' worth, from a 50-mineral start). Modeling the town hall as a one-at-a-time queue gives the standard opening counts (8 Pylon, 9 Depot, 10 Gate, 11 Rax) for every sample player.
- M2: worker counts are taken when a build command is issued, not when the structure is placed. For an expansion the drone or probe walks for several seconds first, so a "12 Hatch" shows as 11 workers in all five samples. Main-base structures match their names exactly (9 Overlord, Overpool at 9). The classifier's "12 Hatch" rule is order-based, so this does not misclassify.
- M1: screp only fills `Computed` (winner, teams, observers, ineffective flags) after an explicit `Compute()` call.

## Notes for M3

From the M2 review. Everything else the review found was fixed in M2 or accepted (see `docs/decisions.md`).

- The pipeline reads files with the same `replay.MaxFileSize` cap and returns `replay.ErrTooLarge` as a `failed` reason.
- `analyze.Analyze` errors wrap `ErrNot1v1` with the filter's reason, but store the reason itself from `Replay.SkipReason()` rather than parsing the error text.
- Wrap the whole parse, analyze, persist step per replay in a `recover()`; only the parser has one today.

## Questions for the owner

Open questions that block or affect upcoming work. Remove each one when answered and record the answer in `docs/decisions.md`.

- A replay with a Random pick, and one with an observer in a `One on One` or `Top vs Bottom` game, would settle the two open findings in `docs/architecture.md`.
