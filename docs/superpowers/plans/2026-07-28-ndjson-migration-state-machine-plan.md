# NDJSON Migration Report State Machine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the NDJSON migration report an explicit per-component workflow state machine and make user-question timing deterministic.

**Architecture:** The report ledger stores each component's phase, high-level status, open-decision count, and next safe action. The skill describes transition guards; `user-decisions.md` owns decision queues. No parser or transition-validation command is introduced.

**Tech Stack:** Markdown, Bash validation commands, existing APM skill package.

## Global Constraints

- Edit only the package source under `agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/`.
- State is per deployable component; components in one monorepo may use different phases.
- `blocked` remains a status overlay, not a phase.
- Placement-probe FAIL is the only immediate user question.
- Other decisions are batched at inventory or review boundaries.
- Do not introduce a report parser, lifecycle command, or automatic transition validator.

---

### Task 1: Make the migration report the state authority

**Files:**
- Modify: `agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/references/migration-report-template.md`

**Interfaces:**
- Produces: Ledger columns `Phase`, `Status`, `Open decisions`, and `Next action`.
- Produces: Phase vocabulary consumed by `SKILL.md`: `pending`, `discovery`, `placement`, `inventory`, `awaiting_decisions`, `migrating`, `gates`, `review`, `smoke`, `migrated`.

- [ ] **Step 1: Add a failing documentation scenario to the template**

Add this scenario directly above the component ledger:

```markdown
Before updating a component row, identify its current phase and next legal
transition. Do not infer a completed phase from partial evidence such as clean
greps or a successful build.
```

- [ ] **Step 2: Add the stateful component ledger**

Replace the current component header/table with:

```markdown
## Deployable components

| Component | Path | Stack | Log config | Phase | Status | Open decisions | Next action |
|-----------|------|-------|------------|-------|--------|----------------|-------------|
| ... | ... | ... | ... | pending / discovery / placement / inventory / awaiting_decisions / migrating / gates / review / smoke / migrated | pending / in-progress / blocked / migrated | 0 | Start discovery |
```

- [ ] **Step 3: Define phases, blocked overlay, and legal transitions**

Insert a `## Component state machine` section after the ledger. Include the exact phase sequence, a transition table matching the design spec, and these rules:

```markdown
`blocked` is a status overlay, not a phase. Preserve the component's phase when
blocked; set `Next action` to the action that clears the block.

No transition may skip a preceding phase. A component resumes at its recorded
phase after its block clears.
```

- [ ] **Step 4: Update status rules**

Keep the existing status meanings, but make `in-progress` apply to any nonterminal phase and state that `migrated` requires:

```markdown
`Phase=migrated`, `Status=migrated`, `Open decisions=0`, and all existing
completion evidence.
```

- [ ] **Step 5: Validate the template manually**

Run:

```bash
cd agent-packages/qubership-ndjson-logging-migration
rg -n 'Phase|Open decisions|Next action|awaiting_decisions|blocked.*overlay' \
  .apm/skills/qubership-ndjson-logging-migrate/references/migration-report-template.md
```

Expected: all four ledger fields, the `awaiting_decisions` phase, and the blocked-overlay rule appear.

- [ ] **Step 6: Commit**

```bash
git add agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/references/migration-report-template.md
git commit -m "docs: add NDJSON migration report phases"
```

### Task 2: Define immediate and batched user-decision queues

**Files:**
- Modify: `agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/references/user-decisions.md`

**Interfaces:**
- Consumes: Component phase and ledger columns from `migration-report-template.md`.
- Produces: Immediate placement procedure and inventory/review queue procedure consumed by `SKILL.md`.

- [ ] **Step 1: State the queue policy at the top of the file**

Replace the opening instruction with:

```markdown
Record every decision in the migration report's component row and decision
tables. Ask placement-probe failures immediately. Collect every other
unresolved decision during inventory or review, then present one batch at that
phase boundary.
```

- [ ] **Step 2: Make placement failure an explicit immediate transition**

In `## Event-field placement unsupported`, require:

```markdown
Set `Phase=placement`, `Status=blocked`, `Open decisions=1`, and `Next action`
to the exact user choice needed. Do not bulk-migrate call sites.
```

Retain the existing evidence, option menu, user-provided option, and re-probe requirements.

- [ ] **Step 3: Add the inventory decision-batch procedure**

Insert `## Inventory decision batch` before `## Returned diagnostics`:

```markdown
During `inventory`, record unresolved shared `{}` templates, logged
preformatted messages, returned diagnostics, ambiguous extraction, and
sensitive response-body choices. When inventory completes, set
`Phase=awaiting_decisions`, set `Open decisions` to the unresolved count, and
ask one grouped question. Do not enter `migrating` until every row is resolved,
blocked, or covered by an explicit repo-wide policy.
```

- [ ] **Step 4: Add the review decision-batch procedure**

Insert this rule before `## Semantic field names`:

```markdown
During review, fix clear defects without asking. For a genuine new semantic
ambiguity, add it to one review decision batch, return to
`awaiting_decisions`, and after the choice re-enter `migrating` followed by
gates and the required review loop.
```

- [ ] **Step 5: Convert later “ask immediately” wording**

