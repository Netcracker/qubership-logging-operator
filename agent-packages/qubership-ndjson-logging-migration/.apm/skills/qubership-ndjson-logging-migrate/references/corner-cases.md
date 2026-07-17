# Corner Cases

Keep this file short. Unique pitfalls and open validation items — not a second copy of policy.

**Canonical policy (do not restate here):**

| Topic | Source of truth |
| ----- | --------------- |
| Hard rules / self-check | [SKILL.md](../SKILL.md) |
| User decisions | [user-decisions.md](user-decisions.md) |
| Confirmed transformation shapes | [pattern-recipes.md](pattern-recipes.md) |
| Gates | [completion-gates.md](completion-gates.md) |
| Java / Go call patterns | [java-quarkus.md](java-quarkus.md), [go-qubership-lib.md](go-qubership-lib.md) |

## Known from logging docs and fixtures

- Brackets or JSON payloads inside legacy `[key=value]` fields can break text parsing; move them to JSON string fields.
- Fields without `=` inside brackets are not structured fields; either remove them or promote them to named JSON fields.
- Long dynamic field names and values can hit OpenSearch limits; keep field names stable and short.
- Multiline stack traces should stay part of one log event where the framework supports it; avoid splitting one logical
  event across many stdout JSON records.
- Preserve audit routing with top-level `logType: audit`.
- Direct GELF appenders bypass stdout/FluentBit; treat them as a separate migration path.
- Keep framework-specific defaults (`msg`, `ts`, `caller`) only if the pipeline accepts them; otherwise map to
  `message`, `time`, and stable source fields.
- When promoting Go `logrus` context with `WithFields`, ensure field values are JSON-safe or stringified; unsupported
  values can otherwise break the whole log line.

## Pilot backlog (unique)

- **Go logrus dual-format:** `LOG_FORMAT=text` may map to a legacy `cloud` bracket formatter; `LOG_FORMAT=json` emits
  NDJSON with `time`/`level`/`message` plus `WithField` context (e.g. `error_code`). Keep chart/env fallbacks during
  rollout. Map `text` consistently for all resolution paths (env, deprecated var, flag).
- **Returned errors vs NDJSON fields:** Keep `fmt.Errorf` return text unchanged for API compatibility; structure at the
  logging boundary — see [user-decisions.md](user-decisions.md) § Returned diagnostics.
- **One error record per failure:** Do not log a wrapper/summary line plus a detail line for the same event (for example
  remove `"see reasons list below"` headers when each reason already has its own structured ERROR). Retry attempts may
  log separately when they include distinct attempt/retry fields. See also [user-decisions.md](user-decisions.md)
  § One error record per failure.
- **Pre-logger stdout:** `fmt.Printf` for `-version`, shutdown banners, and log-rotation bootstrap messages remain plain
  text only when recorded as blocked with a concrete bootstrap-logging reason.
- **Multi-stack monorepo discovery:** A root `pom.xml` may cover only one Java subtree; sibling Go binaries with their
  own `go.mod` and `helm-templates/` are separate runtime components. Scan repo root before claiming completion.
- **Parent go.work:** When the target repo is linked from a parent workspace, run Go tests and smoke with
  `GOWORK=off` so results reflect the component under migration.
- **Infra-only trap:** Logger + Helm + a handful of Go files is not a completed migration; completion gate must show
  zero active `log.*f` in production packages.
- **Bulk Java codemod:** `632→0 {}` with failing `mvn compile` is not done — see [completion-gates.md](completion-gates.md).
- **Java JSON field placement:** `addKeyValue` fields at top level in NDJSON, not only under `mdc.*` — see
  [java-quarkus.md](java-quarkus.md) § Verify JSON output.
- **Go logfields.Format:** Regex re-parse of `key=value` suffixes is fragile; quote whitespace values and protect reserved
  keys (`time`, `level`, `message`, `class`, `request_id`).
- **Skill source:** edit the APM skill package source; reinstall/sync to update deployed `.agents/skills` copies.
- **Migration report:** `.ndjson-migration-report.md` is a worktree ledger — exclude from product PRs unless explicitly
  requested.
- Validate Python production logger choice.
- Validate Log4j/log4j2 JSON pattern.
- Validate Envoy JSON access log field mapping.
- Validate Nginx JSON access log support versus existing FluentBit parser path.
- Validate zerolog canonical field names in a Qubership service.
