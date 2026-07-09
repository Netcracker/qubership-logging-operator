# collection-howto — how to collect each required field

For each `field-id` listed as missing, read its section and use the steps for the
ticket platform. Weave them into the message to the author. Placeholders like
`<pod>` are filled by the author with their own values. Prose-only fields need no
collection — the author just states them in the ticket.

## description

State in the ticket what is wrong, what was expected, and when it started. No collection needed.

## deployment_params

**kubernetes:** Export the CR: `kubectl -n logging get loggingservice -o yaml`.
**vm:** Attach the deploy parameters used for the install (inventory or job parameters).

## logging_version

**kubernetes:** The operator image tag (or the `version` in the CR):
`kubectl -n logging get deploy logging-operator -o jsonpath='{.spec.template.spec.containers[0].image}'`.
**vm:** The `application_version` from the deploy parameters (e.g. the `deploy-logging-service` job).

## steps_to_reproduce

List the exact actions that trigger the problem, in order. No collection needed.

## environment_link

Paste a link to the affected environment (Graylog UI, cluster console, or pipeline). No collection needed.

## expected_behavior

State what you expected to happen and why, so the gap is clear. No collection needed.

## cve_report

Attach the vulnerability report — the CVE list or SCA / scanner output (e.g. XRAY) for the scanned version.

## deployment_type

State whether the install is driven by ArgoCD or by a Groovy pipeline.

## deployment_mode

State whether this was a clean install or a rolling update.

## deployment_logs

**kubernetes:** Attach the pipeline log of the failed deploy, or a link to the run.
**vm:** Attach the `deploy-logging-service` job log, or a link to the run.

## operator_logs

**kubernetes:** `kubectl -n logging logs deploy/logging-operator --tail=2000`.

## service_logs

**kubernetes:** `kubectl -n logging logs -l name=<workload> --previous --tail=2000`
for each restarting service — workloads are `graylog` (StatefulSet),
`logging-fluentd` and `logging-fluentbit` (DaemonSets).
**vm:** `docker logs --tail 2000 <container>` for the failed Graylog-stack
container — `graylog_graylog_1`, `graylog_elasticsearch_1`, `graylog_mongo_1`,
`graylog_web_1`.

## pod_yaml

**kubernetes:** `kubectl -n logging get pod -l name=<workload> -o yaml` for each
restarted service (`graylog`, `logging-fluentd`, `logging-fluentbit`).

## configmap_fluent

**kubernetes:** `kubectl -n logging get cm logging-fluentbit logging-fluentd -o yaml`.
**vm:** Attach the FluentBit / FluentD config files from the agent host that ships to this Graylog.

## dashboards

**kubernetes:** Attach the runtime and Graylog-metrics dashboard screenshots, or a Grafana link covering the incident window.
**vm:** Attach the same metrics screenshots if monitoring is available, or a Grafana link.

## ssh_access

**vm:** Provide SSH access to the VM (keys) so the logs and config can be read directly.

## container_logs

**vm:** `docker logs --tail 2000 <container>` for each failed or restarting
container — e.g. `graylog_graylog_1`, `graylog_elasticsearch_1`,
`graylog_mongo_1`, `graylog_web_1`.

## affected_scope

Name the namespace and service that have no logs. No collection needed.

## lost_logs_example

**kubernetes:** Provide an example of the missing logs from the node: `/var/log/pods/<namespace>_<pod>_<container_id>/`.
**vm:** Provide an example of the missing source logs from the application host.

## where_not_seen

State where you do not see the logs (Graylog UI, Grafana, etc.). No collection needed.

## query_used

Paste the exact query you used to select the logs. No collection needed.

## graylog_fluent_logs

**kubernetes:** `kubectl -n logging logs -l name=graylog` and
`kubectl -n logging logs -l name=logging-fluentbit` (or `logging-fluentd`),
around the incident time.
**vm:** `docker logs graylog_graylog_1` (and `graylog_elasticsearch_1` for
indexing errors), around the incident time. FluentBit / FluentD run on the
source side, not on the Graylog VM.

## source_vs_graylog_logs

Provide an example of the source logs and the same lines as they appear in Graylog, so the delay or gap is visible.
</content>
