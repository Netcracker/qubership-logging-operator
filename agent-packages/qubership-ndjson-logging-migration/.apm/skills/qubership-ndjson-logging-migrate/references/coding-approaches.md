# Coding Approaches (index)

How to implement call-site changes. [completion-gates.md](completion-gates.md) verifies the result.

## Stack playbooks (read one)

| Stack                                      | File                                                 |
| ------------------------------------------ | ---------------------------------------------------- |
| Java / Quarkus / SLF4J / Logback           | [java-quarkus.md](java-quarkus.md)                   |
| Go / logrus / qubership-core-lib-go / logr | [go-qubership-lib.md](go-qubership-lib.md)           |
| Python, Nginx, Envoy                       | [evidence.md](evidence.md) + target repo conventions |

## Cross-cutting rules

**Default strategy:** small batches, hand review, compile/build after each batch. Scripts produce candidates only — every
changed call site still gets semantic review.

| Approach        | When                                                                          |
| --------------- | ----------------------------------------------------------------------------- |
| Hand edit       | Controllers, mappers, multi-line logs, text blocks, < 20 sites                |
| Script + review | Large homogeneous one-line `log.info("...", a, b)` in services                |
| Script-only     | Never — if the diff could delete methods or break annotations, review by hand |

After each batch: `mvn compile` or `go build` → **review diff field names** → spot-check 5–10 call sites →
`_get_`/`_stream_`/`argN` greps (blocking residue) → throwables sweep → text-block inventory.

Java event-field rules (fluent API, no per-call MDC): [java-quarkus.md](java-quarkus.md). Confirmed shapes after user
choice: [pattern-recipes.md](pattern-recipes.md).

## Migration process (done right)

1. **Repo-root discovery** — list every runtime component (sibling `go.mod`, Helm charts) before the first edit.
2. **Placement probe** — [placement-probe.md](placement-probe.md) per component before bulk call-site edits; on FAIL ask
   ([user-decisions.md](user-decisions.md) § Event-field placement unsupported).
3. **Call sites + config** — JSON formatter and `LOG_FORMAT` Helm wiring are necessary but not sufficient; migrate
   formatted log calls in production sources only after placement PASS (or explicit user defer).
4. **Gates, not grep alone** — grepping `{}` to zero while Java does not compile is incomplete; run
   [completion-gates.md](completion-gates.md) in full (and SKILL.md self-check).
5. **`blocked` sparingly** — large/noisy work is batched and continued; `blocked` is for user decisions, missing
   credentials with exact error, or unsafe API changes.
6. **Smoke** — one realistic startup/config path with a captured NDJSON line (`time`, `level`, `message` + top-level
   event fields), not unit tests alone.
6. **Target repo wins** — extend existing logger/config patterns; do not copy another service's stack blindly.
7. **Report** — write `.ndjson-migration-report.md` in the worktree per
   [migration-report-template.md](migration-report-template.md); exclude from product PR unless the team asks for it.

## Per call site checklist

- [ ] Semantic `snake_case` field names — [completion-gates.md](completion-gates.md) §4.1
- [ ] Throwable preserved (`setCause`) when original had one
- [ ] No duplicate `addKeyValue` key in one fluent chain (Java)
- [ ] No new per-call MDC / `StructuredLog` helper for event fields (Java)
- [ ] If user chose structure-at-boundary: consumer text unchanged; `.setMessage(sameVariable)` —
      [pattern-recipes.md](pattern-recipes.md)
- [ ] `message` reads naturally without dangling placeholders
- [ ] Level unchanged unless user approved
- [ ] Non-logging code unchanged (`buildResponse`, endpoints, imports)
