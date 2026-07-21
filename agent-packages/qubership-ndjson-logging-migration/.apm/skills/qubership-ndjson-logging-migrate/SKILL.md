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

## Hard rules (read before any edit)

1. **Inventory first** — shared `{}` constants, preformatted `log.*(msg|message|…)`, Go `log.*f` (include **Trace**),
   and residual printf verbs on non-`f` log methods — before bulk edits. Patterns:
   [preformatted-message-patterns.md](references/preformatted-message-patterns.md).
2. **Java event fields** — SLF4J 2.x fluent API (`addKeyValue`). Never add `StructuredLog` or per-call `MDC.put` for
   event data. Request-scoped MDC in filters stays as-is.
3. **Go fields** — prefer a real field API or repo helper (`WithFields`, `logfields.Format` / `Err`, …). Dropping `f`
   while keeping `log.Error("… key=%v …", key, err)` is **not** done — see
   [go-qubership-lib.md](references/go-qubership-lib.md).
4. **Stop and ask** on shared `{}` template constants and logged preformatted messages — do not guess. Choices:
   [user-decisions.md](references/user-decisions.md). After confirmation, shapes:
   [pattern-recipes.md](references/pattern-recipes.md).
5. **API / throw text** — when a string is also used for `Response.entity`, DTO error fields, or exception detail, keep
   that string unchanged; structure **only** the log line (same variable in `setMessage` when message is conditional).
6. **Do not claim done** while user-decision rows are open, or while `StructuredLog` / templating constants / Go residual
   printf diagnostics remain. A component is **not** `migrated` while any completion gate is FAIL or PARTIAL (see
   [migration-report-template.md](references/migration-report-template.md) § Status rules).

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

## Required outcome

- Useful diagnostic data in **structured fields**, not only inside `message`.
- `message` human-readable; correlation fields preserved (`request_id`, `tenant_id`, trace/span, `logType`).
- Every remaining formatted / variable-message call site inventoried or migrated (or explicit no-action / blocked).
- **Java:** per-log fields via SLF4J 2.x fluent API (`addKeyValue`) — not per-call MDC wrappers.

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
8. **Implement** in small batches — build after each batch.
9. **Re-inventory** — no unaccounted formatted / preformatted / text-block / residual-printf calls.
10. **Self-check** (below) then full [completion-gates.md](references/completion-gates.md).
11. **Smoke** — [smoke-validation.md](references/smoke-validation.md).
12. **Write report** — stage = `migrate`; status rules in
    [migration-report-template.md](references/migration-report-template.md); exclude from product PR unless requested.
13. **Propose skill updates** in the APM package source, not `.agents/skills` copies.

## Self-check (before claiming done)

Run against production sources (adjust paths). Failures must be fixed or listed as blocked / user-decision with counts.

```bash
# Java — forbidden helper / per-call MDC for event fields
grep -rn 'StructuredLog\|MDC\.put' --include='*.java' src/main/java || true

# Java — shared {} still used as templates (misleading zero)
grep -rnE 'WARNING_MESSAGE|MESSAGE_[A-Z_]+\s*=\s*".*\{}' --include='*.java' src/main/java || true

# Java — unreviewed preformatted log sites
grep -rnE 'log\.(warn|error|debug|info)\((message|msg|aggregatedError|errorMsg|warn|e\.getMessage)' \
  --include='*.java' src/main/java || true

# Java — text-block logs (same-line {} grep misses these; open each hit for {})
grep -rnE 'log\.(info|debug|warn|error|trace)\("""' --include='*.java' src/main/java || true

# Java — codemod field-name residue (polish required; target 0)
grep -rnE 'addKeyValue\("[^"]*(_get_|_stream_|e_get_message)' --include='*.java' src/main/java || true
grep -rn '"arg[0-9]\+"' --include='*.java' src/main/java || true

# Go — include Trace; exclude _test.go / commented lines in review
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)f\(' --include='*.go' . || true

# Go — residual diagnostic format verbs after dropping f (incomplete).
# Same-line check; "%s" + field helper is OK. If any hit: review that whole file for
# multi-line concatenations with the same pattern.
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)(C)?\(.*%[vTdoxXefg]' --include='*.go' . || true
```

Then spot-check field names and run [completion-gates.md](references/completion-gates.md).
Go: every residual-verb hit must become a field API / repo helper, or be listed blocked — see
[go-qubership-lib.md](references/go-qubership-lib.md).
Java: text-block `{}` and `_get_`/`_stream_` keys block `migrated` — see completion-gates §3–§4.1.

## Monorepos

One component at a time; update ledger before stopping.

## Definition of done

[completion-gates.md](references/completion-gates.md): build, integrity, pattern + semantic gates, smoke NDJSON valid.
Self-check above is clean or every hit is accounted for in the report.
