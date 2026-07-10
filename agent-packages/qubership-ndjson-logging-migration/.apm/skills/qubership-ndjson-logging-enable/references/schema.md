# Stage 1 schema (JSON envelope)

Use the Qubership logging guide (*log-formats.md*) as the source of truth. **Stage 1** satisfies the **pipeline** contract only.

## NDJSON envelope (required)

Each log event: one JSON object on one stdout line.

Minimum fields (names may be mapped by encoder — verify smoke output):

- `time` — ISO-8601 UTC (or encoder equivalent mapped in config).
- `level` — `ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`.
- `message` — human-readable text; **may still contain** legacy bracket/`key=value`/`{}` residue in stage 1.

## Correlation (config-level)

Promote via encoder/MDC/additional-field when already available:

- `request_id`, `tenant_id`, `traceId`, `spanId`, `logType` (audit)

## Access logs (Nginx / Envoy)

Proxy access logs are **one JSON object per line** but use the **access-log field set** from the logging guide — not the app envelope (`time`, `level`, `message`).

Typical fields (names per repo / guide):

- `time` or `start_time` — request timestamp
- `method`, `path` (or `uri`), `status` (or `response_code`)
- `request_time`, `duration`, `body_bytes_sent`, `remote_addr`, `upstream_cluster` as applicable

Stage 1: switch config to JSON access format and confirm ingestion.

See [enable-nginx-envoy.md](enable-nginx-envoy.md).

## FluentBit

- Pod emits JSON → default JSON parser applies.
- Do not emit JSON while chart annotations force a text parser.

## Example (stage 1 acceptable)

Legacy text inside `message` is OK:

```json
{"time":"2026-01-01T00:00:00.000Z","level":"INFO","request_id":"abc","message":"[request_id=abc] user_id=42 action=login Login succeeded"}
```
