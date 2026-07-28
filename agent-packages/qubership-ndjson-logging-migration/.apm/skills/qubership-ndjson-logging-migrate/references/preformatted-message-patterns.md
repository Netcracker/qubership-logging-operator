# Inventory and Smell Checks

**All check commands live in [scripts/smell-checks.sh](../scripts/smell-checks.sh)** — run the script instead of
copying greps, so inventory runs, completion gates, and review-pass re-checks cannot drift apart:

```bash
<skill-dir>/scripts/smell-checks.sh <component-root>
```

It finds logger calls that pass a variable or prebuilt string, `{}` / printf templating, forbidden helpers, and codemod
residue — separate from returned `fmt.Errorf` / wrapped exceptions.

**After inventory:** classify and ask — [user-decisions.md](user-decisions.md). Confirmed shapes —
[pattern-recipes.md](pattern-recipes.md). Do not invent policy here.

## When to run

| Moment | Purpose |
| ------ | ------- |
| Workflow inventory step | find candidates |
| Completion gates 2.2 / 3 / 4.1 | verify targets ([completion-gates.md](completion-gates.md)) |
| Review-pass re-check | confirm fixes did not regress |

## Checks

| Id | Stack | Finds | At the gate |
| -- | ----- | ----- | ----------- |
| J1 | Java | `StructuredLog` / per-call `MDC.put` for event fields (forbidden) | 0 |
| J2 | Java | same-line SLF4J `{}` in log calls | 0 |
| J3 | Java | shared string constants still containing `{}` | 0 — any hit: queue for inventory decision batch |
| J4 | Java | preformatted message logs (`log.warn(message)`, `e.getMessage()`, …) | 0 unreviewed |
| J5 | Java | text-block logs `log.*("""` — open each hit for `{}` inside | 0 unreviewed |
| J6a/J6b | Java | codemod residue keys (`_get_`, `_stream_`, `e_get_message`, `argN`) | 0 — blocking (§4.1) |
| J7 | Java | illegal single-line text block on log lines | 0 |
| J8 | Java | variable/expression as sole message argument | 0 unreviewed |
| G1 | Go | `log.*f(` including Trace | 0 |
| G2 | Go | residual printf verbs on non-`f` log calls | 0 |
| G3 | Go | variable passed as message | 0 unreviewed |
| P1/P2 | Python | non-literal / f-string logger calls | 0 unreviewed |

Production scope (built into the script): Java = `src/main/java` subtrees; Go excludes `_test.go`, `dev/`, `vendor/`.
Ignore commented lines during review. Expected G3 hits: repo field-helper calls (`logfields.Format` / `Err` or
equivalent) are the **approved** pattern — classify as OK, do not rework.

## Misleading zeros

Clean checks are **necessary, never sufficient** — the goal is queryable top-level fields (SKILL.md § Goal):

- **Go drop-`f`:** G1 → 0 while `log.Error("… key=%v …", key, err)` remains. G2 catches the same-line case only —
  on any hit, review the whole file for multi-line concatenations.
- **Go Sprintf dodge:** `fmt.Sprintf(…)` / string build then `log.X("%s", msg)` — **no grep catches this**; review
  string builds feeding log calls. A single `"%s"` wrapper around a field helper is OK
  ([go-qubership-lib.md](go-qubership-lib.md)).
- **Java shared constants:** J2 → 0 while a `{}` constant still templates at runtime (J3) — queue for the inventory
  decision batch ([user-decisions.md](user-decisions.md) § Java shared `{}` template constants).
- **Java text blocks:** J2 misses `log.info(""" … {} … """)` — open every J5 hit.

## Common patterns

| Pattern                                 | Typical locations                                |
| --------------------------------------- | ------------------------------------------------ |
| `log.warn(message)` / `log.error(msg)`  | Service classes passing a variable built earlier |
| `log.error(aggregatedError)`            | Controllers aggregating validation errors        |
| Text-block summary logged as one string | Batch / job services                             |

List every unreviewed hit under `User decision — logged preformatted messages` with file, count, and one example line —
row format in [migration-report-template.md](migration-report-template.md).
