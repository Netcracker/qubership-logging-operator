# Pattern Recipes (pilot-derived)

Transformation **shapes** from real migrations — not copy-paste templates or grep targets. Read when implementing a
user-confirmed choice from [user-decisions.md](user-decisions.md).

## How to read examples

| Do | Don't |
| -- | ----- |
| Recognize the **invariant** (e.g. same `msg` for log and `Response.entity`) | Search for exact pilot strings or class names in other repos |
| Derive **field names** from variables and message semantics at **this** call site | Reuse example field names when local variables differ |
| Adapt level, framework return type, and DTO setters to the actual code | Assume JAX-RS, `String msg`, Java, or one repo's naming |
| Treat each block as **before/after shape** | Treat examples as an exhaustive inventory of sites |

Placeholders: `varA`, `varB`, `context`, `consumer`, `SHARED_TEMPLATE`, `detailVar` — stand in for whatever exists at
the call
site. The **rules and table invariants** transfer; literal text does not.

Do **not** apply these recipes until the user confirms the choice (or states a repo-wide policy in session). Record the
decision in `.ndjson-migration-report.md`.

## When to use

| User choice | This file |
| ----------- | --------- |
| **Structure at logging boundary** | § Split log vs API text, § Conditional message |
| **Composed path / URL (or ask)** | § Composed path or URL |
| **Fixed allowed values in message** | § Fixed allowed values |
| **Partial fluent helper for mappers** | § Partial fluent helper |
| **Prose-only constant** | § Template constant without `{}` |
| **Prose-only / no change** | § `getMessage()` only at log site |
| **Throw-only exception** | § Throw vs log+return |

---

## Composed path or URL

**When:** Placeholders build one path / URL / similar composed string (segments joined in the template), not independent
filter keys.

**Rule:** Preserve the original event meaning. Prefer the **whole** string in `message` and/or one field. Do **not**
rename the event or split path segments into separate fields unless each segment is independently useful to filter on —
if unsure, ask ([user-decisions.md](user-decisions.md) § Ambiguous meaning or field extraction).

_Before shape:_

```java
log.debug("…prose… {}/…/{}/…/{}", varA, varB, varC);
```

_After shape (default):_

```java
String composed = String.format("%s/…/%s/…/%s", varA, varB, varC);
log.atDebug()
        .setMessage("…same prose… " + composed)   // same intent as before
        .addKeyValue("semantic_composed", composed)
        .log();
```

(Or shorter fixed prose + one `semantic_composed` field — still the same intent; do not invent a different outcome.)

---

## Fixed allowed values

**When:** A template mixes one runtime diagnostic with fixed literals / enums that only name allowed values
(`Only {}, {} or {}`, `expected {}`, etc.).

**Rule:** Extract the **event-varying** value(s) as fields. Keep allowed constants in the **message** via format/concat
(so renames/`toString` stay correct) — do **not** add a field per constant.

_Before shape:_

```java
log.error("… invalid status={}. Only {}, {} or {} …", status, EnumA.A, EnumA.B, EnumA.C);
```

_After shape:_

```java
log.atError()
        .setMessage(String.format("… invalid status. Only %s, %s or %s …", EnumA.A, EnumA.B, EnumA.C))
        .addKeyValue("status", status)
        .log();
```

**Not:** `.addKeyValue("a_status", EnumA.A).addKeyValue("b_status", EnumA.B)…` — those keys never vary and are not useful
filters. Prefer `String.format` / concat with the constants over hard-coding `"A, B or C"` in the string. Name the
event field from message semantics (`status`, not a wrong noun like the surrounding entity type alone).

---

## Split log vs API text (structure at logging boundary)

**When:** `log.*(msg)` where the same `msg` / `message` is also passed to an API consumer (`Response.entity`, DTO setter,
`throw`, error return, etc.).

**Rule:** Keep the **same string** for non-log consumers. Add structured fields **only** on the stdout log line.

_Before shape:_

```java
String msg = …build formatted or conditional string…;
log.error(msg);
consumer.accept(msg);   // HTTP body, DTO, throw arg, etc.
```

_After shape:_

```java
String msg = …unchanged build logic…;
log.atError()
        .setMessage(msg)                       // same variable as consumer
        .addKeyValue("semantic_a", varA)       // from variables in scope here
        .addKeyValue("semantic_b", varB)
        .log();
consumer.accept(msg);   // unchanged
```

| Layer | Change |
| ----- | ------ |
| `msg` / `message` variable and consumer | **Unchanged** |
| Log | Fluent API + semantic `addKeyValue` |

**Incomplete:** `log.atError().setMessage(msg).log()` with **no** `addKeyValue` — cosmetics only; diagnostics still only
in `message`. Always add fields from variables in scope (here: whatever was formatted into `msg`).

Use a **short fixed `setMessage` only when** it does not replace conditional or formatted text the original log emitted
**and** those diagnostics are also attached as fields. When `msg` is shared with a consumer, keep `.setMessage(msg)` and
still add the fields.

