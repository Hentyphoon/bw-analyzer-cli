# Brood War Replay Analyzer

`bwa` is a Go CLI and HTTP service that ingests StarCraft: Brood War replays,
extracts per-player metrics, stores them in PostgreSQL, and answers one
question well: **what does this opponent open with against my race, and how
often does it win?**

Status: milestone M2 (analyze) done. `bwa parse` prints a replay's summary,
APM, EAPM, redundancy, and build orders. The other commands are stubs until
their milestones land.

## Usage

```sh
bwa parse <file> [--json] [--minutes N]
```

`--minutes` limits the build order to the first N game minutes (default 5,
0 for the whole game). Example, trimmed:

```
Map:       Eclipse 1.3
Version:   1.21+ (Top vs Bottom)
Duration:  21:41
Matchup:   TvZ

Team  Race  Result    APM  EAPM  Redundancy  Player
   1  Z     loss      322   257         20%  [z]home
   2  T     win       322   224         31%  IlIIlIlIlIIllII

Build order: [z]home (Z), first 5 minutes
   0:01  Drone
   0:52  Overlord
   1:38  Hatchery
   1:55  Spawning Pool
   2:01  Extractor
   ...
```

Only 1v1 games between two humans lasting at least two minutes are
analyzed. A player who picked Random is listed under the race they played. Anything else is reported with the reason it was skipped. Files
over 8 MB are refused; real replays are a few hundred kilobytes.

## Metrics

Brood War counts time in frames; one frame is 42 ms at Fastest speed.

- **APM**: every command the player issued, divided by the minutes up to
  their last command. This is the definition used by the replay library
  (`icza/screp`), and the tests check that the numbers round to its values.
  Dividing by the player's own last command, not the game length, keeps a
  player who leaves early from being diluted.
- **EAPM**: the same, counting only commands the library classifies as
  effective. A command is ineffective when it is, for example, a too-fast
  repetition, a selection changed before it was used, or a unit ordered into
  a full production queue.
- **Redundancy**: the share of a player's commands that were ineffective,
  from 0 to 1.
- **APM curve**: action and effective-action counts for each whole game
  minute.
- **Build order**: the structures, units, and research a player ordered,
  in order. Kinds are `structure` (build commands and building morphs such
  as Lair), `unit` (training and Zerg unit morphs), and `research` (tech and
  upgrades). Commands are de-noised: ineffective ones and names outside the
  race's allowlist are dropped, and repeats of the same structure on the
  same spot, or the same research, within 10 seconds collapse into one.
- **Matchup**: the two race letters sorted and joined with `v`, so both
  players of a game share it: `PvT`, never `TvP`.

## Known limitation

A Brood War replay is a log of player commands, not of game state. Build
orders are reconstructed from the commands that were issued, so an order
that was cancelled, could not be afforded, or failed to place can still
appear. Winners are inferred by the library from who left the game, and are
sometimes unknown.

## Development

Requires Go 1.25+ (the minimum set by screp), Docker (for Postgres), and
`golangci-lint` v2.

```sh
make build        # bin/bwa
make test         # go test -race ./...  (use RACE= without a C compiler)
make lint         # go vet + golangci-lint
make db-up        # start Postgres 16 via docker compose
```

Configuration: `DATABASE_URL` (defaults to the compose database in the
Makefile); everything else is a flag.

### Sample replays

Nothing under `testdata/` is committed. Tests that read a replay skip
without one, so set up the samples locally to run them:

- `testdata/replays/screp_shieldbattery_zvt.rep`: copy
  `repparser/testdata/shieldbattery_raw_trailing_0x78.rep` from the screp
  module (`go env GOMODCACHE`, then `github.com/icza/screp@v1.13.4`). It is
  screp's public test replay (Apache-2.0), a ShieldBattery ZvT.
- Any other 1v1 replays, for example from the TL.net replay database:
  <https://tl.net/replay/>.

Then generate golden files with
`go test ./cmd/bwa -run TestParseGolden -update`.

Design notes and findings about the replay data are in
[docs/architecture.md](docs/architecture.md).
