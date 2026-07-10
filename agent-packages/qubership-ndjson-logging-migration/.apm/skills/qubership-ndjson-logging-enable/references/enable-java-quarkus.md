# Stage 1 — Java / Quarkus / Logback

Config and Helm only.

## Quarkus

- Add or confirm `quarkus-logging-json` dependency.
- `quarkus.log.console.json=true` (and text profile / `%text.` overrides for legacy bracket mode when `LOG_FORMAT=text`).
- Wire `LOG_FORMAT` / `QUARKUS_PROFILE` in Helm.
- Promote correlation via `quarkus.log.console.json.additional-field.*` (e.g. `%X{requestId}` → `request_id`) — config only.
- Map encoder keys to logging-guide field names if needed (`date-format`, excluded keys).

## Logback / Spring

- Enable JSON encoder (Logstash or native JSON layout) in `logback-spring.xml` / profile XML.
- Conditional `%text` vs JSON profile tied to `LOG_FORMAT`.
- Preserve existing MDC keys in JSON output via encoder config.

## Minimal code (only when required)

- Fix broken format-switch bootstrap if the app cannot start with `LOG_FORMAT=json`.
