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

## Analysis (`internal/analyze`)

Every function is pure: it takes `replay` types and returns results, with
no I/O. `Analyze` runs everything for one replay. It refuses replays that
fail the 1v1 filter, because computer players share player ID 255 and
keying by ID is only safe for two humans.

### APM, EAPM, and redundancy

The definitions are screp's, so the totals can be checked against it.
`internal/parser/analyze_test.go` asserts that, for every sample replay,
our APM and EAPM round to screp's integers and our redundancy matches.

- **Counted commands.** Every command the library attributes to a player
  listed in the header. Observer chat uses IDs 128 and up and is excluded.
- **Effective.** `IneffKind == IneffKindEffective`. screp's heuristics flag
  unit-queue overflow, too-fast cancels, too-fast repetition, too-fast
  reselection, repetition, and repeated hotkey assignment.
- **Divisor.** Minutes up to the player's own last command, not the game
  length. Commands with a frame outside the game are counted but ignored
  for the last-command frame and the curve, as screp does.
- **Curve.** One bucket per whole game minute from minute 0 to the end of
  the game, including empty minutes, so both players' curves line up.

Trade-offs. APM rewards spam, which is why EAPM exists; EAPM in turn depends
on heuristics that do not know game state, so a legitimate burst of clicks
can be marked ineffective and real waste can pass. Dividing by the last
command keeps early leavers comparable but makes APM of a player who
stopped acting (idle while losing) look higher than their game average.
Across the 20 players in the 10 1v1 samples, APM ranges from 168 to 407
and redundancy from 0.05 to 0.46.

### Build orders

A build order is a list of commands the player issued, not of things that
finished. Steps come from production commands in order:

1. Drop commands screp marks ineffective.
2. Drop names not on the race's allowlist (`analyze/units.go`). The names
   are spelled exactly as screp reports them, and a test checks each one
   against screp's tables so a typo cannot silently drop a unit.
3. Collapse a structure ordered again on the same position within 10
   seconds into the first order. The gap is measured from the previous
   repeat, so a long burst of clicks collapses fully.
4. When a different structure is ordered on the same position within 10
   seconds, keep only the later order: the worker was re-sent.
5. Collapse the same research ordered again within 10 seconds.

Units are never collapsed. Training several of the same unit is normal, and
screp already marks orders into a full queue as ineffective.

Why these rules, from the samples (see finding 9 below): repeats of the
same structure on one tile cluster within 10 seconds of each other, while
later repeats are minutes apart and are mostly Hatcheries, Nexuses, Cannons,
and Creep Colonies, which look like rebuilds. One Evolution Chamber was
replaced by a Spawning Pool on the same tile 1.9 seconds later, which would
mislead the opening classifier. Research is spammed heavily: 127 effective
repeats of the same research within 10 seconds, such as Psionic Storm
clicked 0.2 seconds apart. Across the 10 1v1 samples, the 9,698 production
commands of the 20 players reduce to 8,309 steps (85.7% kept).

### Worker estimate

`WorkersBefore` estimates a player's worker count just before a frame: the
four starting workers plus every worker whose production started before
that frame. It ignores workers that died and Zerg drones that became
structures, which matches how players name openings ("9 Pool" is a Spawning
Pool ordered at 9 workers).

When production starts depends on the race:

- **Zerg.** A Drone starts when it is ordered. Each one uses up a larva, so
  drone orders already come at the pace they can be made.
- **Terran and Protoss.** A Command Center or Nexus builds one worker at a
  time, and an SCV or Probe takes 300 frames (about 12.6 s at Fastest).
  Players queue several at once, and orders they cannot afford are still
  recorded, so counting orders overcounts badly. Instead each order starts
  when it arrives or when the previous worker finishes, whichever is later.

The plan's original rule (four plus every worker ordered) was tried first.
Results for the 10 Terran and Protoss players (7 Protoss, 3 Terran) in the
10 1v1 samples, at each player's first structures. The third Terran opened
Command Center first, so only two Barracks openings appear:

