# Qubership Logging Operator — troubleshooting

## Operator and LoggingService

### FluentD reconciliation fails because selector does not match template labels

**Symptoms:**

* The operator reports `DaemonSet.apps "logging-fluentd" is invalid`.
* The error contains `spec.template.metadata.labels` and `selector does not match template labels`.
* `LoggingService` status contains a failed FluentD reconciliation cycle.

**Root cause:**

The FluentD reconciler replaces the desired pod-template labels but leaves the existing DaemonSet selector unchanged.
Kubernetes requires the selector to match the pod-template labels. A label-affecting configuration change, such as a
change between Kubernetes and OpenShift deployment modes, can therefore make an existing selector incompatible with
the newly rendered template. A fresh DaemonSet does not have this mismatch.

**How to check:**

1. Ask the operator to compare the live selector and template labels:

   ```bash
   kubectl -n <namespace> get daemonset logging-fluentd -o jsonpath='{.spec.selector.matchLabels}{"\n"}{.spec.template.metadata.labels}{"\n"}'
   ```

   The selector labels must be present with the same values in the template labels.
2. Inspect the `LoggingService` and the Helm or GitOps source for changes to `openshiftDeploy`, custom labels, and the
   target platform. Do not infer which value changed from the API error alone.
3. Confirm the exact API error in the operator logs and retain the existing DaemonSet YAML for comparison.

**How to fix:**

1. If the desired platform or label value is wrong, correct it in the authoritative Helm or GitOps source so the
   rendered pod-template labels again match the existing selector.
2. If the desired labels are correct and the selector is stale, plan recreation of only `logging-fluentd`.
3. **DANGEROUS — interrupts node log collection; messages produced during the gap may not appear centrally until
   FluentD resumes.** Save the DaemonSet YAML, delete the DaemonSet during a maintenance window, and let the operator
   recreate it. Confirm new pods are Ready before ending the window.

**How to avoid this issue:**

Treat selector-participating labels as immutable after installation. The operator implementation should recreate the
DaemonSet when the desired selector changes or keep mutable labels out of the selector.

**Data to collect:**

* The `LoggingService` YAML and authoritative values diff.
* The live DaemonSet YAML and the operator error excerpt.

**Sources:**

