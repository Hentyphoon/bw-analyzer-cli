# Architecture

## Packages

| Package | Role | May import |
|---|---|---|
| `internal/replay` | Domain types | nothing internal |
| `internal/analyze` | Pure analysis functions | `replay` |
| `internal/parser` | `Parser` interface and screp adapter | `replay` |
| `internal/store` | pgx repository and migrator | `replay` |
| `internal/pipeline` | Concurrent ingest | all of the above |
| `internal/api` | HTTP handlers and middleware | all of the above |
| `cmd/bwa` | Subcommand dispatch | all of the above |

## Replay data findings

Checked against screp v1.13.4. Each finding says where it comes from:
**source** means read in the library code, and **sample** means observed in a
replay. So far the only sample is `testdata/replays/screp_shieldbattery_zvt.rep`,
a 1.21+ ShieldBattery ZvT that ships with screp's own tests. Findings marked
**open** need the owner's sample replays before they can be closed.

| # | Question | Finding | Basis |
|---|---|---|---|
| 1 | Is `Computed` filled automatically? | No. `ParseConfig` never calls `Compute()`; the adapter calls it. `Compute` also sets every command's `IneffKind`, assigns teams from alliance commands, flags observers, and infers the winner. | source |
| 2 | Sections needed | `Commands` (APM, build orders, teams, winners) and `MapData` (start locations: `Compute` only sets `StartLocation` when `MapData` was parsed). `MapGraphics` and `Debug` stay off. | source |
| 3 | Effectiveness and APM | A command is effective when `IneffKind == IneffKindEffective` (0). The library counts every command whose player ID belongs to a header player, so observer chat (ID 128 and up) is excluded. APM = commands ÷ minutes up to that player's **last command frame**, not the game length, rounded to an integer. The adapter keeps the same set of commands; a test checks that the per-player totals equal `CmdCount` and `EffectiveCmdCount`. | source, sample |
| 4 | Coordinate units | Different. `BuildCmd.Pos` is in **tiles** (sample: x from 0 to 127 on a 128-wide map). `StartLocation` is in **pixels** (sample: 3776 = 118 × 32). The adapter multiplies build positions by 32, so `replay.Point` is always pixels. Which tile of the footprint the build position refers to is unverified; at the 40% proxy threshold the difference does not matter. | source, sample |
| 5 | Name strings | Names come from `repcmd` tables (`Units`, `Techs`, `Upgrades`). Sample examples: `Spawning Pool`, `Supply Depot`, `Lair`, `Drone`, `SCV`, `Metabolic Boost (Zergling Speed)`, `Terran Infantry Weapons`. Some units carry a suffix, e.g. `Siege Tank (Tank Mode)`. The full allowlists are built in M2 from those tables. | source, sample |
| 6 | Random race | **Open.** `RaceByID` knows only IDs 0 (Zerg), 1 (Terran), 2 (Protoss); anything else becomes letter `U`, which the adapter maps to `?`, and the 1v1 filter skips the game as "unknown race". Whether a Random player's header holds their resolved race needs a sample. | source |
| 7 | Human vs computer, observers | Human means `Player.Type` is `PlayerTypeHuman`; all computers share ID 255. Observers are detected **only for Melee and UMS game types**: a human with APM < 25 and fewer than 5 build commands. Other types are never checked, including `Top vs Bottom`, which this ShieldBattery 1v1 reports. **Open:** an observer in such a game would count as a third player, so the filter would skip the game. Samples with observers are needed to decide whether to add our own fallback. | source, sample |
| 8 | Winner coverage | Inferred as "largest remaining team wins" from Leave Game commands, plus a virtual leave for the replay saver, who is identified from chat. Sample: 1 of 1 known (team 2). **Open:** coverage across a real set. | source, sample |
| 9 | Repeated build commands | Repeats are only partly marked ineffective. In the sample, a Supply Depot at the same tile was ordered at frames 3256, 3262, 3361, and 3516; only 3262 is marked ineffective (repetition). Separately, an Evolution Chamber and then a Spawning Pool were ordered on the same tile 46 frames apart, a changed mind that also shows up. Train spam beyond the queue is marked `unit queue overflow`. So dropping ineffective commands is not enough, and M2 also needs the same-structure, same-position collapse from section 7. | sample |

Other notes:

- **Train vs. unit morph.** Both decode to `*repcmd.TrainCmd`. The command type (`Train` vs. `Unit Morph`) distinguishes them, so the domain keeps separate kinds.
- **Parse errors.** Commands that fail to parse are kept out of `Cmds` and listed in `ParseErrCmds`. Each one may hide further commands in the same block. The count is surfaced as `Replay.ParseErrors`.
- **Map names.** The header copy is truncated to 26 bytes, so the adapter prefers `MapData.Name` and strips control characters, which map makers use for colors.
- **Start time.** Stored as seconds since the epoch, converted to UTC.
- **Game type is not a 1v1 signal.** ShieldBattery records 1v1s as `Top vs Bottom`, so the filter looks only at players, never at game type.
