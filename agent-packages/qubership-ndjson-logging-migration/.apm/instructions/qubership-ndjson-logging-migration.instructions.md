---
description: >
  Trigger NDJSON logging skills — stage 1 (enable JSON output) or stage 2 (call-site field migration).
applyTo: "**/*"
---

When working on Qubership logging format rollout:

- **Stage 1** — enable NDJSON envelope only (`LOG_FORMAT`, logger/encoder config, Helm, smoke JSON line,
  no mass call-site rewrites): apply `qubership-ndjson-logging-enable`.
- **Stage 2** — complete migration (placement probe for top-level event fields, then move data out of `{}` / `log.*f` /
  bracket text into structured fields, inventories, semantic gates): apply `qubership-ndjson-logging-migrate` after
  stage 1 is done or explicitly scoped.

If the user does not specify a stage, infer from intent: “turn on JSON logging” / “default json per logging guide”
→ stage 1; “migrate log calls” / “extract fields from messages” → stage 2.
