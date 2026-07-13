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
   - structure at the logging boundary;
   - refactor string builder to expose fields;
   - accept prose-only `message` for that category;
   - mark site/pattern `blocked`.
3. Do not classify as `static/no-action` without an explicit choice.

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

## Semantic field names

Rename short local names (`i`, `i_1`, `sbe`, `qName`) to consumer-friendly `snake_case` (`reason_index`, `query_name`).
For validation loops, prefer one aggregate record or meaningful per-item records — not placeholder messages like `.`.
