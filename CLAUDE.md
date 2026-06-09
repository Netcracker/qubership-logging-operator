# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Qubership Logging Operator is a Kubernetes operator that deploys and manages a logging stack: Graylog, FluentD, FluentBit, and a K8S Events Reader. It uses a single CRD (`LoggingService` in API group `logging.netcracker.com/v1`) to reconcile the entire stack.

## Common Commands

### Build & Run
```bash
make all              # Full pipeline: generate → test → build → image → docs → archives
make build-binary     # Compile Go binary to build/_binary/manager
make generate         # Regenerate CRDs and deepcopy (controller-gen v0.16.5)
make image            # Build Docker image
make fmt              # go fmt ./...
make run              # Run operator locally against ~/.kube/config
```

### Testing
```bash
make test                              # Alias for unit-test
make unit-test                         # go test -race with shuffle, excludes e2e-tests
go test -race -run TestName ./controllers/...  # Run a single test
```

Integration tests use Robot Framework in `test/robot-tests/` and run via GitHub Actions.

### Documentation
```bash
make docs             # Generate API docs and copy CRDs to docs/
```

## Architecture

### Go Module Structure
The project uses Go workspaces (`go.work`) with two modules:
- `.` — main operator module (`github.com/Netcracker/qubership-logging-operator`)
- `./api` — CRD types module (independently versioned)

### Entry Point
`cmd/operator/main.go` — sets up controller-runtime manager, scoped to `WATCH_NAMESPACE` env var (defaults to `"logging"`). Exposes metrics on `:8383` and optional pprof on `:9180`.

### Controller Hierarchy
`LoggingServiceReconciler` (`controllers/loggingservice_controller.go`) orchestrates component-specific reconcilers:

| Package | Component | K8s Resource |
|---|---|---|
| `controllers/graylog/` | Graylog + MongoDB sidecar | StatefulSet |
| `controllers/fluentd/` | FluentD | DaemonSet |
| `controllers/fluentbit/` | FluentBit (standard mode) | DaemonSet |
| `controllers/fluentbit-forwarder-aggregator/` | FluentBit HA mode (forwarder + aggregator) | DaemonSet + StatefulSet |
| `controllers/events-reader/` | CloudEventsReader | Deployment |
| `controllers/utils/` | Shared utilities (labels, status, pod management) | — |

Each component reconciler uses embedded YAML templates (`go:embed`) for manifest generation and ConfigMap-based configuration.

### Reconciliation Pattern
- Exponential backoff on failures (starts at 1s, doubles via `TimeoutOnFailedReconcile`)
- Container runtime auto-detection (docker, containerd, cri-o) from cluster nodes; defaults to `containerd`
- Status tracking per component via `StatusUpdater`

### CRD
Single CRD defined in `api/v1/loggingservice_types.go`. Generated CRD YAML lives in `charts/qubership-logging-operator/crds/`. After modifying types, run `make generate`.

### Data Flow
```
App Pods → FluentBit (DaemonSet) → [optional FluentD] → Graylog → OpenSearch/Elasticsearch
                  ↓ (HA mode)
         FluentBit Aggregator (StatefulSet)

K8s Events → CloudEventsReader → FluentBit → Graylog

Alternative outputs: Loki, Splunk, CloudWatch, Kafka, HTTP
```

### Helm Charts
- `charts/qubership-logging-operator/` — main operator chart (values.yaml ~100KB, values.schema.json for validation)
- `charts/qubership-logging-crds/` — standalone CRD chart for installing CRDs independently
