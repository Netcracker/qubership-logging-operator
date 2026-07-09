---
name: logging-l1-outcome
description: First-hop disposition of a classified Qubership logging-stack support ticket. Use after logging-l1-classification has labeled a ticket, to decide the next action — ask the author for missing data (Additional Info Required), flag a suspected known issue with a draft reply (Suspected Known Issue), or hand a structured packet to L2 (Handoff to L2). Reads ticket content and attachments (logs, screenshots); never touches live systems, mutates the ticket, or closes it.
---

# logging-l1-outcome

## What this does

Take a ticket already classified by `logging-l1-classification` and decide ONE
operator-facing disposition. Do not re-classify. First hop of an L1 pipeline; L2
troubleshooting is a later hop.

## Hard rules

- **No system access, no mutation, no ticket closure.** Reading attachments and
  static docs is fine; querying or changing live systems is not.
  `suspected_known_issue` drafts a reply and recommends — the operator confirms
  and closes.
- **One round of data request.** If the first `additional_info_required` round is
  answered and data is still short, hand off — do not keep interviewing.
- **Evidence-backed.** Every fact you pass is quoted verbatim with its source (a
  log line, a one-line screenshot description). Use only described `rca-cases`;
  never invent a cause or a mutating action.

## Inputs

- The `logging-l1-classification` JSON: `intent, component, platform, phase, symptom,
  topic` — the verdict; do not revisit it.
- The ticket text: `Summary` + `Description` + `Steps to Reproduce`.
- Attachments: logs (text) and screenshots/images where you can read images.

## Reading evidence — what counts as present

When you check whether a required fact is present, read the ticket text and
attachments together. Two rules decide presence:

- **Inline evidence is the evidence.** A log line, stack trace, or console block
  pasted into the ticket (`{code}`, `{noformat}`, `Exception`, `Caused by`,
  `Traceback`) IS the "logs" fact. Treat it as present and read it directly.
- **A referenced attachment is a provided fact.** A Jira embed or attachment link
  (`!file.png!`, `[^file.log]`) or a prose reference ("attached", "screenshot",
  "во вложении") means the author supplied that fact. Treat it as present even
  when you cannot open the file. Do not re-request it.

## Step 0 — match known problems (mechanical)

Write the ticket text plus any attachment text to a temp file and run the bundled
matcher (use this skill's base directory, shown when it loads):

```bash
python3 <skill-dir>/scripts/match_rca.py /tmp/ticket.txt
```

It prints the ids of any `references/rca-cases.txt` patterns that fired. For each
matched id, read its cause and draft reply in `references/rca-cases.md`. A match
is a HINT — confirm the case truly applies (its caveat holds) before using it.

## Decision flow — stop at the first that fires

1. **Suspected known issue.** A matched `rca-cases` entry truly applies (its
   caveat holds) → `suspected_known_issue`: emit the probable cause and the draft
   reply. For `consultation`, a documentation answer found in the bundled docs
   counts here too.
2. **Additional info required.** No match → check the collected facts (applying
   the evidence rules above) against `references/facts-required.md` for this
   localization. For each required `field-id` still missing, read its collection
   steps for the ticket platform in `references/collection-howto.md` and weave them
   into the message to the author. → `additional_info_required`, listing only the
   missing field-ids. One round only.
3. **Handoff to L2.** Facts present, no L1 resolution → `handoff_to_l2`.

## Output

Emit one YAML block, `outcome` first. Pick the shape for the outcome.

`suspected_known_issue` — a suggestion the operator confirms:

```yaml
outcome: suspected_known_issue
case_id: index-read-only-after-disk-cleanup
cause: "Index left read-only after a disk fill; OpenSearch does not clear the flag."
draft_reply: |
  <reply to the author, in their language>
recommend: close_after_confirmation
```

`additional_info_required` — the missing field-ids, plus a ready message:

```yaml
outcome: additional_info_required
missing:
  - logging_version
  - configmap_fluent
message_to_author: |
  <a short message in the author's language that asks only for the missing fields
  AND, for each, the collection steps from collection-howto.md for this platform>
```

`handoff_to_l2` — the structured packet (routed by `logging-l2-triage`):

```yaml
outcome: handoff_to_l2
localization: {component: fluentbit, platform: kubernetes, phase: runtime, symptom: no_data}
facts:
  error_text: "connection timeout to tcp://graylog:12201"   # fluentbit.log
  affected_scope: "namespace payments"                       # Description
  logging_version: "14.6.0"                                  # Description
sources: [fluentbit.log]
```

Rules: `facts` are quoted verbatim with their source (the trailing `# source`
comment); a one-line description stands in for a screenshot. No binaries and no
`attachments` field — the raw files stay on the ticket for L2. Priority is L2's
call.
