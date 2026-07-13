# Java / Quarkus / SLF4J Playbook

Read when the target component uses Maven/Quarkus, SLF4J, or Logback-style logging.

## Infrastructure

- Add or enable Quarkus JSON console (`quarkus-logging-json`) or existing Logback JSON encoder.
- Wire `LOG_FORMAT=text|json` in Helm; use `%text` profile or equivalent for legacy bracket format when needed.
- Declare promoted MDC keys in `quarkus.log.console.json.fields.*` so they appear as top-level JSON fields — verify with
  one captured stdout line.

## Structured helper pattern

If the repo already has a thin MDC wrapper (put → log → clear in `finally`), **extend it** — do not add a parallel
stack. If none exists, use short-lived `MDC.put` / `MDC.remove` around the log call (same lifecycle).

**Level guards:** for `debug`/`trace`, check `isDebugEnabled()` / `isTraceEnabled()` before MDC work — original `{}`
calls were lazy.

**Quarkus JSON:** MDC keys often land under `mdc.*` unless promoted via `quarkus.log.console.json.additional-field.*`
(or equivalent in the target repo). After migrating a field, capture one runtime JSON line and confirm where it appears.

## Converting call sites

**Before:**

```java
log.error("Logical backup failed: id={}, error={}", id, msg, throwable);
```

**After** (repo helper with throwable overload — adjust class/method names to match the target):

```java
StructuredLogging.error(log, "Logical backup failed", throwable,
        "backup_id", id, "error_message", msg);
```

- Pass the **exception via the throwable overload** (`log.error(message, throwable)` inside the helper) — do **not**
  put a `Throwable` in MDC as a field value (stringifies without stack trace).
- Field names from **message semantics** (`backup_id`, `namespace`, `status`) — never `arg0`, never duplicate keys in one
  call (later pair overwrites earlier MDC put).
- Equivalent manual pattern when no helper exists:

```java
MDC.put("backup_id", String.valueOf(id));
MDC.put("error_message", msg);
try {
    log.error("Logical backup failed", throwable);
} finally {
    MDC.remove("backup_id");
    MDC.remove("error_message");
}
```

## Exception mappers

Replace `log.warn(WARNING_MESSAGE, class, path, msg)` with structured fields + throwable overload, or centralize in
`Utils.logRequestWarning(...)`. A shared `{}` constant still templates at runtime — migrate to explicit fields or list
under user decision until done.

**Only change logging lines** — preserve `buildResponse(status, supplier)` overloads and response builders. Remove unused
imports if a file no longer calls the structured helper.

## Hand-migrate (do not bulk-codemod)

- REST controllers
- Exception mappers
- Flyway Java migrations (`V1_*__*.java`)
- Multi-line logs and `"""` text blocks — single-line text blocks are illegal Java

## Build gate

`mvn -pl <module> -am compile` must pass before claiming this component done. If blocked by private packages (401),
record under **Blocked validation** — status stays not migrated-complete.
