# User Decision Points

Record every decision in the migration report's component row and decision
tables. Ask placement-probe failures immediately. Collect every other
unresolved decision during inventory or review, then present one batch at that
phase boundary.

## Event-field placement unsupported

When the [placement probe](placement-probe.md) **FAIL**s for a component (any language/stack), **stop bulk call-site
migration** for that component and ask **immediately**.

Set `Phase=placement`, `Status=blocked`, `Open decisions=1`, and `Next action`
to the exact user choice needed. Do not bulk-migrate call sites.

1. Show probe evidence: command, one redacted JSON line, PASS/FAIL reason, and local facts used for the recommendation
   (runtime version, logging deps, failure signature).
2. Present **one recommended** option (from local facts only — see placement-probe § Recommendation) plus the fixed
   alternatives below. Always invite a **user-provided** approach.
3. Do **not** implement a placement fix or continue call-site rewrites until the user chooses.
4. After the user picks: implement that choice only → **re-probe** until PASS (or leave the component `blocked` /
   `in-progress` if they chose defer / accept-unmet-goal).

| Option | Description |
| ------ | ----------- |
| **Add placement infrastructure** | Keep the preferred call-site field API; add/fix formatter, encoder, bridge, or `JsonProvider`-style hook so event fields become **top-level** JSON keys. |
| **Change logging backend / encoder** | Switch or reconfigure the logging stack so the field API is natively supported (e.g. Logback JSON encoder with structured args). |
| **Defer** | Leave component `blocked` / `in-progress`; no bulk call-site migration until placement is fixed later. |
| **Accept message-embedded fields** | Explicitly accept diagnostics only inside `message` for now. **Goal unmet** — do **not** mark the component `migrated`. |
| **User-provided** | Any approach the user describes; treat as first-class once stated. |

Do not offer new per-call `MDC.put` / `StructuredLog`-style wrappers for event fields as the default “fix.” Request-scoped
correlation MDC in filters stays allowed.

Record in the report: probe result, recommended option, user choice, and re-probe result.

## Inventory decision batch

During `inventory`, record unresolved shared `{}` templates, logged
preformatted messages, returned diagnostics, ambiguous extraction, and
sensitive response-body choices. When inventory completes, set
`Phase=awaiting_decisions`, set `Open decisions` to the unresolved count, and
ask one grouped question. Do not enter `migrating` until every row is resolved,
blocked, or covered by an explicit repo-wide policy.

## Returned diagnostics (API / error return paths)

When structured data lives in `fmt.Errorf`, wrapped exceptions, or other values returned across boundaries, ask the user:

- keep error text as-is; add structured fields only at the logging boundary;
- introduce a typed/wrapped error exposing fields;
- marshal selected context when logging;
- mark only that exact case as `blocked`.

Example: `fmt.Errorf("error opening config file %v : %+v", path, err)` — do not silently redesign the return shape.

## Ambiguous meaning or field extraction

Preserve **what the log meant** while extracting **independently useful** fields. When that trade-off is unclear,
**stop and ask** — do not invent a new summary or over-split a composed value.

Typical triggers (non-exhaustive):

- Placeholders assemble one URL / path / query / template (`…/{}/{}/…`) — operators usually need the **whole** string,
  not each segment as its own field.
- Some `{}` args are **fixed** allowed values (enums / literals); only the runtime value is a useful field.
- Unclear whether a value is a filter key vs prose-only detail.
- Extracting fields would force a shorter/different `message` that changes the event meaning.
- Several reasonable field layouts; none is obviously correct from the call site alone.

1. Show the original call (one example) and 2–3 concrete options (e.g. keep full composed string in `message` + optional
   single field; or also extract a true id field; or prose-only / no change).
2. Wait for the user (or a stated repo-wide policy). Record the decision in the report.
3. Do not change level or invent failure/success wording the original line did not express.

Defaults when the case is clear:

- **Composed path/URL:** build the full string once; keep meaning in `message`; add at most one field for the whole
  string. See [pattern-recipes.md](pattern-recipes.md) § Composed path or URL.
- **Fixed allowed values:** extract only the runtime diagnostic; keep allowed constants in `message` via
  format/concat (not as fields). See [pattern-recipes.md](pattern-recipes.md) § Fixed allowed values.

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

When inventory finds `log.warn(SHARED_TEMPLATE, …)`, `log.error(SOME_CONSTANT, …)`, or any `String` constant that still
contains `{}` and is used as an SLF4J message template:

1. **Stop implementation on that component** and queue during inventory and ask at the inventory decision boundary —
   before bulk edits, helper extraction, or claiming `{}` grep is zero.
2. In the question, name the constant, caller count, and one example file (e.g. `Helpers.java:SHARED_TEMPLATE`, N callers).
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

During review, fix clear defects without asking. For a genuine new semantic
ambiguity, add it to one review decision batch, return to
`awaiting_decisions`, and after the choice re-enter `migrating` followed by
gates and the required review loop.

## Semantic field names

Consumer-friendly `snake_case` from message semantics — not positional keys, leaked locals, or codemod residue
(`*_get_*`, `*_stream_*`). Validation: [completion-gates.md](completion-gates.md) §4.1 (residue greps are **blocking**
until polished; spot-check alone is not enough after bulk codemod).
