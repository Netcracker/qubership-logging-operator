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

- **Win:** diagnostic values that are useful to filter on become named JSON fields; **`message` keeps the same meaning**
  as before (what happened); correlation (`request_id`, `tenant_id`, trace/span, `logType`) stays intact.
- **Lose:** rearranging call sites only so inventory greps go to zero while diagnostics remain buried in `message` —
  including `fmt.Sprintf(…)` / string build then `log.X("%s", msg)`, or drop-`f` with `key=%v` still in the format string.
- **Also lose:** changing what the log says (wrong level summary, invented failure text) or splitting a composed value
  (URL/path/query) into path segments when operators need the whole string.

Greps and gates are **smell checks** that the goal may be unmet. Clean greps alone never mean `migrated`.

## Hard rules (read before any edit)

1. **Serve the goal** — every edit should make diagnostics queryable as fields (or record an explicit no-action / blocked
   reason). Do not ship cosmetic rewrites (`log.error(msg)` → `log.atError().setMessage(msg).log()` with no fields, or
   greps-only dodges).
2. **Placement probe before bulk migrate** — for **every** stack/language component, prove the intended event-field API
   yields **top-level** JSON keys before rewriting call sites. See [placement-probe.md](references/placement-probe.md).
   On FAIL: stop and ask ([user-decisions.md](references/user-decisions.md) § Event-field placement unsupported) — do
   **not** guess or implement a placement fix until the user chooses (recommended + alternatives + user-provided).
3. **Inventory first** — run [scripts/smell-checks.sh](scripts/smell-checks.sh); check meanings in
   [preformatted-message-patterns.md](references/preformatted-message-patterns.md). Inventory finds candidates;
   the goal decides what “fixed” means.
4. **Java event fields** — SLF4J 2.x fluent API (`addKeyValue`) so event data lands in JSON for search. Never add
   `StructuredLog` or per-call `MDC.put` for event data. Request-scoped MDC in filters stays as-is. Fluent call sites
   alone are insufficient if the placement probe FAIL (bridge/formatter gap).
5. **Go fields** — prefer a real field API or repo helper so keys appear at JSON top level — see
   [go-qubership-lib.md](references/go-qubership-lib.md). Still require a placement probe.
6. **Stop and ask** on shared `{}` template constants, logged preformatted messages, placement-probe FAIL, and
   **ambiguous meaning / field splits** (composed URL/path, unclear what to extract) — do not guess. Choices:
   [user-decisions.md](references/user-decisions.md). After confirmation, shapes:
   [pattern-recipes.md](references/pattern-recipes.md).
7. **Preserve log meaning** — keep the same event intent and a faithful `message`. Extract only values useful to filter
   on alone (not fixed allowed-value enums/literals; not over-split path segments). When unsure → stop and ask (§ above).
8. **API / throw text** — when a string is also used for `Response.entity`, DTO error fields, or exception detail, keep
   that string unchanged; structure **only** the log line (same variable in `setMessage` when message is conditional).
9. **Do not claim done** while the goal is unmet: diagnostics still only in `message`, open user-decision rows,
   placement probe FAIL, `StructuredLog` / templating constants, any completion gate FAIL/PARTIAL, or the **review
   pass** not finished — see [migration-report-template.md](references/migration-report-template.md) § Status rules.
10. **Preserve indentation** — broken whitespace that fails checkstyle / spotless / gofmt is a gate failure. Practice:
    [java-quarkus.md](references/java-quarkus.md) § Indentation.
11. **Advance one component through recorded phases** — update `Phase`,
    `Status`, `Open decisions`, and `Next action` in the migration report at
    each transition. Do not skip a phase or mark `migrated` unless the report
    transition guard permits it — see migration-report-template.md § Component
    state machine.

## Reference map

| When                 | Read                                                                                                   |
| -------------------- | ------------------------------------------------------------------------------------------------------ |
| Placement probe      | [placement-probe.md](references/placement-probe.md) — before bulk call-site edits (all stacks)         |
| Smell / inventory checks | [scripts/smell-checks.sh](scripts/smell-checks.sh) + [preformatted-message-patterns.md](references/preformatted-message-patterns.md) |
| User choice          | [user-decisions.md](references/user-decisions.md)                                                      |
| Pattern recipes      | [pattern-recipes.md](references/pattern-recipes.md) — after user confirms a decision                   |
| Stack implementation | [java-quarkus.md](references/java-quarkus.md) or [go-qubership-lib.md](references/go-qubership-lib.md) |
| Cross-cutting rules  | [coding-approaches.md](references/coding-approaches.md)                                                |
| Field naming contract | [schema.md](references/schema.md) — when mapping fields                                                |
| Before claiming done | [completion-gates.md](references/completion-gates.md) + § Review pass (below)                          |
| Report               | [migration-report-template.md](references/migration-report-template.md)                                |
| Smoke                | [smoke-validation.md](references/smoke-validation.md)                                                  |
| Pitfalls             | [corner-cases.md](references/corner-cases.md)                                                          |
| Background           | [evidence.md](references/evidence.md)                                                                  |

## Workflow

Advance **one deployable component** at a time. At every phase below, update that
component's ledger row (`Phase`, `Status`, `Open decisions`, `Next action`) per
[migration-report-template.md](references/migration-report-template.md).

