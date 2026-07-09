---
description: Triggers for the logging-l1-triage skills — classifying an incoming logging-stack support ticket, then deciding the next action on it.
applyTo: "**/*"
---

## Skill trigger: `logging-l1-classification`

When classifying what the author of an incoming Qubership logging-stack support
ticket is asking for — a problem on a specific installation, a consultation
about an existing capability, a request for a new feature, or a
security/vulnerability remediation — apply the `logging-l1-classification`
skill. It decides from ticket content and emits one intent label plus a uniform
localization (component / platform / phase / symptom or topic). `unknown` is the
only "not sure" value. It classifies and localizes only: no system access, no
answers, no ticket mutation.

## Skill trigger: `logging-l1-outcome`

After a logging-stack ticket has been classified by `logging-l1-classification`,
apply the `logging-l1-outcome` skill to decide the next action: request missing
data, flag a suspected known issue with a draft reply, or hand a structured
packet to L2. It reads ticket content and attachments, takes no system action,
and never mutates or closes the ticket.
