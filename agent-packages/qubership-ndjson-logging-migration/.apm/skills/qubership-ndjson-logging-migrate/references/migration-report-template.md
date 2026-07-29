# Migration Report Template

ALWAYS create or update `.ndjson-migration-report.md` at the **root of the target worktree** during the migration run.
Use this structure; leave N/A rows explicit rather than omitting them.

## Lifecycle (not part of the product PR)

| Moment | Report in worktree?                         | Commit / upstream PR?                                                                            |
| ---------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Migration run          | **Yes** — coverage ledger and gate evidence | No — working artifact                                                                            |
| Resume across sessions | Yes — update in place                       | Untracked is fine                                                                                |
| Final product PR       | —                                           | **Exclude** `.ndjson-migration-report.md` unless the team explicitly wants an audit file in-repo |

On resume, read **Active component**, **Workflow phase**, and **Next action** before rediscovering the repository.
Repeat placement or full inventory only when the report says that phase is incomplete or a later change invalidated it.
During `implement`, **Next action** names the next batch ID, risk tier, and path/scope; “continue migration” is not enough
to resume safely.

Before opening or updating a product PR, drop the report from the commit (`git restore --staged` / omit from `git add`).
Copy it to the eval workspace or keep a local copy if you need an audit trail. Summarize completion gates and coverage in
the PR description instead.

