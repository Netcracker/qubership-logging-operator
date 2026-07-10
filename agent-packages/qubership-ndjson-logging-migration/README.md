# qubership-ndjson-logging-migration

Two-stage NDJSON rollout for Qubership services.

| Stage | Skill | Goal |
|-------|-------|------|
| **1 — Enable** | `qubership-ndjson-logging-enable` | JSON envelope on stdout — config/Helm/`LOG_FORMAT`; legacy text may stay inside `message` |
| **2 — Migrate** | `qubership-ndjson-logging-migrate` | Extract fields from messages; full completion gates |

Run stage 1 before stage 2. Stage 2 assumes NDJSON output already works (or documents blockers).

Install:

```yaml
# apm.yml devDependencies
- ./qubership-logging-operator/agent-packages/qubership-ndjson-logging-migration
```
