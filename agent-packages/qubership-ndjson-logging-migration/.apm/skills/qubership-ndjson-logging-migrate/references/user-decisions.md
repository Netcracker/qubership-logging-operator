# User Decision Points

Surface these **before** claiming completion. Inventory returned diagnostics and logged preformatted messages separately.

## Returned diagnostics (API / error return paths)

When structured data lives in `fmt.Errorf`, wrapped exceptions, or other values returned across boundaries, ask the user:

- keep error text as-is; add structured fields only at the logging boundary;
- introduce a typed/wrapped error exposing fields;
- marshal selected context when logging;
- mark only that exact case as `blocked`.

Example: `fmt.Errorf("error opening config file %v : %+v", path, err)` — do not silently redesign the return shape.

## Logged preformatted messages

Search patterns: [preformatted-message-patterns.md](preformatted-message-patterns.md).

Examples: `log.warn(message)`, `log.error(aggregatedError)`, `log.debug(e.getMessage())`, Java text-block summaries
logged as one string.

For each pattern:

1. Count and list in the report under `User decision — logged preformatted messages` (file, count, one example).
2. Ask unless the user already gave a repo-wide policy:
   - **structure at the logging boundary** — see [pattern-recipes.md](pattern-recipes.md) § Split log vs API text; **confirm
     with user before implementing**;
   - refactor string builder to expose fields;
   - accept prose-only `message` for that category (no code change when only `e.getMessage()` at site — see
     pattern-recipes § `e.getMessage()` only);
   - mark site/pattern `blocked`.
3. Do not classify as `static/no-action` without an explicit choice.

### Structure at logging boundary (user-confirmed)

Apply only after the user selects this option (or a repo-wide policy). Full shapes and pitfalls:
[pattern-recipes.md](pattern-recipes.md) § Split log vs API text and § Conditional message building.

Record in the report: `structure at boundary — API text unchanged; setMessage(same variable); fields added`.

If the session cannot wait for an answer, stop with the question list in the report — do not mark complete.

## One error record per failure

When an error is returned to a caller, log it at the handling boundary **or** return/wrap without logging — not both
unless layers emit distinct lifecycle events. If both layers currently log, ask which layer owns the error log.

## Response body / sensitive INFO logs

When an existing INFO log prints a full response body and migration would split or redact it, present this table and wait
for a choice:

| Option            | Description                                                              |
| ----------------- | ------------------------------------------------------------------------ |
| **Preserve INFO** | Keep full `body` at INFO with `body_length` (behavior-equivalent).       |
| **Redact**        | Mask sensitive portions at INFO; keep `body_length` and status.          |
| **Truncate**      | Prefix/suffix at INFO with `body_truncated=true` and full `body_length`. |
| **Move to DEBUG** | Status/size at INFO; full `body` only at DEBUG (explicit level change).  |
| **Block**         | Defer until security/ops review.                                         |

Mark affected sites `needs user decision` until answered.

## Java shared `{}` template constants

When inventory finds `log.warn(WARNING_MESSAGE, …)`, `log.error(SOME_TEMPLATE, …)`, or string constants still
containing `{}` used as SLF4J message templates (common in exception mappers):

1. **Stop implementation on that component** and ask the user **immediately** — before bulk edits, helper extraction, or
   claiming `{}` grep is zero.
2. In the question, name the constant, caller count, and one example file (e.g. `Utils.java:WARNING_MESSAGE`, 12 mappers).
3. Offer these choices (unless the user already stated a repo-wide policy in this session):
   - **Inline fluent API** — replace each call with `log.atWarn().setMessage("…").addKeyValue(...).log()`; constant
     becomes a fixed message or is removed.
   - **Partial fluent helper** — shared method that only adds the repeated field block to a `LoggingEventBuilder`;
     callers still own `.atWarn()` / `.atError()`, `.setCause`, and site-specific fields. See
     [pattern-recipes.md](pattern-recipes.md) § Partial fluent helper. Prefer this over a helper that logs for the
     caller.
   - **Prose-only constant** — constant holds fixed text with no `{}`; fields only via fluent API at call sites (confirm
     this matches mapper semantics).
   - **Blocked** — defer that mapper/pattern until reviewed.

   Do **not** offer or introduce a new per-call MDC / `StructuredLog`-style wrapper as the “centralized helper.”

Do not move `{}` into another constant, leave templating in place, or mark the Java component migrated-complete while
these sites await an answer. If the session cannot wait, stop with the question in the report — do not guess.

## Semantic field names

Consumer-friendly `snake_case` from message semantics — not positional keys or leaked locals. Validation:
[completion-gates.md](completion-gates.md) §4.1 (spot-check required; greps are optional heuristics only).
