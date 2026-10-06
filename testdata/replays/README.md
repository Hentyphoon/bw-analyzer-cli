# Sample replays

Owner-supplied 1v1 replays go here (see PLAN.md section 15). Each `.rep` file
gets a golden file in `../golden/`; regenerate with
`go test ./cmd/bwa -run TestParseGolden -update`.

| File | Source |
|---|---|
| `screp_shieldbattery_zvt.rep` | Copied from `github.com/icza/screp` v1.13.4, `repparser/testdata/shieldbattery_raw_trailing_0x78.rep` (Apache-2.0). ShieldBattery ZvT on Eclipse 1.3, 1.21+, 21:41. Its file ends in trailing bytes, which screp tolerates. |
