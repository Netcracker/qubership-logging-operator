# facts-required — what data L1 needs, keyed by localization

Look up by the `logging-l1-classification` localization. The baseline applies to any
`problem`; the phase/symptom rows add fields on top. `platform=kubernetes` is the
in-cluster Logging install; `platform=vm` is External Logging. Each line is
`field-id: description`; the collection steps for each id live in
`collection-howto.md`. Used two ways:

- additional_info_required: list the baseline + matching-row ids that are MISSING.
- handoff_to_l2: those same ids, now collected, go into the packet `facts`.

## Baseline — intent=problem

- description: what is wrong, what was expected, when it started
- deployment_params: deployment parameters or the `LoggingService` CR
- logging_version: the Logging version in use
- steps_to_reproduce: the actions that trigger the problem (if available)
- environment_link: a link to the affected environment (if available)

## intent=consultation

- description: what the reporter wants to know
- logging_version: the Logging version in use
- environment_link: a link to the affected environment (if available)

## intent=feature_request

- description: the requested change
- logging_version: the Logging version in use
- expected_behavior: what the reporter expects and why

## intent=security

- logging_version: the Logging version used for the scan
- cve_report: the vulnerability report (CVE / SCA list)

## phase=deploy, platform=kubernetes

- logging_version: the Logging version being installed
- deployment_type: ArgoCD or Groovy
- deployment_mode: clean install or rolling update
- deployment_params: the deployment parameters
- deployment_logs: deployment logs or an accessible pipeline link
- operator_logs: logs from logging-operator
- service_logs: logs from the failed services

## phase=deploy, platform=vm

- logging_version: the Logging version being installed
- deployment_params: the deployment parameters
- ssh_access: SSH access to the VM (keys)
- deployment_logs: deployment logs or an accessible pipeline link

## symptom=oom_memory or not_running, platform=kubernetes

- logging_version: the Logging version in use
- deployment_type: ArgoCD or Groovy
- deployment_params: the deployment parameters
- service_logs: logs from the restarting services
- pod_yaml: pod YAML of the restarted services
- configmap_fluent: the FluentBit / FluentD ConfigMap
- dashboards: runtime + Graylog metrics screenshots or a Grafana link

## symptom=not_running, platform=vm

- logging_version: the Logging version in use
- deployment_params: the deployment parameters
- ssh_access: SSH access to the VM (keys)
- container_logs: logs from the failed / restarting containers

## symptom=no_data

- logging_version: the Logging version in use
- deployment_params: the deployment parameters
- affected_scope: which namespace and service has no logs
- lost_logs_example: an example of the lost logs
- where_not_seen: where the logs are not seen (Graylog UI, Grafana, etc.)
- query_used: the query used to select logs
- graylog_fluent_logs: logs from Graylog and FluentBit / FluentD
- configmap_fluent: the FluentBit / FluentD ConfigMap

## symptom=performance (log delay / time gap)

- source_vs_graylog_logs: source logs and the same lines from Graylog
- configmap_fluent: the FluentBit / FluentD ConfigMap
