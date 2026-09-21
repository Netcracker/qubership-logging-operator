# AGENTS.md

Guidance for coding agents working in this repository.

## Project overview

Qubership Logging Operator is a Kubernetes operator that deploys and manages a logging stack: Graylog, FluentD,
FluentBit, and a Kubernetes events reader. A single CRD (`LoggingService` in API group `logging.netcracker.com/v1`)
drives reconciliation of the whole stack.

## Common commands

### Build and run

```bash
make all              # Full pipeline: generate → test → build-binary → image → docs → archives
make build-binary     # Run generate and fmt, then compile the binary to build/_binary/manager
make generate         # Regenerate CRDs and deepcopy code (controller-gen v0.20.1)
make image            # Build the Docker image
make fmt              # go fmt ./...
make vet              # go vet ./... (not wired into build-binary; see the TODO in the Makefile)
make run              # Run the operator locally against ~/.kube/config
```

### Testing

```bash
make test                                      # unit-test plus python-test
make unit-test                                 # go test -race -cover with --shuffle=on, excludes /e2e-tests
make python-test                               # unittest discover -s scripts/log-analysis/tests
go test -race -run TestName ./controllers/...  # Run a single Go test
```

Integration tests use Robot Framework in `test/robot-tests/` and run in GitHub Actions.

### Documentation

```bash
make docs             # Generate docs/api.md, refresh docs/crds, copy CRDs into the CRD chart, run helm-docs
```

## Architecture

### Go module structure

The project uses a Go workspace (`go.work`, Go 1.26) with two modules:

- `.` — main operator module (`github.com/Netcracker/qubership-logging-operator`)
- `./api` — CRD types module (`github.com/Netcracker/qubership-logging-operator/api`), versioned independently

### Entry point

`cmd/operator/main.go` sets up the controller-runtime manager, scoped to the namespace in `WATCH_NAMESPACE`
(defaults to `logging`). Default bind addresses: metrics on `:8080`, health and readiness probes on `:8081`
(`/health` and `/ready`), and pprof on `:9180` — pprof is enabled by default and can be turned off with
`--pprof-enable=false`.

### Controller hierarchy

`LoggingServiceReconciler` (`controllers/loggingservice_controller.go`) orchestrates the component reconcilers:

| Package                                       | Component                                         | Kubernetes resource     |
|-----------------------------------------------|---------------------------------------------------|-------------------------|
| `controllers/graylog/`                        | Graylog and its MongoDB sidecar                   | StatefulSet             |
| `controllers/fluentd/`                        | FluentD                                           | DaemonSet               |
| `controllers/fluentbit/`                      | FluentBit (standard mode)                         | DaemonSet               |
| `controllers/fluentbit-forwarder-aggregator/` | FluentBit HA mode (forwarder and aggregator)      | DaemonSet + StatefulSet |
| `controllers/events-reader/`                  | CloudEventsReader                                 | Deployment              |
| `controllers/utils/`                          | Shared utilities (labels, status, pod management) | —                       |

Each component reconciler embeds its YAML templates with `go:embed` and configures the component through ConfigMaps.

### Reconciliation pattern

- A failed reconcile requeues after `TimeoutOnFailedReconcile`, which starts at 1s and doubles on each failure. A
  successful reconcile resets it to 1s.
- `spec.containerRuntimeType` wins when set. Otherwise the operator detects the runtime (docker, containerd, cri-o)
  from the cluster nodes and falls back to `containerd`.
- Per-component status is tracked through `StatusUpdater`.

### CRD

The single CRD is defined in `api/v1/loggingservice_types.go`. Generated CRD YAML lives in
`charts/qubership-logging-operator/crds/`. Run `make generate` after changing the types.

### Data flow

```text
App Pods → FluentBit (DaemonSet) → [optional FluentD] → Graylog → OpenSearch/Elasticsearch
                  ↓ (HA mode)
         FluentBit Aggregator (StatefulSet)

K8s Events → CloudEventsReader → FluentBit → Graylog

Alternative outputs: Loki, Splunk, CloudWatch, Kafka, HTTP
```

### Helm charts

- `charts/qubership-logging-operator/` — main operator chart (large `values.yaml`, validated by `values.schema.json`)
- `charts/qubership-logging-crds/` — standalone chart for installing the CRDs separately
- `charts/qubership-victorialogs/` — VictoriaLogs deployment for Kubernetes; templates only, no operator Go code

### Agent packages

`agent-packages/` holds APM packages with their own `.apm/` sources and `apm.yml` manifests
(`troubleshoot-logging`, `qubership-ndjson-logging-migration`). Edit the sources under `.apm/`, not the files that
`apm compile` generates. The repository root is not APM-managed.
