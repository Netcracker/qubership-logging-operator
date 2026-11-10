# Log categories

The logging stack sorts records into categories, such as application, audit, and system logs. Graylog keeps each
category in its own stream, and the HTTP output can send each category to its own VictoriaLogs instance. Each category
has its own retention and access rules.

The agent decides the basic category of a record: container, audit, or system. The other categories depend on fields
in the record, which your service writes.

## Categories at a glance

<!-- markdownlint-disable line-length -->
| Category          | Records                                                                     | Graylog stream      | Created by default | `X-Log-Type` with HTTP routing |
| ----------------- | --------------------------------------------------------------------------- | ------------------- | ------------------ | ------------------------------ |
| Application       | Container logs that match no other category                                 | `Default Stream`    | Yes                | `out_pods`                     |
| Audit             | Node and Kubernetes audit files, and container records marked as audit      | `Audit logs`        | Yes                | `out_audit`                    |
| System            | `/var/log/messages`, `/var/log/syslog`, or the systemd journal of each node | `System logs`       | Yes                | `out_system`                   |
| Kubernetes events | Records with `kind: KubernetesEvent`, written by the Cloud Events Reader    | `Kubernetes events` | Yes                | `out_k8s_event`                |
| Integration       | Records with `logType` or `log_type` equal to `int`                         | `Integration logs`  | No                 | `out_int`                      |
| Access            | Records with `logType` or `log_type` equal to `access`                      | `Access logs`       | No                 | `out_access`                   |
| Ingress NGINX     | Records from pods whose name contains `ingress-nginx`                       | `Nginx logs`        | No                 | `out_nginx`                    |
| Bill Cycle        | Records with an `nc_bill_cycle_label` field                                 | `Bill Cycle logs`   | No                 | `out_pods`                     |
<!-- markdownlint-enable line-length -->

When a record fits several categories, the HTTP routing picks the first match in this order: audit, Kubernetes
events, Ingress NGINX, access, integration, application. In Graylog, each processing rule routes the record on its own,
so a record can land in more than one stream. Every rule removes the record from `Default Stream`.

## Category field

With the Loki, OpenTelemetry, or HTTP output enabled, FluentBit writes the basic category to the record:

| Value       | Records                                                     |
| ----------- | ----------------------------------------------------------- |
| `container` | Container records, including Kubernetes events              |
| `audit`     | Audit file records, and container records marked as audit   |
| `system`    | System log records                                          |

The field is `stream` with the Loki or OpenTelemetry output, and `log_category` when the HTTP output is enabled. With
only the Graylog output, neither field exists: Graylog uses the rules below instead.

## Application logs

Every container record that no other rule claims stays in `Default Stream`, which Graylog also shows as
`All messages`. FluentBit reads container logs when `fluentbit.containerLogging` is `true`, the default.

## Audit logs

Audit records go to the `Audit logs` stream, which uses its own index set. Administrators and users with the
`AuditViewer` role can read it; the `Operator` role cannot.

### Audit log files

The agent reads these files and marks every record from them as audit:

| File                                                 | Enabled by                            |
| ---------------------------------------------------- | ------------------------------------- |
| `/var/log/audit/audit.log`                           | `fluentbit.systemAuditLogging`        |
| `/var/log/kubernetes/audit/audit.log`                | `fluentbit.kubeAuditLogging`          |
| `/var/log/kubernetes/kube-apiserver-audit.log`       | `fluentbit.kubeAuditLogging`          |
| `/var/log/kube-apiserver/audit.log`                  | `fluentbit.kubeApiserverAuditLogging` |
| `/var/log/openshift-apiserver/audit.log`             | `fluentbit.kubeApiserverAuditLogging` |
| `/var/lib/origin/ocp-audit.log`                      | Read automatically on OpenShift       |

