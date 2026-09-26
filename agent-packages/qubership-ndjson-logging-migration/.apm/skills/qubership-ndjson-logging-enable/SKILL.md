---
name: qubership-ndjson-logging-enable
description: >
  Stage 1 — Enable NDJSON log output for Qubership services without migrating every log call site. Wire LOG_FORMAT,
  logger/encoder config, and Helm so stdout emits one JSON object per line (time, level, message). Use when rolling out
  JSON logging, default json per logging guide, FluentBit JSON parser alignment, Nginx/Envoy JSON access logs, or "switch to structured logs" before field
  extraction. Do NOT use for mass {} / log.*f call-site migration — use qubership-ndjson-logging-migrate instead.
---

# NDJSON Enable (Stage 1)

**Goal:** config-level rollout so stdout emits a valid JSON envelope (`time`, `level`, `message`) per component. Change
logger/encoder, `LOG_FORMAT`, and Helm only; leave existing log call sites and `message` text as-is. Minimal bootstrap
code only when JSON mode fails compile or process startup.

**Envelope ≠ event-field placement:** stage 1 does **not** prove that SLF4J `addKeyValue`, Go field helpers, or similar
appear as top-level JSON keys. That is stage 2’s [placement probe](../qubership-ndjson-logging-migrate/references/placement-probe.md)
in `qubership-ndjson-logging-migrate`.

**Discover before changing:** read the target repo’s existing logger dependencies, config files, and Helm env — extend that
stack; do not copy another service’s pattern. Output shape follows the logging guide (*log-formats.md*); ambiguities go
to the user. For **build and smoke**, follow the repo’s own README / `dev/` / `bootstrap/` / workflow docs
([enable-gates.md](references/enable-gates.md)).

## Reference map

| When                         | Read                                                                                                                                                               |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Envelope schema              | [schema.md](references/schema.md)                                                                                                                                  |
| Stack → config only          | [enable-java-quarkus.md](references/enable-java-quarkus.md), [enable-go.md](references/enable-go.md), or [enable-nginx-envoy.md](references/enable-nginx-envoy.md) |
| Before claiming stage 1 done | [enable-gates.md](references/enable-gates.md)                                                                                                                      |
| Maven 401 / GitHub Packages  | [maven-github-packages.md](references/maven-github-packages.md)                                                                                                    |
| Smoke                        | [smoke-validation.md](references/smoke-validation.md)                                                                                                              |
| Report                       | [migration-report-template.md](references/migration-report-template.md)                                                                                            |
| Pitfalls                     | [corner-cases.md](references/corner-cases.md)                                                                                                                      |

## Required outcome

- One JSON object per log event on stdout (`time`, `level`, `message` or stack-mapped equivalents).
- `LOG_FORMAT=text|json` where dual rollout is needed; default `json` per logging guide.
- Correlation fields from **config** (MDC, additional-field, encoder) where the stack already provides them — not from
  rewriting call sites.
- Chart/env defaults updated for deployable components.

## Workflow

1. Read [schema.md](references/schema.md) — stage 1 contract only.
2. **Repo-root discovery** — list every runtime component (`go.mod`, `pom.xml`, nginx/envoy chart config, charts)
   before editing.
3. **Classify stack** per component → read [enable-java-quarkus.md](references/enable-java-quarkus.md),
   [enable-go.md](references/enable-go.md), or [enable-nginx-envoy.md](references/enable-nginx-envoy.md).
4. **Config-only changes** — encoder, `quarkus-logging-json`, logrus/zap JSON handler, Python JSON handler, `LOG_FORMAT`
   env, Helm values. Code edits only for **bootstrap blockers** (e.g. missing dependency, `LOG_FORMAT` not read, panic in
   `SetLogFormat`) — not call-site logging refactors.
5. **Run gates** — [enable-gates.md](references/enable-gates.md); build and smoke per repo docs.
6. **Smoke** — [smoke-validation.md](references/smoke-validation.md); use repo-documented local run/deploy or CI; one
   captured line parses as JSON (app envelope or access-log fields per [schema.md](references/schema.md)).
7. **Write report** — [migration-report-template.md](references/migration-report-template.md); stage = `enable`. Exclude
   `.ndjson-migration-report.md` from product PR unless requested.

## Monorepos

Enable one component at a time; update the ledger before stopping.

## Definition of done

[enable-gates.md](references/enable-gates.md): build/smoke pass (or explicit blocked), JSON envelope valid, `LOG_FORMAT`
wired.