| First structure | Count of orders | Queue model | Name players use |
|---|---|---|---|
| Pylon (7 players) | 11 to 25 | 8 | 8 Pylon |
| Supply Depot (3 players) | 16 to 21 | 8 or 9 | 9 Depot |
| Gateway (2 players) | 15 to 27 | 10 | 10 Gate |
| Barracks (2 players) | 25 to 30 | 11 | 11 Rax |
| Forge (4 players) | 19 to 33 | 11 or 12 | 11/12 Forge |

A test in `internal/parser` pins the screp sample's Terran at 9 workers
for the first Supply Depot and 11 for the first Barracks.

Zerg, checked the same way on the 10 Zerg players in the samples: the first
Overlord comes at 9 workers for 9 of them ("9 Overlord"), and a Spawning
Pool ordered after it at 9 for all four who did that ("Overpool"). A
Hatchery ordered before the Pool comes at 11 for all five who did that,
where players call the opening "12 Hatch". No drone is lost: in the two of
those games with no ineffective drone orders the count is still 11, and the
12th drone is ordered right after the Hatchery. The difference is timing.
A build command is recorded when the drone is sent, and the walk to the
natural expansion takes several seconds, while players count at placement.
A Pool in the main has a short walk, so its count matches. The drone orders
screp marks ineffective are same-second double taps that an early economy
cannot pay for. This does not affect the classifier: "12 Hatch" in
`PLAN.md` section 8 is defined by order (Hatchery before Pool), not by a
count.

Limits. The 300-frame build time comes from Brood War's unit data, not from
screp, which carries no build times. The model assumes a single town hall:
once an expansion finishes, two queues run in parallel and the estimate runs
low. Openings are decided before that. An order the player could not afford
still enters the modeled queue; the exact opening counts above suggest this
rarely matters early on.

## Replay data findings

Checked against screp v1.13.4. Each finding says where it comes from:
**source** means read in the library code, and **sample** means observed in a
replay. Findings marked **open** could not be settled with the samples on hand.

**Sample set (11 replays).** screp's public ShieldBattery ZvT sample plus
10 owner replays from TL.net. All are local only; nothing under `testdata/`
is committed. Versions: 5 pre-1.18 (`-1.16`), 1 from 1.18 to 1.20,
5 from 1.21 onward. Game types: 5 Use Map Settings, 4 Top vs Bottom
(ShieldBattery), 2 One on One. All three races appear. 10 are 1v1 games;
two of those have one observer each. 1 is a 2v2 game, which the filter
skips. Lengths range from 10 to 61 minutes.

