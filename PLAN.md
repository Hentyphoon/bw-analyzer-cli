# Brood War Replay Analyzer: Build Plan

A Go CLI and HTTP service that ingests StarCraft: Brood War replays, extracts per-player metrics, stores them in PostgreSQL, and answers one question well: **what does this opponent open with against my race, and how often does it win?**

This document is the spec. Build it milestone by milestone (section 14) and stop for review after each one.

---

## 1. Working rules

1. **One milestone at a time.** Finish it, run its acceptance checks, summarize what was built and anything that deviated from this plan, then stop and wait.
2. **Read the library, don't recall it.** Before using any `screp` type, function, or field, open its source in the module cache and confirm the name. Section 6 lists what has already been confirmed and what has not.
3. **Never invent a measurement.** Numbers in `docs/benchmarks.md` come only from runs you executed, with the command and hardware recorded next to them. If an optimization does not help, write that down.
4. **Standard library first.** The only allowed third-party modules are listed in section 3. Ask before adding another.
5. **Stop and ask** if `testdata/replays/` is empty, or if this plan contradicts what the replay data actually contains.
6. **Keep it explainable.** The owner will be interviewed on this code. Prefer plain, idiomatic Go over clever abstractions, and comment the non-obvious concurrency decisions.

## 2. Scope

**In scope**

- `bwa parse`: print one replay's summary and build orders
- `bwa ingest`: concurrent batch ingest of a directory into Postgres
- `bwa eval`: score the opening classifier against hand labels
- `bwa serve`: REST API over the stored data
- `bwa migrate`: apply database migrations
- Reproducible benchmarks comparing ingest configurations

**Out of scope** (do not build): a custom replay decoder, game simulation, supply or economy tracking, web UI, authentication, team games, games against computers, a message queue, Grafana dashboards, cloud deployment.

**Known limitation to state in the README.** A Brood War replay is a log of player commands, not of game state. Build orders here are reconstructed from commands that were issued, so a cancelled or failed order can still appear. Winners are inferred by the library and are sometimes unknown.

## 3. Stack

| Concern | Choice |
|---|---|
| Language | Go, current stable release (minimum 1.22 for `net/http` method routing) |
| Replay decoding | `github.com/icza/screp` |
| Database | PostgreSQL 16+, driver `github.com/jackc/pgx/v5` (`pgxpool`) |
| Concurrency helpers | `golang.org/x/sync/errgroup` |
| HTTP | `net/http` and `http.ServeMux` only, no framework |
| CLI | `flag` with hand-rolled subcommand dispatch, no Cobra |
| Logging | `log/slog` |
| Migrations | Embedded `.sql` files applied in order by a small internal up-only migrator, no migration library |
| Tests | `testing` only, table-driven |
| Lint | `go vet` and `golangci-lint` with default settings |

Configuration: `DATABASE_URL` environment variable, everything else via flags.

## 4. Repository layout

```
bw-analyzer/
  cmd/bwa/main.go           # subcommand dispatch only
  internal/
    replay/                 # domain types, no dependencies on screp or pgx
    parser/                 # Parser interface + screp adapter
    analyze/                # APM, EAPM, build order, classifier (pure functions)
    store/                  # pgx repository layer + migrator
    pipeline/               # worker pool and ingest orchestration
    api/                    # handlers, middleware, routing
  migrations/               # 0001_init.sql, ...
  testdata/replays/         # sample .rep files (supplied by the owner)
  testdata/golden/          # golden JSON for parse output
  docs/architecture.md
  docs/benchmarks.md
  Dockerfile
  docker-compose.yml        # postgres (+ app)
  Makefile                  # build, test, lint, db-up, migrate, bench
  .github/workflows/ci.yml
  README.md
```

Dependency direction: `replay` imports nothing internal. `analyze` imports only `replay`. `parser` and `store` import `replay`. `pipeline` and `api` wire them together.

## 5. CLI

```
bwa parse  <file> [--json]
bwa ingest <dir>  [--workers N] [--write-mode copy|rows] [--timeout 30s]
                  [--force] [--cpuprofile f] [--memprofile f]
bwa eval   --labels labels.csv --dir <dir>
bwa serve  [--addr :8080] [--max-upload-concurrency 4]
bwa migrate
```

- `--workers` defaults to `runtime.NumCPU()`.
- `--write-mode rows` exists so the unoptimized baseline stays reproducible from the same binary. The default is `copy`.
- `--force` re-ingests replays whose hash is already stored (delete, then insert).
- Exit code is non-zero only for fatal errors (bad flags, database unreachable). Individual replay failures are counted, not fatal.

