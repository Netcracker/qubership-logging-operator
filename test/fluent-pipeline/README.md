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

A record the pipeline must drop is a probe: its `_test` block sets `dropped`, and the run fails when any output record
carries its match values. The run also fails when the output holds a record no expected record claims, so every
fixture line the pipeline keeps has to be described, and a pipeline that emits a record twice is caught.

```json
{ "_test": { "id": "coredns-empty-line", "dropped": true }, "time": "2026-09-03T10:20:40.250007468Z" }
```

The fixtures carry no test-only markers. A marker inside a message reaches the pipeline as data: a `[key=value]`
marker becomes a field, which adds one to `parse_field_count` and can turn `parse_status` from `failed` into
`success`, so the expected records would describe the marker rather than the log line.

The `fluentbit` and `fluentbit-ha` scenarios also read the Prometheus exporter of the agent that writes the output.
The `log_to_metrics` filters, which count parse errors and Calico SYN packets, reach no output file: they are
exported on port 2021. The scrape runs in the network namespace of the agent container and is compared with
`testdata/metrics/<scenario>.prom`. Those filters flush every 20 seconds, so the scrape is retried until it carries
every expected line.

The `fluentbit` and `fluentbit-ha` scenarios also run every parser of the rendered `parsers.conf` in isolation, once
per configuration the operator ships: the daemon set, the forwarder, and the aggregator. The three define different
sets of parsers, and several names carry different expressions in each, so a suite runs against one configuration and
its coverage counts for that configuration alone. Parser cases live in `testdata/parser-cases.json`; every parser has
one matching and one non-matching source line. The runner adds `test_case` after parsing, so test identifiers never
change the input being tested. Cases for parsers a configuration does not define are skipped in its suite, and the
generator names them in its log.

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
- `kube-metadata` runs the Fluent Bit daemon set with the real Kubernetes filter against a fake API server.
- `render` renders the configuration for every custom resource under `testdata/assets/render/` and asks the agent to
  validate it without processing any log.

All scenarios validate container, system, and audit inputs. Fluentd stamps its system and audit records with a
`fluentd_time` the fixtures cannot pin: the syslog parser takes the current year, because RFC 3164 carries none, and
audit records get the ingestion time. The helper's `-ignoreFluentdTime` flag names those expected files, and their
records leave `fluentd_time` out.

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

## Kube-metadata scenario

The other scenarios set `mockKubeData: true`, which replaces the Kubernetes filter with a filter that adds fixed
fields. That filter parses nothing, so two things the pipeline does in a cluster never happen: `Merge_Log`, which
feeds the parsed payload to the rest of the chain, and the `fluentbit.io/parser` annotation, which names the parser
for a pod. Records therefore reach the generic chain that would not reach it in a cluster.

The `kube-metadata` scenario runs the rendered configuration with `mockKubeData: false` and answers the filter with a
fake API server: the helper's `kube-api` stage serves the pod metadata under `testdata/kube-metadata/pod-metadata/`,
writes a self-signed certificate for `kubernetes.default.svc`, and the runner mounts that certificate as the agent's
service account directory and points the name at the server with an added host entry. A pod with no metadata file is
answered with a pod that declares no annotations.

Its fixtures live under `testdata/kube-metadata/logs/containers/`, separate from the other scenarios, and the pod name
of a fixture is its path below `containers`. That is what selects the route in filters that read the pod name, so a
directory named `kube-scheduler` reaches the tag rewrite that only Kubernetes system pods reach in a cluster.

## Render scenario

The three pipeline scenarios run one custom resource each, so the template branches they do not select, such as the
Docker runtime, OpenShift, the journald input, or the Loki, HTTP, and OpenTelemetry outputs, never render. The
`render` scenario covers them: for each custom resource under `testdata/assets/render/<agent>/`, the helper renders
the agent's templates the way the `prepare` stage does, and the agent's dry run (`fluent-bit --dry-run`,
`fluentd --dry-run`) loads the result. The forwarder and the aggregator templates both render for a custom
resource under `fluentbit-ha/`. The report lists one row per rendered configuration.

The custom resources take the shape the chart produces, with the multiline expressions and the output block the
chart always sets, and one custom resource per group of branches, so that a failed row points at an area. The
helper stands in for the operator where the templates need more than the custom resource: it fills the output
credentials the operator reads from Secrets with placeholders, and for Fluentd it supplies the environment of the
DaemonSet and the mounted service account and certificate files, which Fluentd opens during the dry run.

A custom resource that documents a known defect starts with three header lines: `# expect-failure: <reason>`,
`# expect-stage: render` or `# expect-stage: validation`, and `# expect-diagnostic: <extended regular expression>`.
The row passes only when the named stage is the one that failed and its log matches the expression, so that an
unrelated renderer or agent error fails the row instead of passing for the defect. A configuration that renders and
validates also fails the row, with a note to remove the three lines, so the row tells the person who fixes the
defect to update the expectation.