| # | Question | Finding | Basis |
|---|---|---|---|
| 1 | Is `Computed` filled automatically? | No. `ParseConfig` never calls `Compute()`; the adapter calls it. `Compute` also sets every command's `IneffKind`, assigns teams from alliance commands, flags observers, and infers the winner. | source |
| 2 | Sections needed | `Commands` (APM, build orders, teams, winners) and `MapData` (start locations: `Compute` only sets `StartLocation` when `MapData` was parsed). `MapGraphics` and `Debug` stay off. | source |
| 3 | Effectiveness and APM | A command is effective when `IneffKind == IneffKindEffective` (0). The library counts every command whose player ID belongs to a header player, so observer chat (ID 128 and up) is excluded. APM = commands ÷ minutes up to that player's **last command frame**, not the game length, rounded to an integer. The adapter keeps the same set of commands; a test checks that the per-player totals equal `CmdCount` and `EffectiveCmdCount`. | source, sample |
| 4 | Coordinate units | Different. `BuildCmd.Pos` is in **tiles** (sample: x from 0 to 127 on a 128-wide map). `StartLocation` is in **pixels** (sample: 3776 = 118 × 32). The adapter multiplies build positions by 32, so `replay.Point` is always pixels. Which tile of the footprint the build position refers to is unverified; at the 40% proxy threshold the difference does not matter. | source, sample |
| 5 | Name strings | Names come from the `repcmd` tables (`Units`, `Techs`, `Upgrades`). The samples contain 133 distinct production names, including 42 build targets and 5 building morphs (Lair, Hive, Greater Spire, Sunken Colony, Spore Colony). Watch for: `Queens Nest` has no apostrophe; Terran add-ons (`ComSat`, `Machine Shop`, `Control Tower`, and so on) arrive as build commands; some units carry a suffix, e.g. `Siege Tank (Tank Mode)`; the Firebat is `Firebat` (`Gui Motang (Firebat)` is the campaign hero). The M2 allowlists in `analyze/units.go` use these exact strings, checked against screp's tables by a test. |  source, sample |
| 6 | Random race | **Resolved by owner decision: record the race played.** The header's race byte is read by screp's `RaceByID`, which knows 0 (Zerg), 1 (Terran), 2 (Protoss). All 32 player slots in the samples hold one of those, but no sample is known to include a Random pick, so whether the header stores the pick (6) or the race played is still unobserved. Either way the adapter records the race played: if the header race is anything else, it takes the race of the player's first structure order (`repcmd.RaceOfUnitID`, which covers structures only). A test rewrites the sample Terran's header race to 6 and checks it is still recorded as Terran. | source, sample |
| 7 | Human vs computer, observers | Human means `Player.Type` is `PlayerTypeHuman`; all computers share ID 255. screp detects observers **only for Melee and UMS game types**: a human with APM < 25 and fewer than 5 build commands. Other types are never checked, including `One on One` and the `Top vs Bottom` that ShieldBattery 1v1s report. Samples: all 32 slots are human; both observers (in UMS games) were flagged correctly and sit on their own team 3. **Closed by owner decision:** replays are assumed to be 1v1 games, so an observer in a One on One or Top vs Bottom game is not handled. If one appears, it counts as a third player and the filter skips the game. | source, sample |
| 8 | Winner coverage | Inferred as "largest remaining team wins" from Leave Game commands, plus a virtual leave for the replay saver, who is identified from chat. Samples: 11 of 11 known, across all three version ranges and all three game types. The M4 corpus will give a real rate. | source, sample |
| 9 | Repeated build commands | Repeats are only partly marked ineffective. In the sample, a Supply Depot at the same tile was ordered at frames 3256, 3262, 3361, and 3516; only 3262 is marked ineffective (repetition). Separately, an Evolution Chamber and then a Spawning Pool were ordered on the same tile 46 frames apart, a changed mind that also shows up. Train spam beyond the queue is marked `unit queue overflow`. Across all samples: 1567 build commands, 77 marked ineffective. Among the effective ones, the same player ordered the same structure on the same tile again 50 times within 10 seconds and 77 times after more than 10 seconds; a different structure followed on the same tile 8 times. So dropping ineffective commands is not enough, and M2 needs the same-structure, same-position collapse from section 7. Gaps from the previous same-structure, same-tile order: 23 within 2 s, 15 at 2 to 5 s, 12 at 5 to 10 s, 10 at 10 to 20 s, 17 at 20 to 60 s, 28 at 1 to 5 min, 22 over 5 min. Repeats past 10 s are mostly Hatchery (19), Photon Cannon (18), Creep Colony (13), Nexus (8), and Extractor (6), which look like rebuilds, so M2 kept the 10 s window. Of the 8 different-structure cases, only one (Evolution Chamber, then Spawning Pool 1.9 s later) is within 10 s; the rest are minutes apart. | sample |

Other notes:

- **Train vs. unit morph.** Both decode to `*repcmd.TrainCmd`. The command type (`Train` vs. `Unit Morph`) distinguishes them, so the domain keeps separate kinds.
- **Parse errors.** Commands that fail to parse are kept out of `Cmds` and listed in `ParseErrCmds`. Each one may hide further commands in the same block. The count is surfaced as `Replay.ParseErrors`.
- **Map names.** The header copy is truncated to 26 bytes, so the adapter prefers `MapData.Name` and strips control characters, which map makers use for colors.
- **Start time.** Stored as seconds since the epoch, converted to UTC.
- **Game type is not a 1v1 signal.** ShieldBattery records 1v1s as `Top vs Bottom`, so the filter looks only at players, never at game type.
