# Placement probe (all stacks)

**Mandatory** before bulk call-site migration for each runtime component. Stage 1 JSON envelope
(`time` / `level` / `message`) is **not** enough — this probe checks that the stack’s application-facing
**event-field API** actually yields **top-level** JSON keys operators can filter on.

Do **not** invent a novel strategy taxonomy each run. Use the fixed decision menu in
[user-decisions.md](user-decisions.md) § Event-field placement unsupported. Highlight a **recommended** option from
**local facts** (probe output + version/classpath). The user decides; invite a **user-provided** alternative.

## When

| Moment | Required? |
| ------ | --------- |
| After stage 1 envelope OK, **before** bulk call-site edits for that component | **Yes, at L1 or higher** |
| After implementing a user-chosen placement fix (infra / backend change) | **Yes** — re-probe before migrating call sites |
| End validation ([smoke-validation.md](smoke-validation.md)) | Reuse L1 evidence; attempt L2/L3 only on practical existing paths and record any higher-fidelity blocker |

Skip only when the component has **no** logging event-field work (e.g. pure Helm/nginx access-log stage 2 N/A) — record
`placement probe: N/A` with reason in the report.

## Evidence requirement

Use the highest already-runnable level. Do not create complex deployment infrastructure solely for this probe.

| Level | Placement use |
| ----- | ------------- |
| **L0 — transformation-only** | Direct formatter/helper unit test. Supporting evidence only; **never placement PASS**. |
| **L1 — logging-runtime integration** | Normal minimum. Initialize the framework/app test context, real public logger API, lifecycle, handler/appender chain, encoder/formatter, and production JSON settings; capture final rendered sink output. |
| **L2 — packaged process stdout** | Valid placement evidence when an existing runnable process path is practical. |
| **L3 — practical deployment logs** | Valid placement evidence when an existing compose/kind/integration deployment is practical. |

At L1, unrelated databases, migrations, schedulers, and external services may be disabled. Do not replace or bypass the
logging graph under test.

## Probe values and pass/fail

Emit through the **same application-facing logger and event-field API** that migration will use. Include representative
values in one or more captured events:

- simple: `alpha`
- quoted/whitespace: `value with "quotes" and spaces`
- braced/map: `{kind=map, count=2}`
- parenthesized/object: `Widget(id=7, label=sample)`

**PASS** at L1 or higher when each captured final NDJSON object:

- Parses as a single JSON object
- Has the configured time field (`time`, `timestamp`, or the component's configured alias), `level`, and
  `message` (or configured equivalents)
- Has readable prose in `message`; diagnostics do not need to be parsed from it
- Exposes every chosen diagnostic as a **top-level** key with its value preserved
- Does not contain those diagnostics only as a leading `key=value` message prefix

**FAIL** when any of:

- Diagnostics appear only inside `message` (e.g. `key=value … prose`, printf leftovers)
- Diagnostics appear only under nested `mdc` / equivalent used as **event** fields (correlation MDC is separate)
- Evidence only calls the formatter/helper directly (L0), bypasses the public logger, or replaces the configured
  handler/appender/encoder graph
- No L1-or-higher path can run — record the exact command/test and error; treat placement as **blocked** until
  resolved or the user chooses defer

### Failure signatures (examples, not exhaustive)

| Stack | Common FAIL signature |
| ----- | --------------------- |
| Java / Quarkus + JBoss SLF4J bridge | `loggerClassName` = `org.slf4j.spi.DefaultLoggingEventBuilder`; message starts with `field=…` prefixes |
| Go message-string logger without helper/formatter | `message` contains `key=%v` / glued diagnostics; no top-level keys |
| Logback without structured encoder support | fields missing or only in formatted message text |

## How to probe at L1 (minimal)

Prefer an existing framework or application integration test. Configure it with the production JSON logging settings,
emit through the public logger, and capture the configured console/output sink after the full logging chain renders it.
Use an already available JSON parser or the component's test framework; never require or install Python or another
runtime solely for validation.

### Java / Quarkus / SLF4J

Run a Quarkus application test context with the component's production JSON logging profile. Capture the configured
console handler output, call the injected/application logger's SLF4J fluent API, and assert the final rendered line.
Do not instantiate the JSON formatter directly.

Also record: Quarkus version, whether `quarkus-logging-json` is present, whether `JsonProvider` (or equivalent) exists
on the classpath — for the recommendation note only.

### Go / qubership-core-lib-go / logrus / zap

Initialize the component's normal logging bootstrap with production JSON settings, redirect/capture its final configured
writer, and log through the public application logger using the same field path migration will use
(`logformat.Msg` / `logfields.Format` / `WithFields` / zap attrs / `logr` values mapped by the adapter). A unit test
that invokes only `Format` or an encoder is L0.

### Logback / Spring

Start the Spring application test context with the production Logback JSON configuration, capture output from the
configured console appender, and emit through the application SLF4J logger using the fluent or structured-argument path
the migration will use. Do not replace the production encoder with a test formatter.

### Python / other

Use the same L1 contract: application test context, public logger API, production logging graph/settings, and final sink
capture. Framework-specific test utilities are acceptable only when they observe rather than replace that graph.

## On FAIL — stop

Do **not** bulk-migrate call sites for that component. Follow
[user-decisions.md](user-decisions.md) § Event-field placement unsupported — it owns the full procedure (probe
evidence → recommended option + alternatives → wait for the user → implement the choice → **re-probe** until PASS)
and what to record in the migration report.

## On PASS

Proceed with inventory and call-site migration using the stack playbook. End validation should show top-level fields on
real migrated lines; attempt L2/L3 only when a practical existing path is available.

## Recommendation (bounded — not open research)

When presenting options after FAIL, the agent **may** mark one option as recommended using **only**:

- Probe output and failure signature
- Local version / dependency facts (pom, go.mod, jar/classpath checks)

Do **not** run open-ended web research to invent a new “best architecture” each session. Optional deeper how-to research
is allowed **after** the user selects an option (to implement that choice), or if the user asks.
