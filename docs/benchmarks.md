# Benchmarks

Every number in this file comes from a run in this repository, with the command and environment recorded beside it. Empty cells mean "not measured yet". Never fill one with an estimate.

## Environment

| | |
|---|---|
| Date of run | |
| CPU model and core count | |
| RAM | |
| Disk type | |
| OS | |
| Go version | |
| PostgreSQL version | |
| Postgres location (same machine or remote) | |
| Git commit | |

## Corpus

| | |
|---|---|
| Source | |
| Files | |
| Total size | |
| Ingested ok | |
| Skipped (not 1v1, too short) | |
| Failed to parse | |

## Ingest throughput

Command: `make bench CORPUS=<dir>`

| Run | Workers | Write mode | Wall time | Replays/sec | Speedup vs baseline |
|---|---|---|---|---|---|
| Baseline | 1 | rows | | | 1.0x |
| + COPY | 1 | copy | | | |
| + workers | 2 | copy | | | |
| + workers | 4 | copy | | | |
| + workers | 8 | copy | | | |
| + workers | NumCPU | copy | | | |

**Headline figures** (the best measured run, and the configuration that produced it):

- Peak throughput:
- Best speedup over baseline:

## Where the time goes

Cumulative time per stage, from the `bwa ingest` summary.

| Run | Parse | Analyze | Persist |
|---|---|---|---|
| Baseline | | | |
| Best configuration | | | |

What this implies for worker count and for the next optimization:

## Profiling finding

Command used to capture the profile:

- **Hotspot found:**
- **Change made:**
- **Before:**
- **After:**
- **If no change was practical, why:**

## Micro-benchmarks

Command: `go test -bench . -benchmem ./internal/analyze/`

| Benchmark | ns/op | B/op | allocs/op |
|---|---|---|---|
| | | | |

## Classifier evaluation

Command: `bwa eval --labels labels.csv --dir <dir>`

| | |
|---|---|
| Labeled replays in total | |
| Used for tuning rules | |
| Held out for this score | |
| Overall accuracy on held-out set | |
| Accuracy, Terran | |
| Accuracy, Protoss | |
| Accuracy, Zerg | |

Confusion matrix:

```
(paste bwa eval output here)
```

## Parse success by version

| Version range | Files | Parsed ok | Failed | Most common failure reason |
|---|---|---|---|---|
| | | | | |

## What did not help

Optimizations that were tried and gave no measurable gain. Record them here instead of leaving them out.

- None recorded yet.