All three settings default to `true`. A file that does not exist on the node is skipped. See
[Log location on different OS](../log-location.md#audit-logs) for the operating systems that have `auditd`.

### Audit records from containers

FluentBit marks a container record as audit with `nc_audit_label: true` when its raw line matches one of these
expressions. The match is case-sensitive.

<!-- markdownlint-disable line-length -->
| Pods                                     | Expression                                                                                                                                                                            |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| All pods                                 | `access-control\|PG_SERVICE\|database\ssystem\sis\|audit_log_type\|AUDIT\|logType[\":\s\t=]*audit\|org\.qubership\.security\.audit.*CEF\|org\.qubership\.cloud\.keycloak\.provider.*` |
| Pod or container name contains `grafana` | `Invalid\susername\sor\spassword\|Successful\sLogin\|Successful\sLogout\|Failed\sto\slook\sup\suser\sbased\son\scookie`                                                               |
| Pod or container name contains `mongo`   | `ACCESS\|CONTROL`                                                                                                                                                                     |
<!-- markdownlint-enable line-length -->

The expressions match anywhere in the line, so any message that contains the word `AUDIT` in capitals is an audit
record.

### Mark your service logs as audit

Add a `logType` field with the value `audit` to the record:

- In the Qubership format, add the `[logType=audit]` group:

  ```text
  [2025-05-02T09:16:53.541Z] [INFO] [logType=audit] [user=alice] Role changed
  ```

- In JSON, add the `logType` key:

  ```json
  {"time":"2025-05-02T09:16:53.541Z","level":"INFO","logType":"audit","message":"Role changed"}
  ```

A record that already has `nc_audit_label: true` in its JSON payload is also treated as audit.

## System logs

System records go to the `System logs` stream, which stores them in `Default index set` together with application
logs. FluentBit reads one source per node, chosen by `fluentbit.systemLogType`:

| `systemLogType`  | Source              | `tag` in the record |
| ---------------- | ------------------- | ------------------- |
| `varlogmessages` | `/var/log/messages` | `/var/log/messages` |
| `varlogsyslog`   | `/var/log/syslog`   | `/var/log/syslog`   |
| `systemd`        | The systemd journal | `systemd`           |

`fluentbit.systemLogging` turns the input on and defaults to `true`. The default type is `varlogmessages`. See
[Log location on different OS](../log-location.md#system-logs) to choose the type for your nodes.

Graylog routes a record to `System logs` when its `tag` field matches `var.log.messages|var.log.syslog|systemd`.

## Kubernetes events

The Cloud Events Reader writes Kubernetes events to its own log as JSON records with `kind: KubernetesEvent`. FluentBit
reads them like any container log, and Graylog routes them to the `Kubernetes events` stream by the `kind` field. See
[Kubernetes events](../user-guides/cloud-events.md) for the reader setup.

A custom event format must keep `"kind":"KubernetesEvent"`, or the events stay in `Default Stream`.

## Integration and access logs

A service marks a record by setting `logType` or `log_type`:

| Value    | Category    | Graylog stream     |
| -------- | ----------- | ------------------ |
| `int`    | Integration | `Integration logs` |
| `access` | Access      | `Access logs`      |

The value must match exactly: `integration` or `Access` does not route the record. For example:

```text
[2025-05-02T09:16:53.541Z] [INFO] [logType=int] [target=billing] Request sent
```

The pipeline never sets these fields itself.

## Ingress NGINX logs

Records from pods whose name contains `ingress-nginx` go to the `Nginx logs` stream. FluentBit parses their access log
lines; see [Third-party components](log-formats.md#third-party-components).

## Bill Cycle logs

Graylog routes a record to `Bill Cycle logs` when it has an `nc_bill_cycle_label` field, or when its `tag` ends with
`var/log/bill-cycle.log`. Neither agent sets the field out of the box, so a service has to write it, or a custom filter
has to add it.

## Configure Graylog streams

Graylog creates `System logs`, `Audit logs`, and `Kubernetes events` when `graylog.streams` is empty. To create other
streams, or to change retention, list them in `graylog.streams`:

```yaml
graylog:
  streams:
    - name: "Audit logs"
      install: true
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M"
      maxNumberOfIndices: 5
    - name: "Kubernetes events"
      install: true
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M"
      maxNumberOfIndices: 5
    - name: "Integration logs"
      install: true
      rotationStrategy: "timeBased"
      rotationPeriod: "P1M"
```

When you set the list, include every stream you want. The operator adds `System logs` and `Audit logs` when the list
leaves them out, but it creates `Kubernetes events` only if the list includes it. See [Graylog Streams](../installation-parameters.md#graylog-streams)
for every parameter.

To route other records to a stream of your own, add a stream and a rule in Graylog, or add a field in the agent with
`extraFields` or a custom filter. See [Configure log parsing](configure-parsing.md).

## FluentD differences

FluentD sets the category in `stream` for every output, and in `log_category` when the HTTP output is enabled. It
marks container records as audit with the same expressions as FluentBit, and sets `nc_audit_label: false` on every
other container record. The Graylog streams and rules are the same.
