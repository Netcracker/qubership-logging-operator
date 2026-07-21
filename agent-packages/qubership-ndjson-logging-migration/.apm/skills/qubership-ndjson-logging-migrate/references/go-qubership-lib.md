# Go / qubership-core-lib-go Playbook

Read when the target uses `go.mod`, qubership-core-lib-go `logging`, logrus, zap, slog, zerolog, or controller-runtime `logr`.

## Infrastructure

- Install NDJSON formatter via `logging.SetLogFormat` or existing repo pattern.
- Read `LOG_FORMAT` from env (default `json` per logging guide).
- Update Helm `LOG_FORMAT` default when the chart owns deployment env.

If the repo lives in a parent `go.work`, run tests and smoke with `GOWORK=off`.

## Prefer structural fields (priority)

Use the highest option the repo already supports:

1. **First-class field API** — logrus `WithFields` / `WithField`, zap/slog attrs, `InfoC`/`ErrorC` with typed fields,
   controller-runtime `logr` key/value pairs mapped in the adapter (not concatenated into `message`).
2. **Repo field helper** — when the logger is message-string-only but the tree has a helper that builds a prose message
   plus extractable fields (e.g. `logfields.Format` / `logfields.Err`), use that helper. Prefer
   `log.Error(helper(...))` or a single `"%s"` wrapper around the helper — not printf of diagnostic values.
3. **Last resort** — hand-built trailing `key=value` in one fully built string (no residual `%v` / `%d` args on the
   log call). Fragile; only when (1) and (2) are unavailable.

Avoid encoding structured data only in the message string when a first-class API or repo helper exists.

### Incomplete — do not claim done

Dropping the `f` while keeping printf args and `key=value` inside the format string is **not** a migration:

```go
// Incomplete (still printf diagnostics in the format string)
log.Error("operation failed key=%v error=%v", key, err)
```

That still fails the residual-verb self-check even when `log.*f` is zero. Zero `log.*f` alone is necessary, not
sufficient.

### Prefer — repo helper (message-string logger)

```go
// Complete when the repo provides Format/Err (or equivalent)
log.Error(logfields.Err("operation failed", err, "resource_key", key))
log.Info(logfields.Format("operation succeeded", "resource_id", id))

// Also OK when the API requires a format string: single %s wrapper around the helper only
log.ErrorC(ctx, "%s", logfields.Err("operation failed", err, "resource_key", key))
```

A custom formatter may regex-parse trailing `key=value` into JSON fields. Safeguards required whenever suffix parsing
is used:

- Quote values containing whitespace
- Never let parsed keys overwrite reserved fields (`time`, `level`, `message`, `class`, `request_id`, …)
- Add unit tests for whitespace, URLs, and reserved-key collision

## logrus pattern

Migrate `log.*f(` to literal message + `WithFields` / `WithField`. Production scope must reach **zero active `log.*f`**
and **zero residual diagnostic format verbs** on non-`f` methods (see [SKILL.md](../SKILL.md) self-check).
Exclude `_test.go`, `dev/`, commented lines.

## logr / controller-runtime

Map key-value pairs to structured fields in the adapter — do not concatenate them into `message`.

## Build gate

`GOWORK=off go build ./...` and relevant `go test` for touched packages before claiming done.

## Smoke

Capture one stdout line and confirm it parses as JSON with `time`, `level`, `message`, and expected diagnostic keys at
top level (not only buried in `message`).
