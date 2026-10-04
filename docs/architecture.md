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

To be filled in during milestone M1 from real sample replays.
