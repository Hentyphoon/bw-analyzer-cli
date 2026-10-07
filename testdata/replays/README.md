# Sample replays

Owner-supplied 1v1 replays go here (see PLAN.md section 15). They are
git-ignored, as are their golden files in `../golden/`, so they stay on the
owner's machine. Tests that need them skip when they are absent, so CI runs
on the tracked sample alone. Regenerate golden files with
`go test ./cmd/bwa -run TestParseGolden -update`.

| File | Source |
|---|---|
| Owner replays (git-ignored) | Downloaded from the TL.net replay database: <https://tl.net/replay/> |
| `screp_shieldbattery_zvt.rep` | Tracked. Copied from `github.com/icza/screp` v1.13.4, `repparser/testdata/shieldbattery_raw_trailing_0x78.rep` (Apache-2.0). ShieldBattery ZvT on Eclipse 1.3, 1.21+, 21:41. Its file ends in trailing bytes, which screp tolerates. |