* Support ticket `PSUPCLPL-14999`, Gold 150 ticket export.
* `controllers/fluentd/handler.go:35-60`.
* [DaemonSet API reference — Kubernetes](https://kubernetes.io/docs/reference/kubernetes-api/apps/daemon-set-v1/).

### Operator reports Graylog is not started although a pod is Running

**Symptoms:**

* Deployment fails with `Deploy of graylog failed. Reason: graylog is not started`.
* A Graylog pod is visible in the cluster and its phase is `Running`.
* `LoggingService` status remains `ReconcileGraylogStatus` or another failed reconciliation state.

**Root cause:**

The operator does not equate the Kubernetes pod phase `Running` with a completed Graylog rollout. It waits for the
Graylog StatefulSet's total and Ready replica counts to equal the desired count before `graylog.startupTimeout`
expires. A pod can be Running while its readiness probe fails, another replica is missing, or the configured startup
timeout is shorter than actual startup.

**How to check:**

1. Compare `.spec.replicas`, `.status.replicas`, and `.status.readyReplicas` on the Graylog StatefulSet.
2. Inspect pod readiness conditions and events. `Running` with `Ready=False` confirms that pod phase alone is not a
   healthy result.
3. Inspect the Graylog and MongoDB container logs and the Graylog readiness probe result.
4. Read `spec.graylog.startupTimeout` in `LoggingService`; zero means the operator default applies.

**How to fix:**

1. Fix the readiness, scheduling, storage, MongoDB, or application error demonstrated by the failed condition or logs.
2. If all containers become healthy but startup consistently exceeds the configured timeout, increase
   `graylog.startupTimeout` in the authoritative values and redeploy.
3. **DANGEROUS — with one Ready replica or a full-replica restart, Graylog UI, search, and ingestion remain
   unavailable until readiness returns.** Restart or roll Graylog only when logs show a transient startup failure and
   the incident owner accepts the topology-specific interruption.

**Data to collect:**

* StatefulSet status, pod conditions and events, `LoggingService` status, and startup timeout.
* Graylog and MongoDB logs covering the failed reconciliation window.

**Sources:**

* Support ticket `PSUPCLPL-9209`, Gold 150 ticket export.
* `controllers/graylog/handler.go:158-178`.
* `controllers/utils/pod_manager.go:145-175`.

## Events reader

### Events reader cannot list cluster events after deployment

**Symptoms:**

* The `events-reader` pod runs but Kubernetes or OpenShift events do not reach the logging pipeline.
* Authorization checks for the `events-reader` service account return `no`.
* A redeployment leaves the service account without the expected cluster role binding.

**Root cause:**

The chart creates the `events-reader` ClusterRoleBinding only when both `cloudEventsReader.install` and
`createClusterAdminEntities` are enabled. When cluster-scoped resources are disabled or removed by another deployment,
the namespaced workload still exists but its service account cannot list cluster events.

**How to check:**

1. Inspect `cloudEventsReader.install` and `createClusterAdminEntities` in the authoritative values.
2. Inspect the `events-reader` service account and the `<namespace>-cloud-events-reader` ClusterRoleBinding. Its subject
   must name the service account in the Logging namespace and its role must grant the intended event reads.
3. Ask the operator to run:

   ```bash
   kubectl auth can-i list events --all-namespaces --as=system:serviceaccount:<namespace>:events-reader
   ```

   `yes` is the expected result.

**How to fix:**

1. If cluster-scoped resources are managed by this chart, enable `createClusterAdminEntities` and redeploy with an
   account permitted to create them.
2. If organizational policy manages cluster RBAC separately, create the equivalent binding through that approved
   process. Bind the service account to the least-privileged role that supplies the required event reads; do not grant
   `cluster-admin` as a shortcut.

**How to avoid this issue:**

Make one deployment owner authoritative for cluster-scoped logging resources and test the service-account permission
after upgrades of either Logging or Monitoring.

**Data to collect:**

* Values, service account, ClusterRoleBinding, `kubectl auth can-i` output, and events-reader logs.

**Sources:**

* Support ticket `PSUPCLPL-5031`, Gold 150 ticket export.
* `charts/qubership-logging-operator/templates/cloud-event-reader/clusterrole.yaml:1-25`.
* `docs/installation-parameters.md:47`.

### Events reader restarts after invalid command-line arguments

**Symptoms:**

* The `events-reader` container exits with code 1 or enters `CrashLoopBackOff` after custom arguments are added.
* Output contains `Error: format string exceeds maximum allowed length of 1024 characters`,
  `Error: metricsPort cannot be empty.`, or `Error: Invalid metrics path`.
* Other matching errors include `could not parse filter events configuration` and
  `error occurred during initialization of logs output`.

**Root cause:**

Logging Operator passes `cloudEventsReader.args` unchanged to the version 2.9.1 container. The reader validates its
flags, metrics settings, and optional filter file before it starts the event controllers. An invalid Go template
terminates startup only when the `logs` output is enabled; metrics-only mode does not parse that template. The output
validator also accepts some malformed strings, so use the exact startup error rather than inferring validation from an
arbitrary value. The stock Deployment mounts no filter configuration, so a nonempty `filtersPath` cannot reach a file
unless a custom image already contains it.

**How to check:**

1. Inspect current and previous `events-reader` logs and retain the first startup error.
2. Compare `LoggingService.spec.cloudEventsReader.args` with the Deployment's effective container arguments.
3. For `filtersPath`, inspect the image and Deployment volume mounts. The stock Deployment has no filter-config
   volume.
4. Compare the failing value with the documented defaults. When `logs` output is enabled, a healthy custom `format` is
   a valid Go `text/template`.

**How to fix:**

1. Remove an unnecessary override or correct the value in the authoritative Helm or GitOps configuration.
2. For custom filtering, add reviewed source-level volume support or use a verified custom image. Do not hand-edit the
   operator-owned Deployment.
3. Render the `LoggingService` and inspect the generated Deployment arguments before applying the correction.
4. **DANGEROUS — if no Ready reader remains during the rollout, events emitted in that gap are not backfilled.**
   Deploy the corrected arguments during a controlled rollout and verify that the reader starts its workers.

**Data to collect:**

* Current and previous container logs, `LoggingService`, Deployment arguments, and any referenced filter artifact.

**Sources:**

* `api/v1/loggingservice_types.go:296-308`.
* `charts/qubership-logging-operator/templates/operator/loggingservice.observability.netcracker.com.yaml:983-993`.
* `controllers/events-reader/assets/deployment.yaml:27-79`.
* [Argument validation in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/main.go#L39-L132).
* [Flag validation in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/pkg/utils/flags.go#L36-L47).

### Events reader probes fail after the health port is overridden

**Symptoms:**

* The `events-reader` pod does not become Ready or restarts after liveness failures on `/health:8080`.
* The effective arguments contain `-pprofAddr=<different-port>`.
* Logs may contain `could not start health endpoint` or `failed to start HTTP server`.

**Root cause:**

The reader's `pprofAddr` argument controls the listener that serves both `/health` and pprof. Logging Operator
hard-codes liveness and readiness probes to port `8080`. A different valid value moves the listener away from the
probes; an invalid or unavailable value prevents the health server from starting.

**How to check:**

1. Compare the effective `pprofAddr` argument with both probe definitions in the Deployment.
2. Inspect current and previous container logs for health-server initialization or bind errors.
3. Confirm that the failures target the events-reader `/health` endpoint on `8080`, not a Fluent or Graylog endpoint.

**How to fix:**

1. Remove the `pprofAddr` override or set it to `8080` in the authoritative configuration.
2. Render the `LoggingService` and inspect both probes before applying the correction.
3. **DANGEROUS — if no Ready reader remains during the rollout, events emitted in that gap are not backfilled.**
   Deploy the corrected argument during a controlled rollout and verify `/health:8080` before proceeding.

**Sources:**

* `controllers/events-reader/assets/deployment.yaml:27-62`.
* [Health endpoint in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/pkg/utils/health.go#L15-L50).
* [Health argument handling in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/main.go#L50-L77).

### Events reader is Ready but zero workers process events

**Symptoms:**

* The pod is Ready and `/health` returns success.
* Logs contain `started workers`, but event records and event metrics do not increase.
* The effective arguments contain `-workers=0` or a negative value.

**Root cause:**

Version 2.9.1 does not validate the worker count. The controller logs `started workers` after iterating over the
configured count, but zero or a negative value starts no `runWorker` goroutines. The health endpoint is independent of
the event workers, so the pod can remain Ready while queued events never reach a sink.

**How to check:**

1. Inspect the effective `workers` argument; the documented default is `2`.
2. Confirm that the pod is healthy but neither stdout records nor event counters increase.
3. Rule out `Events reader cannot list cluster events after deployment` with the read-only authorization check from
   that case.

**How to fix:**

1. Remove the override or set a positive worker count, starting with the documented default of `2`.
2. **DANGEROUS — starting sink processing increases collector and storage load from its current zero-output level.**
   Apply the change while monitoring collector queues and storage ingestion.

**Sources:**

* `charts/qubership-logging-operator/values.yaml:1967-1973`.
* `controllers/events-reader/assets/deployment.yaml:31-62`.
* [Worker startup in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/pkg/controller/events_reader_controller.go#L164-L181).

### Events reader is Ready but metrics-only output produces no event logs

**Symptoms:**

* The pod is Ready and logs show `sink initialized successfully` with `sink=metrics` but not `sink=stdout`.
* Kubernetes event counters can increase, but stdout has no records containing `"kind":"KubernetesEvent"`.
* FluentD or FluentBit therefore has no Kubernetes event records to forward.

**Root cause:**

The reader supports independent `logs` and `metrics` sinks. With only `-output=metrics`, version 2.9.1 initializes no
stdout sink. The Logging pipeline depends on stdout records being collected by FluentD or FluentBit.

**How to check:**

1. Inspect every effective `output` argument and the sink initialization logs.
2. If the metrics endpoint is available, compare increasing counters with the absence of event records in stdout.
3. Confirm that RBAC is allowed and the worker count is positive before assigning this cause.

**How to fix:**

1. Remove all `output` arguments to use the `logs` default, or add a separate `-output=logs` argument.
2. **DANGEROUS — enabling stdout adds allowed Kubernetes events to collector and storage load.** Estimate the event
   volume and monitor collector queues and storage ingestion during rollout.

**Sources:**

* `docs/architecture.md:111-117`.
* `controllers/events-reader/assets/deployment.yaml:27-38`.
* [Sink selection in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/main.go#L92-L135).

### Events reader omits namespaces not listed in its arguments

**Symptoms:**

* Events from one or more configured namespaces appear, but events from another namespace never appear.
* The pod is Ready and its service account can read Events.
* The effective arguments contain one or more `-namespace=<name>` entries.

**Root cause:**

With no namespace arguments, the reader creates one cluster-wide controller. When any namespace arguments are present,
version 2.9.1 creates one namespaced watcher per listed value. Events from unlisted namespaces are outside those
watches.

**How to check:**

1. Compare the missing event's namespace with every effective `namespace` argument.
2. Confirm that events from at least one listed namespace reach stdout.
3. If events from every namespace are absent, check the RBAC case instead.

**How to fix:**

1. Add the missing namespace when collection should remain scoped.
2. Remove every namespace argument when cluster-wide collection is intended.
3. **DANGEROUS — expanding the scope increases API watch, collector, and storage volume.** Confirm capacity and
   monitor ingestion after the change, especially before enabling cluster-wide collection.

**Sources:**

* `charts/qubership-logging-operator/values.yaml:1967-1973`.
* `charts/qubership-logging-operator/templates/operator/loggingservice.observability.netcracker.com.yaml:983-993`.
* [Namespace watcher selection in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/main.go#L138-L149).

### Custom event format keeps Kubernetes events out of their stream

**Symptoms:**

* Reader logs contain `Could not execute template for Event`, and the corresponding event record is empty or absent.
* Alternatively, records exist in stdout but are absent from the Graylog Kubernetes events stream.
* A raw record is not valid JSON or lacks `"kind":"KubernetesEvent"`.

**Root cause:**

The default version 2.9.1 template emits JSON with `kind=KubernetesEvent`. A custom Go template can fail during
execution, produce non-JSON output, omit `kind`, or assign another value. Logging's Graylog processing rule matches the
exact `KubernetesEvent` value, so a different schema does not enter the intended stream.

**How to check:**

1. Inspect the effective `format` argument and one raw event line from reader stdout.
2. Validate that line as JSON and inspect its `kind` value.
3. Search for the same event outside the Kubernetes events stream. Finding it elsewhere confirms a routing-schema
   mismatch.
4. For `Could not execute template for Event`, compare the custom template fields with the default template.

**How to fix:**

1. Remove the custom format to restore the default schema, or retain valid JSON and the exact
   `"kind":"KubernetesEvent"` discriminator.
2. **DANGEROUS — changing the event schema can break downstream parsers, streams, dashboards, and saved searches.**
   Validate the new record against every active output before rollout.

**Sources:**

* `controllers/graylog/config/processing_rules/k8s_event_logs.rule:1-5`.
* `docs/observability.md:101-108`.
* [Event formatting in Qubership kube events reader 2.9.1](https://github.com/Netcracker/qubership-kube-events-reader/blob/2.9.1/pkg/format/event_format.go#L13-L50).

## FluentD

### FluentD pod cannot mount dockerdaemon on a containerd cluster

**Symptoms:**

* The FluentD pod remains unready during installation.
* Pod events contain `Unable to attach or mount volumes: unmounted volumes=[dockerdaemon]`.
* The event ends with `timed out waiting for the condition` on a cluster that uses containerd.

**Root cause:**

When the effective `LoggingService.spec.containerRuntimeType` is `docker`, the generated DaemonSet mounts Docker host
paths including `/etc/docker/daemon.json` and `/var/lib/docker/containers`. Those mounts are not rendered for
`containerd` or `cri-o`. A wrong effective runtime therefore requests Docker-specific paths on non-Docker nodes. A
value entered in a deployment portal is not proof that it reached the rendered `LoggingService`.

**How to check:**

1. Compare `LoggingService.spec.containerRuntimeType` with each node's
   `.status.nodeInfo.containerRuntimeVersion`.
2. Inspect the live FluentD DaemonSet volumes. A `dockerdaemon` volume proves that the effective rendered runtime was
   Docker.
3. Inspect operator logs for `Container Runtime set in custom resource` or the discovery/default messages.
4. Compare the portal, Helm, or GitOps value with the rendered `LoggingService`; this distinguishes propagation failure
   from operator discovery.

**How to fix:**

1. Set `containerRuntimeType` to the runtime actually used by the nodes in the authoritative deployment source.
2. Render the chart or inspect the desired CR and verify that Docker-only volumes disappear before deployment.
3. **DANGEROUS — rolling the FluentD DaemonSet interrupts collection on each node; messages produced during the gap
   may not appear centrally until the agent resumes.** Apply the corrected CR during a controlled rollout and verify
   Ready agents per node.

**Data to collect:**

* `LoggingService`, node runtime versions, FluentD DaemonSet YAML, pod events, and operator runtime-selection logs.

**Sources:**

* Support ticket `PSUPCLPL-10420`, Gold 150 ticket export.
* `controllers/loggingservice_controller.go:155-187`.
* `controllers/fluentd/assets/daemon-set.yaml:174-190` and `controllers/fluentd/assets/daemon-set.yaml:281-302`.

### FluentD exits with a Docker log-driver configuration error

**Symptoms:**

* FluentD logs contain
  `[ERROR] the log-driver found in both config files (/etc/sysconfig/docker, /etc/docker/daemon.json)`.
* Alternatively, logs contain `[ERROR] the log-driver in /etc/docker/daemon.json doesn't match json-file`.
* Alternatively, logs contain `[ERROR] the log-driver in /etc/sysconfig/docker doesn't match json-file`.
* The `logging-fluentd` container exits with code 1 before FluentD starts, so the pod remains unready or restarts.

**Root cause:**

The Qubership FluentD 1.19.2-2 entrypoint reads Docker logging-driver declarations from `/etc/sysconfig/docker` and
`/etc/docker/daemon.json`. It exits when both files declare a driver, even when both values are `json-file`. When only
one file declares a driver, it exits if the declaration does not contain `json-file`.

For Docker deployments, Logging Operator mounts `/etc/docker/daemon.json` and also mounts `/etc/sysconfig/docker`
unless `osKind` is `ubuntu`. The generated Docker input tails JSON container log files, so `json-file` is part of this
integration contract.

**How to check:**

1. Inspect current and previous `logging-fluentd` logs and identify which exact entrypoint error is present.
2. Compare `LoggingService.spec.containerRuntimeType` with the node's reported runtime. If they differ, follow
   `FluentD pod cannot mount dockerdaemon on a containerd cluster` instead.
3. Inspect the DaemonSet volume mounts and effective `osKind`. An Ubuntu rendering mounts only
   `/etc/docker/daemon.json`; other supported Docker renderings mount both files.
4. Through the platform's approved read-only node procedure, inspect both source files. At most one file may expose a
   `log-driver` declaration, and any declaration must contain `json-file`. If neither file declares a driver, do not
   infer health from the entrypoint alone; separately confirm that Docker's effective driver is `json-file` through an
   approved read-only runtime inspection.

**How to fix:**

1. If both files declare `json-file`, choose the node configuration source that is authoritative for the platform and
   remove the duplicate declaration from the other source.
2. If the authoritative source declares another driver, change it to `json-file` through the approved node
   configuration process.
3. **DANGEROUS — if applying the change restarts Docker or recreates containers, workloads on that node are
   interrupted and their logs remain unavailable to FluentD until they return.** Apply the node-runtime change during
   a maintenance window, one node at a time, and verify workload recovery before continuing.
4. Confirm that the container passes entrypoint validation and FluentD opens its monitoring endpoint.

**How to avoid this issue:**

Manage the Docker logging driver in one authoritative node configuration source. Keep the effective driver set to
`json-file` wherever this FluentD Docker input is deployed.

**Data to collect:**

* Current and previous `logging-fluentd` logs and container exit status.
* The `LoggingService`, FluentD DaemonSet, node runtime and OS values, and redacted Docker configuration files.

**Sources:**

* `charts/qubership-logging-operator/templates/_helpers.tpl:313-326`.
* `controllers/fluentd/assets/daemon-set.yaml:173-190` and `controllers/fluentd/assets/daemon-set.yaml:280-300`.
* `controllers/fluentd/fluentd.configmap/conf.d/inputs/input-k8s-container.conf:1-27`.
* [Docker log-driver validation in Qubership FluentD 1.19.2-2](https://github.com/Netcracker/qubership-fluentd/blob/1.19.2-2/entrypoint.sh#L3-L29).

### FluentD probes fail with connection refused on port 24220

**Symptoms:**

* `Liveness probe failed` or `Readiness probe failed` targets `/api/plugins.json` on port `24220`.
* The probe error ends with `connect: connection refused`.
* FluentD pods do not become Ready or restart repeatedly.

**Root cause:**

The DaemonSet probes FluentD's monitoring endpoint at `/api/plugins.json:24220`. Connection refusal proves that the
container network namespace has no listener on that port at probe time; it does not identify why FluentD failed to
start. Common distinguishable paths are a configuration parse error, a plugin/startup failure, or a process that has
not completed startup. The probe error alone does not prove a repository or image defect.

**How to check:**

1. Inspect current and previous FluentD container logs from the same pod. Find the first startup error before the probe
   failures.
2. Inspect the generated `logging-fluentd` ConfigMap and compare custom fragments with the authoritative values.
3. Inspect pod events, exit code, restart count, image, and resource limits.
4. Confirm whether port `24220` ever appears as listening in FluentD startup logs; do not execute into the pod merely to
   test a hypothesis when logs already prove startup failed.

**How to fix:**

1. Correct the configuration, plugin, image, or resource problem shown by the first FluentD startup error.
2. Render and validate the corrected configuration before deployment.
3. **DANGEROUS — a FluentD rollout interrupts node log collection; messages produced during the gap may not appear
   centrally until the agent resumes.** Deploy the corrected source and verify the monitoring endpoint becomes healthy
   on every node.

**Data to collect:**

* Current and previous FluentD logs, ConfigMap, pod YAML/events, image, and custom values.

**Sources:**

* Support ticket `PSUPCLPL-10784`, Gold 150 ticket export.
* `controllers/fluentd/assets/daemon-set.yaml:68-96`.

### FluentD worker exits with SIGKILL

**Symptoms:**

* FluentD logs contain `Worker 1 exited unexpectedly with signal SIGKILL`.
* Node kernel logs identify a Ruby process killed for out-of-memory.
* After the worker restarts, FluentD performs heavy disk reads.

**Root cause:**

The repository source documents a historical deployment where the effective buffer was near 1 GB. In that affected
configuration, a container limit near 1 GiB left no room for the supervisor, worker runtime, plugins, and records, so
the node OOM killer killed the worker. The current chart renders a 512 MB default buffer; this case matches it only when
the effective configuration and kernel OOM evidence reproduce the memory-overcommit condition.

**How to check:**

1. Inspect pod termination reason, exit code, restart count, container limits, and node OOM events.
2. Inspect the effective FluentD buffer `total_limit_size` and compare it with the memory limit.
3. Confirm that the same timestamp appears in FluentD's SIGKILL message and the node's OOM record.

**How to fix:**

1. Increase the FluentD memory limit to leave runtime headroom or reduce the affected output buffer below the limit.
2. Render the new ConfigMap and DaemonSet and confirm the intended values before deployment.
3. **DANGEROUS — rolling FluentD interrupts node log collection; messages produced during the gap may not appear
   centrally until the agent resumes.** Apply the resource or buffer change during a controlled rollout and verify that
   no new OOM events occur.

**How to avoid this issue:**

Size the container for the configured aggregate buffers plus Ruby and plugin overhead; alert on OOM kills and buffer
queue growth.

**Sources:**

* `docs/troubleshooting.superseded.md:825-872`.
* `controllers/fluentd/fluentd.configmap/conf.d/outputs/output-graylog.conf:41-52`.
* [Buffer section — Fluentd documentation](https://docs.fluentd.org/configuration/buffer-section).

### FluentD causes high disk-read load after a worker restart

**Symptoms:**

* FluentD consumes unusually high read IOPS or throughput.
* The load follows `Worker 1 exited unexpectedly with signal SIGKILL` and worker initialization messages.

**Root cause:**

The repository records high disk reads immediately after a worker SIGKILL, but it does not prove which files caused
them. File-buffer replay is possible only when file storage and backlog are present; the current default uses memory
buffering. Treat the disk load as a correlated symptom of the OOM path in `FluentD worker exits with SIGKILL`, not as
an independently confirmed file-buffer diagnosis.

**How to check:**

1. Correlate the start of disk reads with the worker SIGKILL and restart timestamps.
2. Check node OOM evidence, FluentD memory limit, effective buffer type/size, and any configured file-buffer backlog as
   described in `FluentD worker exits with SIGKILL`.
3. If file buffering is disabled or no backlog exists, do not attribute the reads to buffer replay; collect
   per-process and per-file I/O evidence instead.

**How to fix:**

1. Apply the confirmed memory or buffer correction from `FluentD worker exits with SIGKILL`.
2. Let the valid backlog drain while monitoring capacity; do not delete buffered files to reduce disk reads.

**Sources:**

* `docs/troubleshooting.superseded.md:873-889`.
* `charts/qubership-logging-operator/values.yaml:717`.
* [Buffer section — Fluentd documentation](https://docs.fluentd.org/configuration/buffer-section).

### FluentD cannot flush a GELF UDP message because data is too big

**Symptoms:**

* FluentD logs contain `failed to flush the buffer` and `Data too big`.
* The error says the message `would create more than 128 chunks`.
* The affected Graylog output uses GELF over UDP.

**Root cause:**

GELF UDP messages may not exceed 128 chunks, and all chunks must arrive within the protocol window. The historical
repository document records the exact 128-chunk error. The repository does not pin the inner
`fluent-plugin-gelf-hs` and `gelf-rb` versions, so confirm the exact error before applying this case to the current
FluentD image. FluentD buffering cannot make a record that exceeds the active GELF UDP chunk limit deliverable.

**How to check:**

1. Confirm the exact `Data too big (... bytes), would create more than 128 chunks!` error and the output protocol.
2. Inspect the rendered Graylog output configuration and the size of one rejected record.
3. If the output uses TCP, this GELF UDP case does not match.

**How to fix:**

1. Change the Graylog output to TCP in the authoritative FluentD configuration.
2. If UDP is mandatory for an older deployment, constrain the record or chunk configuration so the GELF message stays
   within the documented chunk limit. Validate the rendered configuration before rollout.
3. **DANGEROUS — changing the output protocol interrupts delivery; buffered records remain stranded when their output
   cannot connect to the retained destination.** Coordinate the Graylog input and FluentD rollout and retain the old
   input until the old buffer is drained.

**How to avoid this issue:**

Use TCP for large or unpredictable log messages and enforce record-size limits before the GELF output.

**Sources:**

* `docs/troubleshooting.superseded.md:890-968`.
* [GELF via UDP — Graylog documentation](https://go2docs.graylog.org/5-0/getting_in_log_data/gelf.html?#GELFviaUDP).
* [Fluentd issue 3651](https://github.com/fluent/fluentd/issues/3651).

## FluentBit

### FluentBit times out while connecting to Graylog

**Symptoms:**

* FluentBit logs contain `connection #-1 to tcp://unavailable:0 timed out after 10 seconds`.
* Logs contain `getaddrinfo(host='<graylog_url>', err=12): Timeout while contacting DNS servers`.
* The GELF output reports `no upstream connections available`.

**Root cause:**

The documented failure combines a DNS timeout with an unavailable output connection. In affected deployments CPU
starvation can prevent timely DNS and network work; an unreachable DNS service or incorrect Graylog host can produce
the same surface. The log line does not by itself prove CPU exhaustion.

**How to check:**

1. Inspect FluentBit CPU throttling and usage during the error window.
2. Inspect the rendered Graylog host, port, protocol, and DNS-related output settings.
3. Compare DNS failures from several pods and nodes. Cluster-wide failures point away from one FluentBit container.
4. Inspect service/endpoints for an in-cluster Graylog target or the authoritative external DNS record.

**How to fix:**

1. Correct a wrong host, port, Service, endpoint, or DNS dependency demonstrated by the checks.
2. If CPU throttling coincides with the timeouts, raise FluentBit's CPU limit in authoritative values; the repository
   source suggests `1` CPU for the documented incident.
3. Where the current image supports them, configure connection timeout, worker connections, and DNS transport in the
   authoritative output configuration, then render it before deployment.
4. **DANGEROUS — rolling FluentBit interrupts collection on each node; messages produced during the gap may not appear
   centrally until the agent resumes, and a full filesystem output queue discards its oldest chunks.** Apply the
   correction in a controlled rollout and verify output retries and backlog afterward.

**Data to collect:**

* FluentBit logs and CPU throttling metrics, rendered output configuration, target Service/endpoints, and DNS evidence.

**Sources:**

* `docs/troubleshooting.superseded.md:970-1008`.
* [Buffering — Fluent Bit 4.2 documentation](https://docs.fluentbit.io/manual/4.2/data-pipeline/buffering).

### FluentBit is running but stops sending logs

**Symptoms:**

* FluentBit pods remain Running but no new logs arrive in Graylog.
* Older releases use a `rewrite_tag` emitter named `raw_parsed` and match only `parsed.**` at the output.

**Root cause:**

The repository records a workaround for a legacy pipeline containing the `raw_parsed` rewrite-tag emitter, but it does
not establish the mechanism that stopped delivery. The current built-in configuration no longer contains that block.
Treat this as a version-scoped workaround only when the effective ConfigMap proves the legacy configuration is present;
otherwise check output connectivity, backpressure, and tail state.

**How to check:**

1. Inspect the effective `filter-log-parser.conf` and `output-graylog.conf` for `Emitter_Name raw_parsed`,
   `Emitter_Storage.type filesystem`, and `Match parsed.**`.
2. Inspect FluentBit output retry, storage, and tail-input metrics. If the legacy block is absent, do not use its
   workaround.
3. Compare the deployed Logging image and ConfigMap with the current rendered chart.

**How to fix:**

1. Upgrade to a Logging release whose generated configuration does not contain the affected legacy pipeline.
2. For a legacy release only, make the documented rewrite-tag and output-match correction in the authoritative source;
   do not hand-edit the operator-owned ConfigMap as a permanent fix.
3. **DANGEROUS — rolling FluentBit interrupts collection; messages produced during the gap may not appear centrally
   until the agent resumes, and a full filesystem output queue discards its oldest chunks.** Deploy the corrected
   pipeline during a controlled rollout and verify that both raw and parsed records reach the output.

**Sources:**

* `docs/troubleshooting.superseded.md:1009-1058`.
* `controllers/fluentbit/fluentbit.configmap/conf.d/filters/filter-rewrite-tag.conf:1-40`.
* [Buffering — Fluent Bit 4.2 documentation](https://docs.fluentbit.io/manual/4.2/data-pipeline/buffering).

### HA forwarders cannot connect to aggregators on port 24224

**Symptoms:**

* After HA mode is enabled, forwarder pods remain Running but logs stop reaching the backend.
* Forward output errors name aggregator hosts on TCP port `24224`.
* The report says the forwarder cannot connect to the aggregator.

**Root cause:**

The generated forwarder upstream lists one host per configured aggregator replica:
`logging-fluentbit-aggregator-<n>.logging-fluentbit-aggregator:24224`. Three code-backed paths must be distinguished:

1. `aggregator.replicas` is zero or omitted in a hand-authored CR, so the upstream has no `[NODE]` entries even though
   the StatefulSet template defaults its replica count to two.
2. Aggregator pods are not Ready or the Service has no Ready endpoints.
3. The upstream uses pod-specific StatefulSet DNS names while the governing Service is a normal ClusterIP Service, not
   headless. Pod-specific DNS depends on the governing Service DNS contract and may not resolve as rendered.

The ticket wording alone does not choose among DNS, endpoints, TCP reachability, TLS, or an empty upstream.

**How to check:**

1. Inspect `upstream-forward.conf`. It must contain one `[NODE]` per desired replica and each port must be `24224`.
2. Compare `LoggingService.spec.fluentbit.aggregator.replicas` with StatefulSet desired/ready replicas.
3. Inspect the `logging-fluentbit-aggregator` Service and EndpointSlices. Healthy output requires Ready endpoints on
   port `24224`.
4. Ask for the exact forwarder error. A DNS error, connection refusal, timeout, TLS failure, and missing upstream have
   different causes.
5. If DNS is implicated, compare the generated pod-specific hostnames with `.spec.clusterIP` and the StatefulSet
   `serviceName`. Do not call the DNS-contract defect confirmed merely because the names look suspicious.

**How to fix:**

1. If the upstream is empty, set an explicit positive `aggregator.replicas` value in the authoritative CR or values and
   render the upstream before deployment.
2. If EndpointSlices have no Ready addresses, fix the scheduling, storage, probe, or configuration error shown by the
   aggregator pods and events.
3. If Ready endpoints exist but the generated pod-specific names do not resolve, escalate the repository defect with
   the rendered upstream, Service, StatefulSet, EndpointSlices, and exact DNS error. A source-level fix must either use
   the stable Service name or provide the headless governing Service required by the per-pod names.
4. **DANGEROUS — changing aggregator discovery interrupts the HA path; when queues reach their configured limits,
   FluentBit discards the oldest chunks.** Deploy a reviewed operator fix during a maintenance window and verify
   forward ACKs and backlog drain before removing the previous path.

**Data to collect:**

* Forwarder logs, upstream ConfigMap, `LoggingService`, Service, StatefulSet, EndpointSlices, and relevant
  NetworkPolicy.

**Sources:**

* Support ticket `PSUPCLPL-14264`, Gold 150 ticket export.
* `controllers/fluentbit-forwarder-aggregator/forwarder.configmap/conf.d/upstreams/upstream-forward.conf:1-9`.
* `controllers/fluentbit-forwarder-aggregator/assets/flb-aggregator-service.yaml:1-31`.
* `controllers/fluentbit-forwarder-aggregator/assets/flb-aggregator-stateful-set.yaml:1-23`.
* [Forward output — Fluent Bit documentation](https://docs.fluentbit.io/manual/pipeline/outputs/forward).
* [StatefulSet stable network identity — Kubernetes](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#stable-network-id).
* [Buffering — Fluent Bit 4.2 documentation](https://docs.fluentbit.io/manual/4.2/data-pipeline/buffering).

## Graylog

### Graylog UI is not reachable in a legacy VM deployment

**Symptoms:**

* A browser cannot connect to the Graylog UI.
* Fewer than four legacy Graylog Docker containers are running, or one reports a status other than `Up`.
* The Graylog VM itself may not answer network checks.

**Root cause:**

In the repository's legacy VM topology, the UI depends on the web, Graylog, OpenSearch, and MongoDB containers plus VM
network reachability. A stopped or restarting dependency prevents normal UI access. This case does not apply to the
current Kubernetes StatefulSet unless the report explicitly concerns the legacy Docker deployment.

**How to check:**

1. On the legacy VM, inspect `docker ps -f "name=graylog"`. The documented healthy result contains
   `graylog_web_1`, `graylog_graylog_1`, `graylog_storage_1`, and `graylog_mongo_1`, all `Up`.
2. If SSH is unavailable, confirm network reachability through the environment's approved read-only network check.
3. Inspect logs for the container that is absent or restarting before choosing a fix.

**How to fix:**

1. Correct the dependency-specific error shown by its logs.
2. **DANGEROUS — restarting a Graylog container interrupts the function it provides and may interrupt ingestion or
   search.** Restart only the failed container after preserving its logs and confirming the restart is appropriate.

**Sources:**

* `docs/troubleshooting.superseded.md:8-33`.

### Graylog UI opens but log messages are unavailable

**Symptoms:**

* The Graylog UI is accessible but searches do not show expected messages.
* Graylog `System > Overview` reports errors or FluentD/FluentBit health checks fail.

**Root cause:**

The repository source treats this as a pipeline-localization case: Graylog may reject or fail to index messages, or the
collector may not deliver them. The absence of messages alone does not distinguish collection, transport, processing,
stream routing, index, or search-time causes.

**How to check:**

1. Inspect `System > Overview`, index failures, inputs, journal growth, and the relevant time range.
2. Inspect the deployed FluentD or FluentBit workload health and output errors.
3. Trace one uniquely identifiable message through collector input, output, Graylog input, stream, and index. Stop at
   the first boundary where the count changes or the message disappears.

**How to fix:**

1. Apply the matched collector, input, stream, index, or storage case only after the boundary check confirms it.
2. If no existing case matches, collect the boundary evidence and escalate without inventing a cause.

**Data to collect:**

* One source log, collector logs/configuration, Graylog input and index failures, stream/index set, and search interval.

**Sources:**

* `docs/troubleshooting.superseded.md:34-41`.

### Graylog route loops through HTTP 302 redirects

**Symptoms:**

* Access to the Graylog Route repeatedly returns `302` to the same URL.
* The deployment uses the documented disaster-recovery topology without a virtual IP.
* Separate load balancers terminate HTTPS certificates before the OpenShift Route.

**Root cause:**

The legacy DR topology publishes a Route for the active Graylog UI. When a separate HTTPS load balancer does not carry
the Route host through the expected SNI passthrough mapping, requests return to the same external URL and form a
redirect loop.

**How to check:**

1. Capture the redirect chain and confirm that each `Location` returns to the same Graylog URL.
2. Confirm the deployment is the documented no-vIP DR topology with external TLS load balancers.
3. Inspect the load balancer's `os_sni_passthrough.map` through the approved configuration-review process and check for
   the Graylog Route hostname.

**How to fix:**

1. Add the Route URL to `os_sni_passthrough.map` through the load balancer's controlled configuration process.
2. Validate the map and apply it according to the load balancer's approved operational procedure. The repository does
   not specify whether the implementation needs a reload or what its availability impact is.

**Sources:**

* `docs/troubleshooting.superseded.md:42-55`.

### Graylog stops processing after its legacy VM disk fills

**Symptoms:**

* Graylog stops processing new messages and searches may return HTTP 500.
* The legacy OpenSearch container restarts repeatedly.
* `df -h` shows the Graylog VM filesystem is full.

**Root cause:**

The legacy VM has no remaining capacity for OpenSearch data and Graylog journals. OpenSearch can also place indices in
`read_only_allow_delete` at the flood-stage watermark. This case applies to the documented Docker VM paths; use
Kubernetes PVC and OpenSearch storage evidence for cloud deployments.

**How to check:**

1. Inspect `df -h`, directory usage, Docker container states, and OpenSearch logs on the legacy VM.
2. Query index settings and identify indices with `index.blocks.read_only_allow_delete=true`.
3. Inventory backups and retained indices before any deletion.

**How to fix:**

1. Safely add capacity or remove confirmed disposable data through OpenSearch retention APIs.
2. After utilization is below the high watermark, clear `read_only_allow_delete` on the affected indices.
3. **DANGEROUS — deleting `/srv/docker/graylog/opensearch/nodes/` destroys every locally stored log index.** Stop the
   four Graylog containers, remove that directory only when total data loss is accepted or a tested backup exists,
   restart dependencies in order, and restore required logs from backup.
4. **DANGEROUS — restarting the legacy Graylog stack interrupts ingestion, search, and UI access.** Perform the
   restart only as part of the approved recovery sequence.

**How to avoid this issue:**

Configure index rotation and retention below available storage and alert before OpenSearch reaches its high watermark.

**Sources:**

* `docs/troubleshooting.superseded.md:60-117`.
* [Cluster settings — OpenSearch documentation](https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/cluster-settings/).

### Graylog container is OOMKilled and the UI returns 504

**Symptoms:**

* The legacy Graylog UI is inaccessible or returns HTTP 504.
* Graylog or OpenSearch Docker containers restart repeatedly.
* Container or kernel evidence identifies an out-of-memory kill.

**Root cause:**

The configured Graylog or OpenSearch JVM heap and container memory are insufficient for the workload, or the heap
leaves no headroom for non-heap memory. Repeated restarts without OOM evidence do not prove this cause.

**How to check:**

1. Inspect container state, termination reason, kernel OOM records, effective memory limits, and JVM `-Xms`/`-Xmx`.
2. Compare heap plus other container allocations with physical VM memory.

**How to fix:**

1. Prefer redeploying the legacy service with corrected `graylog_heap_size` and `es_heap_size` parameters.
2. **DANGEROUS — stopping Docker interrupts every container on the host, including Graylog, MongoDB, and OpenSearch.**
   If the legacy manual method is unavoidable, preserve configuration and container IDs, stop Docker, update the
   documented JVM values, start Docker, and verify all containers. Use the deployment pipeline whenever possible.

**How to avoid this issue:**

Keep equal minimum and maximum JVM heap values, leave operating-system and non-heap headroom, and alert on OOM events.

**Sources:**

* `docs/troubleshooting.superseded.md:121-194`.
* [Additional configuration — Graylog documentation](https://go2docs.graylog.org/current/setting_up_graylog/additional_configuration.htm).

### Graylog UI is slow and the journal keeps growing

**Symptoms:**

* Graylog searches lag by 5–15 minutes or the UI is slow.
* The UI reports `Journal utilization is too high`.
* CPU, memory, disk IOPS, or OpenSearch throughput is saturated.

**Root cause:**

OpenSearch is accepting messages more slowly than Graylog receives them, so the disk journal grows. Disk latency is the
repository's first capacity suspect, followed by memory and CPU. The saturated resource must be demonstrated rather
than assumed.

**How to check:**

1. Inspect Graylog journal count and trend, process/input/output buffers, OpenSearch health, CPU, memory, and disk IOPS.
2. Compare deployed capacity with `docs/installation.md#hwe` and measure disk performance using an approved read-only
   benchmark result or existing monitoring data.
3. Confirm that journal growth and recent-log delay occur over the same interval.

**How to fix:**

1. Add or restore the saturated resource and reduce input volume until OpenSearch catches up.
2. **DANGEROUS — stopping Graylog inputs pauses ingestion and may move loss or backlog to upstream senders.** Pause
   inputs only with upstream owners informed, wait for buffers and journal utilization to recover, then resume them.
3. **DANGEROUS — restarting the only Ready Graylog replica, or every replica together, interrupts ingestion, UI, and
   search.** Restart only after collecting evidence and when the incident owner accepts the topology-specific
   interruption; a restart does not correct inadequate capacity.

**How to avoid this issue:**

Size primarily for OpenSearch disk throughput, monitor journal growth, and keep rotation within storage capacity.

**Sources:**

* `docs/troubleshooting.superseded.md:198-238` and `docs/troubleshooting.superseded.md:492-523`.
* [Sending in log data — Graylog documentation](https://go2docs.graylog.org/1380099/getting_in_log_data/getting_in_log_data.html).

### Graylog receives messages but does not make them searchable

**Symptoms:**

* New messages are not available in search or search fails entirely.
* `The journal contains X unprocessed messages` is above 100,000 and continues growing.

**Root cause:**

Graylog accepts messages into its journal but OpenSearch does not index them at the incoming rate. The immediate cause
can be full storage, an OOM condition, insufficient OpenSearch capacity, an unhealthy cluster, or indexing failures.
Journal growth localizes the failure after Graylog input and before successful indexing; it does not select one cause.

**How to check:**

1. Inspect Graylog `System > Nodes`, journal trend, input/output buffers, and index failures.
2. Inspect OpenSearch cluster health, disk watermarks, rejected writes, JVM/resource health, and current write index.
3. Match the evidence to `Graylog stops processing after its legacy VM disk fills`,
   `Graylog container is OOMKilled and the UI returns 504`, or
   `Graylog UI is slow and the journal keeps growing`.

**How to fix:**

1. Apply the storage, memory, capacity, or index correction confirmed by the checks.
2. Do not delete the Graylog journal to make the counter smaller; it contains messages that have not been indexed.

**Data to collect:**

* Journal trend, buffers, index failures, OpenSearch health/allocation, disk, JVM, and write-index settings.

**Sources:**

* `docs/troubleshooting.superseded.md:240-261`.

### One Graylog index grows beyond its configured maximum size

**Symptoms:**

* Logging storage utilization is high.
* One index is larger than the `Max index size` configured on its Graylog index set.

**Root cause:**

The repository documents a Graylog index-rotation failure in which an index is not rotated at the configured size. A
large index can also be explained by a different rotation strategy or a value applied to another index set, so
the effective index-set configuration must be compared before assigning the documented bug.

**How to check:**

1. Query Graylog's indexer API and identify the oversized index and its index set.
2. Inspect that index set's active rotation strategy, maximum size, current write alias, and recent rotation errors.
3. Confirm a usable backup or snapshot before considering deletion.

**How to fix:**

1. Correct the rotation configuration if it does not match the intended index set.
2. Trigger a supported rotation when the current write index is the oversized index and Graylog is otherwise healthy.
3. **DANGEROUS — deleting an index permanently removes every log stored in that index.** With an approved backup and
   after confirming it is not the active write target, delete only the named index through the Graylog indexer API.

**How to avoid this issue:**

Monitor index size against the active rotation policy and alert when a write index crosses its configured limit.

**Sources:**

* `docs/troubleshooting.superseded.md:262-291`.
* [Index model — Graylog documentation](https://go2docs.graylog.org/1380099/setting_up_graylog/index_model.html).

### Graylog shows a negative number of unprocessed journal messages

**Symptoms:**

* The `Disk Journal` section shows a negative unprocessed-message count.
* Journal files were manually removed previously.

**Root cause:**

The repository documents this state after the journal directory is cleaned only partially. On-disk journal contents
and Graylog's journal accounting no longer describe the same data.

**How to check:**

1. Confirm the negative counter and collect Graylog logs around journal recovery.
2. Inspect the configured `message_journal_dir` and record its size and files without deleting anything.
3. Confirm whether a prior manual cleanup occurred and whether any unprocessed logs must be preserved.

**How to fix:**

1. Prefer restoring the journal from a consistent backup or escalating when unprocessed messages must be preserved.
2. **DANGEROUS — clearing the journal permanently loses every message that has not reached OpenSearch and stopping
   Graylog interrupts ingestion.** Stop Graylog, remove the complete configured journal contents only after explicit
   acceptance of that loss, start Graylog, and verify the counter resets.
3. To disable the journal, change `message_journal_enabled=false` in authoritative configuration only after accepting
   the loss of disk-backed protection against downstream outages.

**Sources:**

* `docs/troubleshooting.superseded.md:292-332`.

### Graylog displays inconsistent timestamps

**Symptoms:**

* The `message`, `time`, and `timestamp` fields display different time-zone offsets.
* Events from different nodes appear shifted in searches.

**Root cause:**

The repository source attributes this to inconsistent node time zones or to a Graylog user-display timezone that
differs from UTC. Changing the user setting changes presentation but does not rewrite a timestamp embedded in the
original `message` field.

**How to check:**

1. Compare the raw source timestamp, parsed `timestamp`, Graylog user timezone, and the source node's timezone.
2. Verify time synchronization and UTC settings on all producing nodes.
3. Determine whether the difference is only display formatting or incorrect parsing of the source text.

**How to fix:**

1. Standardize producing nodes and parsers on UTC.
2. Set the Graylog user's display timezone to the intended presentation timezone when stored timestamps are correct.
3. Correct the parser when the source timezone is being interpreted incorrectly; changing the UI setting cannot fix
   the original `message` text.

**Sources:**

* `docs/troubleshooting.superseded.md:333-341`.

### Graylog cannot display OpenSearch node information over TLS

**Symptoms:**

* `System > Nodes` says OpenSearch or Elasticsearch node information is unavailable.
* Opening node details produces a TLS or certificate error.

**Root cause:**

The certificate presented by the endpoint named in the node-details TLS error is expired, untrusted, or lacks the
service DNS name used by that request. The repository example uses `graylog-service.logging.svc`; this case does not
assign an OpenSearch certificate failure unless the error names the OpenSearch endpoint.

**How to check:**

1. Capture the exact certificate error and endpoint name from Graylog logs or node details.
2. Inspect certificate validity, issuer chain, and SANs from the referenced Secret or certificate artifact.
3. Compare the SANs with the service name and namespace in the rendered configuration.

**How to fix:**

1. Renew or replace the certificate with the correct trust chain and SANs through the configured cert-manager or
   secret-management process.
2. **DANGEROUS — when no Ready replica remains during certificate reload, the endpoint is unavailable until a replica
   becomes Ready.** Roll only components that consume the changed Secret, preserve at least one Ready replica when the
   topology supports it, and verify TLS before continuing.

**How to avoid this issue:**

Monitor certificate expiry and render service DNS names into certificate SANs from the same deployment values.

**Sources:**

* `docs/troubleshooting.superseded.md:342-359`.
* `docs/user-guides/tls.md:1-220`.

### Graylog widgets fail because a field has the wrong mapping

**Symptoms:**

* A widget reports `Unable to perform search query: Elasticsearch exception`.
* The reason says `Text fields are not optimized for operations that require per-document field data`.
* The message recommends a keyword field or mentions `fielddata=true` for a named field such as `timestamp`.

**Root cause:**

A custom index or stream targets an index whose dynamic or explicit mapping assigned an incompatible type to a field
used for aggregation or sorting. OpenSearch dynamic mapping can select `text`, while Graylog expects a sortable or
aggregatable type such as `keyword` or `date`.

**How to check:**

1. Extract the field name from the error.
2. Query `GET /_mapping/field/<field>` or `GET /<index>/_mapping/field/<field>` and compare mappings across affected
   indices.
3. Inspect the index template and the stream-to-index-set assignment that controls new indices.

**How to fix:**

1. Correct the index template with the intended explicit field type so newly rotated indices receive it.
2. Rotate to a new index with the corrected template. Preserve the old index until a separately approved migration
   procedure defines how existing records are moved.
3. Do not enable `fielddata` merely because the error suggests it; the message itself warns that it can consume
   significant memory.

**How to avoid this issue:**

Declare fields used by Graylog widgets explicitly and test custom stream/index templates before production ingestion.

**Sources:**

* `docs/troubleshooting.superseded.md:360-428`.
* [Mappings — OpenSearch documentation](https://docs.opensearch.org/latest/mappings/).
* [Index model — Graylog documentation](https://go2docs.graylog.org/1380099/setting_up_graylog/index_model.html).

### Graylog deflector exists as an index instead of an alias

**Symptoms:**

* Graylog Overview reports `Deflector exists as an index and is not an alias`.
* OpenSearch contains `<index_prefix>_deflector` as an index rather than an alias.
* Logs may say `Active write index for index set ... doesn't exist yet` immediately before automatic index creation.

**Root cause:**

Graylog reserves `<index_prefix>_deflector` as the write alias. A user can create a conflicting index manually, or
agents can write during an update after a stream exists but before Graylog creates its target index and alias. With
OpenSearch automatic index creation enabled, the premature write creates a real `_deflector` index.

**How to check:**

1. Query aliases and indices for the exact `_deflector` name and confirm its object type.
2. Correlate Graylog's missing-active-index warning with OpenSearch's automatic index-creation log.
3. Confirm the intended target index and backup any data stored in the conflicting index.

**How to fix:**

1. Stop new writes through the affected input or deployment sequence before correcting the alias.
2. **DANGEROUS — stopping Graylog inputs pauses ingestion and can move backlog or loss upstream.** Coordinate the
   pause with senders and record the start time.
3. **DANGEROUS — deleting the conflicting `_deflector` index permanently deletes any records it contains.** Preserve
   or reindex required records, verify the backup, delete only the confirmed conflicting index, and let Graylog create
   the intended target and alias.
4. Resume inputs only after the alias points to a valid write index.

**How to avoid this issue:**

Never create indices ending in `_deflector`. During updates that create streams and custom indices, pause inputs until
the target indices and aliases exist.

**Sources:**

* `docs/troubleshooting.superseded.md:429-491`.
* [Index model — Graylog documentation](https://go2docs.graylog.org/1380099/setting_up_graylog/index_model.html).

## Graylog auth proxy

### Graylog auth proxy exits after the documented OAuth example is applied

**Symptoms:**

* The `graylog-auth-proxy` sidecar exits and restarts while Graylog is configured for OAuth.
* Its logs contain `Invalid OAuth2 config: attempt to use incorrect scheme for OAuth authorization server`.
* The rendered `graylog-auth-proxy-config` contains `oauth-host: ""` although values contain `oauth.url`.

**Root cause:**

The repository's Keycloak examples use `graylog.authProxy.oauth.url`, but the chart values and ConfigMap template read
`graylog.authProxy.oauth.host`. Helm accepts the unused `url` key, leaves `host` empty, and renders an empty
`oauth-host`. Qubership Graylog auth proxy 0.2.3 validates the URL scheme at startup and exits when it is empty.

**How to check:**

1. Inspect current and previous `graylog-auth-proxy` sidecar logs for the exact validation error.
2. Render the chart and inspect `graylog-auth-proxy-config`. `oauth-host` must contain an `http://` or `https://` URL.
3. Compare the authoritative values with the chart schema. `oauth.url` is unused; `oauth.host` is the supported key.
4. If `oauth-host` is populated, follow the next specific validation error instead of assigning this cause.

**How to fix:**

1. Replace `graylog.authProxy.oauth.url` with `graylog.authProxy.oauth.host` and retain the required OAuth paths,
   client credentials, and TLS settings.
2. Render the ConfigMap and confirm that `oauth-host` contains the intended URL before deployment.
3. **DANGEROUS — restarting the Graylog pod to reload the subPath-mounted config interrupts auth-proxy access and can
   interrupt Graylog UI, search, and ingestion when no other Ready replica serves them.** Roll the StatefulSet during
   a maintenance window and verify both containers become Ready.

**How to avoid this issue:**

Validate examples against the values schema and add a Helm schema rule that rejects unknown OAuth keys.

**Data to collect:**

* Redacted auth-proxy values, rendered ConfigMap, sidecar logs, and Graylog pod container statuses.

**Sources:**

* `docs/user-guides/graylog-auth-proxy.md:394-420` and `docs/user-guides/graylog-auth-proxy.md:422-460`.
* `charts/qubership-logging-operator/values.yaml:580-641`.
* `charts/qubership-logging-operator/templates/graylog/auth-proxy/configmap.yaml:62-95`.
* `controllers/graylog/assets/statefulset.yaml:463-480`.
* [OAuth validation in Qubership Graylog auth proxy 0.2.3](https://github.com/Netcracker/qubership-graylog-auth-proxy/blob/0.2.3/config/oauth.py#L63-L91).

### LDAP user authenticates but Graylog auth proxy returns 401 when memberOf is empty

**Symptoms:**

* Auth-proxy logs contain `Auth OK for user "<user>"` followed by `Authentication failed` for the same user.
* The browser or API receives HTTP 401 even though the LDAP bind succeeded.
* The LDAP entry exists but has no `memberOf` values.

**Root cause:**

Qubership Graylog auth proxy 0.2.3 returns the LDAP user's `memberOf` list after a successful bind. Its request handler
treats both `None` and an empty list as authentication failure. A downstream branch that assigns default Graylog roles
to an empty `memberOf` list is therefore unreachable through the LDAP handler.

**How to check:**

1. Correlate `Auth OK for user "<user>"` and the subsequent `Authentication failed` entry in the same request.
2. Through the approved directory-support process, inspect the user's sanitized `memberOf` attribute.
3. If logs say `No objects found in LDAP` or the bind itself fails, this case does not match.
4. Inspect the image tag; this traced behavior applies to the chart's default auth-proxy version `0.2.3`.

**How to fix:**

1. If identity policy requires group membership, add the user to an approved LDAP group that the proxy may map.
2. **DANGEROUS — changing LDAP group membership changes authorization and may grant access in systems beyond
   Graylog.** Obtain identity-owner approval and verify the effective roles before applying that workaround.
3. Otherwise, patch the handler to reject only `None` and allow the existing default-role branch to handle an empty
   list. Build and review a replacement image before changing `graylog.authProxy.image`.
4. **DANGEROUS — rolling the Graylog StatefulSet interrupts auth-proxy access and can interrupt Graylog services when
   no other Ready replica serves them.** Deploy the reviewed image during a maintenance window.

**Data to collect:**

* Correlated auth-proxy logs, sanitized LDAP attributes, role mapping, image tag, and returned HTTP status.

**Sources:**

* `charts/qubership-logging-operator/templates/_helpers.tpl:432-445`.
* `docs/user-guides/graylog-auth-proxy.md:86-110`.
* [LDAP handler in Qubership Graylog auth proxy 0.2.3](https://github.com/Netcracker/qubership-graylog-auth-proxy/blob/0.2.3/ldap_auth_handler/handler.py#L185-L200).
* [LDAP connector in Qubership Graylog auth proxy 0.2.3](https://github.com/Netcracker/qubership-graylog-auth-proxy/blob/0.2.3/ldap_auth_handler/ldap_connector.py#L93-L130).
* [Graylog default-role branch in Qubership Graylog auth proxy 0.2.3](https://github.com/Netcracker/qubership-graylog-auth-proxy/blob/0.2.3/common/graylog.py#L43-L64).

## OpenSearch

### OpenSearch rejects new fields after the total-fields limit

**Symptoms:**

* OpenSearch or Graylog logs contain `Limit of total fields [1000] in index [<index>] has been exceeded`.
* Messages with newly introduced fields fail to index.

**Root cause:**

Dynamic mappings accumulated the configured maximum number of fields. In the repository-documented incident, an agent
incorrectly parsed arbitrary `key=value` fragments from message text into new field names. External senders and custom
parsers can produce the same mapping explosion.

**How to check:**

1. Inspect the affected index mapping and count or identify rapidly changing field names.
2. Compare suspicious fields with the original message and the effective FluentD/FluentBit parser configuration.
3. Identify which collector or direct sender first introduced those fields.
4. Inspect `index.mapping.total_fields.limit`; increasing it does not correct runaway field generation.

**How to fix:**

1. Upgrade or correct the parser/sender that creates dynamic garbage fields, then verify a new sample document.
2. Rotate to a clean index with an explicit template for expected fields.
3. **DANGEROUS — `_update_by_query` rewrites documents, can consume substantial CPU and I/O, and an incorrect script
   can corrupt indexed data.** Snapshot the index, test the field-removal script on a small copy, and run it only with
   monitored capacity when historical cleanup is required.
4. Clear a write block only after its independent cause is fixed; see `OpenSearch indices become read-only after disk
   pressure`.

**How to avoid this issue:**

Use explicit mappings, constrain dynamic parsing, and alert on field-count growth before the limit is reached.

**Sources:**

* `docs/troubleshooting.superseded.md:539-641`.
* [Mapping explosion — OpenSearch documentation](https://docs.opensearch.org/latest/mappings/mapping-explosion/).

### OpenSearch logs no such index for opendistro-ism-config

**Symptoms:**

* Logs contain `IndexNotFoundException[no such index [.opendistro-ism-config]]`.
* The message comes from `ManagedIndexCoordinator` when no ISM policy has been created.

**Root cause:**

The OpenSearch Index Management plugin observes a cluster change before its own configuration index exists. Upstream
confirmed that this was intended behavior with an inappropriate ERROR log level and changed it to DEBUG in plugin
release `2.10.0.0`. By itself this message has no negative effect.

**How to check:**

1. Confirm the exact index name and logger and check whether any separate indexing or ISM operation actually failed.
2. Determine the shipped OpenSearch/index-management version and whether ISM is configured.
3. If user-visible behavior fails, this cosmetic message is not enough to explain it; continue diagnosis.

**How to fix:**

1. Ignore the message when it is the only symptom.
2. If ISM is required, create the intended policy through the normal configuration process rather than creating the
   system index manually.
3. Upgrade to a compatible release containing the log-level correction when the deployed Logging release supports it.

**Sources:**

* `docs/troubleshooting.superseded.md:642-691`.
* [OpenSearch index-management issue 697](https://github.com/opensearch-project/index-management/issues/697).
* [Index-management release 2.10.0.0](https://github.com/opensearch-project/index-management/releases/tag/2.10.0.0).

### OpenSearch performance worsens with a heap above 32 GB

**Symptoms:**

* OpenSearch becomes slower or OOM failures continue after heap is raised beyond approximately 32 GB.
* Throughput is lower than with a heap below the compressed ordinary object pointer threshold.

**Root cause:**

Large JVM heaps can lose compressed ordinary object pointers and use wider references, reducing effective memory
efficiency. The repository's Graylog guidance recommends roughly half of system memory for OpenSearch without crossing
approximately 32 GB. The actual threshold is JVM-dependent, so the effective JVM mode is stronger evidence than the
configured number alone.

**How to check:**

1. Inspect effective `-Xms`/`-Xmx`, container or VM memory, JVM compressed-oops mode, GC, and process RSS.
2. Confirm Graylog, MongoDB, the operating system, and filesystem cache still have headroom.
3. Compare throughput and GC before and after the heap change using the same workload window.

**How to fix:**

1. Set equal minimum and maximum OpenSearch heap below the compressed-oops threshold and leave substantial host memory
   for other processes and filesystem cache.
2. **DANGEROUS — restarting every OpenSearch node, or the only node, stops Graylog indexing and search until storage
   returns.** Apply the heap correction through the deployment source and use the topology's supported rolling
   procedure when enough healthy nodes exist.

**Sources:**

* `docs/troubleshooting.superseded.md:692-738`.
* [OpenSearch setup — Graylog documentation](https://go2docs.graylog.org/current/setting_up_graylog/opensearch.htm).

### OpenSearch indices become read-only after disk pressure

**Symptoms:**

* Graylog index failures contain `index <name> is read-only`.
* Input or output counters advance but recent logs are absent from search.
* Index settings contain `index.blocks.read_only_allow_delete=true`.

**Root cause:**

OpenSearch places indices on a node above the flood-stage disk watermark into `read_only_allow_delete` protection. The
current upstream default flood stage is 95%, and the block is released after utilization falls below the high
watermark. Old installations or overrides can use different thresholds, so inspect effective cluster settings.

**How to check:**

1. Inspect filesystem/PVC use, effective disk watermarks, shard allocation, and index blocks.
2. List indices and sizes and identify retention candidates without deleting them.
3. Confirm backups and the active Graylog write indices before planning cleanup.

**How to fix:**

1. Add capacity or let configured retention remove expired data.
2. **DANGEROUS — deleting indices permanently removes all logs in them.** With an approved retention decision and a
   verified backup, delete only named expired indices through the OpenSearch or Graylog API.
3. Once disk use is safely below the high watermark, clear `index.blocks.read_only_allow_delete` on affected indices
   if the deployed version has not released it automatically.
4. **DANGEROUS — disabling disk allocation thresholds removes protection against a completely full disk, which can
   make the cluster unavailable and risk data loss.** Do not disable the allocator in production; if an emergency
   exception is approved, time-bound it, monitor free space continuously, and restore the protection immediately.

**How to avoid this issue:**

Keep total Graylog retention below available storage and alert before the high and flood-stage watermarks.

**Sources:**

* `docs/troubleshooting.superseded.md:739-823`.
* [Cluster settings — OpenSearch documentation](https://docs.opensearch.org/latest/install-and-configure/configuring-opensearch/cluster-settings/).
* [ISM blocked-index resolution — OpenSearch documentation](https://docs.opensearch.org/latest/im-plugin/ism/error-prevention/resolutions/).

## ConfigMap reloader

### Fluent container restarts after a ConfigMap change

**Symptoms:**

* A FluentD or FluentBit container restarts after its ConfigMap changes.
* FluentD logs contain `Fluent::ConfigParseError` and identify a file and line, for example
  `unmatched end tag at filter-add-hostname.conf line 6,12`.

**Root cause:**

The ConfigMap reloader causes the collector to load the changed generated configuration. A syntax error in a custom or
manually edited fragment makes FluentD exit; the workload then restarts it. The restart is an effect of the parser
failure, not proof that the reloader itself is faulty.

**How to check:**

1. Inspect current and previous collector logs and record the first parse error, file, line, and column.
2. Compare that rendered file with the authoritative custom configuration value.
3. Render or parse the full configuration offline; a healthy result contains no unmatched tags or invalid sections.

**How to fix:**

1. Correct the malformed fragment in the authoritative values, not only in the live operator-owned ConfigMap.
2. Validate the full generated configuration before deployment.
3. **DANGEROUS — reloading or rolling the collector interrupts collection; messages produced during the gap may not
   appear centrally until the collector resumes.** Deploy the corrected source and verify the main container stays
   Ready after the reload.

**Data to collect:**

* Previous container logs, rendered ConfigMap, source custom values, pod restart history, and reloader logs.

**Sources:**

* `docs/troubleshooting.superseded.md:1059-1082`.
