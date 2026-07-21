# Migrate call sites — qubership-log-exporter (stage 2)

**Repository:** `qubership-log-exporter` (worktree assigned by operator)

**Prerequisite:** JSON logging envelope is **already enabled** (formatter, Helm `LOG_FORMAT`, smoke JSON line).
Do not redo stage-1 config work unless something is broken.

**Issue:** [Netcracker/qubership-logging-operator#289](https://github.com/Netcracker/qubership-logging-operator/issues/289)

**Specification:** [docs/cookbook/log-formats.md](https://github.com/Netcracker/qubership-logging-operator/blob/main/docs/cookbook/log-formats.md)

## Ask

Migrate **log call sites** so variable data lives in structured fields, not in formatted message strings
(`log.Infof`, `log.Errorf`, etc.).

1. Use the existing Go/logrus stack where possible.
2. Preserve log levels unless you document an explicit product decision.
3. Inventory preformatted-message and `fmt.Errorf` return paths; flag user-decision items.
4. Validate with Go tests and at least one realistic startup smoke path producing parseable NDJSON.
5. Report coverage and completion gates per the migrate skill.

Do not commit (eval / draft).
