# Configure log parsing

You can change how FluentBit parses and categorizes logs without changing the operator. Set the parameters below in
the Helm values, or in the `LoggingService` custom resource.

The built-in behavior is described in [Supported log formats](log-formats.md), [Log fields reference](log-fields.md),
and [Log categories](log-categories.md). For all agent parameters, see [FluentBit](../installation-parameters.md#fluentbit)
in the installation parameters.

## Choose a parser for a pod

Add the `fluentbit.io/parser` annotation to the pod template to make FluentBit parse its logs with a parser you name:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    metadata:
      annotations:
        fluentbit.io/parser: logfmt
```

To set the parser for one container of the pod, use `fluentbit.io/parser-<container>`. See
[the Kubernetes filter documentation](https://docs.fluentbit.io/manual/pipeline/filters/kubernetes)
for all forms of the annotation.

The value must name a parser of the agent, such as `json`, `logfmt`, `qubership`, `postgres`, `cassandra`, `consul`,
`coredns`, `opensearch`, `fluentbit`, `k8s-nginx-ingress`, or `klog_entry`. In the HA mode, the forwarder reads the
annotation, so the parser must exist in the forwarder configuration.

An annotation changes the result in these ways:

- The record gets `parse_format: unknown`, unless the line is a JSON object.
- The message is the field the parser produced, such as `short_message`, or the raw line. A logfmt `msg` key stays a
  field of its own.
- The other parsers do not run on the record.

Use the annotation when the automatic selection picks the wrong parser for a pod. For the formats it picks
automatically, see [How FluentBit chooses a parser](log-formats.md#how-fluentbit-chooses-a-parser).

FluentBit ignores the `fluentbit.io/exclude` annotation. To stop collecting a pod's logs, use
[`excludePath`](#choose-the-log-sources).

## Change multiline rules

Two expressions decide how FluentBit joins the lines of a Qubership-format message:

| Parameter                             | Default                                        | Meaning                                         |
| ------------------------------------- | ---------------------------------------------- | ----------------------------------------------- |
| `fluentbit.multilineFirstLineRegexp`  | `/^(\[\d{4}\-\d{2}\-\d{2}).*/`                 | A line that starts a new message                |
| `fluentbit.multilineOtherLinesRegexp` | `/^(?!\[\d{4}\-\d{2}\-\d{2}\|\s*\{).*/`        | A line that continues the current message       |

In the HA mode, set `fluentbit.aggregator.multilineFirstLineRegexp` and
`fluentbit.aggregator.multilineOtherLinesRegexp` instead. The forwarder uses them.

Write each expression between slashes. The value is inserted into the configuration as is, so it must not contain a
colon followed by a space or a `#` after a space.

For example, to also start a message at a date that has no bracket, such as `2025-05-02 09:16:53 INFO ...`:

```yaml
fluentbit:
  multilineFirstLineRegexp: '/^(\[?\d{4}-\d{2}-\d{2}).*/'
  multilineOtherLinesRegexp: '/^(?!\[?\d{4}-\d{2}-\d{2}|\s*\{).*/'
```

Keep these rules when you change the expressions:

- The continuation expression must reject every line that the first expression accepts. Otherwise a new message is
  appended to the previous one.
- Keep `\s*\{` in the continuation expression. Without it, JSON lines that follow a text message are appended to it.

The Helm chart fills in the defaults. A `LoggingService` you create without the chart must set both expressions;
otherwise the multiline rules are empty. Upgrading the operator does not change values already stored in the custom
resource.

Java, Go, and Python stack traces are joined by built-in rules that you cannot change.

## Choose the log sources

| Parameter                             | Default          | Effect                                                                         |
| ------------------------------------- | ---------------- | ------------------------------------------------------------------------------ |
| `fluentbit.containerLogging`          | `true`           | Reads container logs                                                           |
| `fluentbit.systemLogging`             | `true`           | Reads system logs                                                              |
| `fluentbit.systemLogType`             | `varlogmessages` | The system log source: `varlogmessages`, `varlogsyslog`, or `systemd`          |
| `fluentbit.systemAuditLogging`        | `true`           | Reads `/var/log/audit/audit.log`                                               |
| `fluentbit.kubeAuditLogging`          | `true`           | Reads the Kubernetes audit logs under `/var/log/kubernetes/`                   |
| `fluentbit.kubeApiserverAuditLogging` | `true`           | Reads the kube-apiserver and OpenShift API server audit logs                   |
| `fluentbit.excludePath`               | Empty            | Comma-separated file patterns that FluentBit does not read                     |
| `fluentbit.selfLogging`               | `false`          | Reads the logs of the FluentBit pods themselves                                |

See [Log categories](log-categories.md) for the files each input reads. For example, to skip every pod in the
`kube-system` namespace on containerd or CRI-O nodes:

```yaml
fluentbit:
  excludePath: "/var/log/pods/kube-system_*/*/*.log"
```

On Docker nodes, container logs are under `/var/log/containers/<pod>_<namespace>_<container>-<id>.log`.

## Add fields to every record

`fluentbit.extraFields` adds fields to every record, for example, to tell clusters apart in a shared Graylog:

```yaml
fluentbit:
  extraFields:
    cluster: prod-eu-1
```

The values replace payload fields with the same name. In the HA mode, set `fluentbit.aggregator.extraFields`;
`fluentbit.extraFields` has no effect there.

## Add your own processing

`fluentbit.customFilterConf` adds Fluent Bit filters to the pipeline. They run after parsing and audit detection, and
before level normalization and format classification. At that point:

- The raw `log` field of a container record is gone. Read the message from `short_message`.
- The payload fields are already at the top level of the record.
- The `stream` or `log_category` field, if any, is already set.
- A `level` you set is normalized afterward, like any other level.

Container record tags have the form `pods.var.log.pods.<namespace>_<pod>_<uid>.<container>.<n>.log` on containerd and
CRI-O nodes. In the HA mode, the forwarder host name follows `pods.`. The expression
`^pods\.(?:[^.]+\.)?var\.log\.pods\.<namespace>_` matches a namespace in both modes.

### Route records to the Bill Cycle stream

The agents do not detect Bill Cycle records out of the box. This filter marks the records of the `billing` namespace
whose message starts with `Invoice`:

```yaml
fluentbit:
  customFilterConf: |-
    [FILTER]
        Name         modify
        Match_Regex  ^pods\.(?:[^.]+\.)?var\.log\.pods\.billing_
        Condition    Key_value_matches  short_message  ^Invoice\s
        Set          nc_bill_cycle_label true
```

Enable the `Bill Cycle logs` stream in `graylog.streams` to store them separately; see
[Configure Graylog streams](log-categories.md#configure-graylog-streams).

### Transform records with Lua

`fluentbit.customLuaScriptConf` adds Lua scripts to the agent configuration. FluentBit stores each script under
`/fluent-bit/etc/<name>`; call it from a custom filter. This example takes the level from a `severity` field when a
service does not write `level`:

```yaml
fluentbit:
  customLuaScriptConf:
    severity_to_level.lua: |
      function severity_to_level(tag, timestamp, record)
        if record["level"] == nil and record["severity"] ~= nil then
          record["level"] = record["severity"]
          return 2, timestamp, record
        end
        return 0, timestamp, record
      end
  customFilterConf: |-
    [FILTER]
        Name    lua
        Match   pods*
        Script  /fluent-bit/etc/severity_to_level.lua
        Call    severity_to_level
```

See [the Lua filter documentation](https://docs.fluentbit.io/manual/pipeline/filters/lua) for the return values.

### Custom processing in the HA mode

The HA mode splits the pipeline between two agents:

| Agent      | Runs                                                                 | Custom parameters                                                                                                            |
| ---------- | -------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| Forwarder  | Inputs, multiline joining, the Kubernetes filter, `extraFields`      | `fluentbit.customInputConf`, `fluentbit.customFilterConf`, `fluentbit.customLuaScriptConf`                                   |
| Aggregator | Parsing, audit detection, level normalization, outputs               | `fluentbit.aggregator.customFilterConf`, `fluentbit.aggregator.customOutputConf`, `fluentbit.aggregator.customLuaScriptConf` |

Put filters that read parsed fields, like the examples above, in `fluentbit.aggregator.customFilterConf`. A forwarder
filter runs before parsing and sees the raw `log` field.

For custom inputs and outputs, see [Agents pipeline customization](../user-guides/agents-pipeline-customization.md).

## Check parsing results

Find lines that no parser matched by searching for `parse_status:failed` in Graylog, or for `parse_status:=failed` in
VictoriaLogs.

FluentBit also counts them in the `fluentbit_parse_error_total` metric on port `2021`, labeled with `namespace`,
`pod`, and `container`. A growing counter for a pod means the pod writes a format the pipeline does not recognize:
switch the service to a [supported format](log-formats.md#application-log-formats), or
[choose a parser](#choose-a-parser-for-a-pod) for the pod.

`parse_level_unknown: true` marks records whose level is missing or not recognized.

## FluentD

FluentD has fewer settings:

- `fluentd.multilineFirstLineRegexp` sets the expression that starts a multiline record. The default is
  `/(^\[\d{4}\-\d{2}\-\d{2})|(^\{")|(^*0m\d{2}\:\d{2}\:\d{2})/`.
- `fluentd.containerLogging`, `fluentd.systemLogging`, `fluentd.systemLogType`, `fluentd.systemAuditLogging`,
  `fluentd.kubeAuditLogging`, `fluentd.kubeApiserverAuditLogging`, and `fluentd.excludePath` choose the sources.
- `fluentd.extraFields` adds fields to every record.
- `fluentd.customInputConf`, `fluentd.customFilterConf`, and `fluentd.customOutputConf` add configuration of your own.

FluentD ignores the `fluentbit.io/parser` annotation. See
[Agents pipeline customization](../user-guides/agents-pipeline-customization.md#fluentd) for examples.