```markdown
# NDJSON Logging Migration Report — <repo-name>

| Field | Value |
|-------|-------|
| **Run start HEAD** | `<git rev-parse HEAD at run start>` |
| **Branch** | `<branch>` |
| **Skill** | `qubership-ndjson-logging-migrate` |
| **Stage** | migrate (stage 2) |
| **Date** | YYYY-MM-DD |
| **Active component** | `<path or none>` |
| **Workflow phase** | `pending` / `discovery` / `placement` / `inventory` / `awaiting_decisions` / `migrating` / `gates` / `review` / `smoke` / `migrated` |
| **Next action** | `<one concrete action; during migrating: batch ID + tier + path/scope>` |
| **Last updated** | `YYYY-MM-DD — <short note>` |

Before updating a component row, identify its current phase and next legal
transition. Do not infer a completed phase from partial evidence such as clean
greps or a successful build.

## Deployable components

| Column | Meaning |
| --- | --- |
| `Phase` | Current workflow checkpoint from the phase set below. |
| `Status` | High-level outcome: `pending`, `in-progress`, `blocked`, or `migrated`. |
| `Open decisions` | Number of unresolved user decisions for this component. |
| `Next action` | A specific imperative instruction that allows safe resume. |

| Component | Path | Stack | Log config | Phase | Status | Open decisions | Next action |
|-----------|------|-------|------------|-------|--------|----------------|-------------|
| example-java | `service/` | Java / Quarkus | JSON enabled | placement | blocked | 1 | Await user choice: add placement infrastructure / change backend / defer / accept message-embedded fields |
| ... | ... | ... | ... | pending / discovery / placement / inventory / awaiting_decisions / migrating / gates / review / smoke / migrated | pending / in-progress / blocked / migrated | 0 | Start discovery |

The placement-failure example row records probe FAIL in **User decision — event-field placement** below.

**Placement probe outcomes:**

| User choice | Phase | Status | Next transition |
| --- | --- | --- | --- |
| Probe **PASS** | `placement` → `inventory` | `in-progress` | Enter inventory. |
| **Defer** | stays `placement` | `blocked` | End work on this component until placement is fixed later. Do **not** enter `inventory`. |
| **Accept message-embedded fields** | stays `placement` | `blocked` | End work on this component; goal unmet. Do **not** enter `inventory` or mark `migrated`. |
| **Add placement infrastructure** / **Change logging backend** / **User-provided** | stays `placement` | `in-progress` or `blocked` | Implement choice → re-probe; on PASS → `inventory`. |

Do not enter `inventory` or bulk `migrating` until the user chooses (for FAIL) and re-probe passes (when applicable).

## Component state machine

State is per deployable component. Components in one repository may advance
independently.

    pending
      -> discovery
      -> placement
      -> inventory
      -> awaiting_decisions
      -> migrating
      -> gates
      -> review
      -> smoke
      -> migrated

| From | To | Required condition |
| --- | --- | --- |
| `pending` | `discovery` | Component is recorded in the ledger. |
| `discovery` | `placement` | Stack and logging path are identified. |
| `placement` | `inventory` | Placement probe passes. |
| `inventory` | `awaiting_decisions` | Candidates are classified and unresolved decisions are recorded (`Open decisions` > 0). |
| `inventory` | `migrating` | Candidates are classified and **no** unresolved decisions remain (`Open decisions=0`). Do **not** enter `awaiting_decisions` or ask a vacuous question. |
| `awaiting_decisions` | `migrating` | Every queued decision has a user choice or applies a recorded repo-wide policy. |
| `migrating` | `gates` | Every candidate is migrated, static/no-action, blocked, or decision-accounted. |
| `gates` | `review` | Build, integrity, pattern, and semantic gates pass, or a concrete validation block is recorded. |
| `review` | `smoke` | The required review -> fix -> re-check loop is recorded. |
| `review` | `awaiting_decisions` | Review finds genuine new semantic ambiguities; queue them in one review decision batch (`Open decisions` > 0). |
| `gates` | `migrating` | A gate failure requires more migration work (fix candidates, polish fields, re-apply patterns) before re-running gates; after a review decision batch is resolved, re-enter migration work before another gate/review cycle. |
| `smoke` | `migrated` | Smoke passes, or is environment-blocked while every other gate passes. |

`blocked` is a status overlay, not a phase. Preserve the component's phase when
blocked; set `Next action` to the action that clears the block. For placement
probe FAIL, record the probe in **User decision — event-field placement** and
name the awaiting choice using the fixed option menu in
[user-decisions.md](user-decisions.md) § Event-field placement unsupported — do
not restate probe mechanics here.

Permitted transitions are only those listed in the table above — including the
zero-decision `inventory -> migrating` path when `Open decisions=0`. Do not infer
or invent other phase jumps. A component resumes at its recorded phase after its
block clears.

**Re-entry paths (only allowed backward transitions):** `review -> awaiting_decisions`
(when review queues new semantic ambiguities) and `gates -> migrating` (when gate
failure requires more migration work, or after a review decision batch is resolved,
before another gate/review cycle). No other backward or skip transitions are
permitted.

## Status rules

| Status | When allowed |
| ------ | ------------ |
| **migrated** | `Phase=migrated`, `Status=migrated`, `Open decisions=0`, and all existing completion evidence: all gates for that component are PASS (or smoke BLOCKED with a concrete env reason **and** every other gate PASS, including placement probe PASS and **review pass** finished). |
| **in-progress** | Work has begun in a nonterminal phase (`discovery` through `smoke`); any gate FAIL/PARTIAL, open user-decision rows (excluding placement when already `blocked`), review pass not done, or polish follow-up remaining. Never use with `Phase=pending`. |
| **blocked** | Cannot proceed safely (auth, missing cluster, placement probe FAIL awaiting user choice, unsafe API change) — record exact error; preserve phase; set `Next action` to the choice or fix that clears the block (see **User decision — event-field placement** or **Blocked validation** as appropriate). |
| **pending** | `Phase=pending` and `Status=pending` only — component is recorded in the ledger but work has not started. Do not use `in-progress` while `Phase=pending`. |

**Do not** mark a component `migrated` while any non-smoke completion-gate row is **FAIL** or **PARTIAL** (including
field-name polish, placement below L1, or **review pass** skipped). Prefer `in-progress` and list the follow-up
(e.g. “polish 200 `_get_` keys”). L2/L3 smoke may stay `UNAVAILABLE` or `BLOCKED` only when every non-smoke gate
passes and the report records exact L1 evidence plus the higher-fidelity blocker.

Stage 1 envelope-only enablement may leave diagnostics inside `message`; that is
not a Stage 2 completion failure for the whole repo. Stage 2 component migration
does **not** accept a component-wide placement-probe failure as `migrated` — keep
`Phase=placement`, `Status=blocked`. An individually accepted message-embedded
diagnostic at a single call site is a completed migrated log-entry exception —
record it in the decision tables.

**Greps clean ≠ migrated** if diagnostics are still only inside `message` at
sites that were not individually accepted (unqueryable) — see [SKILL.md](../SKILL.md)
§ Goal. Fluent/`addKeyValue` call sites with placement probe FAIL are **not** `migrated`.

## Progress — `<active component>`

Update this section after each batch so a fresh session can resume without reading prior chat history.

| Batch | Risk tier | Scope | Build / review evidence | State |
|-------|-----------|-------|-------------------------|-------|
| 1 | R1 / R2 | `<path or glob>` | `<command result; spot-check or diff review>` | pending / in-progress / done / blocked |

## Completion gates

Run manual greps and builds from [completion-gates.md](completion-gates.md) per component.

| Gate | Command / check | Before | Result / blocker | PASS |
|------|-----------------|--------|------------------|------|
| Placement probe | L1+ test/command; application logger → final configured sink | | level, top-level keys, validator / BLOCKED error | |
| Java compile | `mvn -pl <module> compile` | | exit 0 / BLOCKED | |
| Go build | `GOWORK=off go build ./...` | | exit 0 | |
| Java `{}` inline | smell-checks.sh J2 + J5 hits opened | | 0 | |
| Java field names | spot-check + smell-checks.sh J6a/J6b | | OK (0 residue) | |
| Java event fields | manual: fluent API + JSON top-level; no new MDC wrapper | | OK | |
| Go `log.*f` (production) | smell-checks.sh G1 | | 0 | |
| Go residual printf | smell-checks.sh G2 | | 0 | |
| Throwables | manual sweep | | fixed | |
| Integrity | git diff review | | no stray deletions | |
| Review pass | SKILL.md § Review pass — fix + re-check | | done (note fixes) | |
| Smoke NDJSON | existing L2/L3 path; otherwise L1 evidence | | PASS / UNAVAILABLE / BLOCKED + higher-fidelity blocker | |

## Runtime evidence

Use the highest already-runnable level. L0 is direct formatter/helper testing and never placement evidence. L1
initializes the real application-facing logger, lifecycle, configured handler/appender and encoder/formatter, and
production JSON settings, then captures final rendered sink output. L2 is packaged process stdout. L3 is an existing
practical deployment or integration log path. Do not create complex deployment infrastructure solely for this
evidence. When L1 is the highest achieved level, record `UNAVAILABLE` or `BLOCKED` for L2/L3 and state the concrete
higher-fidelity reason.

| Component | Achieved level | Command / test | Result | Validator | Higher-fidelity blocker |
| --------- | -------------- | -------------- | ------ | --------- | ----------------------- |
| | L0 / L1 / L2 / L3 | | exact parse, required aliases, top-level diagnostics, readable message | existing parser/test assertion | L2/L3 UNAVAILABLE or BLOCKED reason |

For placement PASS, `Achieved level` must be L1 or higher. Record the chosen diagnostic keys/values, including simple,
quoted/whitespace, braced/map, and parenthesized/object values. The result must show that diagnostics are top-level and
not only a leading `key=value` message prefix. Name the already available JSON parser or component test assertion used;
never require or install a runtime solely for validation.

## User decision — event-field placement

| Component | Probe result | Recommended | User choice | Re-probe |
|-----------|--------------|-------------|-------------|----------|
| | PASS / FAIL / N/A | | | |

## User decision — logged preformatted messages

| Pattern | Count | Example files | Decision |
|---------|-------|---------------|----------|
| log.warn(message) | | | structure / prose-only / blocked |

## User decision — returned diagnostics

| Pattern | Count | Example files | Decision |
|---------|-------|---------------|----------|
| fmt.Errorf with embedded fields | | | keep at boundary / typed error / blocked |

## Blocked validation

| Component | Gate / evidence level | Command / test | Error or unavailable reason |
| --------- | --------------------- | -------------- | --------------------------- |
| | | | |

## Validation commands

| Component | Evidence level | Command / test | Result | Higher-fidelity blocker |
| --------- | -------------- | -------------- | ------ | ----------------------- |
| | | | | |

## Lessons (target-specific)

1. ...
```

Record **blocked** with the exact error when Maven or private registry auth prevents compile — do not mark the Java
component migrated-complete. Same for FAIL/PARTIAL gates (field polish, text-block `{}`, residual printf).
