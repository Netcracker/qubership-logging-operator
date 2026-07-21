# Completion Gates (Stage 2 — Definition of Done)

**Prerequisite:** Stage 1 JSON envelope (`qubership-ndjson-logging-enable`) or equivalent repo config.

**Goal first:** operators can filter on top-level JSON fields with a readable `message` (see [SKILL.md](../SKILL.md)
§ Goal). Pattern greps are **smell checks** — necessary evidence that candidates remain, **not** the win condition.
A migration is complete only when **build**, **integrity**, **pattern smells**, and **semantic quality** gates pass and
smoke shows queryable fields (or each failure is explicitly `blocked` with a concrete reason). Clean greps with
diagnostics still only inside `message` is **not** done.

Lessons from pilot migrations: bulk Java codemods can drive `{}` greps to zero while leaving **non-compiling** code, **deleted
endpoints**, and **unusable `arg0` field keys**.

## Gate order (run in this sequence)

1. **Build** — compile/test per runtime component
2. **Integrity** — no accidental method/endpoint deletion; imports still resolve
3. **Pattern smells** — zero unaccounted formatted/variable-message candidates in production scope
4. **Semantic quality** — field names, throwables, messages, duplicate keys (goal: queryable fields)
5. **Smoke** — realistic startup emits valid NDJSON with diagnostic keys at top level (see [smoke-validation.md](smoke-validation.md))

Do not claim completion if an earlier gate failed unless the failure is recorded as `blocked` and unrelated work is still
valid.

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

These patterns must be **zero** in `src/main/java`:

```bash
# Illegal single-line text block (opening """ must be followed by newline)
grep -rn 'log\.at\w\+([^)]*""" [^"]' --include='*.java' src/main/java
grep -rn '""" [^"]' --include='*.java' src/main/java | grep -E 'log\.(at|info|debug|warn|error)'

# Orphan annotations / broken method stubs (manual review)
# e.g. @APIResponses without following @GET method body
```

### 2.3 Imports and annotations

- If `@Slf4j` remains on the class, `import lombok.extern.slf4j.Slf4j` must be present.
- Codemods must not strip Lombok/logger imports while leaving generated `log` usage.

### 2.4 API behavior preserved

- Exception mappers must still call the same `buildResponse(...)` overloads with the same suppliers/builders.
- Do not reduce `buildResponse(status)` to a single-arg form when only two-arg overloads exist.

---

## 3. Pattern gates (necessary, not sufficient)

Record **before/after counts** in the migration report.

| Stack               | Production scope check                                                           | Target            |
| ------------------- | -------------------------------------------------------------------------------- | ----------------- |
| Go/logrus           | Active `log.*f(` in non-test `.go` (exclude `//` comments, `dev/`, `_test.go`)   | **0**             |
| Go residual printf  | Non-`f` `log.*(C)?(` with diagnostic `%v`/`%d`/… in the call (see SKILL self-check) | **0**          |
| Java/SLF4J          | Same-line `log.(info\|…).*\{` **and** text-block `log.*( """` with `{}` inside   | **0**             |
| Java field polish   | Codemod residue keys (`_get_`, `_stream_`, `e_get_message`, `argN`) — §4.1       | **0** or polish pass done |
| Logged preformatted | Patterns in [preformatted-message-patterns.md](preformatted-message-patterns.md) | **0** unreviewed  |

**Misleading zero (Go):** `log.*f` → 0 while `log.Error("… key=%v …", key, err)` remains is **not** done — see
[go-qubership-lib.md](go-qubership-lib.md).

**Misleading zero (Java):** `{}` in a **shared string constant** still templates at runtime — **stop and ask
the user immediately** per [user-decisions.md](user-decisions.md); do not treat same-line `{}` grep zero as fully
structured.

**Misleading zero (Java text blocks):** same-line `{}` grep → 0 while `log.info(""" … {} … """)` remains is **not** done —
inventory text-block opens and open each hit.

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
3. **Codemod residue greps (blocking until 0 or an explicit polish follow-up is finished):**

```bash
grep -rnE 'addKeyValue\("[^"]*(_get_|_stream_|e_get_message)' --include='*.java' src/main/java
grep -rn '"arg[0-9]\+"' --include='*.java' src/main/java
```

Mark the field-names gate **PARTIAL** (and the component **not** migrated) while these hits remain. Polish to semantic
names before claiming done. Spot-check alone is not enough after a bulk codemod.

Same rule for Go: semantic names + spot-check. See [user-decisions.md](user-decisions.md) § Semantic field names.

### 4.2 No duplicate keys in one log call

Fluent API: repeating `addKeyValue("status", …)` twice in one chain **overwrites** the earlier value. Review every
multi-field migration manually.

**Bad:** `.addKeyValue("status", COMPLETED).addKeyValue("status", FAILED)`  
**Good:** `.addKeyValue("completed_status", COMPLETED).addKeyValue("failed_status", FAILED)`

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

### 4.5 Java event fields in JSON (Quarkus / Logback)

Per-log fields must use the SLF4J 2.x fluent API (`addKeyValue`) or encoder structured args — see
[java-quarkus.md](java-quarkus.md). After migration:

- Capture one runtime JSON line and verify `addKeyValue` fields appear at the **top level**.
- Correlation fields (`request_id`, `tenant_id`) may still use request-scoped MDC + `%X{...}` in config — that is
  expected.
- **Manual review (diff):** no new `StructuredLog`-style helper and no new per-call `MDC.put` for event fields.
  Request-scoped MDC in filters/interceptors is OK.

If diagnostic fields appear only under `mdc.*`, the call sites are still MDC-shaped — rework to fluent API.

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

## Report template (paste into migration report)

```markdown
## Completion gates

| Gate | Command / check | Before | After | PASS |
|------|-----------------|--------|-------|------|
| Java compile | `mvn -pl ... compile` | — | exit 0 | |
| Go build | `GOWORK=off go build ./cmd/` | — | exit 0 | |
| Java `{}` inline | same-line + text-block inventory | N | 0 | |
| Java field names | spot-check + `_get_`/`_stream_`/`argN` greps | — | OK (0 residue) | |
| Java event fields | manual: fluent API + JSON top-level; no new MDC wrapper | — | OK | |
| Go `log.*f` | grep production .go | N | 0 | |
| Go residual printf | SKILL self-check residual verbs | N | 0 | |
| Throwables | manual sweep | N dropped | N fixed | |
| Integrity | git diff review | — | no stray deletions | |
| Smoke NDJSON | see smoke-validation.md | — | OK | |
```

Migration is **not complete** while any **blocking** row is FAIL or PARTIAL without a concrete `blocked` reason.
Do not mark a component `migrated` in the coverage ledger while any gate for that component is FAIL/PARTIAL — see
[migration-report-template.md](migration-report-template.md) § Status rules. Pattern smells cleared without queryable
top-level fields still fail the goal ([SKILL.md](../SKILL.md)).