1. `discovery` — stage 1 confirmation, repo-root discovery, stack classification.
   - Confirm stage 1 — JSON smoke passed or document config blocker. Envelope ≠ event-field placement.
   - **Repo-root discovery** — coverage ledger for all runtime components.
   - **Classify stack** → [java-quarkus.md](references/java-quarkus.md) or [go-qubership-lib.md](references/go-qubership-lib.md).
2. `placement` — run probe; on FAIL block and ask immediately.
   - **Placement probe** — [placement-probe.md](references/placement-probe.md) for that component (all languages); on
     FAIL apply hard rule 2 (stop, ask immediately, re-probe until PASS or leave `blocked` / `in-progress`).
     `blocked` is a status overlay — preserve `Phase=placement`.
3. `inventory` — run smell checks and classify candidates.
   - **Inventory** — run [scripts/smell-checks.sh](scripts/smell-checks.sh)
     ([preformatted-message-patterns.md](references/preformatted-message-patterns.md)).
   - **Classify** sites: `migrate`, `static/no action`, `needs user decision`, `blocked`.
   - **Queue decisions** — record unresolved shared `{}` templates, logged preformatted messages, returned diagnostics
     ([user-decisions.md](references/user-decisions.md) § Returned diagnostics), ambiguous extraction, and sensitive
     response-body choices in the report. Do **not** ask individually; count them for the inventory batch.
4. `awaiting_decisions` — present one inventory batch; apply explicit policy.
   - **User decisions** — present one grouped question for all queued inventory items per
     [user-decisions.md](references/user-decisions.md) § Inventory decision batch. After confirmation, read
     [pattern-recipes.md](references/pattern-recipes.md) before editing those sites.
5. `migrating` — map fields and implement small batches.
   - **Map fields** — [schema.md](references/schema.md) + stack playbook + [coding-approaches.md](references/coding-approaches.md).
   - **Implement** in small batches — build after each batch; spot-check that new fields are queryable, not only that
     greps shrank.
6. `gates` — re-inventory, smell checks, completion gates.
   - **Re-inventory** — re-run [scripts/smell-checks.sh](scripts/smell-checks.sh); no unaccounted candidates.
   - **Smell checks** (below) then full [completion-gates.md](references/completion-gates.md).
7. `review` — required review -> fix -> re-check loop.
   - **Review pass (blocking)** — see § Review pass below. Fix clear defects without asking. For genuine new semantic
     ambiguities, queue them in one **review decision batch**, return to `awaiting_decisions`, and after the choice
     re-enter `migrating` followed by `gates` and this review loop
     ([user-decisions.md](references/user-decisions.md)). Do **not** write the report or mark `migrated` until this
     finishes (or remaining hits are explicitly `blocked` / user-decision).
8. `smoke` — capture top-level diagnostic fields.
   - **Smoke** — [smoke-validation.md](references/smoke-validation.md); confirm diagnostic keys at JSON top level
     (placement probe criterion again on a real migrated line).
9. `migrated` — update the report only when the transition guard passes.
   - **Write report** — stage = `migrate`; status rules in
     [migration-report-template.md](references/migration-report-template.md); mark `migrated` only when the transition
     guard permits. Exclude from product PR unless requested.
   - **Propose skill updates** in the APM package source, not `.agents/skills` copies.

## Smell checks (before claiming done)

Run [scripts/smell-checks.sh](scripts/smell-checks.sh) against each component root — check meanings, production scopes,
and misleading zeros: [preformatted-message-patterns.md](references/preformatted-message-patterns.md). Hits suggest the
**goal** is unmet — fix toward queryable fields, or list as blocked / user-decision with counts. **Clean checks are not
sufficient** (e.g. `fmt.Sprintf` then `log.X("%s", msg)` with diagnostics inside `msg` still fails the goal — no grep
catches it). Spot-check field names and JSON placement after the script run.

Then run [completion-gates.md](references/completion-gates.md). Semantic + smoke gates decide `migrated`, not pattern
counts alone — see [go-qubership-lib.md](references/go-qubership-lib.md) and completion-gates §3–§4.1.

## Review pass (blocking)

After smells + completion gates look green, **do not** stop. Greps clean still miss indent breaks, wrong key↔value,
no-op fluent wraps, and leftover Go printf.

1. **Diff review** — read the full migrate diff for the component (not only grep hits). Judge against existing hard
   rules and [completion-gates.md](references/completion-gates.md) (especially integrity §2.5, semantic §4). Do not
   invent a second checklist.
2. **Fix** clear violations in place (indent, field names, residual format verbs, `setMessage` without fields when
   diagnostics exist in scope). Stop and ask only for true ambiguities ([user-decisions.md](references/user-decisions.md)).
3. **Re-check** — re-run smell greps + any gate that the fixes could affect (build / integrity / semantic spot-check).
4. **One loop** — one review→fix→re-check cycle is required; a second only if the re-check still finds clear defects.
   Then proceed to smoke and the report. Record in the report that the review pass ran (and what was fixed, briefly).

## Monorepos

One component at a time; update ledger before stopping.

On resume, read each component's ledger row first. Continue only the recorded
`Next action`; do not restart completed phases. Components may be at different
phases.

## Definition of done

The **goal** is met for each component: placement probe PASS, queryable fields, readable `message`, correlation
preserved; build/integrity OK; smell checks clean or accounted for; **review pass** finished;
[completion-gates.md](references/completion-gates.md) PASS (or blocked with reason). Clean greps without queryable
fields is **not** done.
