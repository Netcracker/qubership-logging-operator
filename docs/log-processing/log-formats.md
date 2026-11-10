# Supported log formats

FluentBit recognizes three application log formats, the Kubernetes klog format, and the native formats of a set of
third-party components. A line in any other format still reaches the output, unparsed, with its raw text as the
message.

This page describes FluentBit, the default agent. FluentD parses far fewer formats; see
[FluentD differences](#fluentd-differences).

For the fields a parsed record carries, see [Log fields reference](log-fields.md). To change how logs are parsed, see
[Configure log parsing](configure-parsing.md).

## Application log formats

Write application logs in one of these formats to get structured fields in the log storage.

| Format      | `parse_format` | Use it when                                                         |
| ----------- | -------------- | ------------------------------------------------------------------- |
| Qubership   | `qubership`    | You write text logs and want a timestamp, a level, and named fields |
| JSON        | `json`         | Your logging library writes one JSON object per line                |
| logfmt      | `logfmt`       | Your logging library writes `key=value` pairs, as Go libraries do   |

### Qubership format

The Qubership format is a bracketed text format:

```text
[TIMESTAMP] [LEVEL] [key=value]... MESSAGE
```

Example:

```text
[2025-05-02T09:16:53.541Z] [WARN] [request_id=1682710824367.0.7602] [thread=kafka-producer] Broker disconnected
```

The parser accepts the following:

- `TIMESTAMP` is `YYYY-MM-DD`, then `T`, `t`, or a space, then `HH:MM:SS`. Fractional seconds with `.` or `,` and up to
  nine digits are optional. The time zone may be `Z`, `z`, `+HH`, `+HHMM`, or `+HH:MM` and is optional.
- `LEVEL` is one of `TRACE`, `DEBUG`, `INFO`, `WARN`, `WARNING`, `ERROR`, or `FATAL`, in any case. A line with another
  level, such as `NOTICE`, is not a Qubership record.
- Spaces between the brackets are optional, so `[2025-05-02T09:16:53.541Z][WARN]` also matches.
- `MESSAGE` is everything after the last field. It can span several lines; see [Multiline logs](#multiline-logs).

The pipeline turns the line into these fields:

- `date` holds the timestamp text exactly as written. The event time of the record stays the time the container
  runtime wrote the line; the pipeline does not parse the application timestamp.
- `level` holds the level, normalized as described in [Log levels](log-fields.md#log-levels).
- `short_message` holds the message. A line with no message gets `<Empty message>`.
- Each `[key=value]` group becomes a field named `key`.

The `[key=value]` groups follow these rules:

- The groups must come right after `[LEVEL]`. Parsing stops at the first bracketed group that is not `key=value`, so
  `[XNIO-1 task-1]` and everything after it become part of the message.
- A key cannot contain spaces, `[`, `]`, or `=`.
- A value can contain spaces and `=`, but not `[` or `]`. Escaping brackets is not supported.
- Leading and trailing spaces of a value are trimmed.
- A group with an empty value, or with a `"` in its key or value, is skipped.
- A key that names a field the pipeline sets, such as `namespace`, `pod`, or `level`, is stored as `parsed_<key>`.
  See [Field names to avoid](log-fields.md#field-names-to-avoid).

For the recommended set of fields, such as `request_id` and `tenant_id`, see
[Logging guidelines for applications](../cookbook/log-formats.md).

### JSON

A JSON record is a single line that holds one JSON object:

```json
{"time":"2025-05-02T09:16:53.018Z","level":"INFO","request_id":"9f09fc89","message":"Order created"}
```

The pipeline treats the line as JSON when it starts with `{"` after optional spaces and ends with `}`.

- Every top-level key becomes a field. Nested objects stay nested, so prefer flat objects: most log storages index
  nested fields poorly.
- `msg`, or else `message`, becomes `short_message` when it holds a non-empty string. Otherwise the raw line is the
  message.
- `level` becomes the record level.
- A `time` field in the payload does not change the event time. It is kept as `parsed_time`; see
  [Reserved fields](log-fields.md#reserved-fields).

A pretty-printed JSON object that spans several lines is not parsed as JSON.

### logfmt

A logfmt record is a line of `key=value` pairs separated by spaces, as described in
[the logfmt specification](https://brandur.org/logfmt):

```text
time=2025-05-02T09:16:53Z level=info msg="Order created" request_id=9f09fc89
```

- The whole line must consist of pairs. A key starts with a letter or `_`. A value with spaces must be quoted.
- A line that also contains a `[key=value]` group is not treated as logfmt.
- `msg`, or else `message`, becomes `short_message`.
- A `time` key does not change the event time.

### Lines in other formats

A line that no parser matches is still sent to the output:

- `parse_status` is `failed` and `parse_format` is `unknown`.
- `short_message` holds the raw line.
- If the line contains a bracketed level token such as `[ERROR]` or `[ warn ]`, that token becomes the level.
  Otherwise the level is `info` and `parse_level_unknown` is `true`.
- The `fluentbit_parse_error_total` metric counts the line, labeled with its namespace, pod, and container.

Empty lines and lines with only whitespace are dropped.

## Kubernetes system logs

Kubernetes components write logs in the klog format:

```text
I0502 09:16:53.541234       1 controller.go:123] Starting controller
```

FluentBit tries the klog parsers on every container log first, so any container that writes klog lines gets
`parse_format: klog`. Pods whose names start with `kube-` or `clickhouse-operator-` take a dedicated path: their
multiline klog entries, such as traces, are joined before parsing.

The parser stores the date in `klog_date`, the process ID in `pid`, and the source file and line in `file`. The level
letter becomes the level: `I` is `info`, `W` is `warning`, `E` is `err`, `F` is `alert`, and `V` is `debug`.

## Third-party components

FluentBit selects the parser of a third-party component by the pod name or the container name. The name only has to
contain the text in the table, and the namespace is never checked: a pod named `pg-patroni-node1` selects the
PostgreSQL parser, and a namespace named `pgadmin` does not.

<!-- markdownlint-disable line-length -->
| Component                    | Pod or container name contains                                        | `parse_format`       | Notes                                                                                                                                           |
| ---------------------------- | --------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| CoreDNS                      | `coredns`                                                             | `coredns`            | Extracts `module`, and `domain` and `record_type` for query logs                                                                                |
| ingress-nginx                | `nginx`                                                               | `k8s-nginx-ingress`  | Parses the default access log format: `method`, `path`, `code`, upstream fields. Other lines stay unparsed                                      |
| Apache Cassandra             | `cassandra`                                                           | `cassandra`          | Extracts `method`                                                                                                                               |
| Consul                       | `consul`                                                              | `consul`             |                                                                                                                                                 |
| PostgreSQL and Patroni       | `postgres`, `patroni`, or `pg`                                        | `postgres`           | PostgreSQL levels such as `LOG` are not standard levels; see [Log levels](log-fields.md#log-levels)                                             |
| Jaeger                       | `jaeger`                                                              | `json`               | Parsed as JSON; the `ts` field is removed                                                                                                       |
| MongoDB                      | `cnfrs` in the pod name, or `mongo` in a container of a `graylog` pod | `mongodb_structured` | Parsed as JSON. `ctx`, `c`, `s`, `attr`, `msg`, and `t.$date` become `context`, `component`, `level`, `attributes`, `short_message`, and `date` |
| Argo CD                      | `argocd-`, except `argocd-redis`                                      | `logfmt`             |                                                                                                                                                 |
| Fluent Bit                   | `fluentbit`                                                           | `fluentbit`          | Extracts `component`. The agent pods are not collected unless `selfLogging` is enabled                                                          |
| OpenSearch                   | `opensearch-<N>`, where `<N>` is one or two digits                    | `opensearch`         | Extracts `logger` and `opensearch_node`. The Qubership parser is skipped for these pods                                                         |
| Calico                       | `calico-node`                                                         | `unknown`            | Parses SYN packet logs and counts them in the `calico_syn_packets_total` metric                                                                 |
<!-- markdownlint-enable line-length -->

A component parser runs only on the pods it selects. When it does not match a line, the line goes on to the generic
parsers, so a JSON line from a PostgreSQL pod is still parsed as JSON.

## Host logs

FluentBit also reads logs from the node file system. See [Log categories](log-categories.md) for the inputs and the
settings that enable them.

| Source                                                     | Format                                                                                         |
| ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `/var/log/messages`, `/var/log/syslog`                     | Syslog in RFC 3164 or RFC 3339 layout. The message inside is then parsed as klog or logfmt     |
| systemd journal                                            | Structured journal entries; `MESSAGE` becomes the message                                      |
| `/var/log/audit/audit.log`                                 | Linux audit (`auditd`) records                                                                 |
| Kubernetes, kube-apiserver, and OpenShift audit log files  | Kubernetes audit events in JSON                                                                |

Syslog records carry `syslog_time`, `syslog_host`, `syslog_ident`, and `syslog_pid`. An RFC 3164 timestamp has no time
zone, so the record keeps the time the agent read the line.

Host records do not get `parse_status` or `parse_format`.

## How FluentBit chooses a parser

FluentBit tries the parsers in a fixed order and stops at the first one that matches:

1. **Pod annotation.** If the pod has the `fluentbit.io/parser` annotation, FluentBit uses the parser it names. See
   [Choose a parser for a pod](configure-parsing.md#choose-a-parser-for-a-pod).
2. **Embedded JSON.** Without the annotation, the Kubernetes filter parses the line if it is a JSON object.
3. **klog.** The klog parsers run on every remaining line.
4. **Component parsers.** CoreDNS, ingress-nginx, Cassandra, Consul, and PostgreSQL, each on the pods it selects.
5. **JSON.** The generic JSON parser.
6. **Argo CD, Fluent Bit, and OpenSearch,** each on the pods it selects.
7. **Qubership.** On every pod except OpenSearch pods.
8. **logfmt.** On every pod.
9. **Calico.** On `calico-node` pods.

In the HA mode, the aggregator runs the generic JSON parser before the Cassandra, Consul, and PostgreSQL parsers.

## Multiline logs

FluentBit joins the lines of one message into a single record before parsing. For container logs, it tries these rules
in order:

1. **Qubership.** A line that starts with `[YYYY-MM-DD` opens a record. Every following line that does not start with
   `[YYYY-MM-DD` or `{` is appended. You can change both expressions; see
   [Change multiline rules](configure-parsing.md#change-multiline-rules).
2. **Java** stack traces, with the built-in Fluent Bit `java` parser.
3. **Go** panics and stack traces, with the built-in `go` parser.
4. **Python** tracebacks, including Python 3.11 error markers and frames without a source line.

A record is flushed when the next record starts or after two seconds without new lines.

A line that starts with `{` always opens a new record, so JSON lines are never appended to a text message. Write JSON
records on a single line.

## FluentD differences

FluentD supports a smaller set of formats:

- JSON lines. Every key becomes a field.
- Any line that starts with a bracketed token and contains a `[LEVEL]` token later. FluentD extracts only the level.
- klog lines. FluentD extracts the level, `thread`, and `class`.
- `[key=value]` groups anywhere in the line, through `fluent-plugin-fields-parser`. A key cannot contain spaces, `[`,
  `]`, or `=`.

FluentD has no third-party component parsers, except an extra JSON and Calico packet parser for `calico-node` pods. It
does not set `parse_format` or `parse_status`, and it keeps the message in `log` rather than `short_message`. It
ignores the `fluentbit.io/parser` annotation.

FluentD joins multiline records with `fluentd.multilineFirstLineRegexp`. A record starts at a line that begins with
`[YYYY-MM-DD`, `{"`, or a colored `HH:MM:SS` time. It also merges stack traces of common languages.
