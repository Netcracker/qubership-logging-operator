# NDJSON Migration Report State Machine

## Goal

Make the migration report a durable, per-component workflow checkpoint. An agent
should resume a migration by reading the component ledger row, rather than
reconstructing completed work, open questions, and safe next steps from prose.

This is a report-backed state machine only. It adds no parser, command wrapper,
or automatic transition validator.

## Component ledger

The existing `Deployable components` table gains these columns:

| Column           | Meaning                                                                    |
| ---------------- | -------------------------------------------------------------------------- |
| `Phase`          | Current workflow checkpoint from the phase set below.                      |
| `Status`         | High-level outcome: `pending`, `in-progress`, `blocked`, or `migrated`.    |
| `Open decisions` | Number of unresolved user decisions for this component.                    |
| `Next action`    | A specific imperative instruction that allows safe resume.                 |

The state machine is per deployable component. Components in one repository may
advance independently.

`blocked` is a status overlay, not a phase. It preserves the component's current
phase and records what must happen before the component can resume.

## Phases and transitions

```text
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
```

| From                 | To                   | Required condition                                                                               |
| -------------------- | -------------------- | ------------------------------------------------------------------------------------------------ |
| `pending`            | `discovery`          | Component is recorded in the ledger.                                                             |
| `discovery`          | `placement`          | Stack and logging path are identified.                                                           |
| `placement`          | `inventory`          | Placement probe passes.                                                                          |
| `inventory`          | `awaiting_decisions` | Candidates are classified and unresolved decisions are recorded.                                 |
| `awaiting_decisions` | `migrating`          | Every queued decision has a user choice or applies a recorded repo-wide policy.                  |
| `migrating`          | `gates`              | Every candidate is migrated, static/no-action, blocked, or decision-accounted.                   |
| `gates`              | `review`             | Build, integrity, pattern, and semantic gates pass, or a concrete validation block is recorded.  |
| `review`             | `smoke`              | The required review -> fix -> re-check loop is recorded.                                         |
| `smoke`              | `migrated`           | Smoke passes, or is environment-blocked while every other gate passes.                           |

No transition may skip a preceding phase. A `blocked` component resumes at its
recorded phase after its block clears.

## User-decision queues

### Immediate: placement failure

A placement-probe failure blocks all safe call-site migration. The agent must:

1. Leave `Phase=placement`, set `Status=blocked`, and record one open decision.
2. Ask the placement question immediately using the existing fixed option menu.
3. Wait for the user's choice, implement only that choice, and re-probe.

### Batched: inventory decisions

During `inventory`, collect these decisions in the existing report tables:

- shared Java `{}` template constants;
- logged preformatted messages;
- returned diagnostics;
- ambiguous field extraction;
- sensitive response-body log handling.

At the `inventory -> awaiting_decisions` boundary, ask for all unresolved
non-placement decisions in one grouped request. Do not enter `migrating` until
the queue is resolved, blocked, or covered by an explicit repo-wide policy.

### Batched: review discoveries

Fix clear defects during `review`. Queue genuine new semantic ambiguities as a
single review decision batch before returning to `migrating`. Repeat gates and
the review loop after implementing the user's choice.

## Documentation changes

- `migration-report-template.md` defines phases, ledger columns, transition
  guards, and example rows.
- `SKILL.md` replaces the linear workflow with phase-driven instructions and
  points to the report template as the state authority.
- `user-decisions.md` distinguishes the immediate placement decision from
  queued inventory and review decisions.

The existing gate, placement-probe, and pattern-recipe references retain their
judgment criteria. The state machine controls sequencing, not field naming or
message semantics.

## Failure handling

- Validation failures keep the component at its current phase with
  `Status=in-progress`, unless work cannot continue safely; then set
  `Status=blocked` and record the exact error.
- An unresolved user decision prevents the transition to `migrating`; it never
  permits a `migrated` status.
- A smoke environment block may permit `migrated` only under the existing
  report rule: placement and every other gate already pass.

## Verification

1. Check that the template permits two components in different phases.
2. Simulate a placement-probe FAIL: verify the report says `placement`,
   `blocked`, one open decision, and an immediate next action.
3. Simulate several inventory findings: verify they produce one batched
   decision request before migration.
4. Simulate a review ambiguity: verify clear fixes continue, while ambiguity
   re-enters the decision queue and requires gates/review again.
5. Review the resulting report for a completed component: its phase must be
   `migrated`, status `migrated`, zero open decisions, and completion evidence.
