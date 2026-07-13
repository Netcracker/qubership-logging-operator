# Eval protocol (skill-creator alignment)

This package follows [skill-creator](https://github.com/anthropics/skills/tree/main/skills/skill-creator) iteration
practice. Workspace: `../qubership-ndjson-logging-migration-workspace/`.

The **published skills** have no bundled scripts — completion gates are manual:

- **Stage 1 — Enable:** [enable-gates.md](.apm/skills/qubership-ndjson-logging-enable/references/enable-gates.md)
- **Stage 2 — Migrate:** [completion-gates.md](.apm/skills/qubership-ndjson-logging-migrate/references/completion-gates.md)

Eval-only helpers live under `../qubership-ndjson-logging-migration-workspace/scripts/`.

## Two-stage eval model

Run **stage 1 (enable)** before **stage 2 (migrate)**. Migrate evals assume JSON envelope is already configured.

| Stage | Skill | Package evals | Objective checks | Workspace iteration |
| ----- | ----- | ------------- | ---------------- | ------------------- |
| Enable | `qubership-ndjson-logging-enable` | `evals/evals-enable.json` | `evals/objective_checks-enable.json` | `iteration-7-enable/` |
| Migrate | `qubership-ndjson-logging-migrate` | `evals/evals-migrate.json` | `evals/objective_checks-migrate.json` | `iteration-8-migrate/` |

Index: `evals/evals.json` lists both stages and skill paths.

## Eval definitions

Each stage file follows skill-creator shape: top-level `skill_name` + `evals[]` with `prompt`, `expected_output`,
`expectations` (process + outcome).

## Per iteration (skill-creator Steps 1–5)

1. **Spawn runs in one turn** — for each eval, launch **with-skill** and **without-skill** subagents in parallel.
2. **Executor isolation** — worktree + natural prompt only. Executors must NOT read:
   - `iteration-*/grading/`, `acceptance-criteria.json`, `evals/evals*.json`
   - Prior `**/outputs/**`, `vault/`, iteration READMEs
   - Sibling worktrees or primary repo checkouts
3. **Capture timing** — on each subagent completion, write `timing.json` (`total_tokens`, `duration_ms`) into the run directory.
4. **Grade** — process expectations via grader; objective checks via workspace scripts (heuristic supplement only):

   ```bash
   WS="../qubership-ndjson-logging-migration-workspace"

   # Stage 1 enable
   python3 "$WS/scripts/check_enable_gates.py" <worktree> [--java-path ...] [--go-path ...]
   python3 "$WS/iteration-7-enable/scripts/grade_objective.py" with_skill go-log-exporter

   # Stage 2 migrate
   python3 "$WS/scripts/check_migration_gates.py" <worktree> [--java-path ...] [--go-path ...]
   python3 "$WS/scripts/validate_ndjson_line.py" sample.log
   python3 "$WS/iteration-8-migrate/scripts/grade_objective.py" with_skill java-dbaas-monorepo
   ```

5. **Aggregate** — prefer skill-creator `scripts/aggregate_benchmark.py` when available (pass the **stage skill name**):

   ```bash
   python -m scripts.aggregate_benchmark \
     <workspace>/iteration-7-enable \
     --skill-name qubership-ndjson-logging-enable

   python -m scripts.aggregate_benchmark \
     <workspace>/iteration-8-migrate \
     --skill-name qubership-ndjson-logging-migrate
   ```

6. **Viewer** — `eval-viewer/generate_review.py` before revising the skill from results.
7. **Feedback** — read `feedback.json`; improve skill; rerun into `iteration-(N+1)/`.

## Discriminating assertions

**Enable:** baseline should fail `check_enable_gates.py` (no JSON config / Helm format / smoke evidence). With-skill should
pass or document explicit `blocked` credentials.

**Migrate:** baseline on a post-enable worktree should **fail** objective checks on dbaas (hundreds of `{}`, no operator
migration) even when process expectations pass. With-skill should pass gates or document explicit `blocked` items.

## History

Track versions in workspace `history.json`.

## Description optimization

Run skill-creator `run_loop.py` / `improve_description.py` only after an iteration shows skill lift on objective checks.
Run separately per stage skill when both are stable.