`ingest` ends by printing a summary: files seen, ok, failed, skipped, duplicates, wall time, replays/sec, and cumulative time spent in parse, analyze, and persist.

## 6. Parsing

### Domain model (`internal/replay`)

Define plain structs with no `screp` types leaking out:

- `Replay`: content hash, file name, map name, version range, game type, duration in frames, start time, players, and normalized command slices.
- `Player`: in-game player ID, name, race (`T`/`P`/`Z`), team, start location, result.
- `Action`: frame, player ID, effective (bool). One per counted command.
- `ProductionCmd`: frame, player ID, kind (build, train, building morph, tech, upgrade), name, position (build only).

Time: Brood War counts frames. Use one helper for frames to seconds and use it everywhere.

### Parser interface (`internal/parser`)

```go
type Parser interface {
    Parse(data []byte) (*replay.Replay, error)
}
```

The adapter passes the library a quiet logger (the library logs parse problems to the default `log` logger otherwise), keeps a `recover()` as a second line of defense, and surfaces the count of commands that failed to parse.

### Confirmed against the screp source

- `repparser.ParseConfig(data []byte, cfg repparser.Config)` parses from bytes. `Config` has `Commands`, `MapData`, `MapGraphics`, `Debug`, and `Logger`. The replay ID and header are always parsed.
- Errors: `repparser.ErrNotReplayFile` and `repparser.ErrParsing`. The parser already converts internal panics into `ErrParsing`.
- The README states the package is safe for concurrent use and supports both modern (1.18+) and legacy replays.
- `rep.Replay` has `Header`, `Commands`, `MapData`, `Computed`.
- `Header`: `Version` (a range such as `-1.16`, `1.18-1.20`, `1.21+`), `Frames`, `StartTime`, `Map`, `Type`, `Speed`, `Players` (team order), `Duration()`, `Matchup()`. Its doc comment gives 1 frame = 42 ms.
- `Player`: `SlotID`, `ID` (all computer players share ID 255), `Type`, `Race` (with `Letter`), `Team`, `Name`, `Observer` (computed, not stored).
- `Commands.Cmds` is a `[]repcmd.Cmd`. Each has `BaseCmd()` returning `Frame`, `PlayerID`, `Type`, and `IneffKind`. `Commands.ParseErrCmds` lists commands that failed to parse.
- Concrete command types include `BuildCmd` (`Unit`, `Pos`, `Order`), `TrainCmd` (`Unit`; used for both train and unit morph), `BuildingMorphCmd` (`Unit`), `TechCmd`, `UpgradeCmd`, `CancelTrainCmd`, `HotkeyCmd`, `LeaveGameCmd`.
- `Computed`: `WinnerTeam` (0 means unknown), and `PlayerDescs` / `PIDPlayerDescs` with `APM`, `EAPM`, `CmdCount`, `EffectiveCmdCount`, `LastCmdFrame`, `StartLocation`, `Redundancy()`.

### Not confirmed: verify with real replays in milestone 1

Dump a sample replay and check each of these before relying on it. Record the findings in `docs/architecture.md`.

1. **Computed data.** Whether parsing fills `Computed` automatically or `Compute()` must be called.
2. **Sections needed.** Whether start locations require `MapData` to be parsed. Parse the minimum sections the analysis needs, and never enable `MapGraphics` or `Debug`.
3. **Effectiveness.** How `IneffKind` marks a command as effective, which commands the library counts toward APM, and what duration it divides by.
4. **Coordinate units.** Whether `BuildCmd.Pos` and `StartLocation` use the same units (tiles versus pixels). Convert to one unit in the adapter.
5. **Name strings.** The exact unit, tech, and upgrade names the library reports, for the allowlists and classifier.
6. **Random race.** Whether a player who picked Random is reported with their actual race.
7. **Player types.** How human versus computer is expressed, and how reliable the `Observer` flag is on the samples.
8. **Winner coverage.** What fraction of sample replays have a known `WinnerTeam`.
9. **Repeated build commands.** How spam-clicked or failed building placements appear in the command stream. This drives the de-noising rule in section 7.

### 1v1 filter

Keep a replay only if it has exactly two non-observer players, both human, on different teams, and lasts at least two minutes. Otherwise mark it `skipped` with a reason.

## 7. Analysis (`internal/analyze`)