---

## Conditional message building

**When:** `message` / `msg` is built with `if`, ternary, or `String.format` branches before `log.*(message)`.

**Rule:** Build `message` **once**. Use that **exact string** for both the consumer and `setMessage(message)`. Add
`addKeyValue` for fields already in scope — do not drop branches or invent a shorter log-only summary.

_Before shape:_

```java
String message = condition
        ? String.format("…text without detail…", context)
        : String.format("…text with detail…", context, detailVar);
log.error(message);
consumer.accept(message);
```

_After shape:_

```java
String message = …same condition and format logic as before…;
var logBuilder = log.atError()
        .setMessage(message)                   // not a different summary string
        .addKeyValue("semantic_context", context)
        .setCause(e);                          // when original had a throwable in scope
if (/* detail applies — same condition as the "with detail" branch */) {
    logBuilder.addKeyValue("semantic_detail", detailVar);
}
logBuilder.log();
consumer.accept(message);
```

Extracting a private `build…Message(context, detailVar)` helper is fine when the same conditional appears multiple times
— it must preserve the **same branches and format strings** as before.

| Pitfall | Wrong | Right |
| ------- | ----- | ----- |
| Empty / absent detail | Fixed short `setMessage` + null field | `setMessage(message)` from same builder as consumer |
| Optional detail field | Always emit a field | Emit only when the original branch included that detail |

---

## Throw vs log+return

| Pattern | Log migration? | Why |
| ------- | -------------- | --- |
| `throw new SomeException(…formatted…)` only | **No** at throw site | Logging at exception mapper / filter |
| `log.error(msg);` then consumer/`return` | **Yes** at log line | Stdout separate from API body |
| `log.error(msg); throw …(msg)` | **Yes** at log line | Keep `msg` for `throw`; structure log only |

Do not add a second structured log before `throw` if the mapper already logs the same failure.

---

## Partial fluent helper (mapper shared fields)

**When:** Many call sites share the same **field block** (e.g. exception mappers). User chooses a helper that enriches a
builder — not a full log wrapper. Caller still owns level, `setCause`, and extra fields.

_Helper shape:_

```java
static LoggingEventBuilder withSharedFields(
        LoggingEventBuilder builder,
        …typed args for each repeated field…) {
    return builder
            .setMessage("…fixed summary or passed-in summary…")
            .addKeyValue("field_a", argA)
            .addKeyValue("field_b", argB);
}
```

_Call site shape:_

```java
withSharedFields(log.atWarn(), …args…)
        .addKeyValue("optional_extra", extra)   // site-specific
        .setCause(e)                            // when needed
        .log();
```

Name the helper for the **domain** (`withSharedExceptionFields`, `withFailureFields`, …) — do not copy a pilot name
unless it fits the target repo.

---

## Template constant without `{}` (prose-only constant)

**When:** A `private static final String SHARED_TEMPLATE = "…{}…"` is still used as an SLF4J template. User chooses **keep
constant, remove placeholders**.

_Constant shape:_ fixed prose, no `{}`.

_Call site shape:_

```java
log.atLevel()
        .setMessage(SHARED_TEMPLATE)
        .addKeyValue("semantic_a", varA)    // one field per former placeholder
        .addKeyValue("semantic_b", varB)
        .log();
```

---

## `log.*(throwable.getMessage())` only — prose-only / no change

**When:** The **only** diagnostic at the log site is `throwable.getMessage()` — no other variables in scope to structure.

**Rule:** **No code change required** for NDJSON (stage 1 envelope already adds `time`, `level`, `message`). Fluent API
with `.setMessage(e.getMessage())` alone is optional style — record as `static/no-action` in the report.

Do not parse `getMessage()` into invented fields.

---

## Go: drop-`f` / printf still burying diagnostics

**When:** Diagnostics remain inside the log string — either residual printf on `log.Info`/`Error`, or a
`fmt.Sprintf` / string build followed by `log.X("%s", msg)`.

**Incomplete (goal unmet — do not ship as migrated):**

```go
log.Error("operation failed key=%v error=%v", key, err)

msg := fmt.Sprintf("operation failed key=%s error=%s", key, err)
log.Error("%s", msg)
```

**Prefer** — first-class fields (`WithFields`) when available; else a repo helper:

```go
log.Error(logfields.Err("operation failed", err, "resource_key", key))
// or, if the API requires a format string:
log.ErrorC(ctx, "%s", logfields.Err("operation failed", err, "resource_key", key))
```

A single `"%s"` wrapper around the **helper** is OK. Pre-baking diagnostics into `msg` then logging `"%s"` is not.
Details: [go-qubership-lib.md](go-qubership-lib.md).

---

## Report line (user-confirmed)

```markdown
| log.error(message) | N | path/Service.java | **structure at boundary** — consumer text unchanged; setMessage(same var); fields added |
| log.debug(e.getMessage()) | N | path/Migration.java | **prose-only / no change** |
```
