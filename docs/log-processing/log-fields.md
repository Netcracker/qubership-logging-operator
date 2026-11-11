# Log fields reference

FluentBit adds a fixed set of fields to every record it sends. It also reserves some field names for its own use and
renames payload fields that collide with them. Use this page to know which fields you can search on and which names
your services must not write.

This page describes FluentBit. For the formats that produce these fields, see [Supported log formats](log-formats.md).

## Fields in a container record

A record read from a container log carries the fields below, plus the fields the parser extracts from the line.

<!-- markdownlint-disable line-length -->
| Field                 | Value                                                                                                                                   |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `short_message`       | The message. Taken from the parsed message field, or the raw line when the parser produced none                                         |
| `level`               | The severity as a syslog name: `emerg`, `alert`, `crit`, `err`, `warning`, `notice`, `info`, or `debug`. See [Log levels](#log-levels)  |
| `detected_level`      | The severity as a Grafana name: `trace`, `debug`, `info`, `warn`, `error`, or `critical`. Used by VictoriaLogs and Loki-style queries   |
| `source_level`        | The level as the application wrote it, before normalization. A value the payload already sets is kept                                   |
| `parse_level_unknown` | `true` when the record has no level, or a level the pipeline does not recognize. Absent otherwise                                       |
| `parse_status`        | `success` when a parser matched the line, `failed` otherwise                                                                            |
| `parse_format`        | The format that matched. See the list in [Supported log formats](log-formats.md)                                                        |
| `namespace`           | The namespace of the pod                                                                                                                |
| `pod`                 | The pod name                                                                                                                            |
| `pod_id`              | The pod UID                                                                                                                             |
| `container`           | The container name                                                                                                                      |
| `labels`              | The pod labels                                                                                                                          |
| `stdio_stream`        | `stdout` or `stderr`, as reported by the container runtime                                                                              |
| `time`                | The time the container runtime wrote the line. The pipeline never replaces it with a timestamp from the line                            |
| `hostname`            | The FluentBit pod that sent the record. In the HA mode, the forwarder pod that read it                                                  |
| `nodename`            | The node the FluentBit pod runs on                                                                                                      |
| `nc_audit_label`      | `true` when the record is an audit record. Absent otherwise. See [Audit logs](log-categories.md#audit-logs)                             |
| `stream`              | The category of the record: `container`, `audit`, or `system`. See [Log categories](log-categories.md)                                  |
<!-- markdownlint-enable line-length -->

The parsers add fields of their own. For example, the Qubership parser adds `date` with the application timestamp, the
klog parser adds `klog_date`, `pid`, and `file`, and the ingress-nginx parser adds `method`, `path`, and `code`.

With the HTTP output enabled, `log_category` holds a copy of `stream`, for every output of the agent. See
[Log categories](log-categories.md) for how the category is chosen.

Records from the Cloud Events Reader are container records too. When the Cloud Events Reader is installed,
`namespace` holds the namespace of the object the event is about, and `pod`, `container`, and `labels` are removed.

### In Graylog

The GELF output sends `hostname` as the GELF host, so Graylog shows the FluentBit pod in its `source` field. Graylog
also turns each pod label into a field of its own, with `/` in the label name replaced by `_`.

## Fields in a host record

Records read from node files and the systemd journal differ from container records:

- They have no `namespace`, `pod`, `container`, `labels`, `stdio_stream`, `parse_status`, or `parse_format`.
- `tag` holds the path of the file the record came from, such as `/var/log/audit/audit.log`. Journal records have
  `tag: systemd`.
- They keep the raw text in `log`.
- Syslog records add `syslog_time`, `syslog_host`, `syslog_ident`, and `syslog_pid`. When the message inside is
  logfmt, its `time` key becomes `log_time`.
- Audit records always have `nc_audit_label: true`.
- `parse_level_unknown` is set only when the record has a level the pipeline does not recognize.

## Log levels

FluentBit normalizes every level to a syslog name in `level` and a Grafana name in `detected_level`. It keeps the
original in `source_level`. The decision uses the first character of the value, in any case:

| Original level starts with                  | `level`   | `detected_level` | Examples                    |
| ------------------------------------------- | --------- | ---------------- | --------------------------- |
| `0` or `p`                                  | `emerg`   | `critical`       | `PANIC`                     |
| `e`, followed by a letter other than `r`    | `emerg`   | `critical`       | `EMERG`, `EMERGENCY`        |
| `1`, `a`, `f`, or `s`                       | `alert`   | `critical`       | `ALERT`, `FATAL`, `SEVERE`  |
| `2` or `c`                                  | `crit`    | `critical`       | `CRIT`, `CRITICAL`          |
| `3`, `e` alone, or `er`                     | `err`     | `error`          | `ERROR`, `ERR`, klog `E`    |
| `4` or `w`                                  | `warning` | `warn`           | `WARN`, `WARNING`           |
| `5` or `n`                                  | `notice`  | `info`           | `NOTICE`                    |
| `6` or `i`                                  | `info`    | `info`           | `INFO`                      |
| `7`, `d`, or `v`                            | `debug`   | `debug`          | `DEBUG`, `VERBOSE`          |
| `t`                                         | `debug`   | `trace`          | `TRACE`                     |
| Anything else, or no level                  | `info`    | `info`           | `LOG`, `Metadata`           |

The last row also sets `parse_level_unknown: true`.

Because only the first character counts, levels outside the usual set can map unexpectedly. For example, the
PostgreSQL `STATEMENT` level becomes `alert`, and `CONTEXT` becomes `crit`.

## Internal fields

The pipeline uses these fields while it processes a record and removes them before the output:

| Field                   | Purpose                                                               |
| ----------------------- | --------------------------------------------------------------------- |
| `log`                   | The raw line. Removed from container records after parsing            |
| `log_parsed`            | The payload that the Kubernetes filter parsed                         |
| `_record_metadata`      | Holds the pipeline's own fields while payload fields are extracted    |
| `_parser_input`         | The copy of the line that the generic parsers read                    |
| `original_log`          | The raw line of a record that the Kubernetes filter parsed            |
| `logfmt_candidate`      | Marks a line that looks like logfmt                                   |
| `_logfmt_input`         | The copy of a logfmt candidate that the logfmt parser reads           |
| `__qubership_candidate` | Marks a line that matched the Qubership parser                        |

A service must not write these names. The pipeline discards a payload `_record_metadata` before it extracts the
payload, and moves a payload `_parser_input` to `parsed_parser_input`. A payload `__qubership_candidate` is removed
unless the raw line really is a Qubership line.

## Reserved fields

The pipeline owns the fields below. When a JSON payload or an annotation-selected parser produces one of these names,
FluentBit moves the payload value to the matching `parsed_` field and keeps its own value:

| Pipeline field  | Payload value moves to   |
| --------------- | ------------------------ |
| `namespace`     | `parsed_namespace`       |
| `pod`           | `parsed_pod`             |
| `pod_id`        | `parsed_pod_id`          |
| `container`     | `parsed_container`       |
| `stdio_stream`  | `parsed_stdio_stream`    |
| `source`        | `parsed_source`          |
| `labels`        | `parsed_labels`          |
| `log`           | `parsed_log`             |
| `time`          | `parsed_time`            |
| `parse_status`  | `parsed_parse_status`    |
| `_parser_input` | `parsed_parser_input`    |

`level` is an exception: a payload `level` becomes the record level, because it is the application's own severity.

Do not write the `parsed_` names in your payloads either; the pipeline uses them for these values.

## Field names to avoid

The protection above covers JSON payloads that the Kubernetes filter parses. Other fields are set by the pipeline
without that protection, so a payload value with the same name is replaced or changes how the record is handled:

| Field                         | What happens to a payload value                                                                   |
| ----------------------------- | ------------------------------------------------------------------------------------------------- |
| `hostname`, `nodename`        | Replaced by the agent's values                                                                    |
| Keys of `extraFields`         | Replaced by the configured values                                                                 |
| `stream`, `log_category`      | Replaced by the category. `log_category` is replaced only when the HTTP output is enabled         |
| `short_message`               | Used as the message instead of `msg` or `message`                                                 |
| `parse_format`                | Replaced when the pipeline detects a format; kept instead of `unknown` otherwise                  |
| `source_level`                | Kept as the original level, even if it differs from `level`                                       |
| `detected_level`              | Replaced by the normalized level                                                                  |
| `nc_audit_label`              | Marks the record as audit when it is `true`. See [Audit logs](log-categories.md#audit-logs)       |
| `kind`                        | Routes the record to the Kubernetes events category when it is `KubernetesEvent`                  |
| `logType`, `log_type`         | Route the record to the access or integration category. See [Log categories](log-categories.md)   |

Qubership `[key=value]` groups follow the same rule as JSON payloads, with no exception for `level`. A group that
names a field the record already has, such as `[pod=checkout]`, is stored as `parsed_pod`, and the record keeps its
own `pod`. The same applies to `hostname`, `nodename`, `stream`, `log_category`, `parse_format`, `detected_level`,
`source_level`, and `parse_level_unknown`, which the pipeline sets later. Use names that do not appear on this page.

Do not start new field names with `log_`. The pipeline uses this prefix for `log_parsed`, `log_category`, and
`log_time`. The `log_type` marker is the one exception, and it keeps its meaning.

## FluentD fields

FluentD sends a different set of fields:

- The message stays in `log`. There is no `short_message`, `parse_status`, `parse_format`, or `detected_level`.
- `level`, `source_level`, and `parse_level_unknown` follow the same idea as in FluentBit, but the mapping differs:
  `e` always maps to `err`, `t` maps to `debug`, and a missing level becomes `info`.
- `namespace`, `pod`, `container`, `labels`, `hostname`, and `kubernetes_host` describe the source.
- `nc_audit_label` is `true` or `false` on every container record.
- `stream` holds the category, and `log_category` replaces it when the HTTP output is enabled.
- `extraFields` keys replace payload fields with the same name.
