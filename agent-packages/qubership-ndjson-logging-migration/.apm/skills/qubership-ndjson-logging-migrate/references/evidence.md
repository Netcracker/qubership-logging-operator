# Pattern Inventory

Use this file as background when choosing an implementation pattern. It does not rank target repositories; the repo
being edited is always the source of truth.

## Known Patterns

### Java Logback / Spring legacy text baseline

- Logging guide requires `LOG_FORMAT=text|json`, default `json`, schema per *log-formats.md*.
- Typical `logback-spring.xml` uses a dense legacy bracket pattern with MDC-style fields such as request ID,
  trace/span IDs, chain/session fields, and log type.
- Agent/profiler-style services often use conditional Logback configuration with the same bracket text pattern.

Migration note: preserve MDC fields and map them to top-level JSON fields. Re-check the current target before choosing a
JSON encoder.

### Go zap

- Kubernetes operators and controllers commonly use zap JSON with `time`, `level`, and `message`.

### Go slog

- Newer services use `slog.NewJSONHandler` with customized timestamp/source fields.

### Go dual-format switch

- Some binaries support `cloud` (legacy brackets), `json`, and `text` format names — not always exactly
  `LOG_FORMAT=text|json`, but useful evidence for safe format switching during rollout.

### Go zerolog

- Bootstrap/sync utilities may use zerolog default JSON or a custom `ConsoleWriter` that emits bracket text.
- Keep field names aligned with `time`, `level`, and `message` unless the target pipeline already accepts framework
  defaults.

### Nginx text/parser baseline

- Logging pipeline docs list Nginx as a third-party parser case.
- Existing deployments often use classic text `log_format` until migrated.

## Shape Examples

### Python flat JSON

- POC/demo manifests sometimes show flat JSON log records on stdout.
- Treat as output-shape reference only; prefer the actual Python logging setup in the target repo.

## Frameworks To Cover

- Python production logging implementation.
- Java Log4j/log4j2 JSON implementation.
- Pure Java / `java.util.logging` implementation.
- Quarkus JSON logging implementation.
- Nginx JSON access logs.
- Envoy JSON access logs.

When touching one of these stacks, document target-specific constraints in the migration report. If the lesson should
improve this reusable skill, propose a source-package update rather than editing deployed `.agents/skills` copies.