All functions here are pure: domain types in, results out, no I/O. That makes them testable with hand-built command slices.

**APM and EAPM.** APM counts every player command; EAPM counts only commands the library classifies as effective. Follow the library's definitions so totals can be checked against `Computed`. The curves are action counts and effective-action counts per whole minute. Redundancy is the share of commands that were ineffective. Document the definitions in the README.

**Build order.** An ordered list of `(frame, kind, name)` per player, built from issued commands:

- `structure`: build commands and building morphs (Lair, Hive, Sunken Colony, and so on)
- `unit`: train commands and Zerg unit morphs
- `research`: tech and upgrade commands

De-noising, as a starting point to refine after checking item 9 above: drop ineffective commands, and collapse repeats of the same structure at the same position within 10 seconds into the first one. Filter names with an explicit allowlist per race in `analyze/units.go`. Text output looks like `1:22  Spawning Pool`.

**Worker count estimate.** Four starting workers plus the number of worker train commands issued so far. The classifier uses this for supply-named openings. It is an estimate, not game state.

**Matchup.** Two race letters sorted alphabetically and joined with `v`: `PvP`, `PvT`, `PvZ`, `TvT`, `TvZ`, `ZvZ`.

**Result.** `win` or `loss` from the library's winner team, `unknown` when it is 0.

## 8. Opening classifier (`internal/analyze`)

Rule-based, evaluated top to bottom per race, first match wins. These rules are a starting point: tune thresholds against the labeled set. The owner may change the label set before labeling begins; after that, keep the names fixed. Define all label names in one Go file.

A structure counts as a **proxy** when its distance from the player's own start location is more than 40% of the distance between the two start locations.

| Race | Label | Rule |
|---|---|---|
| Z | `4 Pool` | Spawning Pool ordered with 6 or fewer estimated workers |
| Z | `12 Hatch` | Second Hatchery ordered before the Spawning Pool |
| Z | `Overpool` | Spawning Pool at 9 or fewer estimated workers, after an Overlord |
| Z | `9 Pool` | Spawning Pool at 9 or fewer estimated workers, before any Overlord |
| Z | `12 Pool` | Spawning Pool before the second Hatchery, 10 or more estimated workers |
| T | `BBS` | Two Barracks ordered before the first Supply Depot |
| T | `14 CC` | Second Command Center before the first Barracks |
| T | `2 Rax` | Two Barracks before any Factory or second Command Center |
| T | `1 Rax FE` | Second Command Center before any Factory |
| T | `2 Fact` | Two Factories before the second Command Center |
| T | `1-1-1` | Factory and Starport before the second Command Center |
| T | `1 Fact FE` | One Factory, then the second Command Center |
| P | `Cannon Rush` | Forge before Gateway and a proxy Photon Cannon |
| P | `Proxy Gateway` | A proxy Gateway ordered before 2:30 |
| P | `Forge FE` | Forge before Gateway, second Nexus before Cybernetics Core |
| P | `Nexus First` | Second Nexus before the first Gateway |
| P | `2 Gate` | Two Gateways before a Cybernetics Core |
| P | `1 Gate FE` | Cybernetics Core, then the second Nexus before any tech structure |
| P | `Robo` | First tech structure after Cybernetics Core is a Robotics Facility |
| P | `Citadel` | First tech structure is a Citadel of Adun |
| P | `Stargate` | First tech structure is a Stargate |
| any | `Other` | Nothing matched |

`bwa eval` reads a CSV with columns `file,player,label`, parses each replay, classifies the named player, and prints overall accuracy, per-race accuracy, and a confusion matrix. It needs no database.

## 9. Storage (`internal/store`)

