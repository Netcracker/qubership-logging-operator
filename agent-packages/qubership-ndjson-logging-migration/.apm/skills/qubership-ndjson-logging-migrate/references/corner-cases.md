# Corner Cases

Keep this file short. Add lessons from real pilot migrations as one or two lines each.

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

## Pilot backlog

- **Go logrus dual-format:** `LOG_FORMAT=text` may map to a legacy `cloud` bracket formatter; `LOG_FORMAT=json` emits
  NDJSON with `time`/`level`/`message` plus `WithField` context (e.g. `error_code`). Keep chart/env fallbacks during
  rollout. Map `text` consistently for all resolution paths (env, deprecated var, flag).
- **Returned errors vs NDJSON fields:** Keep `fmt.Errorf` return text unchanged for API compatibility; log
  `config_path`, `operation`, and `error` once at the config I/O failure site (same pattern as `ValidateConfig`).
  Callers should exit/return without emitting a second summary line for the same failure.
- **One error record per failure:** Do not log a wrapper/summary line plus a detail line for the same event (for example
  remove `"see reasons list below"` headers when each reason already has its own structured ERROR). Retry attempts may
  log separately when they include distinct attempt/retry fields.
- **Pre-logger stdout:** `fmt.Printf` for `-version`, shutdown banners, and log-rotation bootstrap messages remain plain
  text only when recorded as blocked with a concrete bootstrap-logging reason.
- Validate Python production logger choice.
- Validate Log4j/log4j2 JSON pattern.
- **Multi-stack monorepo discovery:** A root `pom.xml` may cover only one Java subtree; sibling Go binaries with their
  own `go.mod` and `helm-templates/` are separate runtime components. Scan repo root before claiming completion.
- **Parent go.work:** When the target repo is linked from a parent workspace, run Go tests and smoke with
  `GOWORK=off` so results reflect the component under migration.
- **Infra-only trap:** Logger + Helm + a handful of Go files is not a completed migration; completion gate must show
  zero active `log.*f` in production packages.
- **Logged preformatted messages:** Sites such as `log.warn(message)`, `log.error(aggregatedError)`, and `log.error(msg)`
  are not static/no-action. Inventory count, list in the report, and ask whether to structure at the logging boundary or
  keep prose-only `message`.
- **Bulk Java codemod:** `632→0 {}` with failing `mvn compile` is not done. Regex codemods can delete REST handlers,
  break `"""` text blocks, drop `@Slf4j` imports, use `arg0`/`arg1` MDC keys, duplicate keys in one call, and drop
  throwables. Require build + integrity + semantic gates from [completion-gates.md](completion-gates.md).
- **Java WARNING_MESSAGE constants:** Moving `{}` into a shared string constant does not structure runtime values; list
  under user decision or migrate call sites to pass fields explicitly.
- **Quarkus MDC promotion:** MDC keys may appear only under `mdc.*` unless declared in `quarkus.log.console.json.fields.*`;
  verify one runtime JSON line after migration.
- **Go logfields.Format:** Regex re-parse of `key=value` suffixes is fragile; quote whitespace values and protect reserved
  keys (`time`, `level`, `message`, `class`, `request_id`).
- **Skill source:** edit the APM skill package source; reinstall/sync to update deployed `.agents/skills` copies.
- **Migration report:** `.ndjson-migration-report.md` is a worktree ledger — exclude from product PRs unless explicitly
  requested.
- Validate Envoy JSON access log field mapping.
- Validate Nginx JSON access log support versus existing FluentBit parser path.
- Validate zerolog canonical field names in a Qubership service.
