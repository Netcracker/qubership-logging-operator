# Placement probe (all stacks)

**Mandatory** before bulk call-site migration for each runtime component, and again as part of end-of-migration smoke.
Stage 1 JSON envelope (`time` / `level` / `message`) is **not** enough — this probe checks that the stack’s **event-field
API** actually yields **top-level** JSON keys operators can filter on.

Do **not** invent a novel strategy taxonomy each run. Use the fixed decision menu in
[user-decisions.md](user-decisions.md) § Event-field placement unsupported. Highlight a **recommended** option from
**local facts** (probe output + version/classpath). The user decides; invite a **user-provided** alternative.

## When

| Moment | Required? |
| ------ | --------- |
| After stage 1 envelope OK, **before** bulk call-site edits for that component | **Yes** |
| After implementing a user-chosen placement fix (infra / backend change) | **Yes** — re-probe before migrating call sites |
| End-of-migration smoke ([smoke-validation.md](smoke-validation.md)) | **Yes** — same pass criterion |

Skip only when the component has **no** logging event-field work (e.g. pure Helm/nginx access-log stage 2 N/A) — record
`placement probe: N/A` with reason in the report.

## Pass / fail

Emit **one** log line that uses the stack’s **intended** event-field API (not a prose-only `log.info("hello")`).

**PASS** when the captured stdout NDJSON line:

- Parses as a single JSON object
- Has a readable `message` (or stack-mapped equivalent) that does **not** require parsing to recover diagnostics
- Exposes the probe’s diagnostic values as **top-level** keys (same names the call will use in migration)

**FAIL** when any of:

- Diagnostics appear only inside `message` (e.g. `key=value … prose`, printf leftovers)
- Diagnostics appear only under nested `mdc` / equivalent used as **event** fields (correlation MDC is separate)
- Probe cannot run (no entrypoint, compile blocked) — record exact error; treat as **blocked** for placement until
  resolved or the user chooses defer

### Failure signatures (examples, not exhaustive)

| Stack | Common FAIL signature |
| ----- | --------------------- |
| Java / Quarkus + JBoss SLF4J bridge | `loggerClassName` = `org.slf4j.spi.DefaultLoggingEventBuilder`; message starts with `field=…` prefixes |
| Go message-string logger without helper/formatter | `message` contains `key=%v` / glued diagnostics; no top-level keys |
| Logback without structured encoder support | fields missing or only in formatted message text |

## How to probe (minimal)

Use the repo’s documented run/smoke path when possible. Otherwise a tiny one-off main / `quarkus:dev` / `go run` is fine.
Delete temporary probe mains unless the user wants them kept.

### Java / Quarkus / SLF4J

```java
slog.atInfo()
    .setMessage("placement probe")
    .addKeyValue("probe_field", "probe_value")
    .log();
```

Expect top-level `"probe_field":"probe_value"` (not only inside `message`).

Also record: Quarkus version, whether `quarkus-logging-json` is present, whether `JsonProvider` (or equivalent) exists
on the classpath — for the recommendation note only.

### Go / qubership-core-lib-go / logrus / zap

Log with the **same** field path migration will use (`logformat.Msg` / `logfields.Format` / `WithFields` / zap attrs /
`logr` values mapped by the adapter). Expect those keys at JSON top level.

### Logback / Spring

Same fluent or `StructuredArguments` pattern the playbook will use; confirm encoder output.

### Python / other

One structured call matching the repo convention; confirm top-level keys in NDJSON.

## On FAIL — stop

1. Do **not** bulk-migrate call sites for that component.
2. Ask the user per [user-decisions.md](user-decisions.md) § Event-field placement unsupported:
   - **Recommended** option + 1–3 sentences of evidence from this probe / local facts
   - Fixed **alternatives** from that section
   - Invite **user-provided** approach
3. Record probe command, sample JSON line (redact secrets), PASS/FAIL, and the user’s choice in the migration report.
4. Implement placement infra / backend change **only after** the user picks. Then **re-probe** (must PASS) before
   call-site migration.

## On PASS

Proceed with inventory and call-site migration using the stack playbook. End smoke must still show top-level fields on
real migrated lines (not only the probe).

## Recommendation (bounded — not open research)

When presenting options after FAIL, the agent **may** mark one option as recommended using **only**:

- Probe output and failure signature
- Local version / dependency facts (pom, go.mod, jar/classpath checks)

Do **not** run open-ended web research to invent a new “best architecture” each session. Optional deeper how-to research
is allowed **after** the user selects an option (to implement that choice), or if the user asks.
