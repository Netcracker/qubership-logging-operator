# Iteration 8 — stage 2 migrate eval

Skill: `qubership-ndjson-logging-migrate`.

Measures **call-site field extraction** and completion gates — not JSON envelope rollout (see `iteration-7-enable`).

## Prerequisite

Worktrees must be **post-enable**. Reset baseline/with-skill worktrees from `iteration-7-enable`
with-skill outputs before spawning migrate runs so both arms start from an enabled baseline.

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
python3 iteration-8-migrate/scripts/grade_objective.py with_skill go-log-exporter
python3 iteration-8-migrate/scripts/grade_objective.py without_skill java-dbaas-monorepo

# Process expectations — human or grader reads grading/expectations.json
```

### Aggregate + viewer

```bash
python -m scripts.aggregate_benchmark \
  iteration-8-migrate \
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
| Reset post-enable worktrees | Pending |
| Full 4-arm run | Pending |
| Objective grading | Pending |
| Process grading | Pending |
| Benchmark + viewer | Pending |

## Supersedes

`iteration-6/` — stale monolithic skill path and removed `log-level-response-body` eval. Keep iteration-6 for history only.
