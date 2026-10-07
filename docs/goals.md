# Project goals

## Why this exists

This is a portfolio project for the owner's resume, aimed at Go backend roles. It has two jobs:

1. Back up a specific set of resume claims with code and measurements a reviewer can check in the repo.
2. Be something the owner can explain line by line in an interview.

A feature that serves neither job is out of scope, even if it would be nice to have.

## The claims this repo must support

These are the draft resume bullets. Numbers marked *target* were estimated before any code existed.

1. Built a concurrent pipeline in Go that parses StarCraft: Brood War replays into build orders and APM/EAPM curves, processing up to **300 replays/sec** (*target*) with a bounded goroutine worker pool, channel-based backpressure, and context-driven timeouts and cancellation.
2. Cut ingest time for 10,000 replays by up to **30x** (*target*), guided by pprof CPU and heap profiles and `go test` benchmarks: parallelized parsing across workers and batched PostgreSQL writes with pgx `CopyFrom`.
3. Shipped a REST API on Go's standard `net/http` serving matchup win rates and opponent opening tendencies from a rule-based build-order classifier (**85% accuracy** on 100 hand-labeled replays, *target*), with graceful shutdown, table-driven tests, and GitHub Actions CI.

## How to treat the targets

- **They are guesses, not requirements.** Measure honestly and report what you get. The owner will change the resume to match the repo, not the other way round.
- **Don't shape a measurement to reach a target.** No slowing the baseline, no dropping slow replays from the corpus, no fitting classifier rules to the same labels used to report accuracy.
- **"Up to" means the best run that was actually measured.** Record the exact figure and the configuration that produced it.
- **Classifier accuracy is reported on labels that were not used for tuning.** If the owner provides a single label set, split it and say how in `docs/benchmarks.md`.
- **If a named technique turns out not to matter**, for example `CopyFrom` gives no measurable gain, say so plainly at the milestone review so the bullet can be reworded.

## Where each claim must be visible

Every technique named in a bullet needs code a reviewer can find and something that proves it works.

| Claim | Lives in | Proven by |
|---|---|---|
| Bounded goroutine worker pool | `internal/pipeline` | Unit or integration test; worker-scaling rows in `docs/benchmarks.md` |
| Channel-based backpressure | `internal/pipeline` | Comment at the channel; explained in `docs/architecture.md` |
| Context-driven timeouts and cancellation | `internal/pipeline`, `cmd/bwa` | Integration test that cancels mid-batch |
| Replays/sec figure | `bwa ingest` summary | `docs/benchmarks.md` |
| Ingest speedup | `--write-mode` and `--workers` flags | Baseline and best rows in `docs/benchmarks.md` |
| pprof-guided optimization | `--cpuprofile`, `--memprofile` | One documented finding with before and after |
| `go test` benchmarks | `internal/analyze` | `Benchmark*` functions and their output |
| pgx `CopyFrom` | `internal/store` | Test that both write modes produce identical rows |
| `net/http` REST API | `internal/api` | `httptest` suite |
| Win rates and opening tendencies | `internal/api`, `internal/store` | Endpoint tests against ingested samples |
| Classifier accuracy | `internal/analyze`, `bwa eval` | Accuracy and confusion matrix in `docs/benchmarks.md` |
| Graceful shutdown | `cmd/bwa`, `internal/api` | Test or documented manual check |
| Table-driven tests | All packages | The tests themselves |
| GitHub Actions CI | `.github/workflows/ci.yml` | Passing run on the default branch |

## Interview questions the repo should answer

Write `docs/architecture.md` so the owner can answer these from it:

- Why a bounded worker pool, and how was its size chosen?
- How do backpressure and graceful shutdown work, step by step?
- Where does ingest time go (parsing, analysis, database), and what does that imply for the worker count?
- How are legacy and Remastered replays handled?
- How are APM and EAPM defined, and what are the trade-offs?
- What was the most surprising thing in the replay data?
- What would change at 100 times the scale?

When something in the replay data surprises you, note it in `docs/progress.md` at the time. Those notes become the answer to the sixth question.

## Who reads the repo

Reviewers skim. They look at the README, the directory layout, whether CI is green, and one or two files. The README needs a working quick start, the metric definitions, the known limitation about command-based build orders, and the headline benchmark results with a link to the full table.
