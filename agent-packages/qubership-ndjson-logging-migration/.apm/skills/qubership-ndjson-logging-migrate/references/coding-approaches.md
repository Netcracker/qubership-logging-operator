# Coding Approaches (index)

How to implement call-site changes. [completion-gates.md](completion-gates.md) verifies the result.

## Stack playbooks (read one)

| Stack                                      | File                                                 |
| ------------------------------------------ | ---------------------------------------------------- |
| Java / Quarkus / SLF4J / Logback           | [java-quarkus.md](java-quarkus.md)                   |
| Go / logrus / qubership-core-lib-go / logr | [go-qubership-lib.md](go-qubership-lib.md)           |
| Python, Nginx, Envoy                       | [evidence.md](evidence.md) + target repo conventions |

## Cross-cutting rules

Choose the batch tier before editing. The tier controls batch size and immediate checks; it never replaces the required
component-wide review pass.

| Tier | Use for | Batch and required check |
| ---- | ------- | ------------------------ |
| **R0 — decide** | Placement FAIL, shared `{}` templates, or a user-decision row | Record the exact scope and pause for the user decision. |
| **R1 — hand** | Controllers, mappers, migrations, text blocks, multi-line calls, error paths, or a small heterogeneous group | One related package or ≤15 calls. Hand-edit, build, then review the full batch diff. |
| **R2 — reviewed batch** | Homogeneous one-line service calls with clear scalar diagnostics | One module subtree or ≤50 calls. Build, review field names/indentation, and spot-check 5–10 calls. |

Never use script-only migration. Scripts produce candidates; the agent must still review each changed event for meaning,
key/value correctness, throwable preservation, and non-logging behavior.
Large inventory count alone is never an R0 blocker: divide homogeneous R2 candidates into consecutive bounded batches and
continue. Review each changed event while applying the batch, then use the 5–10-call spot-check to verify the batch
pattern before the required component-wide review.

### Conservative field extraction

| Prefer as fields | Keep whole or reduce | Stop and ask |
| ---------------- | -------------------- | ------------ |
| Event-varying scalar IDs, names, types, statuses, booleans, and semantically named counts | A safe composed URL/path remains one field; collections become a specific count | Extraction changes meaning, exposes a potentially sensitive value, or needs an unclear field split |
| Existing request correlation fields | Objects, maps, DTOs, request/response bodies, settings, connection properties, credentials, passwords, and tokens are omitted unless an existing safe scalar is available | Shared templates, logged preformatted messages, or placement gaps — follow [user-decisions.md](user-decisions.md) |

Preserve API/exception consumer text and structure only the logging boundary. Preserve the original throwable association.
Keep fixed allowed-value enums in `message`; do not create redundant fixed-value fields.

Java event-field rules (fluent API, no per-call MDC): [java-quarkus.md](java-quarkus.md). Confirmed shapes after user
choice: [pattern-recipes.md](pattern-recipes.md).

## Migration process (done right)

1. **Repo-root discovery** — list every runtime component (sibling `go.mod`, Helm charts) before the first edit.
2. **Placement probe** — [placement-probe.md](placement-probe.md) per component before bulk call-site edits; on FAIL ask
   ([user-decisions.md](user-decisions.md) § Event-field placement unsupported).
3. **Call sites + config** — JSON formatter and `LOG_FORMAT` Helm wiring are necessary but not sufficient; migrate
   formatted log calls in production sources only after placement PASS (or explicit user defer).
4. **Gates, not grep alone** — grepping `{}` to zero while Java does not compile is incomplete; run
   [completion-gates.md](completion-gates.md) in full, then the **review pass** ([SKILL.md](../SKILL.md) § Review pass)
   before marking `migrated`.
5. **`blocked` sparingly** — large/noisy work is batched and continued; `blocked` is for user decisions, missing
   credentials with exact error, or unsafe API changes.
6. **Smoke** — one realistic startup/config path with a captured NDJSON line (`time`, `level`, `message` + top-level
   event fields), not unit tests alone.
7. **Target repo wins** — extend existing logger/config patterns; do not copy another service's stack blindly.
8. **Report** — write `.ndjson-migration-report.md` in the worktree per
   [migration-report-template.md](migration-report-template.md); exclude from product PR unless the team asks for it.
   Note that the review pass ran.

## Per call site checklist

- [ ] `message` preserves original event meaning (no invented summaries)
- [ ] Composed URL/path kept whole unless user approved segment fields —
      [pattern-recipes.md](pattern-recipes.md) § Composed path or URL
- [ ] Fixed allowed-value enums kept in `message` via format/concat (not separate fields) —
      [pattern-recipes.md](pattern-recipes.md) § Fixed allowed values
- [ ] Ambiguous extraction asked — [user-decisions.md](user-decisions.md) § Ambiguous meaning
- [ ] Semantic `snake_case` field names — [completion-gates.md](completion-gates.md) §4.1
- [ ] Throwable preserved (`setCause`) when original had one
- [ ] No duplicate `addKeyValue` key in one fluent chain (Java)
- [ ] No new per-call MDC / `StructuredLog` helper for event fields (Java)
- [ ] Not a no-op fluent wrap — `setMessage(msg).log()` alone is incomplete; add fields (or record prose-only /
      blocked) — [pattern-recipes.md](pattern-recipes.md) § Split log vs API text
- [ ] If user chose structure-at-boundary: consumer text unchanged; `.setMessage(sameVariable)` **and** `addKeyValue` —
      [pattern-recipes.md](pattern-recipes.md)
- [ ] `message` reads naturally without dangling placeholders
- [ ] Indentation matches surrounding code (no `->` padding; fluent chain columns consistent)
- [ ] Level unchanged unless user approved
- [ ] Non-logging code unchanged (`buildResponse`, endpoints, imports)
