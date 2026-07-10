# Smoke Validation Checklist

Run at least one realistic startup or config-check path **after** call-site migration, not only unit tests. Record the exact command and result in the coverage ledger.

**Build before smoke:** For Java, `mvn compile` must pass (or be explicitly blocked) before claiming the component migrated. Smoke on non-compiling code is invalid.

## Go / logrus

```bash
GOWORK=off go test ./...
LOG_FORMAT=json go run . -check-config -config-path examples/config.yaml -log-level error 2>&1 | head -1
```

Confirm the captured line parses as JSON and includes `time`, `level`, and `message`. If the binary has no `-check-config`, use the closest documented startup path that emits logs.

When the repo lives inside a parent `go.work`, prefix tests and smoke with `GOWORK=off`.

Quick manual check:

```bash
LOG_FORMAT=json go run . -check-config ... 2>&1 | head -1 | python3 -c "import json,sys; r=json.load(sys.stdin); assert all(k in r for k in ('time','level','message'))"
```

## Java / Quarkus

```bash
# Required before claiming Java migration complete
mvn -pl <module> -am compile

# Optional when credentials available
mvn -pl <module> quarkus:dev   # or documented integration smoke
# Capture one stdout JSON line; confirm time/level/message fields
```

If Maven compile is blocked (private packages, 401), record under **Blocked validation** with the exact error — **do not mark the Java component migrated-complete** and do not claim JVM smoke passed.

After bulk edits, also run semantic quality greps from [completion-gates.md](completion-gates.md) (`argN` keys, illegal text blocks).

## Python

```bash
LOG_FORMAT=json python -m <app.module> --help   # or documented entrypoint
# Capture one log line; confirm NDJSON schema (time, level, message)
```

## Fixture-only edits (eval / single-file scope)

Build a representative logrus/zap JSON line from the migrated `WithFields` call and confirm it parses with `time`, `level`, `message`. Note in the report that full runtime smoke was out of scope.

## What to record in the migration report

| Field | Example |
|-------|---------|
| Command | `LOG_FORMAT=json go run . -check-config ...` |
| Result | PASS — single-line JSON with time/level/message |
| Validator | manual JSON parse — OK |

Unit tests alone do not satisfy the smoke requirement for full-repo migrations.
