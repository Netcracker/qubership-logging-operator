# Stage 1 report template

`.ndjson-migration-report.md` at worktree root during the run. Exclude from product PR unless requested.

```markdown
# NDJSON Enable Report — <repo-name> (stage 1)

| Field | Value |
|-------|-------|
| **Stage** | enable |
| **Run start HEAD** | `<git rev-parse HEAD>` |
| **Skill** | `qubership-ndjson-logging-enable` |
| **Date** | YYYY-MM-DD |

## Deployable components

| Component | Path | Stack | LOG_FORMAT / encoder | Status |
|-----------|------|-------|----------------------|--------|
| ... | ... | ... | ... | enabled / blocked / pending |

## Stage 1 gates

See [enable-gates.md](enable-gates.md).

| Gate | Check | PASS |
|------|-------|------|
| Build | repo-documented build or CI workflow | |
| LOG_FORMAT + Helm | env + chart | |
| Smoke NDJSON | repo-documented run/deploy; app envelope or access-log JSON line | |

## Blocked validation

| Component | Command | Error |
|-----------|---------|-------|

## Next step

Run `qubership-ndjson-logging-migrate` when ready for call-site field extraction.
```
