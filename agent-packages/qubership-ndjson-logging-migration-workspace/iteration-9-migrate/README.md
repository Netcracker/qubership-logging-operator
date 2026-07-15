# Iteration 9 — stage 2 migrate eval

Skill: `qubership-ndjson-logging-migrate` (fluent API, semantic field names, immediate user-decision stops).

Measures **call-site field extraction** and completion gates — not JSON envelope rollout (see `iteration-7-enable`).

## Prerequisite — shared stage-1 with-skill baseline

Both arms start from the **iteration-7 with-skill commit** (JSON envelope enabled). Reset **baseline** and
**with-skill** worktrees to the same `enable_with_skill_commit` before each migrate run:

```bash
SHA=<enable_with_skill_commit from config.json>
for wt in dbaas-baseline dbaas-with-skill; do
  git -C worktrees/$wt checkout --force "$SHA"
  git -C worktrees/$wt clean -fd
done
```

Then run migrate: `without_skill` edits `*-baseline`, `with_skill` edits `*-with-skill`. Only the migrate skill
differs — not the starting tree.

| Eval | Stage-1 with-skill commit | Status |
| ---- | ------------------------- | ------ |
| java-dbaas-monorepo | `78a11353` on `feat/ndjson-logging-enable` | Frozen |
| go-log-exporter | `7af2cb5` on `feat/ndjson-logging-enable` | Frozen |

## Protocol

Same isolation as [EVAL_PROTOCOL.md](../../qubership-ndjson-logging-migration/EVAL_PROTOCOL.md), with these names:

| Concept | Location |
| ------- | -------- |
| Executor ticket | `prompts/migrate-*.md` — stage-2 wording + post-enable assumption |
| Process rubric | `grading/expectations.json` (executors must **not** read) |
| Package evals | `../../qubership-ndjson-logging-migration/evals/evals-migrate.json` |
| Objective supplement | `scripts/check_migration_gates.py` via `scripts/grade_objective.py` |

### Executor rules

- Worktree path from `config.json` only
- Ticket from `prompts/<name>.md` only
- Do **not** read: `grading/`, `iteration-*/`, `vault/`, workspace `README.md`, sibling worktrees, eval scripts

### Spawn (one turn per eval)

| Arm | Skill | Output dir |
| --- | ----- | ---------- |
| with_skill | `qubership-ndjson-logging-migrate` | `runs/<eval>/with_skill/outputs/` |
| without_skill | No skill | `runs/<eval>/without_skill/outputs/` |

Work happens in the configured worktree; copy transcripts/reports into `outputs/` when the run completes.

### Grade (after both arms)

```bash
# Objective (heuristic)
python3 iteration-9-migrate/scripts/grade_objective.py with_skill go-log-exporter
python3 iteration-9-migrate/scripts/grade_objective.py without_skill java-dbaas-monorepo

# Process expectations — human or grader reads grading/expectations.json
```

### Aggregate + viewer

```bash
python -m scripts.aggregate_benchmark \
  iteration-9-migrate \
  --skill-name qubership-ndjson-logging-migrate
```

## Evals

| Name | Worktrees | Components |
| ---- | --------- | ---------- |
| `go-log-exporter` | `log-exporter-baseline` / `log-exporter-with-skill` | 1× Go |
| `java-dbaas-monorepo` | `dbaas-baseline` / `dbaas-with-skill` | Java aggregator + Go operator |

## Status

| Phase | Status |
| ----- | ------ |
| Scaffold | Done |
| Skill commit (pre-iter-9) | Pending |
| Reset post-enable worktrees | Pending |
| Full 4-arm run | Pending |
| Objective grading | Pending |
| Process grading | Pending |

## Parent

`iteration-8-migrate` — prior migrate skill (MDC wrapper playbook). Iteration 9 tests fluent-API + semantic-field revisions.
