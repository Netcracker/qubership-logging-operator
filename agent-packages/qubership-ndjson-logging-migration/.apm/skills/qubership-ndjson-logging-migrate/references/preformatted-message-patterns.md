# Logged Preformatted Message Patterns

Inventory searches (workflow inventory step and again at the completion gate). Finds logger calls that pass a variable
or prebuilt string instead of a string literal — separate from returned `fmt.Errorf` / wrapped exceptions.

**After inventory:** classify and ask — [user-decisions.md](user-decisions.md). Confirmed shapes —
[pattern-recipes.md](pattern-recipes.md). Do not invent policy here.

## Go

```bash
# Formatted log calls (migrate to WithFields + literal message) — include Trace
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)f\(' --include='*.go' .

# Variable passed as message (logged preformatted)
grep -rnE 'log\.(Trace|Debug|Info|Warn|Error|Fatal|Panic)\([^"'\'']' --include='*.go' .
```

Exclude `_test.go` and `dev/` from production counts unless tests emit runtime logs. Ignore commented lines in review.

## Java / SLF4J / Quarkus

```bash
# {} interpolation in production sources
grep -rnE 'log\.(info|debug|warn|error|trace)\([^)]*\{' --include='*.java' src/main/java

# Variable or expression as sole message argument (no literal string)
grep -rnE 'log\.(info|debug|warn|error|trace)\(\s*[^"'\''{]' --include='*.java' src/main/java

# Common preformatted patterns
grep -rnE 'log\.(warn|error)\((message|msg|errorMsg|aggregatedError)' --include='*.java' .
grep -rnE '\.getMessage\(\)' --include='*.java' src/main/java | grep -E 'log\.(info|debug|warn|error)'

# Shared {} template constants (misleading zero — ask immediately; see user-decisions.md)
grep -rnE 'WARNING_MESSAGE|MESSAGE_[A-Z_]+\s*=\s*".*\{}' --include='*.java' src/main/java
```

## Python

```bash
grep -rnE 'logger\.(debug|info|warning|error|critical)\([^f"'\'']' --include='*.py' .
grep -rnE 'logger\.(debug|info|warning|error|critical)\(f"' --include='*.py' .
```

## Common patterns

| Pattern                                 | Typical locations                                |
| --------------------------------------- | ------------------------------------------------ |
| `log.warn(message)` / `log.error(msg)`  | Service classes passing a variable built earlier |
| `log.error(aggregatedError)`            | Controllers aggregating validation errors        |
| Text-block summary logged as one string | Backup/restore or batch job services             |

List every hit under `User decision — logged preformatted messages` with file, count, and one example line.

## Report template

```markdown
## User decision — logged preformatted messages

| Pattern | Count | Example files | Decision |
|---------|-------|---------------|----------|
| log.warn(message) | 3 | path/File.java:142 | structure at boundary / prose-only / blocked |
```
