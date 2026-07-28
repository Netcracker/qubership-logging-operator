# Completion Gates (Stage 2 — Definition of Done)

**Prerequisite:** Stage 1 JSON envelope (`qubership-ndjson-logging-enable`) or equivalent repo config.

**Goal first:** operators can filter on top-level JSON fields with a readable `message` (see [SKILL.md](../SKILL.md)
§ Goal). Pattern greps are **smell checks** — necessary evidence that candidates remain, **not** the win condition.
A migration is complete only when **build**, **integrity**, **pattern smells**, **semantic quality**, the **review
pass**, and **smoke** pass (or each failure is explicitly `blocked` with a concrete reason). Clean greps with
diagnostics still only inside `message` is **not** done.

Lessons from pilot migrations: bulk Java codemods can drive `{}` greps to zero while leaving **non-compiling** code, **deleted
endpoints**, and **unusable `arg0` field keys**.

## Gate order (run in this sequence)

0. **Placement probe** — before bulk call-site edits ([placement-probe.md](placement-probe.md)); all stacks
1. **Build** — compile/test per runtime component
2. **Integrity** — no accidental method/endpoint deletion; imports still resolve
3. **Pattern smells** — zero unaccounted formatted/variable-message candidates in production scope
4. **Semantic quality** — field names, throwables, messages, duplicate keys (goal: queryable fields)
5. **Review pass** — full migrate diff vs hard rules + these gates; fix; re-check ([SKILL.md](../SKILL.md) § Review pass)
6. **Smoke** — realistic startup emits valid NDJSON with diagnostic keys at top level (see [smoke-validation.md](smoke-validation.md))

Do not claim completion if an earlier gate failed unless the failure is recorded as `blocked` and unrelated work is still
valid. Do **not** bulk-migrate while placement probe is FAIL without a recorded user decision.

---

## 1. Build gates (blocking)

| Stack              | Required command                                                                | Pass criterion                                 |
| ------------------ | ------------------------------------------------------------------------------- | ---------------------------------------------- |
| **Java / Maven**   | `mvn -pl <module> -am compile` (or repo-documented equivalent)                  | **Exit 0**                                     |
| **Java / Quarkus** | Same; optionally `mvn -pl <module> test` when CI credentials exist              | Compile **must** pass before merge             |
| **Go**             | `GOWORK=off go build ./...` and `GOWORK=off go test ./...` for touched packages | **Exit 0** for build; test failures documented |

**If Maven compile is blocked locally** (e.g. GitHub Packages 401): record under `Blocked validation` with the exact
error. **Do not mark the Java component migrated-complete.** Continue only on components that build, or stop with the
blocker named.

**After every bulk codemod batch:** re-run the build gate for that component before the next batch.

---

## 2. Integrity gates (blocking)

Run after large automated edits or any edit touching controllers, mappers, or multi-line log calls.

### 2.1 No accidental code deletion

```bash
# Suspicious: large pure deletions outside test/resources
git diff --stat HEAD
git diff HEAD -- '*.java' | grep '^-.*\(public \|private \|@GET\|@POST\)' 
```

- Restore any **removed endpoint handlers**, service methods, or mapper logic that was not an intentional logging-only
  change.
- If a file's only change would be an unused import, **remove the import**.

### 2.2 Java syntax sanity (post-codemod)

Illegal single-line text blocks must be **zero** in `src/main/java` — smell-checks.sh **J7** (and open every **J5**
hit). Orphan annotations / broken method stubs (e.g. `@APIResponses` without a following `@GET` method body) still
need manual review — no grep catches them.

### 2.3 Imports and annotations

- If `@Slf4j` remains on the class, `import lombok.extern.slf4j.Slf4j` must be present.
- Codemods must not strip Lombok/logger imports while leaving generated `log` usage.

### 2.4 API behavior preserved

- Exception mappers must still call the same `buildResponse(...)` overloads with the same suppliers/builders.
- Do not reduce `buildResponse(status)` to a single-arg form when only two-arg overloads exist.

### 2.5 Indentation and formatting

Match surrounding indent; fluent chain columns consistent; no padding after `->` when expanding one-liners. Run the
repo formatter / checkstyle if present. Details: [java-quarkus.md](java-quarkus.md) (Indentation practice).

---

## 3. Pattern gates (necessary, not sufficient)

Run [scripts/smell-checks.sh](../scripts/smell-checks.sh); record **before/after counts** in the migration report.
Check meanings: [preformatted-message-patterns.md](preformatted-message-patterns.md).

| Stack               | Check                                          | Target                    |
| ------------------- | ---------------------------------------------- | ------------------------- |
| Go/logrus           | **G1** — active `log.*f(` in production `.go`  | **0**                     |
| Go residual printf  | **G2** — diagnostic verbs on non-`f` log calls | **0**                     |
| Java/SLF4J          | **J2** same-line `{}` **and** **J5** text-block hits opened | **0**        |
| Java field polish   | **J6a/J6b** — codemod residue keys — §4.1      | **0** or polish pass done |
| Logged preformatted | **J4/J8/G3** (helper calls are expected hits)  | **0** unreviewed          |

**Misleading zeros** (Go drop-`f`, Sprintf-then-`%s` dodge, Java shared `{}` constants — **queue for the inventory
decision batch**, Java text blocks): [preformatted-message-patterns.md](preformatted-message-patterns.md) § Misleading
zeros.

---

## 4. Semantic quality gates (blocking for merge-quality)

### 4.1 Semantic field names (primary gate)

Every structured field must use consumer-friendly **`snake_case` derived from message semantics** (`resource_id`,
`namespace`, `status`) — not positional placeholders, leaked locals, or expression-derived keys.

