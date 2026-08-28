# Stage 1 — Go

Config, Helm, and logger bootstrap only.

## logrus / qubership-core-lib-go

- `logging.SetLogFormat` or existing repo JSON formatter path.
- `LOG_FORMAT=json` (and legacy `text` / `cloud` mapping for dual rollout).
- Helm env for deployable binaries.
- When a parent `go.work` is present, prefix repo-documented Go commands with `GOWORK=off` so build/smoke target this
  module only.

## zap / slog / zerolog

- Confirm JSON handler already active or switch handler in bootstrap.
- Normalize field keys to `time`, `level`, `message` in handler config if the pipeline requires logging-guide names.

## Minimal code

- Wire missing format flag reading from env.
- Fix startup if JSON mode panics.
