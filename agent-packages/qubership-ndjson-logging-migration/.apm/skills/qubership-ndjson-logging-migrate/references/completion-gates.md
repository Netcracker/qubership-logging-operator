# Completion Gates (Stage 2 — Definition of Done)

**Prerequisite:** Stage 1 JSON envelope (`qubership-ndjson-logging-enable`) or equivalent repo config.

Pattern-count gates alone are **not sufficient**. A migration is complete only when **build gates**, **integrity gates**,
**pattern gates**, and **semantic quality gates** all pass (or each failure is explicitly `blocked` with a concrete
reason).

Lessons from pilot migrations: bulk Java codemods can hit `632→0 {}` while leaving **non-compiling** code, **deleted
endpoints**, and **unusable `arg0` field keys**.

## Gate order (run in this sequence)

1. **Build** — compile/test per runtime component
2. **Integrity** — no accidental method/endpoint deletion; imports still resolve
3. **Pattern** — zero unaccounted formatted/variable-message calls in production scope
4. **Semantic quality** — field names, throwables, messages, duplicate keys
5. **Smoke** — realistic startup emits valid NDJSON (see [smoke-validation.md](smoke-validation.md))

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
- If a file's only change would be an unused import, **remove the import** — do not leave `StructuredLog` imported
  unused.

### 2.2 Java syntax sanity (post-codemod)

These patterns must be **zero** in `src/main/java`:

```bash
# Illegal single-line text block (opening """ must be followed by newline)
grep -rn 'StructuredLog\.\w\+([^)]*""" [^"]' --include='*.java' src/main/java

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
| Java/SLF4J          | `log.(info\|debug\|warn\|error).*` with `{}` in `src/main/java`                  | **0** inline `{}` |
| Logged preformatted | Patterns in [preformatted-message-patterns.md](preformatted-message-patterns.md) | **0** unreviewed  |

**Misleading zero:** `{}` in a **shared constant** (e.g. `WARNING_MESSAGE`) still templates at runtime — list those sites
under user decision; do not treat as fully structured.

---

## 4. Semantic quality gates (blocking for merge-quality)

### 4.1 No placeholder field keys

```bash
grep -rn '"arg[0-9]\+"' --include='*.java' src/main/java
```

**Target: 0** in production Java. Use semantic `snake_case` names (`backup_id`, `namespace`, `status`, `portion_number`).

Same rule for Go: no `arg0`-style keys in `logfields.Format` calls.

### 4.2 No duplicate keys in one log call

`StructuredLog` / MDC: repeating the same key in one call **overwrites** earlier values. Review every multi-argument
migration manually or with a linter script.

**Bad:** `"backup", COMPLETED, "backup", FAILED, "backup", DELETED`  
**Good:** `"completed_status", COMPLETED, "failed_status", FAILED, "deleted_status", DELETED`

### 4.3 Throwables preserved

When the original call passed an exception as the final SLF4J argument (`log.error("...", a, b, throwable)`), use the
**throwable overload** of your structured helper (e.g. `StructuredLog.error(log, msg, t, fields...)`).

Sweep: count removed `error`/`warn` calls that had a throwable vs conversions using the throwable overload — gaps must be
fixed or listed.

### 4.4 Human-readable messages

After extracting fields, `message` must not contain:

- Dangling `=` or `, ,` gaps
- Empty placeholder holes (`logicalBackup=, error=`)
- Placeholder-only text (`.`)

### 4.5 Java MDC → JSON field promotion (Quarkus)

Arbitrary MDC keys appear under `mdc.*` unless declared in `quarkus.log.console.json.fields.*` / `application.properties`.
After migration:

- Declare promoted keys in Quarkus JSON config, **or**
- Capture one runtime JSON line and verify fields appear at the expected top level.

### 4.6 Go `logfields` / regex formatters

See [go-qubership-lib.md](go-qubership-lib.md) (core-lib-go + message suffix). Minimum gates:

- Quote values containing whitespace
- Do not let parsed fields overwrite reserved keys (`time`, `level`, `message`, `class`, `request_id`, …)
- Prefer structural field APIs when the platform logger supports them

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
| Java `{}` inline | grep src/main/java | N | 0 | |
| Java `argN` keys | grep `"arg[0-9]"` | N | 0 | |
| Go `log.*f` | grep production .go | N | 0 | |
| Throwables | manual sweep | N dropped | N fixed | |
| Integrity | git diff review | — | no stray deletions | |
| Smoke NDJSON | see smoke-validation.md | — | OK | |
```

Migration is **not complete** while any **blocking** row is FAIL without a concrete `blocked` reason.
