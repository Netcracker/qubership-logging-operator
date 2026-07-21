# Iteration 7 — stage 1 enable eval

Skill: `qubership-ndjson-logging-enable`.

Measures **JSON envelope rollout** (config/Helm/`LOG_FORMAT`, smoke) — not call-site field extraction.
Stage 2 uses `iteration-8-migrate` and `evals/evals-migrate.json`.

## Protocol

Same isolation as [EVAL_PROTOCOL.md](../../qubership-ndjson-logging-migration/EVAL_PROTOCOL.md), with these names:

| Concept              | Location                                                              |
| -------------------- | --------------------------------------------------------------------- |
| Executor ticket      | `prompts/*.md` — minimal product wording + spec link                  |
| Process rubric       | `grading/expectations.json` (executors must **not** read)             |
| Objective supplement | `scripts/check_enable_gates.py` via `scripts/grade_objective.py`      |

### Executor rules

- Worktree path from `config.json` only
- Ticket from `prompts/<name>.md` only
- Do **not** read: `grading/`, `iteration-*/`, `vault/`, workspace `README.md`, sibling worktrees, eval scripts

### Spawn (one turn per eval)

| Arm           | Skill                              | Output dir                            |
| ------------- | ---------------------------------- | ------------------------------------- |
| with_skill    | `qubership-ndjson-logging-enable`  | `runs/<eval>/with_skill/outputs/`     |
| without_skill | No skill                           | `runs/<eval>/without_skill/outputs/`  |

Work happens in the configured worktree; copy transcripts/reports into `outputs/` when the run completes.

### Grade (after both arms)

```bash
# Objective (heuristic)
python3 iteration-7-enable/scripts/grade_objective.py with_skill go-log-exporter
python3 iteration-7-enable/scripts/grade_objective.py without_skill java-dbaas-monorepo

# Process expectations — human or grader reads grading/expectations.json
```

## Evals

| Name | Worktrees | Components |
| ---- | --------- | ---------- |
| `go-log-exporter` | `log-exporter-baseline` / `log-exporter-with-skill` | 1× Go + Helm |
| `java-dbaas-monorepo` | `dbaas-baseline` / `dbaas-with-skill` | Java aggregator + Go operator |

## Status

| Phase                                                 | Status                          |
| ----------------------------------------------------- | ------------------------------- |
| Scaffold                                              | Done                            |
| Full 4-arm run (reset worktrees + parallel executors) | Done                            |
| Objective grading                                     | Done — see `runs/SUMMARY.md`    |
| Process grading (expectations.json)                   | Pending                         |

## Quick manual try (dbaas pilot)

```bash
# Agent: worktree + prompts/enable-java-dbaas-monorepo.md + enable skill
python3 scripts/check_enable_gates.py worktrees/dbaas-with-skill \
  --java-path dbaas/dbaas-aggregator --go-path dbaas-operator
```
