# qubership-ndjson-logging-migration — skill-creator workspace

Formal eval workspace. Protocol: [EVAL_PROTOCOL.md](../qubership-ndjson-logging-migration/EVAL_PROTOCOL.md).

The published skills have **no bundled scripts** — manual gates in each skill's references are the contract.
This workspace keeps eval-only helpers under `scripts/`.

## Two-stage eval model

| Stage | Skill | Workspace iteration | Package evals |
| ----- | ----- | ------------------- | ------------- |
| 1 — Enable | `qubership-ndjson-logging-enable` | `iteration-7-enable/` | `evals/evals-enable.json` |
| 2 — Migrate | `qubership-ndjson-logging-migrate` | `iteration-9-migrate/` | `evals/evals-migrate.json` |

Run enable before migrate. Migrate worktrees must be post-enable.

## Pinned fixtures

| Eval | Repo | Commit |
| ---- | ---- | ------ |
| go-log-exporter | `qubership-log-exporter` | `4bf5465c88634cd18ea22fbd30a960d4bca79d13` |
| java-dbaas-monorepo | `qubership-dbaas` | `f0d45d69f309bc0492def72ac9d924ea5e8e8e75` |

Worktrees live under `worktrees/`. Do not use primary repo checkouts for eval runs.

## History

See `history.json` for iteration lineage and current best snapshot per stage.

## Iterations

| Dir | Notes |
| --- | ----- |
| `iteration-1`–`3` | Early runs; contamination / shared worktrees |
| `iteration-4` | Isolated worktrees; 100% process assertions; baseline gaps on dbaas objective checks |
| `iteration-5` | Natural blind — C/D cancelled (executor contamination) |
| `iteration-6` | **Superseded** — monolithic migrate scaffold |
| **`iteration-7-enable`** | **Stage 1** — enable skill, `check_enable_gates.py`, 4-arm run done |
| **`iteration-8-migrate`** | Stage 2 — MDC wrapper skill; dbaas discriminates |
| **`iteration-9-migrate`** | **Stage 2 current** — fluent API skill; pending runs |

## Eval-only scripts

See [scripts/README.md](scripts/README.md). Heuristic supplements — not part of the published skill.

```bash
# Stage 1
python3 scripts/check_enable_gates.py worktrees/dbaas-with-skill \
  --java-path dbaas/dbaas-aggregator --go-path dbaas-operator
iteration-7-enable/scripts/grade_objective.py with_skill java-dbaas-monorepo

# Stage 2
python3 scripts/check_migration_gates.py worktrees/dbaas-with-skill \
  --java-path dbaas/dbaas-aggregator --go-path dbaas-operator
iteration-8-migrate/scripts/grade_objective.py without_skill java-dbaas-monorepo
```
