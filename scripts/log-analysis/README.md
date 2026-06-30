# Log Storage Analysis

`log_storage_report.py` produces a JSON report for a Graylog or VictoriaLogs
backend. It collects current filesystem usage from VictoriaMetrics and runs
aggregated backend queries for log categories and normalized severity levels.

## Usage

VictoriaLogs example:

```bash
python scripts/log-analysis/log_storage_report.py \
  --backend-type victorialogs \
  --backend-url http://vlsingle-example.victorialogs:9428 \
  --vl-user user \
  --vl-pass password \
  --victoriametrics-url http://victoria-metrics:8428 \
  --vm-user vm-user \
  --vm-pass vm-password \
  --time-range 24h \
  --output victorialogs-report.json
```

Graylog example:

```bash
python scripts/log-analysis/log_storage_report.py \
  --backend-type graylog \
  --backend-url https://graylog.example.com \
  --graylog-user user \
  --graylog-pass password \
  --victoriametrics-url http://victoria-metrics:8428 \
  --vm-user vm-user \
  --vm-pass vm-password \
  --time-range 24h \
  --output graylog-report.json
```

Use `--time-offset` to analyze a completed window in the past instead of the
latest window. For example, this analyzes one hour of logs ending two hours ago:

```bash
python scripts/log-analysis/log_storage_report.py \
  --backend-type graylog \
  --backend-url https://graylog.example.com \
  --graylog-user user \
  --graylog-pass password \
  --victoriametrics-url http://victoria-metrics:8428 \
  --time-range 1h \
  --time-offset 2h \
  --output graylog-report.json
```

Use `--dry-run` to inspect the generated PromQL, LogsQL, or Graylog aggregate
requests without calling any backend.

Use `--html-output` to generate a self-contained HTML report for sharing with
customers:

```bash
python scripts/log-analysis/log_storage_report.py \
  --backend-type victorialogs \
  --backend-url http://vlsingle-example.victorialogs:9428 \
  --victoriametrics-url http://victoria-metrics:8428 \
  --time-range 24h \
  --output log-storage-report.json \
  --html-output log-storage-report.html
```

For VictoriaLogs, use `--include-vl-block-stats` to add diagnostic LogsQL
queries based on `block_stats` for top streams and fields by occupied disk
space. This option is disabled by default.

For VictoriaLogs, use `--include-detailed-levels` only when per-level
top-source sections are needed. By default, the report uses a faster level
overview. Graylog reports already include an aggregated levels section, so this
flag is rejected for `--backend-type graylog` to avoid a silent no-op.

For Graylog, independent report sections and per-stream aggregate requests run
in parallel by default. Use `--no-parallel-queries` or
`PARALLEL_QUERIES=false` on very large or sensitive environments when reducing
backend load is more important than report runtime. When parallel queries are
enabled, `--graylog-query-workers` / `GRAYLOG_QUERY_WORKERS` limits the total
number of concurrent Graylog HTTP requests. The default is `4`.

If VictoriaMetrics is protected by `vmauth`, pass its Basic Auth credentials
with `--vm-user` and `--vm-pass`.

All CLI options can also be provided through environment variables. Use
`log_storage_report.env.example` as a template:

```bash
cp scripts/log-analysis/log_storage_report.env.example log-storage.env
vi log-storage.env
set -a
. ./log-storage.env
set +a
python scripts/log-analysis/log_storage_report.py
```

The example file uses dotenv-style assignments without `export`. `set -a`
exports the sourced variables for the Python process, and `set +a` disables
that behavior afterwards.

The script uses only the Python standard library, so no `requirements.txt` is
needed for the current implementation.

## Outputs

The script always writes the JSON report to the required `--output` / `OUTPUT`
file path. Stdout output is intentionally disabled because the report can be
large.

When `--html-output` or `HTML_OUTPUT` is set, the script also writes a
self-contained HTML report with summary cards, tables for top lists,
skipped/error sections, and the raw JSON embedded at the bottom for debugging.

## Storage Metrics

For Graylog, filesystem capacity is queried from VictoriaMetrics through:

```promql
node_filesystem_size_bytes
node_filesystem_free_bytes
node_filesystem_avail_bytes
```

These metrics must be narrowed to the relevant OpenSearch or Graylog journal
filesystem to be meaningful. Use a PromQL selector such as:

```bash
--filesystem-selector 'instance="logging-node:9100",mountpoint="/data"'
```

