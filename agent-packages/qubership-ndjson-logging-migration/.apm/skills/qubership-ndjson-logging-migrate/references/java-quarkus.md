# Java / Quarkus / SLF4J Playbook

Read when the target component uses Maven/Quarkus, SLF4J, or Logback-style logging.

## Two roles — do not conflate

| Role | What | How |
| ---- | ---- | --- |
| **Correlation** (thread/request scope) | `request_id`, `tenant_id`, trace/span | Set once in a filter/interceptor; Quarkus JSON `additional-field` with `%X{...}` is OK |
| **Event fields** (this log line only) | `resource_id`, `namespace`, `status`, … | **SLF4J 2.x fluent API** — not per-call `MDC.put` |

Fluent API exists so operators can **search** event data in JSON — not merely to clear `{}` greps.

**Do not** add new `StructuredLog`-style wrappers or per-call MDC for diagnostic fields. MDC is not a structured-logging
API.

## Infrastructure (stage 1 — usually done)

- Quarkus JSON console (`quarkus-logging-json`) or Logback JSON encoder enabled.
- `LOG_FORMAT=text|json` in Helm; `%text` profile for legacy bracket format when needed.
- Correlation fields promoted via `quarkus.log.console.json.additional-field.*` (e.g. `%X{requestId}`).

## Preferred: SLF4J 2.x fluent API (event fields)

Verify the target uses SLF4J 2.x (`org.slf4j.Logger` with `atInfo()` / `addKeyValue()`).

**Do not assume** Quarkus / JBoss Logging promotes `addKeyValue` to top-level JSON. The JBoss SLF4J bridge often lacks a
fluent `LoggingEventBuilder`, so SLF4J’s `DefaultLoggingEventBuilder` may prefix `key=value` onto `message` while
`quarkus-logging-json` only serializes that string. **Always** run the [placement probe](placement-probe.md) before bulk
migrate. On FAIL → [user-decisions.md](user-decisions.md) § Event-field placement unsupported.

**Before:**

```java
log.error("operation failed: id={}, error={}", id, msg, throwable);
```

**After:**

```java
log.atError()
    .setMessage("operation failed")
    .addKeyValue("resource_id", id)
    .addKeyValue("error_message", msg)
    .setCause(throwable)
    .log();
```

**Practices:**

- **`setMessage()`** — short **what happened** summary when migrating `{}` templates; see §4.4 in
  [completion-gates.md](completion-gates.md). When the same string is also an API/DTO/exception consumer, keep that
  string — [pattern-recipes.md](pattern-recipes.md).
- Field names from **message semantics** — [completion-gates.md](completion-gates.md) §4.1.
- Use `setCause(throwable)` when the original SLF4J call passed an exception — do not put `Throwable` in a field value.
- `atDebug()` / `atTrace()` are lazy — prefer them over guarded `log.debug(...)` when using the fluent API.
- Chain multiple `addKeyValue` calls; do not repeat the same key in one event.

**Levels:** `atTrace`, `atDebug`, `atInfo`, `atWarn`, `atError` — match the original level unless user approved a change.

## Verify JSON output

**Before bulk migrate:** [placement-probe.md](placement-probe.md) must PASS for this component.

After migrating a batch, capture one runtime stdout JSON line and confirm:

- `time`, `level`, `message` present
- `addKeyValue` fields appear at the **top level** (not only under `mdc.*`, not glued into `message`)
- Correlation fields (`request_id`, `tenant_id`) still populate from request-scoped MDC / `%X{...}` config

If fields land under `mdc.*` only, the implementation is wrong for event data — fix the call site (fluent API), not by
promoting hundreds of keys in `application.properties`. If fields are glued into `message` with
`DefaultLoggingEventBuilder`, that is a **placement** failure — stop and ask; do not keep rewriting call sites hoping
the bridge will catch up.

## Logback / Spring (non-Quarkus)

When the stack is Logback + SLF4J 2.x, use the same fluent API. If the repo already ships **logstash-logback-encoder**,
`StructuredArguments.kv("field", value)` is acceptable — still not per-call MDC. Extend what the repo already uses; do
not introduce a parallel pattern.

## Exception mappers

Sites such as `log.warn(SHARED_TEMPLATE, class, path, msg)` use a **shared `{}` template constant** — grep hits zero
inline `{}` while values still interpolate at runtime.

**Stop and ask the user before editing these call sites** — do not pick an approach silently and do not defer the
question to the end of the migration. See [user-decisions.md](user-decisions.md) § Java shared `{}` template constants.

**Only change logging lines** — preserve response-builder / `buildResponse`-style overloads.

## Anti-patterns (do not introduce)

| Anti-pattern | Why |
| ------------ | --- |
| New `StructuredLog` / per-call `MDC.put` helper | Wrong abstraction; fields hide under `mdc.*`; leak and overwrite bugs |
| `MDC.put` at every call site (manual or wrapped) | Leak risk, duplicate keys, not event-scoped |
| Bulk regex codemod to any helper without `mvn compile` | Deleted endpoints, illegal text blocks, generic field keys |

If the repo **already** has a legacy per-call MDC helper from an earlier migration, prefer **replacing call sites with
fluent API** over extending the helper. Remove the helper when no callers remain.

## Hand-migrate (do not bulk-codemod)

- REST controllers
- Exception mappers
- Schema / DB migration Java sources (`V1_*__*.java` and similar)
- Multi-line logs and `"""` text blocks — single-line text blocks are illegal Java

## Build gate

`mvn -pl <module> -am compile` must pass before claiming this component done. If blocked by private packages (401),
record under **Blocked validation** — status stays not migrated-complete.
