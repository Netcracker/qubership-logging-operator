# Eval protocol (skill-creator alignment)

This package follows [skill-creator](https://github.com/anthropics/skills/tree/main/skills/skill-creator) iteration practice. Workspace: `../qubership-ndjson-logging-migration-workspace/`.

The **published skill** has no bundled scripts — completion gates are manual ([completion-gates.md](.apm/skills/qubership-ndjson-logging-migration/references/completion-gates.md)). Eval-only helpers live under `../qubership-ndjson-logging-migration-workspace/scripts/`.

## Eval definitions

- `evals/evals.json` — prompts + human-readable `expectations` (process + outcome)
- `evals/objective_checks.json` — programmatic supplements (workspace scripts + grep)

## Per iteration (skill-creator Steps 1–5)

1. **Spawn runs in one turn** — for each eval, launch **with-skill** and **without-skill** subagents in parallel.
2. **Executor isolation** — worktree + natural prompt only. Executors must NOT read:
   - `iteration-*/grading/`, `acceptance-criteria.json`, `evals/evals.json`
   - Prior `**/outputs/**`, `vault/`, iteration READMEs
   - Sibling worktrees or primary repo checkouts
3. **Capture timing** — on each subagent completion, write `timing.json` (`total_tokens`, `duration_ms`) into the run directory.
4. **Grade** — process expectations via grader; objective checks via workspace scripts (heuristic supplement only):
   ```bash
   WS="../qubership-ndjson-logging-migration-workspace"
   python3 "$WS/scripts/check_migration_gates.py" <worktree> [--java-path ...] [--go-path ...]
   python3 "$WS/scripts/validate_ndjson_line.py" sample.log
   ```
   Or `iteration-6/scripts/grade_objective.py <arm> <eval-name>`.
5. **Aggregate** — prefer skill-creator `scripts/aggregate_benchmark.py` when available:
   ```bash
   python -m scripts.aggregate_benchmark <workspace>/iteration-N --skill-name qubership-ndjson-logging-migration
   ```
6. **Viewer** — `eval-viewer/generate_review.py` before revising the skill from results.
7. **Feedback** — read `feedback.json`; improve skill; rerun into `iteration-(N+1)/`.

## Discriminating assertions

Baseline should **fail** objective checks on dbaas (hundreds of `{}`, no operator migration) even when process expectations pass. With-skill should pass gates or document explicit `blocked` credentials.

## History

Track versions in workspace `history.json`.

## Description optimization

Run skill-creator `run_loop.py` / `improve_description.py` only after an iteration shows skill lift on objective checks.
