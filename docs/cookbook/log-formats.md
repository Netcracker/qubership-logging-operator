# Logging guidelines for applications

Write logs to standard output in the Qubership text format or as one JSON object per line. The logging agent then
splits each record into searchable fields, joins multiline messages, and routes special records, such as audit logs,
to their own storage.

This page is for service developers. For the exact parser rules, see [Supported log formats](../log-processing/log-formats.md).

## How logs reach the storage

1. The service writes each record to standard output or standard error.
2. The container runtime (containerd, CRI-O, or Docker) writes the lines to a file on the node.
3. FluentBit, or FluentD, reads the file, joins multiline records, parses each record, and adds the Kubernetes
   metadata.
4. The agent sends the record to Graylog, or to another configured output such as VictoriaLogs, Loki, or Splunk.

Do not write logs to files inside the container: the agent reads only what the runtime writes to the node.

## Choose a format

| Format                           | Choose it when                                                     |
| -------------------------------- | ------------------------------------------------------------------ |
| [Qubership](#qubership-format)   | You write human-readable text logs                                 |
| [JSON](#json-format)             | Your logging library can write one JSON object per line            |
| logfmt                           | Your library writes `key=value` pairs and you cannot change it     |

All three give you a parsed level, a message, and named fields. Use one format per container.

## Key principles

- **Rational granularity:** log only valuable events.
- **Troubleshooting first:** logs must be enough to restore the context of an issue and find its root cause.
- **Structured:** every record is either Qubership text with `[key=value]` fields or valid JSON.
- **Correlation ready:** technical and business identifiers are always present.
- **Lightweight:** logging must not slow the service down noticeably.
- **Secure:** logs never contain passwords, tokens, or personal data.

## Qubership format

```text
[TIMESTAMP] [LEVEL] [key=value]... MESSAGE
```

Only `TIMESTAMP` and `LEVEL` are required:

- `TIMESTAMP` is an ISO 8601 date and time, such as `2025-05-02T09:16:53.541Z`. A space instead of `T`, a comma
  instead of the decimal point, and a missing time zone are also accepted.
- `LEVEL` is `TRACE`, `DEBUG`, `INFO`, `WARN`, `WARNING`, `ERROR`, or `FATAL`, in any case.
- Each `[key=value]` group becomes a field. The groups must come right after the level.
- `MESSAGE` is the rest of the record and can span several lines, such as a stack trace.

A line that does not match is still stored, but as unparsed text: its fields are not extracted, and its level is
found only if a bracketed level token appears in the line.

### Recommended fields

Include the correlation and source fields that are available:

```text
[TIMESTAMP] [LEVEL] [request_id=?] [tenant_id=?] [thread=?] [class=?]
[method=?] [version=?] [error_code=?] [originating_bi_id=?]
[business_identifiers={key=value}] [traceId=?] [spanId=?] MESSAGE
```

The fields can appear in any order.

<!-- markdownlint-disable line-length -->
| Field                  | When to use               | Note                     | Example                                                  |
| ---------------------- | ------------------------- | ------------------------ | -------------------------------------------------------- |
| `request_id`           | Request correlation       | Request ID               | `request_id=9f09fc89-…`                                  |
| `tenant_id`            | Tenant context            | Current tenant or `-`    | `tenant_id=b0c3…`                                        |
| `thread`               | Execution context         | Thread or goroutine name | `thread=TaskScheduler-1`                                 |
| `class`                | Log source                | Class or module name     | `class=o.apa.kaf.cli.NetworkClient`                      |
| `method`               | Java only                 | Method name              | `method=createOrder`                                     |
| `version`              | Blue/Green deployments    | `x-version` header       | `version=v2`                                             |
| `error_code`           | Error catalog             | Comma-separated list     | `error_code=PAY-401,DB-34`                               |
| `originating_bi_id`    | Root business interaction | UUID                     | `originating_bi_id=c771005e-1a46-44ab-b004-6c2f6b9c13d7` |
| `business_identifiers` | Up to 3 key-value pairs   | Runtime only             | `business_identifiers={object_id=12345,layer_id=98765}`  |
| `traceId`, `spanId`    | Distributed tracing       | B3 or W3C headers        | `traceId=4a1ceb… spanId=7b2fae…`                         |
| Custom `[key=value]`   | Exceptional needs         | Must follow parser rules | `[shard=eu-west-1a]`                                     |
<!-- markdownlint-enable line-length -->

Follow these rules for every field:

- A key cannot contain spaces, `[`, `]`, or `=`.
- A value cannot contain `[` or `]`. Do not put JSON arrays or bracketed thread names in a field.
- A field with an empty value, or with `"` in it, is skipped.
- Do not use the names of the fields the pipeline sets, such as `namespace`, `pod`, `container`, `level`, or
  `hostname`. See [Field names to avoid](../log-processing/log-fields.md#field-names-to-avoid).
- Keep values short and their number of distinct values small. See [Limits and restrictions](restrictions.md).

### Examples

```text
[2025-05-02T09:16:53.541Z] [WARN] [request_id=1682710824367.0.7602] [tenant_id=-] [thread=kafka-producer] [class=o.apa.kaf.cli.NetworkClient] Bootstrap broker kafka:9092 disconnected
```

```text
[2025-05-22T11:25:17.004Z] [INFO] [request_id=5f3e4e16] [tenant_id=demo] [thread=http-nio-8080-exec-3] [class=com.qubership.api.RequestController] [method=createRequest] [version=v2] [traceId=da2fae2d1] [spanId=7b2fae2d]
Request created successfully
```

### Logging library patterns

Log4j 2:

```text
[%d{yyyy-MM-dd'T'HH:mm:ss.SSSXXX}] [%p] [request_id=%X{requestId}] [thread=%t] [class=%c{1.}] %m%n
```

Logback:

```text
[%d{yyyy-MM-dd'T'HH:mm:ss.SSSXXX}] [%level] [request_id=%X{requestId:-}] [thread=%thread] [class=%logger{36}] %msg%n
```

The record must start with `[` and the date: the agent uses that start to tell a new record from the next line of a
multiline message.

## JSON format

Write one JSON object per line:

```json
{"time":"2025-05-02T09:16:53.018Z","level":"INFO","request_id":"9f09fc89","tenant_id":"demo","message":"Order created"}
```

- Use `time`, `level`, and `message` (or `msg`) for the core fields, and the [recommended fields](#recommended-fields)
  as top-level keys.
- Keep the object flat. Nested objects are stored as is, and most log storages search them poorly.
- Do not pretty-print: a JSON object that spans several lines is not parsed.
- The stored event time is the time the runtime wrote the line. The `time` key is kept as `parsed_time`.

The pipeline owns some field names and moves payload values with those names aside. See
[Reserved fields](../log-processing/log-fields.md#reserved-fields).

## Log levels

| Level     | Purpose                                                            |
| --------- | ------------------------------------------------------------------ |
| `FATAL`   | Critical, unrecoverable error causing the application to terminate |
| `ERROR`   | Error that prevents further processing                             |
| `WARN`    | Potential issue that does not stop execution                       |
| `WARNING` | Alias for `WARN`                                                   |
| `INFO`    | Business-level milestones _(default)_                              |
| `DEBUG`   | Detailed execution flow for development and troubleshooting        |
| `TRACE`   | Extremely fine-grained diagnostic information                      |

Run production at `INFO`. At `INFO`, log the outcome of successful operations briefly and the details of failed ones.
Enable `DEBUG` or `TRACE` only to investigate a problem.

The agent stores the level as a syslog name, so `FATAL` appears as `alert` and `ERROR` as `err`. See
[Log levels](../log-processing/log-fields.md#log-levels) for the full mapping.

## Request tracing

Log the `X-Request-ID` HTTP header as `request_id` in every record, so you can follow a request across services. With
Logback, read it from the MDC: `[request_id=%X{requestId:-}]`.

Add `traceId` and `spanId` when the service uses distributed tracing.

## Audit, integration, and access logs

Mark special records with a `logType` field so they go to their own stream:

| `logType` value | Category    | Example                                         |
| --------------- | ----------- | ----------------------------------------------- |
| `audit`         | Audit       | `[logType=audit]` or `"logType":"audit"`        |
| `int`           | Integration | `[logType=int]` or `"logType":"int"`            |
| `access`        | Access      | `[logType=access]` or `"logType":"access"`      |

Audit records have restricted access and longer retention. See [Log categories](../log-processing/log-categories.md)
for every category and for the text patterns that mark a record as audit automatically.

## Logs from outside Kubernetes

Services that do not run in the cluster can send logs to Graylog directly:

- Use a GELF appender of your logging library, such as the Log4j or Logback GELF appenders, and point it to the
  Graylog GELF input. These records carry no Kubernetes metadata.
- Forward syslog to Graylog; see [Syslog](../integrations/syslog.md).

Do not write to OpenSearch or Elasticsearch directly. Graylog owns its indices.

## Best practices

- Add context to every record: user, transaction, and thread identifiers help trace the cause of an error.
- Keep stack traces.
- Write messages a person can read and act on. Avoid messages such as `error` or `operation failed` with no details.
- Log calls to other services and systems.
- Do not log in tight loops. A high log volume slows down the whole logging stack.

## Change the log level at runtime

Let operators change the log level without changing the code or redeploying the service. Read the logging
configuration from environment variables or a ConfigMap, and reload it when it changes, for example through a REST
endpoint that reinitializes the logging configuration.
