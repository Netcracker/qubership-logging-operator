# logging-l1-triage

L1 triage pipeline for incoming Qubership logging-stack support tickets
(Graylog, FluentD, FluentBit, OpenSearch, MongoDB, the logging operator,
the Helm chart, and the Ansible installer). It runs in two coordinated
steps: first classify the ticket, then decide what to do with it.

The pipeline is **read-only against live systems**. It does not run
`kubectl`, SSH, or any diagnostic command; it does not change
configuration; and it never mutates or closes the ticket. State-changing
work belongs to L2.

## Skills

| Skill                                                                         | Step     | What it does                                                                                                                                                                                                  |
| ----------------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [`logging-l1-classification`](.apm/skills/logging-l1-classification/SKILL.md) | Classify | Reads the ticket and emits one `intent` (`problem` / `consultation` / `feature_request` / `security` / `unknown`) plus a uniform localization — `component`, `platform`, `phase`, and a `symptom` or `topic`. |
| [`logging-l1-outcome`](.apm/skills/logging-l1-outcome/SKILL.md)               | Dispose  | Takes the classified ticket and picks one disposition: `additional_info_required`, `suspected_known_issue` (cause + draft reply), or `handoff_to_l2` (structured packet).                                     |

`logging-l1-classification` runs first; `logging-l1-outcome` consumes its
verdict. The split is for clarity — the value the package delivers is the
end-to-end triage.

## Layout

```text
agent-packages/logging-l1-triage/
├── apm.yml
├── README.md
└── .apm/
    ├── instructions/
    │   └── logging-l1-triage.instructions.md   # both triggers, merged into AGENTS.md / CLAUDE.md
    └── skills/
        ├── logging-l1-classification/
        │   ├── SKILL.md
        │   ├── references/signals.txt          # signal dictionary
        │   └── scripts/extract_signals.py      # mechanical signal extractor (step 1)
        └── logging-l1-outcome/
            ├── SKILL.md
            ├── references/
            │   ├── rca-cases.md                # known cases: cause + draft reply
            │   ├── rca-cases.txt               # matcher patterns
            │   ├── facts-required.md           # field-ids needed per localization
            │   └── collection-howto.md         # how to collect each field-id, per platform
            └── scripts/match_rca.py            # mechanical rca-cases matcher (step 0)
```

## Install

```sh
apm install Netcracker/qubership-logging-operator/agent-packages/logging-l1-triage
apm compile
```

`apm compile` merges both triggers into your local `AGENTS.md` /
`CLAUDE.md`.

## Scope and limits

- One round of data requests. If the first `additional_info_required`
  round comes back still short, the pipeline hands off to L2 rather than
  interviewing further.
- Evidence-backed. Every fact passed on is quoted verbatim with its
  source; the pipeline uses only described `rca-cases` and never invents
  a cause or a mutating action.
- `unknown` is the only "not sure" value in classification — precision
  over recall, and it stops the localization drilldown for that axis.