```sql
CREATE TABLE replays (
  id            BIGSERIAL PRIMARY KEY,
  content_hash  TEXT NOT NULL UNIQUE,             -- sha256 hex of file bytes
  file_name     TEXT NOT NULL,
  status        TEXT NOT NULL CHECK (status IN ('ok','failed','skipped')),
  status_reason TEXT,
  map_name      TEXT,
  version       TEXT,                             -- version range reported by the library
  game_type     TEXT,
  frames        INT,
  played_at     TIMESTAMPTZ,
  matchup       TEXT,
  ingested_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE players (
  id            BIGSERIAL PRIMARY KEY,
  replay_id     BIGINT NOT NULL REFERENCES replays(id) ON DELETE CASCADE,
  ingame_id     SMALLINT NOT NULL,
  name          TEXT NOT NULL,
  race          CHAR(1) NOT NULL CHECK (race IN ('T','P','Z')),
  opponent_race CHAR(1) NOT NULL CHECK (opponent_race IN ('T','P','Z')),
  result        TEXT NOT NULL CHECK (result IN ('win','loss','unknown')),
  apm           REAL NOT NULL,
  eapm          REAL NOT NULL,
  redundancy    REAL NOT NULL,
  opening       TEXT NOT NULL,
  UNIQUE (replay_id, ingame_id)
);
CREATE INDEX players_name_idx    ON players (lower(name));
CREATE INDEX players_matchup_idx ON players (race, opponent_race);

CREATE TABLE build_events (
  player_id BIGINT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  seq       INT NOT NULL,
  frame     INT NOT NULL,
  kind      TEXT NOT NULL CHECK (kind IN ('structure','unit','research')),
  name      TEXT NOT NULL,
  PRIMARY KEY (player_id, seq)
);

CREATE TABLE apm_minutes (
  player_id         BIGINT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  minute            INT NOT NULL,
  actions           INT NOT NULL,
  effective_actions INT NOT NULL,
  PRIMARY KEY (player_id, minute)
);
```

`opponent_race` is denormalized onto `players` so win-rate queries need no self-join. Brood War replays carry no account ID, so a player's identity is their name.

Each replay is persisted in a single transaction. The repository exposes one write path with two implementations selected by write mode: `rows` (one `INSERT` per row) and `copy` (`pgx.CopyFrom` for the two event tables). Failed and skipped replays still get a `replays` row so they are not retried.

## 10. Ingest pipeline (`internal/pipeline`)

```
walk dir --> bounded channel (cap 2 x workers) --> N workers --> Postgres
```

Per file, each worker: reads bytes (reject files over 8 MB), computes the sha256, skips if the hash is already stored, parses, applies the 1v1 filter, analyzes, persists.

Requirements:

- **Bounded pool.** One producer goroutine plus N workers under an `errgroup.WithContext`. The producer blocks when the channel is full; that is the backpressure.
- **Failure isolation.** A bad replay (error or recovered panic) is recorded as `failed` with a reason and the worker moves on. Workers return an error only for fatal conditions such as a lost database.
- **Cancellation.** Root context from `signal.NotifyContext` for SIGINT and SIGTERM. On cancel the producer stops enqueuing and workers finish the replay in hand, then exit.
- **Per-replay timeout.** Each replay gets its own deadline for database work, derived with `context.WithoutCancel` so an in-flight write completes during shutdown. Library parsing cannot be preempted; check the context between stages and say so in a comment.
- **Dedup.** Check the hash before parsing so duplicates cost almost nothing. The unique constraint covers the race where two workers get the same file.
- **Pool sizing.** `pgxpool` max connections must be at least the worker count.
- **Stage timing.** Accumulate parse, analyze, and persist durations with atomic counters for the summary.

## 11. HTTP API (`internal/api`)

| Method and path | Purpose |
|---|---|
| `GET /healthz` | Liveness plus a database ping |
| `POST /replays` | Multipart upload of one replay. `201` with the new ID, `200` with the existing ID if the hash is known, `422` if unparseable or not 1v1, `429` if the upload semaphore is full |
| `GET /replays` | List with filters `matchup`, `map`, `player`; keyset pagination via `limit` (default 20, max 100) and `cursor` |
| `GET /replays/{id}` | Replay with both players, APM, EAPM, redundancy, opening, result |
| `GET /replays/{id}/build-orders` | Build order for both players |
| `GET /replays/{id}/timeline` | APM and EAPM per minute for both players |
| `GET /stats/matchups` | Games and win rate for each non-mirror race pairing |
| `GET /stats/openings?matchup=TvZ` | Frequency and win rate of each opening in a matchup |
| `GET /players/{name}/openings?vs=Z` | Scouting report: that player's openings against a race, with games and win rate, most common first |

Conventions:

- JSON everywhere. Errors are `{"error": "message"}` with the right status code.
- Validate every parameter: numeric IDs, `limit` range, `vs` in `T|P|Z`, `matchup` in the six valid values. Invalid input is a `400`.
- List responses are `{"items": [...], "next_cursor": "..."}`; omit `next_cursor` on the last page.
- Player lookup is case-insensitive on name.
- Win rates exclude games with an unknown result. Every win-rate response includes both the total games and the games with a known result.
- Uploads go through the same parse, analyze, persist function as `ingest`, wrapped in `http.MaxBytesReader` (8 MB) and a channel semaphore.
- Middleware: request logging with `slog`, panic recovery.
- Server sets read-header, read, write, and idle timeouts, and on SIGTERM calls `Shutdown` with a 10 second deadline.

