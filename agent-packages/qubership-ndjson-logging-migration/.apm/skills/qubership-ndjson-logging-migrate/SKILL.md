---
name: qubership-ndjson-logging-migrate
description: >
  Use when migrating Qubership log call sites after NDJSON is already enabled (stage 1), or when the user asks for
  full NDJSON / structured-field migration. Triggers on SLF4J {}, Go log.*f (including Trace), residual Go printf
  verbs after dropping f (key=%v style), preformatted log.warn(message) / log.error(msg), shared {} template
  constants, monorepo call-site rollout, or "extract fields from messages". Not for config-only LOG_FORMAT / JSON
  envelope rollout — use qubership-ndjson-logging-enable.
---

# NDJSON Migrate (Stage 2)

Extract structured data from log **messages** into JSON **fields** while NDJSON output is already enabled (stage 1).

**Prerequisite:** JSON envelope on stdout — from `qubership-ndjson-logging-enable` or existing repo config. If not, run
stage 1 first or record config gaps in the report.

## Goal (optimize for this, not for greps)

Operators must **filter and aggregate** on stable top-level JSON keys (`resource_id`, `error`, `namespace`, …), not by
parsing prose inside `message`. Each event should still read as a clear human summary.

- **Win:** diagnostic values are named JSON fields; `message` says what happened; correlation
  (`request_id`, `tenant_id`, trace/span, `logType`) stays intact.
- **Lose:** rearranging call sites only so inventory greps go to zero while diagnostics remain buried in `message` —
  including `fmt.Sprintf(…)` / string build then `log.X("%s", msg)`, or drop-`f` with `key=%v` still in the format string.

Greps and gates are **smell checks** that the goal may be unmet. Clean greps alone never mean `migrated`.

## Hard rules (read before any edit)

1. **Serve the goal** — every edit should make diagnostics queryable as fields (or record an explicit no-action / blocked
   reason). Do not ship cosmetic rewrites that only silence greps.
2. **Inventory first** — find work via [preformatted-message-patterns.md](references/preformatted-message-patterns.md)
   (shared `{}` constants, preformatted logs, text blocks, Go `log.*f` / residual printf). Inventory finds candidates;
   the goal decides what “fixed” means.
3. **Java event fields** — SLF4J 2.x fluent API (`addKeyValue`) so event data lands in JSON for search. Never add
   `StructuredLog` or per-call `MDC.put` for event data. Request-scoped MDC in filters stays as-is.
4. **Go fields** — prefer a real field API or repo helper so keys appear at JSON top level — see
   [go-qubership-lib.md](references/go-qubership-lib.md).
5. **Stop and ask** on shared `{}` template constants and logged preformatted messages — do not guess. Choices:
   [user-decisions.md](references/user-decisions.md). After confirmation, shapes:
   [pattern-recipes.md](references/pattern-recipes.md).
6. **API / throw text** — when a string is also used for `Response.entity`, DTO error fields, or exception detail, keep
   that string unchanged; structure **only** the log line (same variable in `setMessage` when message is conditional).
7. **Do not claim done** while the goal is unmet: diagnostics still only in `message`, open user-decision rows,
   `StructuredLog` / templating constants, or any completion gate FAIL/PARTIAL — see
   [migration-report-template.md](references/migration-report-template.md) § Status rules.

## Reference map

| When                 | Read                                                                                                   |
| -------------------- | ------------------------------------------------------------------------------------------------------ |
| Inventory patterns   | [preformatted-message-patterns.md](references/preformatted-message-patterns.md)                        |
| User choice          | [user-decisions.md](references/user-decisions.md)                                                      |
| Pattern recipes      | [pattern-recipes.md](references/pattern-recipes.md) — after user confirms a decision                   |
| Stack implementation | [java-quarkus.md](references/java-quarkus.md) or [go-qubership-lib.md](references/go-qubership-lib.md) |
| Cross-cutting rules  | [coding-approaches.md](references/coding-approaches.md)                                                |
| Field naming contract | [schema.md](references/schema.md) — when mapping fields                                                |
| Before claiming done | [completion-gates.md](references/completion-gates.md)                                                  |
| Report               | [migration-report-template.md](references/migration-report-template.md)                                |
| Smoke                | [smoke-validation.md](references/smoke-validation.md)                                                  |
| Pitfalls             | [corner-cases.md](references/corner-cases.md)                                                          |
| Background           | [evidence.md](references/evidence.md)                                                                  |

