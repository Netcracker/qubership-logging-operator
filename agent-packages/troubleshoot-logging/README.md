# troubleshoot-logging

A single operational skill that diagnoses problems with Qubership Logging Operator (a Kubernetes operator that
deploys and configures log collection, processing, storage, and access components).

The skill is **read-only and advisory**. It does not run `kubectl`, SSH, or Ansible, and it never changes a system. It
reads a support ticket or pasted problem description plus attached evidence, matches the symptom against a curated
reference, and supplies a diagnosis, permitted remediation, and the data to collect when the match is uncertain.

## Contents

| Path | Purpose |
| ---- | ------- |
| [`SKILL.md`](.apm/skills/troubleshoot-logging/SKILL.md) | The diagnosis procedure. |
| [`references/troubleshooting.md`](.apm/skills/troubleshoot-logging/references/troubleshooting.md) | Symptom-indexed failure catalog. |
| [`scripts/show_cases.py`](.apm/skills/troubleshoot-logging/scripts/show_cases.py) | Symptom-catalog and section reader. |

The reference is also exposed at `docs/troubleshooting.md` in the repository root via a symlink.