## 12. Testing

- **Analysis:** table-driven unit tests on hand-built command slices. Cover edge cases: no commands, a game shorter than one minute, all commands ineffective, repeated build commands, unknown unit names.
- **Classifier:** one table-driven test per rule, including ordering conflicts between rules and the proxy distance boundary.
- **Parser:** golden tests. `bwa parse --json` output for each file in `testdata/replays/` is compared to `testdata/golden/`, with a `-update` flag to regenerate. Add a test that a truncated or garbage file returns an error rather than panicking. Add a test that computed APM and EAPM totals agree with the library's `Computed` values.
- **Store and pipeline:** integration tests that run when `TEST_DATABASE_URL` is set and skip otherwise. Cover: ingest then re-ingest produces zero new rows, both write modes produce identical rows, a corrupt file among good ones is recorded as failed without stopping the batch, cancellation mid-batch exits cleanly.
- **API:** `httptest` against a real test database for happy paths, plus validation and pagination cases.
- Run everything with `-race`.

## 13. Benchmarks (`docs/benchmarks.md`)

Provide `make bench CORPUS=<dir>`, which truncates the tables and times `bwa ingest` for each configuration, then writes a results table.

| Run | Workers | Write mode |
|---|---|---|
| Baseline | 1 | rows |
| + COPY | 1 | copy |
| + workers | 2, 4, 8, NumCPU | copy |

For each run record wall time, replays/sec, and the per-stage time split. Also record: corpus size, CPU model and core count, Go version, Postgres version and whether it ran locally.

Then profile the optimized configuration with `--cpuprofile` and `--memprofile`, identify the top hotspot in code this project owns, fix it if practical, and document before and after. Add `go test -bench` micro-benchmarks for the analysis functions.

Brood War replays are small, so expect the database to dominate rather than parsing. If the profile shows per-transaction commit cost is the bottleneck, propose multi-replay batching to the owner before implementing it. Report results as they are.

## 14. Milestones

**M0. Scaffold.** Module, layout, Makefile, `docker-compose.yml` with Postgres, CI running vet, lint, and `go test -race`.
*Done when:* `make build test lint` passes locally and in CI.

**M1. Parse.** Domain model, screp adapter, `bwa parse`. Resolve every item in "Not confirmed" (section 6).
*Done when:* `bwa parse <file>` prints map, version, duration, both players with race and result for every sample replay; golden tests pass; findings are written to `docs/architecture.md`.

**M2. Analyze.** APM, EAPM, redundancy, build order, matchup.
*Done when:* `bwa parse` also prints APM, EAPM, and each player's first five minutes of build order; unit tests cover the cases in section 12; computed totals agree with the library's.

**M3. Store and ingest.** Migrator, schema, repository with both write modes, pipeline, `bwa migrate`, `bwa ingest`.
*Done when:* ingesting the sample directory twice yields the same row counts; integration tests pass; Ctrl-C mid-run exits cleanly with a summary.

**M4. Benchmarks.** `make bench`, profiling flags, `docs/benchmarks.md`.
*Done when:* the benchmark table is generated from a real run on the owner's corpus and one profiling finding is documented.

**M5. Classifier.** Rules, `opening` populated at ingest, `bwa eval`.
*Done when:* rule tests pass and `bwa eval` prints accuracy and a confusion matrix for the owner's labels file.

**M6. API.** All endpoints in section 11 with validation, pagination, middleware, graceful shutdown.
*Done when:* API tests pass and the scouting endpoint returns correct results against ingested data.

**M7. Package.** Multi-stage Dockerfile producing a small static image, compose file running app plus Postgres, README with quick start, metric definitions, the known limitation from section 2, architecture summary, and benchmark highlights.
*Done when:* `docker compose up` followed by an upload and a query works from a clean checkout.

## 15. Owner's responsibilities

These need a person and are not the agent's job:

- Put 5 to 10 1v1 replays covering all three races, including at least one pre-1.18 and one Remastered replay, in `testdata/replays/` before M1.
- Supply a large corpus (target 10,000 replays) for M4.
- Review the label set in section 8, then hand-label about 100 replays into `labels.csv` before M5.
- Review each milestone before the next begins.
- Deploy, if wanted.
