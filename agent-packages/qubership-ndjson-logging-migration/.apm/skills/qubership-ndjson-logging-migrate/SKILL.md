---
name: qubership-ndjson-logging-migrate
description: >
  Stage 2 — Complete Qubership NDJSON migration: move diagnostic data from log messages into structured JSON fields,
  satisfy semantic completion gates. Use after stage 1 (qubership-ndjson-logging-enable) or when JSON output already
  works. Triggers on SLF4J {}, Go log.*f, bracket key=value in messages, preformatted log.warn(message), monorepo
  call-site rollout, or "full NDJSON migration". Not for config-only LOG_FORMAT rollout.
---

# NDJSON Migrate (Stage 2)

Extract structured data from log **messages** into JSON **fields** while NDJSON output is already enabled (stage 1).

**Prerequisite:** JSON envelope on stdout — from `qubership-ndjson-logging-enable` or existing repo config. If not, run stage 1 first or record config gaps in the report.

## Reference map

| When | Read |
|------|------|
| Full field schema | [schema.md](references/schema.md) |
| Stack implementation | [java-quarkus.md](references/java-quarkus.md) or [go-qubership-lib.md](references/go-qubership-lib.md) |
| Cross-cutting rules | [coding-approaches.md](references/coding-approaches.md) |
| Before claiming done | [completion-gates.md](references/completion-gates.md) |
| Report | [migration-report-template.md](references/migration-report-template.md) |
| User choice | [user-decisions.md](references/user-decisions.md) |
| Inventory patterns | [preformatted-message-patterns.md](references/preformatted-message-patterns.md) |
| Smoke | [smoke-validation.md](references/smoke-validation.md) |
| Pitfalls | [corner-cases.md](references/corner-cases.md) |
| Background | [evidence.md](references/evidence.md) |

## Required outcome

- Useful diagnostic data in **structured fields**, not only inside `message`.
- `message` human-readable; correlation fields preserved (`request_id`, `tenant_id`, trace/span, `logType`).
- Every remaining formatted / variable-message call site inventoried or migrated.

## Workflow

1. Confirm stage 1 — JSON smoke passed or document config blocker.
2. Read [schema.md](references/schema.md) — full contract (field extraction).
3. **Repo-root discovery** — coverage ledger for all runtime components.
4. **Classify stack** → [java-quarkus.md](references/java-quarkus.md) or [go-qubership-lib.md](references/go-qubership-lib.md).
5. **Inventory** log sources and preformatted diagnostics — [preformatted-message-patterns.md](references/preformatted-message-patterns.md).
6. **Classify** sites: `migrate`, `static/no action`, `needs user decision`, `blocked`.
7. **User decisions** — [user-decisions.md](references/user-decisions.md).
8. **Map fields** — semantic names; preserve qualifiers in field names.
9. **Implement** in small batches — [coding-approaches.md](references/coding-approaches.md); build after each batch.
10. **Re-inventory** — no unaccounted formatted calls.
11. **Run gates** — [completion-gates.md](references/completion-gates.md).
12. **Smoke** — [smoke-validation.md](references/smoke-validation.md).
13. **Write report** — stage = `migrate`; exclude from product PR unless requested.
14. **Propose skill updates** in the APM package source, not `.agents/skills` copies.

## Monorepos

One component at a time; update ledger before stopping.

## Definition of done

[completion-gates.md](references/completion-gates.md): build, integrity, pattern + semantic gates, smoke NDJSON valid.
