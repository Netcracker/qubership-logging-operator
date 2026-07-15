# Java / Quarkus / SLF4J Playbook

Read when the target component uses Maven/Quarkus, SLF4J, or Logback-style logging.

## Two roles — do not conflate

| Role | What | How |
| ---- | ---- | --- |
| **Correlation** (thread/request scope) | `request_id`, `tenant_id`, trace/span | Set once in a filter/interceptor; Quarkus JSON `additional-field` with `%X{...}` is OK |
| **Event fields** (this log line only) | `backup_id`, `namespace`, `status`, … | **SLF4J 2.x fluent API** — not per-call `MDC.put` |

**Do not** add new `StructuredLog`-style wrappers or per-call MDC for diagnostic fields. MDC is not a structured-logging
API.

## Infrastructure (stage 1 — usually done)

- Quarkus JSON console (`quarkus-logging-json`) or Logback JSON encoder enabled.
- `LOG_FORMAT=text|json` in Helm; `%text` profile for legacy bracket format when needed.
- Correlation fields promoted via `quarkus.log.console.json.additional-field.*` (e.g. `%X{requestId}`).

## Preferred: SLF4J 2.x fluent API (event fields)

Verify the target uses SLF4J 2.x (`org.slf4j.Logger` with `atInfo()` / `addKeyValue()`). Quarkus 3 + JBoss Logging
typically supports this through the SLF4J bridge.

**Before:**

```java
log.error("Logical backup failed: id={}, error={}", id, msg, throwable);
```

**After:**

```java
log.atError()
    .setMessage("Logical backup failed")
    .addKeyValue("backup_id", id)
    .addKeyValue("error_message", msg)
    .setCause(throwable)
    .log();
```

**Practices:**

- **`setMessage()`** — short **what happened** summary (action/outcome); no `{}` placeholders after migration. Put
  identifiers and diagnostics in `addKeyValue`, not duplicated in the message. The line should still make sense when
  someone reads only `message` in a dashboard — not `"."`, label stubs (`backup_id=`), or empty holes after extraction
  (see [completion-gates.md](completion-gates.md) §4.4).
- Field names from **message semantics** (`backup_id`, `namespace`, `status`) — not positional or generic keys
  (`arg0`, `argument1`, `param2`, `value0`) and not leaked locals (`i`, `ns`, `sbe`). Greps cannot catch every bad
  name; spot-check migrated sites per [completion-gates.md](completion-gates.md) §4.1.
- Use `setCause(throwable)` when the original SLF4J call passed an exception — do not put `Throwable` in a field value.
- `atDebug()` / `atTrace()` are lazy — prefer them over guarded `log.debug(...)` when using the fluent API.
- Chain multiple `addKeyValue` calls; do not repeat the same key in one event.

**Levels:** `atTrace`, `atDebug`, `atInfo`, `atWarn`, `atError` — match the original level unless user approved a change.

## Verify JSON output

After migrating a batch, capture one runtime stdout JSON line and confirm:

- `time`, `level`, `message` present
- `addKeyValue` fields appear at the **top level** (not only under `mdc.*`)
- Correlation fields (`request_id`, `tenant_id`) still populate from request-scoped MDC / `%X{...}` config

If fields land under `mdc.*`, the implementation is wrong for event data — fix the call site (fluent API), not by
promoting hundreds of keys in `application.properties`.

## Logback / Spring (non-Quarkus)

When the stack is Logback + SLF4J 2.x, use the same fluent API. If the repo already ships **logstash-logback-encoder**,
`StructuredArguments.kv("field", value)` is acceptable — still not per-call MDC. Extend what the repo already uses; do
not introduce a parallel pattern.

## Exception mappers

Sites such as `log.warn(WARNING_MESSAGE, class, path, msg)` use a **shared `{}` template constant** — grep hits zero
inline `{}` while values still interpolate at runtime.

**Stop and ask the user before editing these call sites** — do not pick an approach silently and do not defer the
question to the end of the migration. See [user-decisions.md](user-decisions.md) § Java shared `{}` template constants.

**Only change logging lines** — preserve `buildResponse(status, supplier)` overloads and response builders.

## Anti-patterns (do not introduce)

| Anti-pattern | Why |
| ------------ | --- |
| New `StructuredLog` / per-call `MDC.put` helper | Wrong abstraction; fields hide under `mdc.*`; PR #551-class bugs |
| `MDC.put` at every call site (manual or wrapped) | Leak risk, duplicate keys, not event-scoped |
| Bulk regex codemod to any helper without `mvn compile` | Deleted endpoints, illegal text blocks, generic field keys |

If the repo **already** has a legacy per-call MDC helper from an earlier migration, prefer **replacing call sites with
fluent API** over extending the helper. Remove the helper when no callers remain.

## Hand-migrate (do not bulk-codemod)

- REST controllers
- Exception mappers
- Flyway Java migrations (`V1_*__*.java`)
- Multi-line logs and `"""` text blocks — single-line text blocks are illegal Java

## Build gate

`mvn -pl <module> -am compile` must pass before claiming this component done. If blocked by private packages (401),
record under **Blocked validation** — status stays not migrated-complete.