## Workflow

1. Confirm stage 1 — JSON smoke passed or document config blocker.
2. **Repo-root discovery** — coverage ledger for all runtime components.
3. **Classify stack** → [java-quarkus.md](references/java-quarkus.md) or [go-qubership-lib.md](references/go-qubership-lib.md).
4. **Inventory** — [preformatted-message-patterns.md](references/preformatted-message-patterns.md) (constants,
   preformatted, text-block `{}`, `log.*f` including Trace, residual `%v`/`%d`/… on non-`f` log calls).
5. **Classify** sites: `migrate`, `static/no action`, `needs user decision`, `blocked`.
6. **User decisions** — [user-decisions.md](references/user-decisions.md). After confirmation, read
   [pattern-recipes.md](references/pattern-recipes.md) before editing those sites.
7. **Map fields** — [schema.md](references/schema.md) + stack playbook + [coding-approaches.md](references/coding-approaches.md).
8. **Implement** in small batches — build after each batch; spot-check that new fields are queryable, not only that
   greps shrank.
9. **Re-inventory** — no unaccounted formatted / preformatted / text-block / residual-printf candidates.
10. **Smell checks** (below) then full [completion-gates.md](references/completion-gates.md).
11. **Smoke** — [smoke-validation.md](references/smoke-validation.md); confirm diagnostic keys at JSON top level.
12. **Write report** — stage = `migrate`; status rules in
    [migration-report-template.md](references/migration-report-template.md); exclude from product PR unless requested.
13. **Propose skill updates** in the APM package source, not `.agents/skills` copies.

## Smell checks (before claiming done)

Run against production sources (adjust paths). Hits suggest the **goal** is unmet — fix toward queryable fields, or list
as blocked / user-decision with counts. **Clean greps are not sufficient** (e.g. `fmt.Sprintf` then `log.X("%s", msg)`
with diagnostics inside `msg` still fails the goal). Spot-check field names and JSON placement after greps.

```bash
# Java — forbidden helper / per-call MDC for event fields
grep -rn 'StructuredLog\|MDC\.put' --include='*.java' src/main/java || true

# Java — shared string constants that still contain SLF4J {} (misleading zero — ask)
grep -rnE 'String\s+[A-Z][A-Z0-9_]*\s*=\s*"[^"]*\{\}' --include='*.java' src/main/java || true

# Java — unreviewed preformatted log sites
grep -rnE 'log\.(warn|error|debug|info)\((message|msg|aggregatedError|errorMsg|warn|e\.getMessage)' \
  --include='*.java' src/main/java || true

# Java — text-block logs (same-line {} grep misses these; open each hit for {})
grep -rnE 'log\.(info|debug|warn|error|trace)\("""' --include='*.java' src/main/java || true

# Java — codemod field-name residue (after bulk rewrite; polish required; target 0)
grep -rnE 'addKeyValue\("[^"]*(_get_|_stream_|e_get_message)' --include='*.java' src/main/java || true
grep -rn '"arg[0-9]\+"' --include='*.java' src/main/java || true

# Go — include Trace; exclude _test.go / commented lines in review
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)f\(' --include='*.go' . || true

# Go — residual diagnostic format verbs after dropping f (smell).
# Same-line check; if any hit: review that whole file for multi-line concatenations.
# Also review fmt.Sprintf / string build then log.X("%s", msg) — greps miss that dodge.
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)(C)?\(.*%[vTdoxXefg]' --include='*.go' . || true
```

Then run [completion-gates.md](references/completion-gates.md). Semantic + smoke gates decide `migrated`, not pattern
counts alone — see [go-qubership-lib.md](references/go-qubership-lib.md) and completion-gates §3–§4.1.

## Monorepos

One component at a time; update ledger before stopping.

## Definition of done

The **goal** is met for each component: queryable fields, readable `message`, correlation preserved; build/integrity OK;
smell checks clean or accounted for; [completion-gates.md](references/completion-gates.md) PASS (or blocked with reason).
Clean greps without queryable fields is **not** done.
