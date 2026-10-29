# Repository agent instructions

## Project Overview

Qubership Logging Operator is a Kubernetes operator that deploys and manages a logging stack: Graylog, FluentD, FluentBit, and a K8S Events Reader. It uses a single CRD (`LoggingService` in API group `logging.netcracker.com/v1`) to reconcile the entire stack.

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
make test-fluent-pipeline              # Run the Fluent Bit pipeline test
make test-fluent-pipeline FLUENT_PIPELINE_SCENARIO=fluentbit-ha  # Run another pipeline scenario
go test -race -run TestName ./controllers/...  # Run a single test
```

Fluent pipeline tests support the `fluentbit`, `fluentbit-ha`, `fluentd`, `kube-metadata`, and `render` scenarios.
`kube-metadata` runs the Kubernetes filter against a fake API server; `render` only renders and validates the agent
configurations for the custom resources under `test/fluent-pipeline/testdata/assets/render/`. In CI they run as the
`fluent_pipeline` job of `.github/workflows/integration-tests.yaml` and answer to `Integration Gate`. Integration tests use Robot
Framework in `test/robot-tests/` and run via GitHub Actions.

### Documentation

```bash
make docs             # Generate API docs and copy CRDs to docs/
```

## Commands

- Run `go test ./api/...` as well as `make unit-test` when you change `api/`. `api/` is a separate Go module in
  `go.work`, and `make unit-test` covers only the root module.
- Check changes to `charts/qubership-logging-operator/` with `helm lint charts/qubership-logging-operator -f <file>`
  for the `*-values.yaml` files in `docs/examples/*/`. CI lints the chart with each of those files as values, so a
  values or schema change can break an example.
- Run the scripts in `scripts/chart-tests/` after changing a chart: `test-monitoring-selectors.sh` covers
  `charts/qubership-logging-operator/`, and `test-victorialogs-rendering.sh` covers `charts/qubership-victorialogs/`.
- Run `make test-fluent-pipeline FLUENT_PIPELINE_SCENARIO=<scenario>` after changing FluentBit or FluentD
  configuration under `controllers/`. Go unit tests do not run the rendered configuration; this target runs it in
  Docker against log fixtures. The default scenario is `fluentbit`; `test/fluent-pipeline/README.md` lists the others.

## Non-obvious invariants

- Regenerate the CRD and deepcopy code with `make generate` after changing `api/v1/loggingservice_types.go`. Do not
  run `controller-gen` directly or edit `charts/qubership-logging-operator/crds/` by hand: the target also adds the
  Helm hook annotations, the operator version annotation, and the common labels.
- `make generate` updates only `charts/qubership-logging-operator/crds/`. Copy the result to the other two locations
  with `make update-crds` (`charts/qubership-logging-crds/crds/`) and `make -B docs/crds` (`docs/crds/`). Without
  `-B`, `make docs/crds` and `make docs` skip the copy because the `docs/crds` directory already exists.
- A new `LoggingService` field is not configurable through Helm until you wire it by hand. Map it in
  `charts/qubership-logging-operator/templates/operator/loggingservice.observability.netcracker.com.yaml`, which
  builds the CR field by field, then add it to `values.yaml`, `values.schema.json`, and
  `docs/installation-parameters.md`.
- Do not edit `charts/qubership-logging-operator/README.md` or `docs/api.md`; both are generated. Change the `# --`
  comments in `values.yaml` or `README.md.gotmpl` and run `make docs/helm`, or change the Go doc comments in
  `api/v1/` and run `make docs/api.md`.
- The operator creates the Graylog, FluentD, FluentBit, and events reader workloads and their configuration from
  templates embedded in `controllers/<component>/` (`assets/`, `*.configmap/`, `config/`). The component directories
  in `charts/qubership-logging-operator/templates/` hold supporting resources such as RBAC, certificates, and
  monitoring. The exception is the Graylog auth proxy configuration, which lives in the chart under
  `templates/graylog/auth-proxy/`.
- FluentBit configuration exists separately for each mode: `controllers/fluentbit/fluentbit.configmap/` for the
  standard mode, and `forwarder.configmap/` and `aggregator.configmap/` under
  `controllers/fluentbit-forwarder-aggregator/` for the HA mode. The copies have diverged, so check each one when
  you change a parser, filter, output, or Lua script, and apply the change wherever the same logic exists.
- A change to a parser, a filter, or the emitted fields must update the expected results in
  `test/fluent-pipeline/testdata/` (`output/<scenario>/` and `parser-cases.json`) in the same change. Remove the
  cases of a parser you delete.
- `Dockerfile` copies only `api/`, `controllers/`, `cmd/operator/main.go`, and `go.*` into the build stage. Add a
  `COPY` line when you add another top-level Go package directory or a second file in `cmd/operator/`; otherwise
  local builds pass and the image build fails.
- `docs/troubleshooting.md` is a symlink to
  `agent-packages/troubleshoot-logging/.apm/skills/troubleshoot-logging/references/troubleshooting.md`. Edit the
  target. In a checkout without symlink support, Git reports the link as a type change; do not commit that change.
- Do not run `apm compile` or `apm install` in the repository root. Only the directories in `agent-packages/` are
  APM packages; the root has no `apm.yml`, so edit the root `AGENTS.md` and `CLAUDE.md` directly.
- Keep the delimiter row of a Markdown table aligned with its header row after you edit the table. Super-Linter
  enforces this in CI with `.github/linters/.markdownlint.yaml`.
