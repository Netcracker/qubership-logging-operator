---
name: logging-l1-classification
description: First-hop classification and localization of incoming Qubership logging-stack support tickets. Use when triaging what a ticket author is asking for (a problem on an installation, a consultation, a feature request, or a security/vulnerability remediation) and where it sits (which component, on VM or Kubernetes, at runtime or during deploy, and the specific symptom or topic). Decides from ticket content; uses the author's Ticket Type only as a tie-breaker and ignores Resolution/Root Cause. Research-phase skill; classifies only, takes no system action and answers nothing.
---

# logging-l1-classification

## What this does

For one logging-stack support ticket, decide from its **content**: (1) the
author's **intent**, then (2) a uniform **localization** drilldown. First hop of
an L1 pipeline; later hops (answering, remediation) are out of scope.

## Hard rules

- **No system access, no answers, no mutation.** Only label.
- **`unknown` is the only "not sure".** No confidence fields. Prefer `unknown`
  over a guess (precision over recall). `unknown` on an axis stops the drilldown.
- Ignore `Resolution` / `Root Cause` — not available at intake.

## Procedure — follow these four steps in order

**Step 1 — attach signals (mechanical).** Write `Summary` + `Description` +
`Steps to Reproduce` to a temp file and run the bundled extractor (use this
skill's base directory, shown when it loads):

```bash
python3 <skill-dir>/scripts/extract_signals.py /tmp/ticket.txt
```

It prints hits per namespace: `problem` / `consultation` / `feature_request` /
`security` (intent) and `component` / `platform` / `phase` / `symptom` / `topic`
(localization). In the research batch these are pre-attached to each record.

**Step 2 — read first, signals later.** Read `Summary` + `Description` + `Steps`
and form your own one-line read of what the author wants. Do NOT look at the
signal hits yet.

**Step 3 — reconcile with the signals.** Now compare your read to the hits and
adjust. Signals are strong evidence, but you weigh them differently per decision
(see Decision 1 and Decision 2 below).

**Step 4 — tie-break with `ticket_type` (last resort only).** If content plus
signals still leave it genuinely two-way, use the author's `ticket_type`
(`Defect` → problem, `Inquiry` → consultation, `Action Item` → neutral). Never
over a clear content signal.

## Decision 1 — intent

`problem | consultation | feature_request | security | unknown`. Pick by the
**dominant ask**:

- something is broken / failing / degraded now → `problem`
- a question — "how to", "is it possible", "where is" → `consultation`
- asking to add or change a product capability → `feature_request`
- asking to remediate a CVE / vulnerability / scan finding → `security`

Signal rule (asymmetric — this matters):

- A `consultation` / `feature_request` / `security` signal is **strong**: if one
  fired, take that intent seriously even if your first read said `problem`. These
  are the classes a quick read under-detects.
- A bare `problem` signal is **weak**: failure words ("error", "fail") appear
  inside questions too, so they do not by themselves make a `problem`. The form
  of the ask wins — a question is `consultation` even if it cites an error.
- Still two-way after content + signals → `ticket_type` (Step 4). Otherwise
  `unknown`.

## Decision 2 — localization (only when `intent != unknown`)

Walk `component → platform → phase → leaf`. **Here signals are a strong prior:**
if an axis signal fired and the text does not contradict it, use that value — do
NOT fall back to `unknown`. Use `unknown` only when no signal fired and the text
gives no clear cue.

- **`component`** — where in the stack: `graylog | fluentd | fluentbit |
  opensearch | operator | mongodb | events-reader | all | unknown`. The surface
  the author points at, not a root cause. A Ruby error/stacktrace ⇒ `fluentd`
  (Fluentd is Ruby; FluentBit is C). `all` = cross-cutting (common for
  `security`).
- **`platform`** — where it runs: `vm | kubernetes | unknown`. OpenShift / EKS =
  `kubernetes`. Forwarders and operator are effectively always `kubernetes`; the
  backend (graylog / mongodb / opensearch) may be either.
- **`phase`** — lifecycle: `runtime | deploy | unknown`. `deploy` = install /
  upgrade / restore (ansible `TASK [...]`, install/upgrade jobs,
  `external-logging-installer`). `operator` vs `deploy`: `operator` is a
  *component* (the running controller); `deploy` is a *phase*. They are
  orthogonal — "operator failed to reconcile during an upgrade" =
  `component=operator` + `phase=deploy`.
- **leaf:**
  - `problem` → **`symptom`** (the chief complaint, a surface not a root cause).
    Base: `not_running | no_data | performance | disk_space | oom_memory |
    data_correctness | auth_cert | config_error`. A component-specific symptom
    (e.g. `graylog.stream_index`, `opensearch.cluster_red_yellow`,
    `fluentd.buffer_overflow`) only when it matches the chosen `component`. A
    deploy-specific symptom (e.g. `deploy.prereq_check`, `deploy.image_unavailable`,
    `deploy.artifact_fetch`, `deploy.task_error`, `deploy.config_validation`) only
    when `phase == deploy`. Two apply → take the one the author leads with. Else
    `unknown`.
  - `consultation` / `feature_request` / `security` → **`topic`**:
    `output_target | retention | backup_restore | sizing_resources | integration |
    rbac_auth`, else `unknown`. `security` is usually `component=all`,
    `topic=unknown`.

## Output

Emit exactly one JSON object per ticket, matching the worklist `uid`, with
`rationale` first:

```json
{"uid": "PSUPCLOBS#12",
 "rationale": "Problem: author states 'graylog doesn't show logs'; no question or consultation/feature/security cue to override; ticket_type not needed.",
 "intent": "problem", "component": "graylog", "platform": "kubernetes",
 "phase": "runtime", "symptom": "no_data", "topic": ""}
```

- `rationale` (1–2 lines) comes first and traces the path: your content read →
  the signal that adjusted it (if any) → whether `ticket_type` broke a tie. **Quote
  the decisive cue verbatim** — the exact phrase that fixed the intent (and the
  cue for a localization axis when not obvious) — so the basis is visible, not
  paraphrased. It is the model's reasoning, formed before the labels.
- Localization fields are filled only when `intent != unknown`.
- A leaf is empty (`""`) when it does not apply to the intent — `symptom` for
  non-problem intents, `topic` for `problem`. `unknown` means the axis applies
  but the text did not determine it. There are no confidence fields.
