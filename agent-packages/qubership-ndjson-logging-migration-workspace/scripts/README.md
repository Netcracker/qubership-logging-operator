# Eval-only scripts

**Not part of the published skill.** Used by `qubership-ndjson-logging-migration-workspace` for skill-creator grading.

Stage skills rely on manual gates in the package:

- Enable: `.apm/skills/qubership-ndjson-logging-enable/references/enable-gates.md`
- Migrate: `.apm/skills/qubership-ndjson-logging-migrate/references/completion-gates.md`

| Script | Purpose | Reliability |
| ------ | ------- | ----------- |
| `validate_ndjson_line.py` | One-line JSON smoke check (`time`/`level`/`message`) | Good for smoke samples |
| `check_migration_gates.py` | Stage **2** grep counts + `go build` | Supplement only — false +/- |
| `check_enable_gates.py` | Stage **1** report, JSON config, smoke documented | Supplement only — not zero `{}` |

Requires Python 3.8+ and `go` on PATH for Go checks.

```bash
python3 scripts/validate_ndjson_line.py sample.log
python3 scripts/check_migration_gates.py worktrees/dbaas-with-skill \
  --java-path dbaas/dbaas-aggregator --go-path dbaas-operator
python3 scripts/check_enable_gates.py worktrees/dbaas-baseline \
  --java-path dbaas/dbaas-aggregator --go-path dbaas-operator
iteration-7-enable/scripts/grade_objective.py without_skill java-dbaas-monorepo
```
