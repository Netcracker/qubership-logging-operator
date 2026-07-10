# Logged Preformatted Message Patterns

Use these searches during inventory (workflow step 5–6) and again at the **completion gate**. They find logger calls that pass a variable or prebuilt string instead of a string literal — separate from returned `fmt.Errorf` / wrapped exceptions.

## Go

```bash
# Formatted log calls (migrate to WithFields + literal message)
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)f\(' --include='*.go' .

# Variable passed as message (logged preformatted)
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)\([^"'\'']' --include='*.go' .

# logrus WithError only — usually OK; review if message is also variable
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error)f\(' --include='*_test.go' .
```

Exclude `_test.go` from production counts unless tests emit runtime logs.

## Java / SLF4J / Quarkus

```bash
# {} interpolation in production sources
grep -rnE 'log\.(info|debug|warn|error|trace)\([^)]*\{' --include='*.java' src/main/java

# Variable or expression as sole message argument (no literal string)
grep -rnE 'log\.(info|debug|warn|error|trace)\(\s*[^"'\''{]' --include='*.java' src/main/java

# Common preformatted patterns
grep -rnE 'log\.(warn|error)\((message|msg|errorMsg|aggregatedError)' --include='*.java' .
grep -rnE '\.getMessage\(\)' --include='*.java' src/main/java | grep -E 'log\.(info|debug|warn|error)'
```

## Python

```bash
grep -rnE 'logger\.(debug|info|warning|error|critical)\([^f"'\'']' --include='*.py' .
grep -rnE 'logger\.(debug|info|warning|error|critical)\(f"' --include='*.py' .
```

## Common patterns

| Pattern | Typical locations |
|---------|-------------------|
| `log.warn(message)` / `log.error(msg)` | Service classes passing a variable built earlier |
| `log.error(aggregatedError)` | Controllers aggregating validation errors |
| Text-block summary logged as one string | Backup/restore or batch job services |

List every hit under `User decision — logged preformatted messages` with file, count, and one example line. Do not classify as `static/no-action` without an explicit user choice.

## Report template

```markdown
## User decision — logged preformatted messages

| Pattern | Count | Example files | Decision |
|---------|-------|---------------|----------|
| log.warn(message) | 3 | path/File.java:142 | structure at boundary / prose-only / blocked |
```

## Semantic quality (completion gate — not optional)

Run after pattern gate; see [completion-gates.md](completion-gates.md).

```bash
# Placeholder field keys (must be 0 in src/main/java)
grep -rn '"arg[0-9]\+"' --include='*.java' src/main/java

# Illegal single-line Java text blocks after codemod
grep -rn '""" [^"]' --include='*.java' src/main/java | grep -i StructuredLog

# Shared template constants still using {} at runtime (review, not auto-zero)
grep -rn 'WARNING_MESSAGE\|String.*=.*"\{}"' --include='*.java' src/main/java
```

Manually verify: throwable overload when original SLF4J call passed an exception; no duplicate MDC key in one `StructuredLog` call.
