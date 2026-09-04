# Fluent pipeline tests

These tests run the Fluent Bit and Fluentd configurations from the current checkout against representative log files.
They compare each processed record with a checked-in JSON result identified by `logId`.

The comparator first looks for an extracted `logId`. If a parser keeps the marker inside the message, the comparator
uses `[logId=<value>]` there instead. As a final fallback, it uses the fixture's unique, fixed `time`. In the fallback
cases, `logId` is test metadata and is not expected to be present in the processed record.

The `fluentbit` and `fluentbit-ha` scenarios also run every parser from the daemon set and forwarder `parsers.conf`
files in isolation. Parser cases live in `testdata/parser-cases.json`; every parser has one matching and one
non-matching source line. The runner adds `test_case` after parsing, so test identifiers never change the input being
tested. Cases for parsers that are not present in a specific rendered configuration are skipped in that scenario.

Parser contract expectations are partial. `expected` lists fields that must be present, while `absent` lists fields
that the parser must not produce. This keeps the expected result focused on the parser contract instead of generated
host and pipeline metadata.

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

| Variable                     | Default                                         |
| ---------------------------- | ----------------------------------------------- |
| `FLUENTBIT_IMAGE`            | `docker.io/fluent/fluent-bit:5.1.0`             |
| `FLUENTD_IMAGE`              | `ghcr.io/netcracker/qubership-fluentd:1.19.3-1` |
| `FLUENT_PIPELINE_TEST_IMAGE` | `qubership-fluent-pipeline-tests:local`         |
| `HELPER_USER`                | `$(id -u):$(id -g)`                             |
| `CFG_TIMEOUT`                | `2` seconds                                     |
| `PARSE_TIMEOUT`              | `20` seconds                                    |
| `PARSER_CONTRACT_TIMEOUT`    | `5` seconds                                     |
| `INT_TESTS_IGNORE`           | Empty                                           |

The runner starts the helper container as `HELPER_USER`, so the rendered configuration and the generated container logs
belong to the calling user. The logging agents run as root and read those files without extra permissions.

## Add a test case

1. Add a CRI-formatted source record under `testdata/logs/containers` or a system input under `testdata/input`.
2. Give the record a unique `logId`. Prefer a field that survives processing; otherwise, retain the marker in the
   message and use a unique, fixed `time`.
3. Add the expected JSON record to the matching file under `testdata/output/fluentbit`,
   `testdata/output/fluentbit-ha`, or `testdata/output/fluentd`.
4. Run every affected scenario locally.

Do not replace expected files with actual output without reviewing each changed field. A broad golden-file update can
hide a pipeline regression.

## Add or change a parser

Add both a matching and a non-matching case to `testdata/parser-cases.json`. Set `match` to `true` or `false`, describe
the significant parsed fields in `expected`, and list fields that would indicate an incorrect match in `absent`.
`TestManifestCoversEveryFluentBitParser` reports a missing pair when `parsers.conf` gains a parser.
