# Migration Report Template

ALWAYS create or update `.ndjson-migration-report.md` at the **root of the target worktree** during the migration run.
Use this structure; leave N/A rows explicit rather than omitting them.

## Lifecycle (not part of the product PR)

| Phase                  | Report in worktree?                         | Commit / upstream PR?                                                                            |
| ---------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Migration run          | **Yes** — coverage ledger and gate evidence | No — working artifact                                                                            |
| Resume across sessions | Yes — update in place                       | Untracked is fine                                                                                |
| Final product PR       | —                                           | **Exclude** `.ndjson-migration-report.md` unless the team explicitly wants an audit file in-repo |

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

Before updating a component row, identify its current phase and next legal
transition. Do not infer a completed phase from partial evidence such as clean
greps or a successful build.

## Deployable components

| Component | Path | Stack | Log config | Phase | Status | Open decisions | Next action |
|-----------|------|-------|------------|-------|--------|----------------|-------------|
| ... | ... | ... | ... | pending / discovery / placement / inventory / awaiting_decisions / migrating / gates / review / smoke / migrated | pending / in-progress / blocked / migrated | 0 | Start discovery |

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
| `inventory` | `awaiting_decisions` | Candidates are classified and unresolved decisions are recorded. |
| `awaiting_decisions` | `migrating` | Every queued decision has a user choice or applies a recorded repo-wide policy. |
| `migrating` | `gates` | Every candidate is migrated, static/no-action, blocked, or decision-accounted. |
| `gates` | `review` | Build, integrity, pattern, and semantic gates pass, or a concrete validation block is recorded. |
| `review` | `smoke` | The required review -> fix -> re-check loop is recorded. |
| `smoke` | `migrated` | Smoke passes, or is environment-blocked while every other gate passes. |

`blocked` is a status overlay, not a phase. Preserve the component's phase when
blocked; set `Next action` to the action that clears the block.

No transition may skip a preceding phase. A component resumes at its recorded
phase after its block clears.

## Status rules

| Status | When allowed |
| ------ | ------------ |
| **migrated** | `Phase=migrated`, `Status=migrated`, `Open decisions=0`, and all existing completion evidence: all gates for that component are PASS (or smoke BLOCKED with a concrete env reason **and** every other gate PASS, including placement probe PASS and **review pass** finished). |
| **in-progress** | Any nonterminal phase; work underway; any gate FAIL/PARTIAL, open user-decision rows (including placement), review pass not done, or polish follow-up remaining. |
| **blocked** | Cannot proceed (auth, missing cluster, placement probe FAIL awaiting user choice, unsafe API change) — record exact error. |
| **pending** | Not started. |

**Do not** mark a component `migrated` while any completion-gate row is **FAIL** or **PARTIAL** (including field-name
polish, **placement probe**, or **review pass** skipped). Prefer `in-progress` and list the follow-up (e.g. “polish 200 `_get_` keys”). Smoke may
stay BLOCKED without a cluster; that alone does not force `migrated` if other gates are incomplete.

**Greps clean ≠ migrated** if diagnostics are still only inside `message` (unqueryable) — see [SKILL.md](../SKILL.md)
§ Goal. Fluent/`addKeyValue` call sites with placement probe FAIL are **not** `migrated`.

## Completion gates

Run manual greps and builds from [completion-gates.md](completion-gates.md) per component.

| Gate | Command / check | Before | After | PASS |
|------|-----------------|--------|-------|------|
| Placement probe | see placement-probe.md | | top-level keys / BLOCKED | |
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
| Smoke NDJSON | captured stdout line → JSON with time/level/message + top-level event fields | | OK / BLOCKED | |

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

| Component | Command | Error |
|-----------|---------|-------|
| | | |

## Validation commands

| Command | Result |
|---------|--------|
| | |

## Lessons (target-specific)

1. ...
```

Record **blocked** with the exact error when Maven or private registry auth prevents compile — do not mark the Java
component migrated-complete. Same for FAIL/PARTIAL gates (field polish, text-block `{}`, residual printf).