**Reject (non-exhaustive):**

| Category | Examples |
| -------- | -------- |
| Positional / generic | `arg0`, `argument1`, `param2`, `value0`, `field1` |
| Leaked locals / abbreviations | `i`, `ns`, `err`, `sbe`, `qName`, `lbName` |
| Codemod / expression residue | `resource_get_id`, `items_stream_map_to_list`, `e_get_message` |

**How to verify (required):**

1. **Spot-check** 5–10 migrated call sites per batch: original `{}` message → each key matches the semantic label
   (e.g. `resource_id`, not `id` or `arg0`). Also check key↔value (do not name a field `*_address` if the value is an id).
2. **Review the diff** for `addKeyValue`, `WithField`, `StructuredArguments.kv`, `logfields.Format`.
3. **Codemod residue check** — smell-checks.sh **J6a/J6b** (blocking until 0 or an explicit polish follow-up is
   finished).

Mark the field-names gate **PARTIAL** (and the component **not** migrated) while these hits remain. Polish to semantic
names before claiming done. Spot-check alone is not enough after a bulk codemod.

Same rule for Go: semantic names + spot-check. See [user-decisions.md](user-decisions.md) § Semantic field names.

### 4.2 Extract only event-varying fields; no duplicate keys

**Fields:** extract values that **vary per event** and are useful to filter on. Fixed literals / enum constants that only
list allowed values stay in **`message`** (format/concat with the constants) — do not turn them into
`completed_status` / `failed_status`-style fields.

**Duplicate keys:** repeating `addKeyValue("status", …)` twice in one chain **overwrites** the earlier value. Do not
“fix” that by inventing parallel keys for fixed constants; keep the constants in `message` instead.

**Bad:** `.addKeyValue("status", status).addKeyValue("completed_status", COMPLETED).addKeyValue("failed_status", FAILED)`  
**Good:** `.addKeyValue("status", status)` + `String.format(…, COMPLETED, FAILED, …)` in `setMessage`

Review every multi-field migration manually.

### 4.3 Throwables preserved

When the original call passed an exception as the final SLF4J argument (`log.error("...", a, b, throwable)`), use
`setCause(throwable)` on the fluent builder (or the repo's equivalent throwable-aware helper).

Sweep: count removed `error`/`warn` calls that had a throwable vs conversions with `setCause` — gaps must be fixed or
listed.

### 4.4 Human-readable messages

After extracting fields, `message` must not contain:

- Dangling `=` or `, ,` gaps
- Empty placeholder holes (`resource=, error=`)
- Placeholder-only text (`.`)

`message` must also **preserve the original event meaning** (same intent as before migration). Invented summaries,
over-split URL/path segments, or fields for fixed allowed-value enums fail this gate — see
[user-decisions.md](user-decisions.md) § Ambiguous meaning and [pattern-recipes.md](pattern-recipes.md) § Fixed allowed
values.

### 4.5 Java event fields in JSON (Quarkus / Logback)

Per-log fields must use the SLF4J 2.x fluent API (`addKeyValue`) or encoder structured args — see
[java-quarkus.md](java-quarkus.md).

- **Before bulk migrate:** [placement-probe.md](placement-probe.md) must PASS (or user chose defer / accept-unmet-goal —
  then do not mark `migrated`).
- Capture one runtime JSON line and verify `addKeyValue` fields appear at the **top level**.
- Correlation fields (`request_id`, `tenant_id`) may still use request-scoped MDC + `%X{...}` in config — that is
  expected.
- Diff smell: `setMessage(msg).log()` with no `addKeyValue` while `msg` was built from diagnostics — **incomplete**
  (same as pre-migration for operators).
- **Manual review (diff):** no new `StructuredLog`-style helper and no new per-call `MDC.put` for event fields.
  Request-scoped MDC in filters/interceptors is OK.

If diagnostic fields appear only under `mdc.*`, the call sites are still MDC-shaped — rework to fluent API.
If diagnostics are glued into `message`, that is placement FAIL ([placement-probe.md](placement-probe.md) § Failure
signatures) — stop and ask per [user-decisions.md](user-decisions.md) § Event-field placement unsupported.

### 4.6 Go field APIs / `logfields` / regex formatters

See [go-qubership-lib.md](go-qubership-lib.md). Minimum gates:

- Prefer first-class fields or a repo helper (`logfields.Format` / `Err`); do not treat “drop `f`, keep printf args”
  as complete
- Quote values containing whitespace when using message-suffix parsing
- Do not let parsed fields overwrite reserved keys (`time`, `level`, `message`, `class`, `request_id`, …)
- Prefer structural field APIs when the platform logger supports them
- Smoke: diagnostic keys appear at JSON top level, not only inside `message`

---

## 5. Automation boundaries (reinforced)

- Scripts produce **candidates** only; **build + semantic review** is mandatory.
- **Multi-line** Java log calls and **text blocks** (`"""`) require hand review or an AST-based tool — regex codemods often
  break them.
- Remove temporary `migrate_*.py` from the PR unless the user explicitly wants them; never leave them in runtime packages.

---

## Recording results

Record per-gate before/after results in the migration report — the canonical gate-table skeleton lives in
[migration-report-template.md](migration-report-template.md) § Completion gates (single home; do not maintain a copy
here).

Migration is **not complete** while any **blocking** row is FAIL or PARTIAL without a concrete `blocked` reason.
Do not mark a component `migrated` in the coverage ledger while any gate for that component is FAIL/PARTIAL — see
[migration-report-template.md](migration-report-template.md) § Status rules. Pattern smells cleared without queryable
top-level fields still fail the goal ([SKILL.md](../SKILL.md)).