For Graylog, the storage section contains calculated `filesystem_usage` rows.
Size values are rendered in GB fields such as `size_gb`, `available_gb`,
`free_gb`, and `used_gb`.

VictoriaMetrics storage queries are evaluated at the end of the selected
analysis window. With `--time-range 1h --time-offset 2h`, the report queries
logs for the completed window ending two hours ago and asks VictoriaMetrics for
metric values at that same window end.

For VictoriaLogs, the report uses native VictoriaLogs metrics instead of
`node_filesystem*`:

```promql
vl_total_disk_space_bytes
vl_free_disk_space_bytes
sum(vl_uncompressed_data_size_bytes)
sum(vl_compressed_data_size_bytes)
sum(vl_uncompressed_data_size_bytes) by (type)
sum(vl_compressed_data_size_bytes) by (type)
```

It renders `victorialogs_disk_usage` with `total_gb`, `free_gb`, `used_gb`,
and `used_percent`, and `victorialogs_data_size` with total and per-type
compressed/uncompressed data size.

VictoriaLogs data types:

- `storage/inmemory`: recently ingested data still held in memory.
- `storage/small`: small on-disk parts created after flushing inmemory data.
- `storage/big`: merged larger on-disk parts used for long-term storage.

Labels for VictoriaLogs disk capacity are reduced to useful identifiers only:
`cluster`, `cluster_name`, `job`, `instance`, `path`, `namespace`, `pod`, and
`service`. Standalone VictoriaLogs usually has `job`, `instance`, and `path`;
Kubernetes scrapes can additionally add `namespace`, `pod`, and `service`.

Optional VictoriaLogs `block_stats` analysis can be enabled with:

```bash
--include-vl-block-stats
```

or:

```bash
INCLUDE_VL_BLOCK_STATS=true
```

It adds `storage.victorialogs_block_stats` with:

- `top_streams_by_disk_usage`: top `_stream` values by
  `values_bytes + bloom_bytes`.
- `top_fields_by_disk_usage`: top fields by `values_bytes + bloom_bytes`.

`block_stats` is a diagnostic LogsQL pipe. It is useful for investigating disk
usage, but the script runs it only when explicitly requested.

VictoriaLogs stores values for every log field in separate compressed data
blocks. `values_bytes` is the on-disk size of stored field values, while
`bloom_bytes` is the on-disk size of bloom-filter data used to skip irrelevant
data blocks during queries.

References:

- [VictoriaLogs storage overview](https://docs.victoriametrics.com/victorialogs/faq/#how-does-victorialogs-work)
- [block_stats pipe](https://docs.victoriametrics.com/victorialogs/logsql/#block_stats-pipe)

## Log Categories

The report uses four logical categories:

- `system`: `log_category=system`.
- `audit`: `log_category=audit` or `nc_audit_label=true`.
- `container`: `log_category=container`, excluding `kind=KubernetesEvent` and
  `nc_audit_label=true`.
- `k8s_events`: `kind=KubernetesEvent`.

New logs sent through the Fluent Bit output path that populates `log_category`
are assigned `log_category=k8s_events`. The report intentionally continues to
identify events by `kind=KubernetesEvent`, so it also covers older stored
records.

For each category it retrieves the total message count, top sources by message
count, and top sources by estimated whole-record size. Results are nested under
the category name in JSON, so `log_category` is represented by the parent key
rather than repeated in every top-list row.

For `container` category, top sources are grouped by `namespace` and the
configured `--source-field` value. The default source field is `container`.
For Graylog reports this value must be a simple field name containing only
letters, digits, and underscores because it is also used in Graylog query
predicates. VictoriaLogs reports additionally support dotted field names.

For `system` and `audit` categories, top sources use dedicated comma-separated
field lists:

```bash
--system-source-fields nodename
--audit-source-fields namespace,container
```

or:

```bash
SYSTEM_SOURCE_FIELDS=nodename
AUDIT_SOURCE_FIELDS=nodename,user.username,verb
```

Use fields that exist in the target environment. Audit logs may be emitted by
containers or by node/control-plane audit collectors, so `AUDIT_SOURCE_FIELDS`
should match the actual source shape. For container-emitted audit logs,
`namespace,container` is usually useful. For node/control-plane audit logs,
fields such as `nodename,user.username,verb` may be better. For `k8s_events`,
top sources are grouped by `namespace`, `involvedObjectKind`, and
`involvedObjectName`, because Kubernetes event records do not contain a
container field.

VictoriaLogs message-size tables use `_msg` length only. They do not estimate
the full parsed record size. Graylog size impact is derived from the built-in
`gl2_accounted_message_size` field.

The existing Graylog pipeline routes system, audit, and Kubernetes event logs
to streams but does not itself guarantee a `log_category` field. Graylog
category breakdown therefore also requires category enrichment to be added in
the same follow-up processing-rules task.

## Severity Levels

The report retrieves a fast level overview: total counts by actual `level`
values and top `level`/source combinations. For VictoriaLogs, detailed
per-level top-source queries are disabled by default and can be enabled with:

```bash
--include-detailed-levels
```

Detailed mode uses these normalized levels:

```text
emerg, alert, crit, err, warning, notice, info, debug
```

`err` is used instead of `error` because it is the normalized value produced by
the logging pipeline.

For Graylog, level filters also include GELF/syslog numeric values from `0` to
`7`, because Graylog commonly stores `level` as a number. For example, `info`
matches `level:6 OR level:info`, and `debug` matches
`level:7 OR level:debug OR level:trace`.

## Schema Quality

The `schema_quality` section shows top sources by max observed
`parse_field_count`. It helps find logs that expand into too many parsed
fields after Fluent Bit processing. Kubernetes events are excluded from this
section.

The detected-problems check uses `FIELDS_COUNT_THRESHOLD` to decide when max
`parse_field_count` should be highlighted. The threshold defaults to `20` and
can be changed with:

```bash
--fields-count-threshold 30
```

or:

```bash
FIELDS_COUNT_THRESHOLD=30
```

This section uses `parse_field_count`, which is produced by the Fluent Bit
pipeline after parsing/post-processing. Fluentd and older logging versions may
not add this field; in that case the `schema_quality` tables are expected to be
empty and no `Too many parsed fields` problem is reported.

## Detected Problems

The HTML report starts with a `Detected Problems` section. It is calculated
from already collected report data and highlights common log storage risks:

- Graylog records with large `gl2_accounted_message_size`.
  Default: `60 KB`.
  Configure with `--graylog-large-record-threshold-kb` or
  `GRAYLOG_LARGE_RECORD_THRESHOLD_KB`.
- VictoriaLogs messages with large `_msg`.
  Default: `60 KB`.
  Configure with `--vl-large-message-threshold-kb` or
  `VL_LARGE_MESSAGE_THRESHOLD_KB`.
- Error-level logs are too large a share of logs with `level`.
  Default: `10%`.
  Configure with `--error-level-percent-threshold` or
  `ERROR_LEVEL_PERCENT_THRESHOLD`.
- Debug/trace logs are present.
  Default: any matching log.
  There is no threshold.
- One container source produces too many logs.
  Default: `20%`.
  Configure with `--single-source-percent-threshold` or
  `SINGLE_SOURCE_PERCENT_THRESHOLD`.
- Source has too many parsed fields.
  Default: `20`.
  Configure with `--fields-count-threshold` or `FIELDS_COUNT_THRESHOLD`.

The Graylog large-message check is based on `max(gl2_accounted_message_size)`
per namespace/container source. The VictoriaLogs large-message check is based
on `max(len(_msg))`, so it covers only message payload size, not the full parsed
record. Threshold values accept either a plain KB value, for example `60`, or an
explicit unit such as `60KB` or `1MB`.

`LARGE_MESSAGE_THRESHOLD_KB` is still accepted as a deprecated fallback when
backend-specific threshold variables are not set.

## Collected Sections

The report collects:

- `namespace_logs`: total log count and top sources by `namespace` plus
  `--source-field`.
- `levels`: actual values found in `level` and top level/source combinations.
- `detailed_levels`: optional per-level top-source sections when
  `--include-detailed-levels` is set.
- `debug_trace`: top sources using `level=debug` or `level=trace`.
- `log_patterns`: top normalized `_msg` patterns calculated with
  `collapse_nums prettify`.
- `k8s_events`: Kubernetes events selected by `kind=KubernetesEvent`.
- `message_size`: VictoriaLogs uses `_msg` length.
- `categories`: VictoriaLogs log counts grouped by `log_category`.
- `schema_quality`: top sources by max `parse_field_count`.
- `storage.victorialogs_block_stats`: optional VictoriaLogs-only disk usage
  diagnostics.
