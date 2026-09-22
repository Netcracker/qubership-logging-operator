# Fluent pipeline tests

These tests run the Fluent Bit and Fluentd configurations from the current checkout against representative log files.
They compare each processed record with a checked-in JSON result.

Every expected record carries a `_test` block, and the comparator uses it alone to find the output record it belongs
to. `id` names the record in the report. `matchOn` lists the expected fields whose values select the output record,
and defaults to `time`, the timestamp the container runtime wrote and the pipeline keeps. Give each fixture record a
timestamp no other fixture uses, and the default selects it. Records that reach the output without a `time` field
declare their own match fields:

```json
{
  "_test": { "id": "syslog-1", "matchOn": ["log"] },
  "log": "ExecSync for ... failed",
  "tag": "/var/log/syslog"
}
```

The fixtures carry no test-only markers. A marker inside a message reaches the pipeline as data: a `[key=value]`
marker becomes a field, which adds one to `parse_field_count` and can turn `parse_status` from `failed` into
`success`, so the expected records would describe the marker rather than the log line.

The `fluentbit` and `fluentbit-ha` scenarios also run every parser from the daemon set and forwarder `parsers.conf`
files in isolation. Parser cases live in `testdata/parser-cases.json`; every parser has one matching and one
non-matching source line. The runner adds `test_case` after parsing, so test identifiers never change the input being
tested. Cases for parsers that are not present in a specific rendered configuration are skipped in that scenario.

Parser contract expectations are partial: their generated `_test` block sets `partial` and matches on `test_case`.
`expected` lists fields that must be present, while `absent` lists fields that the parser must not produce. This
keeps the expected result focused on the parser contract instead of generated host and pipeline metadata.

The contract manifest covers 25 regular parsers. Existing end-to-end fixtures cover the two multiline parsers and CRI
partial-record concatenation. The isolated cases fill the previous content-format gaps for CoreDNS, Consul,
PostgreSQL, OpenSearch, Calico, RabbitMQ, and the input-only system and audit formats.

Two configuration details are intentional in the contract baseline:

- `rabbitmq` is valid as an isolated parser, but no production filter selects it.
- `mongodb_structured` is valid as an isolated parser, while the pipeline handles MongoDB records through generic JSON
  parsing followed by field renames.

The `syslog` and `varlogmessages` matching cases expect no `time` field. Their regular expressions capture a
19-character timestamp without a time-zone offset, while `Time_Format` requires `%z`; Fluent Bit therefore rejects
the parsed timestamp. The contract retains this behavior so a parser fix produces a focused expectation change.

## Scenarios

- `fluentbit` runs the Fluent Bit daemon set pipeline.
- `fluentbit-ha` runs the Fluent Bit forwarder and aggregator pipeline.
- `fluentd` runs the Fluentd daemon set pipeline.

The Fluent Bit scenarios validate container, system, and audit inputs. The current Fluentd baseline validates container
records; its system and audit inputs do not reach the test file output with the supported Fluentd image.

Each scenario performs three operations:

1. Render the agent configuration from the current `LoggingService` API and templates.
2. Run the logging agent in Docker and feed it the files from `testdata/input` and `testdata/logs`.
3. Compare the output with the JSON records in the matching directory under `testdata/output`.

The runner changes only the rendered file-input discovery interval from 60 seconds to 1 second. This keeps parser and
filter behavior unchanged while avoiding a one-minute wait for system and audit fixtures in every scenario.

The runner waits for what it needs instead of sleeping. It reads the agent log for the line that shows the inputs are
open before it appends the system and audit fixtures, and it reads the output file until it holds as many records as
the expected files and the count stops changing. The test custom resources therefore set `logLevel: info`, which
the readiness lines need, and flush every second so records do not wait in a buffer. A wait that runs out reports
what it waited for and the last count it saw; the comparison then names the missing records.

## Requirements

- Docker
- A Unix-like shell

## Run locally

Build the helper image from the repository root:

```bash
docker build \
  --tag qubership-fluent-pipeline-tests:local \
  --file test/fluent-pipeline/Dockerfile \
  .
```

Run one of the scenarios:

```bash
test/fluent-pipeline/run.sh fluentbit
test/fluent-pipeline/run.sh fluentbit-ha
test/fluent-pipeline/run.sh fluentd
```

The runner stores generated configuration and actual output in `build/fluent-pipeline` by default. Set
`TEST_CONTENT_PATH` to use another directory.

The following environment variables override the defaults:

| Variable                     | Default                                         | Meaning                         |
| ---------------------------- | ----------------------------------------------- | ------------------------------- |
| `FLUENTBIT_IMAGE`            | `docker.io/fluent/fluent-bit:5.1.0`             | Fluent Bit image under test     |
| `FLUENTD_IMAGE`              | `ghcr.io/netcracker/qubership-fluentd:1.19.3-1` | Fluentd image under test        |
| `FLUENT_PIPELINE_TEST_IMAGE` | `qubership-fluent-pipeline-tests:local`         | Helper image                    |
| `HELPER_USER`                | `$(id -u):$(id -g)`                             | User the helper runs as         |
| `STARTUP_TIMEOUT`            | `30`                                            | Seconds to wait for open inputs |
| `OUTPUT_TIMEOUT`             | `60`                                            | Seconds to wait for the records |
| `OUTPUT_SETTLE_POLLS`        | `3`                                             | Polls with an unchanged count   |
| `INT_TESTS_IGNORE`           | Empty                                           | Expected files to skip          |

The runner starts the helper container as `HELPER_USER`, so the rendered configuration and the generated container logs
belong to the calling user. The logging agents run as root and read those files without extra permissions.

## Add a test case

1. Add a CRI-formatted source record under `testdata/logs/containers` or a system input under `testdata/input`.
   Leave the message as the application writes it, and give the record a CRI timestamp that no other fixture uses.
2. Add the expected JSON record to the matching file under `testdata/output/fluentbit`,
   `testdata/output/fluentbit-ha`, or `testdata/output/fluentd`, and give it a `_test` block with an `id`.
3. Run the scenario and read the report. A record the pipeline strips the timestamp from is reported as not found;
   add `matchOn` with fields that survive processing, such as `log` for a syslog record.
4. Run every affected scenario locally.

Do not replace expected files with actual output without reviewing each changed field. A broad golden-file update can
hide a pipeline regression.

## Add or change a parser

Add both a matching and a non-matching case to `testdata/parser-cases.json`. Set `match` to `true` or `false`, describe
the significant parsed fields in `expected`, and list fields that would indicate an incorrect match in `absent`.
`TestManifestCoversEveryFluentBitParser` reports a missing pair when `parsers.conf` gains a parser, and
`TestManifestHasNoCaseForARemovedParser` reports the cases left behind when a parser is renamed or removed.
