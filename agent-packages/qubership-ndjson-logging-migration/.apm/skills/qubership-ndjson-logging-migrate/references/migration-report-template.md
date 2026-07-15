# Migration Report Template

ALWAYS create or update `.ndjson-migration-report.md` at the **root of the target worktree** during the migration run.
Use this structure; leave N/A rows explicit rather than omitting them.

## Lifecycle (not part of the product PR)

| Phase                  | Report in worktree?                         | Commit / upstream PR?                                                                            |
| ---------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Migration run          | **Yes** — coverage ledger and gate evidence | No — working artifact                                                                            |
| Resume across sessions | Yes — update in place                       | Untracked is fine                                                                                |
| Final product PR       | —                                           | **Exclude** `.ndjson-migration-report.md` unless the team explicitly wants an audit file in-repo |

Before opening or updating a product PR, drop the report from the commit (`git restore --staged` / omit from `git add`).
Copy it to the eval workspace or keep a local copy if you need an audit trail. Summarize completion gates and coverage in
the PR description instead.

```markdown
# NDJSON Logging Migration Report — <repo-name>

| Field | Value |
|-------|-------|
| **Run start HEAD** | `<git rev-parse HEAD at run start>` |
| **Branch** | `<branch>` |
| **Skill** | `qubership-ndjson-logging-migrate` |
| **Stage** | migrate (stage 2) |
| **Date** | YYYY-MM-DD |

## Deployable components

| Component | Path | Stack | Log config | Status |
|-----------|------|-------|------------|--------|
| ... | ... | ... | ... | migrated / blocked / pending |

## Completion gates

Run manual greps and builds from [completion-gates.md](completion-gates.md) per component.

| Gate | Command / check | Before | After | PASS |
|------|-----------------|--------|-------|------|
| Java compile | `mvn -pl <module> compile` | | exit 0 / BLOCKED | |
| Go build | `GOWORK=off go build ./...` | | exit 0 | |
| Java `{}` inline | grep `src/main/java` | | 0 | |
| Java field names | semantic review + spot-check; optional `"arg[0-9]"` grep | | OK | |
| Go `log.*f` (production) | grep non-test `.go` | | 0 | |
| Throwables | manual sweep | | fixed | |
| Integrity | git diff review | | no stray deletions | |
| Smoke NDJSON | captured stdout line → JSON with time/level/message | | OK | |

## User decision — logged preformatted messages

| Pattern | Count | Example files | Decision |
|---------|-------|---------------|----------|
| log.warn(message) | | | structure / prose-only / blocked |

## User decision — returned diagnostics

| Pattern | Count | Example files | Decision |
|---------|-------|---------------|----------|
| fmt.Errorf with embedded fields | | | keep at boundary / typed error / blocked |

## Blocked validation

| Component | Command | Error |
|-----------|---------|-------|
| | | |

## Validation commands

| Command | Result |
|---------|--------|
| | |

## Lessons (target-specific)

1. ...
```

Record **blocked** with the exact error when Maven or private registry auth prevents compile — do not mark the Java
component migrated-complete.
