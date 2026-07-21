# Iteration 6 — skill-creator aligned eval (superseded)

> **Superseded by `iteration-8-migrate/`** for stage-2 migrate evals. This directory is kept for history only.

Original intent: v1 monolithic migrate skill A/B with objective checks.

## Why superseded

- Stale `skill_path` pointed at removed `qubership-ndjson-logging-migration` skill
- Included eval #3 (`log-level-response-body`) whose fixture was removed from the package
- Two-stage split requires separate enable (`iteration-7-enable`) and migrate (`iteration-8-migrate`) iterations

## Protocol (historical)

See package [EVAL_PROTOCOL.md](../../qubership-ndjson-logging-migration/EVAL_PROTOCOL.md) — use
`iteration-8-migrate` for current migrate protocol.

## Status

| Phase | Status |
| ----- | ------ |
| Scaffold | Done |
| Runs | Cancelled — use iteration-8-migrate |