For shared Java `{}` template constants, preserve the requirement to ask before
editing those call sites, but replace “ask immediately” with “queue during
inventory and ask at the inventory decision boundary.” Placement is the only
immediate category.

- [ ] **Step 6: Validate policy consistency**

Run:

```bash
cd agent-packages/qubership-ndjson-logging-migration
rg -n 'ask immediately|awaiting_decisions|Open decisions|review decision' \
  .apm/skills/qubership-ndjson-logging-migrate/references/user-decisions.md
```

Expected: “ask immediately” appears only in the placement section; both queue boundaries and ledger fields appear.

- [ ] **Step 7: Commit**

```bash
git add agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/references/user-decisions.md
git commit -m "docs: batch NDJSON migration decisions"
```

### Task 3: Make the primary workflow phase-driven

**Files:**
- Modify: `agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/SKILL.md`

**Interfaces:**
- Consumes: Phase and transition definitions from `migration-report-template.md`.
- Consumes: Immediate/batched question policy from `user-decisions.md`.
- Produces: The agent's primary execution workflow.

- [ ] **Step 1: Add the state-machine rule to Hard rules**

Add:

```markdown
11. **Advance one component through recorded phases** — update `Phase`,
    `Status`, `Open decisions`, and `Next action` in the migration report at
    each transition. Do not skip a phase or mark `migrated` unless the report
    transition guard permits it — see migration-report-template.md § Component
    state machine.
```

- [ ] **Step 2: Replace the linear workflow with phase-driven steps**

Keep the existing technical operations, but group them under these headings in
this order:

```markdown
1. `discovery` — stage 1 confirmation, repo-root discovery, stack classification.
2. `placement` — run probe; on FAIL block and ask immediately.
3. `inventory` — run smell checks and classify candidates.
4. `awaiting_decisions` — present one inventory batch; apply explicit policy.
5. `migrating` — map fields and implement small batches.
6. `gates` — re-inventory, smell checks, completion gates.
7. `review` — required review -> fix -> re-check loop.
8. `smoke` — capture top-level diagnostic fields.
9. `migrated` — update the report only when the transition guard passes.
```

At every heading, instruct the agent to update the component ledger.

- [ ] **Step 3: Add resume instructions**

Under `## Monorepos`, add:

```markdown
On resume, read each component's ledger row first. Continue only the recorded
`Next action`; do not restart completed phases. Components may be at different
phases.
```

- [ ] **Step 4: Validate cross-reference and state vocabulary**

Run:

```bash
cd agent-packages/qubership-ndjson-logging-migration
rg -n 'awaiting_decisions|Open decisions|Next action|placement.*immediately|inventory batch' \
  .apm/skills/qubership-ndjson-logging-migrate/SKILL.md \
  .apm/skills/qubership-ndjson-logging-migrate/references/migration-report-template.md \
  .apm/skills/qubership-ndjson-logging-migrate/references/user-decisions.md
```

Expected: the same phase and decision terminology appears in all three files.

- [ ] **Step 5: Commit**

```bash
git add agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/SKILL.md
git commit -m "docs: guide NDJSON migration by component phase"
```

### Task 4: Run report-state scenarios and update installed copy

**Files:**
- Modify: `.agents/skills/qubership-ndjson-logging-migrate/SKILL.md`
- Modify: `.agents/skills/qubership-ndjson-logging-migrate/references/migration-report-template.md`
- Modify: `.agents/skills/qubership-ndjson-logging-migrate/references/user-decisions.md`

**Interfaces:**
- Consumes: Final package-source skill from Tasks 1–3.
- Produces: Installed skill identical to the package source.

- [ ] **Step 1: Verify a placement-failure report row**

Create a temporary report fixture outside the product tree containing:

```markdown
| example-java | `service/` | Java / Quarkus | JSON enabled | placement | blocked | 1 | Await user choice: add placement infrastructure / change backend / defer / accept message-embedded fields |
```

Verify it represents the mandatory immediate question without entering
`inventory` or `migrating`.

- [ ] **Step 2: Verify an inventory batch report row**

Create a temporary report fixture containing:

```markdown
| example-go | `operator/` | Go | JSON enabled | awaiting_decisions | in-progress | 3 | Present inventory batch: preformatted messages, returned diagnostics, response body |
```

Verify one grouped request can resolve the queue before `migrating`.

- [ ] **Step 3: Verify a completed report row**

Create a temporary report fixture containing:

```markdown
| example-done | `app/` | Java | JSON enabled | migrated | migrated | 0 | Complete |
```

Verify all completion gates, review pass, and smoke evidence would be required
in the report before this row is valid.

- [ ] **Step 4: Synchronize the installed skill**

Run:

```bash
rsync -a --delete \
  agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate/ \
  ../.agents/skills/qubership-ndjson-logging-migrate/
diff -rq \
  agent-packages/qubership-ndjson-logging-migration/.apm/skills/qubership-ndjson-logging-migrate \
  ../.agents/skills/qubership-ndjson-logging-migrate
```

Expected: `diff -rq` prints no differences.

Do not commit the installed `.agents` copy from this repository; it belongs to
the parent workspace's installation state. Task 3 commits the final
package-source change.