Fluent Bit refuses a configuration that includes a file no template rendered with a bare `configuration file
contains errors`, which names no file. The suite therefore lists the missing `@INCLUDE` targets in the log before
the dry run, and a fixture for such a defect matches that line.

Two limits of the validators: Fluent Bit's dry run does not open the TLS files an output names, and Fluentd's
`kubernetes_metadata` filter connects to the API server when it starts, so the Fluentd custom resources set
`mockKubeData` and the real-metadata branch renders without a Fluentd check.

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
test/fluent-pipeline/run.sh kube-metadata
test/fluent-pipeline/run.sh render
```

The runner stores generated configuration and actual output in `build/fluent-pipeline` by default. Set
`TEST_CONTENT_PATH` to use a new or empty directory. After the first run, a marker identifies the directory as owned
by the test runner, and later runs can safely replace its contents.

The agent images default to the ones `charts/qubership-logging-operator/templates/_helpers.tpl` deploys, read from
that file at run time: Renovate updates the chart and has no manager for shell files, so a second copy of the version
here would drift without anyone noticing.

The following environment variables override the defaults:

| Variable                     | Default                                 | Meaning                          |
| ---------------------------- | --------------------------------------- | -------------------------------- |
| `FLUENTBIT_IMAGE`            | the image the chart deploys             | Fluent Bit image under test      |
| `FLUENTD_IMAGE`              | the image the chart deploys             | Fluentd image under test         |
| `FLUENT_PIPELINE_TEST_IMAGE` | `qubership-fluent-pipeline-tests:local` | Helper image                     |
| `HELPER_USER`                | `$(id -u):$(id -g)`                     | User the helper runs as          |
| `STARTUP_TIMEOUT`            | `30`                                    | Seconds to wait for open inputs  |
| `METRICS_TIMEOUT`            | `60`                                    | Seconds to wait for the exporter |
| `OUTPUT_TIMEOUT`             | `60`                                    | Seconds to wait for the records  |
| `OUTPUT_SETTLE_POLLS`        | `3`                                     | Polls with an unchanged count    |
| `INT_TESTS_IGNORE`           | Empty                                   | Expected files to skip           |

The runner starts the helper container as `HELPER_USER`, so the rendered configuration and the generated container logs
belong to the calling user. The logging agents run as root and read those files without extra permissions.

## Add a test case

1. Add a CRI-formatted source record under `testdata/logs/containers` or a system input under `testdata/input`.
   Leave the message as the application writes it, and give the record a CRI timestamp that no other fixture uses.
2. Add the expected JSON record to the matching file under `testdata/output/fluentbit`,
   `testdata/output/fluentbit-ha`, or `testdata/output/fluentd`, and give it a `_test` block with an `id`.
3. Run the scenario and read the report. A record the pipeline strips the timestamp from is reported as not found;
   add `matchOn` with fields that survive processing, such as `log` for a syslog record.

A fixture for a third-party format, such as `text/single/coredns`, holds two lines from the same pod: one in the
format its parser expects and one that is not, so the run shows both that the pod name selects the parser and that
the parser leaves other lines alone. The pod name comes from the directory path, so `opensearch-0` matches the
`opensearch-\d{1,2}_` selector while `opensearch` would not.
4. Run every affected scenario locally.

A run that fails names the fields that moved, as `expected -> produced`, with a nested value under the path that
leads to it, so a mismatch reads without opening the records. The whole records are in the output file the run leaves
in `TEST_CONTENT_PATH`, next to the rendered configuration, the agent's own log, and the report of the run; the CI
workflow uploads that directory, together with the expected records, when a scenario fails, and puts the report on
the job summary.

The scenarios run as the `fluent_pipeline` job of the `Tests: Run logging integration tests` workflow, beside the
cluster tests, and answer to `Integration Gate` with them: a pipeline that stopped parsing a supported format fails
the acceptance of the change. The `changes` job of that workflow decides which of the two families a pull request
needs, so a change to the chart does not run the parsers and a change to a fixture does not start a cluster.

Do not replace expected files with actual output without reviewing each changed field. A broad golden-file update can
hide a pipeline regression.

## Add or change a parser

Add both a matching and a non-matching case to `testdata/parser-cases.json`. Set `match` to `true` or `false`, describe
the significant parsed fields in `expected`, and list fields that would indicate an incorrect match in `absent`.

The non-matching line is the nearest line the parser has to reject, not an unrelated one: it differs from the
matching line in the one element the regular expression requires, such as a lowercase severity letter for
`klog_entry` or `PROTO=UDP` for `calico_tcp`. A far-away line passes whether the expression is right or wrong. Where
a parser accepts a near miss, the case that records it is a matching one, as
`cassandra-timestamp-taken-as-method` does for a regular expression that takes a timestamp bracket for the method.
`TestManifestCoversEveryFluentBitParser` reports a missing pair when `parsers.conf` gains a parser, and
`TestManifestHasNoCaseForARemovedParser` reports the cases left behind when a parser is renamed or removed.
