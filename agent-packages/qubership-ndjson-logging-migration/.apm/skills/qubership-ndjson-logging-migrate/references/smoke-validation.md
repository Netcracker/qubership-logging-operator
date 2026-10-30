# Smoke Validation Checklist

After call-site migration, discover the component's existing runtime validation capabilities and use the
**highest already-runnable** evidence level. Do not create complex compose, kind, cluster, or service infrastructure
solely for logging validation.

| Level | Evidence |
| ----- | -------- |
| **L0 — transformation-only** | Direct formatter/helper unit test. Never placement evidence and not end smoke. |
| **L1 — logging-runtime integration** | Real application-facing logger through initialized production logging graph to final captured sink. Normal placement minimum, not an L2/L3 smoke PASS. |
| **L2 — packaged process stdout** | Built application/process emits configured JSON stdout. |
| **L3 — practical deployment logs** | Existing compose, kind, or integration deployment exposes application logs. |

**Placement probe first:** Before bulk call-site edits, run [placement-probe.md](placement-probe.md) for every
stack/language component at L1 or higher.

**Build before smoke:** The component's build gate must pass before L2/L3 evidence is accepted. A smoke attempt on
non-compiling code is invalid.

## Capability discovery

1. Inspect existing test profiles, packaged run/config-check commands, compose/kind assets, and integration jobs.
2. Choose the highest level that is already runnable with available credentials and environment.
3. Attempt L2 or L3 only when that practical path exists. Do not build new deployment infrastructure for this task.
4. If L2/L3 is unavailable or blocked, retain exact L1 placement evidence and record the higher-fidelity blocker.

A component may be `migrated` without L2/L3 only when placement passed at L1 or higher and **every non-smoke gate**
passes. Record smoke as `UNAVAILABLE` or `BLOCKED`, not as a fabricated PASS.

## Tool-neutral validation contract

Parse the **final rendered output** with a JSON parser already available in the component or its test framework.
Never require or install Python, Node, `jq`, or another runtime solely for this skill.

For each selected real migrated line:

- Require a configured time field (`time`, `timestamp`, or the component's documented alias), level, and message.
- Require the chosen migrated diagnostics as top-level keys with values preserved.
- Reject diagnostics that occur only in a leading `key=value` message prefix.
- Require `message` (or its configured alias) to remain readable prose.
- Preserve correlation fields according to the component's established configuration.

## Go / logrus

```bash
# Build/test gate only; generic test output is not placement or smoke evidence unless it captures the final sink at L1.
GOWORK=off go test ./...

# Existing practical L2 path when it emits configured stdout.
LOG_FORMAT=json go run . -check-config -config-path examples/config.yaml -log-level error
```

Use the packaged binary instead of `go run` when the normal build/run path produces one. If the binary has no
`-check-config`, use the closest existing startup path that emits logs. Capture complete lines without truncating or
rewriting the output before validation.

When the repo lives inside a parent `go.work`, prefix tests and smoke with `GOWORK=off`.

If no practical process path exists, use the L1 Go integration pattern from
[placement-probe.md](placement-probe.md) and record L2/L3 as unavailable.

## Java / Quarkus

```bash
# Required before claiming Java migration complete
mvn -pl <module> -am compile

# Existing practical path when credentials/config are available
mvn -pl <module> quarkus:dev   # or documented integration smoke
```

If Maven compile is blocked (private packages, 401), record the exact error. This is a non-smoke build block, so the
component cannot be marked migrated.

When no practical process/deployment path exists, use the Quarkus L1 application-test pattern from
[placement-probe.md](placement-probe.md). Do not describe that as L2.

After bulk edits, also run [completion-gates.md](completion-gates.md) §4.1 (semantic field-name review + spot-check;
smell-checks.sh J6a/J6b — blocking) and §2.2 (illegal text blocks — J5/J7).

## Python

```bash
LOG_FORMAT=json python -m <app.module> --help   # or documented entrypoint
```

Use this only when Python is already the component runtime. Validate final output with the component's existing parser
or test framework.

## Fixture-only edits (eval / single-file scope)

Directly formatting a representative log event is L0. It can support transformation checks but cannot pass placement
or replace end smoke. Record the achieved level and scope limitation. Placement remains PASS only with separate
L1-or-higher evidence, or N/A with a valid no-event-field reason.

## What to record in the migration report

| Field | Example |
| ----- | ------- |
| Achieved evidence | `L2 — packaged process stdout` |
| Command/test | `LOG_FORMAT=json <existing packaged-run command>` |
| Result | `PASS — final line parsed; configured timestamp, level, message, resource_id present` |
| Placement | `PASS at L1 — application logger -> configured console sink` |
| Validator | `<existing component parser/test assertion>` |
| Higher-fidelity blocker | `L3 unavailable — repository has no compose/kind/integration deployment` |

For L1-only completion, record the exact L1 test/command and result plus why each practical L2/L3 path was unavailable
or blocked. L0 unit tests alone do not satisfy placement.
