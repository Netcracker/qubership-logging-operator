# Go / qubership-core-lib-go Playbook

Read when the target uses `go.mod`, qubership-core-lib-go `logging`, logrus, zap, slog, zerolog, or controller-runtime `logr`.

## Infrastructure

- Install NDJSON formatter via `logging.SetLogFormat` or existing repo pattern.
- Read `LOG_FORMAT` from env (default `json` per logging guide).
- Update Helm `LOG_FORMAT` default when the chart owns deployment env.

If the repo lives in a parent `go.work`, run tests and smoke with `GOWORK=off`.

## Prefer structural fields

Use `WithFields`, `InfoC`/`ErrorC` context APIs, or zap/slog field APIs — attach typed fields directly to the record.

Avoid encoding structured data only in the message string when a first-class API exists.

## logrus pattern

Migrate `log.*f(` to literal message + `WithFields` / `WithField`. Production scope must reach **zero active `log.*f`** (exclude `_test.go`, `dev/`, commented lines).

## core-lib-go + message suffix

When the logger accepts only a message string:

```go
log.InfoC(ctx, "%s", logfields.Format("database provisioned",
    "microserviceName", name, "trackingId", id))
```

A custom formatter regex-parses trailing ` key=value` into JSON fields. Safeguards required:

- Quote values containing whitespace
- Never let parsed keys overwrite reserved fields (`time`, `level`, `message`, `class`, `request_id`, …)
- Add unit tests for whitespace, URLs, and reserved-key collision

## logr / controller-runtime

Map key-value pairs to structured fields in the adapter — do not concatenate them into `message`.

## Build gate

`GOWORK=off go build ./...` and relevant `go test` for touched packages before claiming done.

## Smoke

Capture one stdout line and confirm it parses as JSON with `time`, `level`, `message`.
