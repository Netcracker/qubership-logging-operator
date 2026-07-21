# Logging Schema (stage 2 — full contract)

**Stage 1** (`qubership-ndjson-logging-enable`) only requires the JSON envelope. This file is the **stage 2** target.

Use the Qubership logging guide (*log-formats.md*) as the source of truth. This file is a short working
summary, not a replacement for the logging guide.

## NDJSON baseline

Each log event should be one JSON object on one stdout line.

Recommended baseline fields:

- `time` — ISO-8601 timestamp in UTC.
- `level` — `ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`; preserve `FATAL` / `OFF` if the framework emits them.
- `message` — human-readable text only.

Common Qubership/correlation fields:

- `request_id`
- `tenant_id`
- `thread`
- `class`
- `method`
- `version`
- `error_code`
- `originating_bi_id`
- `business_identifiers`
- `traceId`
- `spanId`
- `logType` (`audit` for audit logs)

## Core migration rule

Move structured data from message text into JSON context fields.

Before:

```text
[INFO] [request_id=abc] user_id=42 action=login Login succeeded
```

After:

```json
{"time":"2026-01-01T00:00:00.000Z","level":"INFO","request_id":"abc","user_id":"42","action":"login","message":"Login succeeded"}
```

## FluentBit pipeline assumptions

Use the Qubership FluentBit pipeline guide (*fluentbit-log-pipeline.md*) as the source of truth.

- Default parser is JSON when no pod annotation overrides it.
- Prefer flat JSON fields.
- Keep parser annotations consistent with emitted format; do not emit JSON while forcing a text parser.
