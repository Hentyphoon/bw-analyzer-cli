# Decisions

Why the plan is the way it is. These were settled with the owner before any code was written.

If you think one is wrong, say so at a milestone review and propose the alternative. Don't change it silently. When you make a new decision during the build, append it to the log at the bottom with the milestone it came from.

## Product

- **Brood War, not StarCraft II.** The owner's choice. The project was first planned around SC2 and then switched, so ignore any SC2 concepts (tracker events, supply snapshots, game loops).
- **The headline feature is an opponent scouting report.** The opening classifier is core, not a stretch goal. It gives the API a purpose beyond storing and listing rows.
- **No supply or economy metrics.** A Brood War replay is a command log with no game state. Recovering supply would mean simulating the game. EAPM and redundancy take that place.
- **Build orders are commands issued, lightly de-noised.** The approximation is accepted and documented, not hidden.
- **1v1 human games of at least two minutes only.** Team games, computer opponents, and instant leaves add noise without helping the scouting question.
- **The classifier is rule-based, not learned.** The labeled set is small and the owner must be able to explain every prediction.
- **Proxy detection uses a relative distance (40% of the distance between starts).** A fixed distance would need tuning per map.

## Engineering

- **Use `icza/screp` instead of writing a decoder.** The time goes into the pipeline, analysis, and API. The parser sits behind an interface so it can be replaced or faked in tests.
- **One binary with subcommands, built in the order parse, ingest, serve.** If time runs out after ingest, the project is still a complete CLI.
- **Standard library first: `net/http`, `flag`, `log/slog`, `testing`.** No Gin, Echo, Cobra, or assertion libraries. The resume is Go-focused, and fewer dependencies means fewer things the owner must explain.
- **A small embedded up-only migrator instead of a migration library.** The schema is four tables.
- **The `rows` write mode stays in the binary.** It makes the unoptimized baseline reproducible from the same build, which is more honest than pointing at an old commit.
- **One transaction per replay.** It keeps failure isolation simple. Batching several replays per transaction is allowed only if profiling shows commit cost dominates and the owner agrees.
- **Hash before parse.** Deduplication is only worth having if it saves the parsing work.
- **Failed and skipped replays are stored as rows.** They are not retried on the next run, and parse-success rates can be queried.
- **`opponent_race` is denormalized onto `players`.** Win-rate queries then need no self-join.
- **A player's identity is their name.** Replays carry no account ID. Name collisions are an accepted limitation.
- **Uploads are synchronous behind a semaphore and return 429 when full.** A job queue would add moving parts for no benefit at this scale.
- **Integration tests use `TEST_DATABASE_URL` rather than testcontainers.** One less dependency, and CI already provides a Postgres service.
- **The per-replay timeout covers database work only.** Library parsing cannot be interrupted, and wrapping it in a goroutine to fake a timeout would leak work.
- **No cloud deployment, dashboards, or metrics endpoint.** They were cut to keep the scope finishable.

## Log

Append new decisions here: milestone, decision, reason.

- M1: keep owner replays and their golden files out of git. They are large, they aren't ours to publish, and the golden files carry player names. Tests that need them skip when they are absent.
- M1: track screp's public ShieldBattery ZvT sample (Apache-2.0) so CI has a real replay to test against.
- M1: the golden test runs end to end through `bwa parse --json` in `cmd/bwa`. That covers the adapter and the CLI's JSON output together.
- M1: when a race ID is unknown, map it to `?` and let the 1v1 filter skip the game. Losing a game is better than storing the wrong race.
- M2: APM, EAPM, and redundancy use screp's definitions exactly (divide by the player's last command, not the game length), so tests can assert agreement with the library. Redundancy is stored as a 0 to 1 share, not screp's percent.
- M2: build-order de-noising rules come from the sample data (see `docs/architecture.md`, finding 9). The 10 s repeat window is measured from the previous repeat, so a spam burst collapses fully. A different structure on the same position within 10 s replaces the earlier order. Research repeats within 10 s collapse. Units are never collapsed.
- M2: allowlists are spelled exactly as screp reports names, and a test in `internal/parser` checks every name against screp's tables. That is the only package allowed to import screp, so the check lives there.
- M2: `analyze.Analyze` refuses replays that fail the 1v1 filter, so nothing downstream keys by player ID before the filter has run.
- M2: the worker estimate follows the plan as written and is documented as Zerg-only. Whether to correct it for Terran and Protoss is left to the owner before M5.
- M2 (owner approved): replace the plan's worker estimate for Terran and Protoss with a one-at-a-time town hall queue, 300 frames per SCV or Probe. Counting orders put first Pylons and Depots at 11 to 25 workers; the queue model gives 8 or 9 for every sample player, which matches the names players use. Zerg keeps counting drone orders, since larvae already limit them. Supersedes the previous entry that kept the plan's rule as Zerg-only.
- M2 (owner): assume replays are valid Brood War files. Keep the 8 MB size cap, the library's errors, the parser's `recover()`, and the per-replay failure isolation the plan requires, but add no defensive code for corrupt or adversarial data. A review flagged that an implausible game length in a corrupt header could allocate millions of per-minute buckets; that is accepted under this assumption.
- M2: the size cap lives in `internal/replay` (`MaxFileSize`, `ErrTooLarge`) so `bwa parse` and the M3 pipeline share it. The file is read through a limit of the cap plus one byte, so an oversized file is never loaded in full.
- M2 (owner): CLAUDE.md was written from PLAN.md ahead of the build, so some lines describe later steps and may not match the code yet. Leave them as written. Examples at the end of M2: the golden-file command names `./internal/parser/` while the golden test currently lives in `cmd/bwa`, and `make test` lists no `RACE=` override. The one rule the owner changed is replay files; see the next entry.
- M2 (owner): commit nothing under `testdata/`, including screp's public sample and every golden file. Supersedes the M1 entries that tracked the screp sample. Tests that need a replay skip when it is absent, so CI no longer exercises real replays; they run locally. The files were removed from the index only; earlier commits still contain the screp sample and its golden file.
