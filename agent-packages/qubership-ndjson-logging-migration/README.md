# qubership-ndjson-logging-migration

Two-stage NDJSON rollout for Qubership services.

| Stage           | Skill                              | Goal                                                                                      |
| --------------- | ---------------------------------- | ----------------------------------------------------------------------------------------- |
| **1 — Enable**  | `qubership-ndjson-logging-enable`  | JSON envelope on stdout — config/Helm/`LOG_FORMAT`; legacy text may stay inside `message` |
| **2 — Migrate** | `qubership-ndjson-logging-migrate` | Extract fields from messages; full completion gates                                       |

Run stage 1 before stage 2. Stage 2 assumes NDJSON output already works (or documents blockers).

## Evals (skill-creator)

| Stage | Skill | Eval definitions | Objective checks |
| ----- | ----- | ---------------- | ---------------- |
| Enable | `qubership-ndjson-logging-enable` | `evals/evals-enable.json` | `evals/objective_checks-enable.json` |
| Migrate | `qubership-ndjson-logging-migrate` | `evals/evals-migrate.json` | `evals/objective_checks-migrate.json` |

Index: `evals/evals.json`. Protocol: [EVAL_PROTOCOL.md](EVAL_PROTOCOL.md). Workspace:
`../qubership-ndjson-logging-migration-workspace/` (`iteration-7-enable`, `iteration-8-migrate`).

Install:

```yaml
# apm.yml devDependencies
- ./qubership-logging-operator/agent-packages/qubership-ndjson-logging-migration
```

## Authoring

Markdown under this package must pass markdownlint (120 cols) per
`.github/linters/.markdownlint.yaml`.

Verify with:

```bash
npx markdownlint-cli2 'agent-packages/qubership-ndjson-logging-migration/**/*.md' --config .github/linters/.markdownlint.yaml
```
