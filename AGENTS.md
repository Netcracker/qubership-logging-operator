# Repository agent instructions

## Scope

- Qubership Logging Operator manages Graylog, FluentD, FluentBit, and the Kubernetes Events Reader through one
  `LoggingService` custom resource in `logging.netcracker.com/v1`.
- These instructions apply repository-wide. Put narrower guidance next to the affected subtree.

## Repository map

- `cmd/operator/main.go` configures the controller-runtime manager. It watches `WATCH_NAMESPACE` (default: `logging`),
  serves metrics on `:8080`, and can serve pprof on `:9180`.
- The Go workspace contains the root operator module and the independently versioned `api/` module for the
  `LoggingService` API. Generated CRDs live under `charts/qubership-logging-operator/crds/`.
- `controllers/loggingservice_controller.go` orchestrates reconcilers in `controllers/graylog/`,
  `controllers/fluentd/`, `controllers/fluentbit/`, `controllers/fluentbit-forwarder-aggregator/`, and
  `controllers/events-reader/`. Shared reconciliation helpers live in `controllers/utils/`.
- Graylog runs with a MongoDB sidecar, and the FluentBit HA aggregator uses a StatefulSet. FluentD, standard FluentBit,
  and the HA forwarder use DaemonSets; the Events Reader uses a Deployment.
- Component `manifest.go` files embed their adjacent YAML and configuration assets. Keep the assets and the Go code
  that consumes them consistent.
- Application and Kubernetes event logs flow through FluentBit, optional FluentD or an HA aggregator, and Graylog with
  OpenSearch or Elasticsearch. Other output integrations include Loki, Splunk, CloudWatch, Kafka, and HTTP.
- `charts/qubership-logging-operator/` is the operator chart; validate values against its `values.schema.json`.
  `charts/qubership-logging-crds/` packages the CRDs separately.
- `agent-packages/` contains APM packages that are versioned separately from the operator runtime.

## Commands

- Format Go code: `make fmt`.
- Run controller tests: `go test -race ./controllers/...`.
- Run one Go test by name: `go test -race -run TestName ./controllers/...`; replace `TestName` with the test name.
- Run API module tests: `go test -race ./api/...`.
- Run the root Go unit-test suite: `make unit-test`.
- Run the repository source tests: `make test`. This runs the Go race suite and the Python log-analysis tests.
- Run Go static analysis: `make vet`. The build target does not include this check.
- Regenerate CRDs and deepcopy code after API changes: `make generate`.
- Build the manager binary: `make build-binary`.
- Build the Docker image: `make image`.
- Generate API, CRD, and Helm documentation: `make docs`.
- Run the full artifact pipeline only when images, documentation, and archives are in scope: `make all`.
- Run the operator against the active kubeconfig only when cluster-backed verification is in scope: `make run`.

## Non-obvious invariants

- Treat `api/v1/loggingservice_types.go` and its Kubebuilder markers as the source for the API. Do not edit
  `api/v1/zz_generated.deepcopy.go` or chart CRDs directly; run `make generate` and review every generated change.
- Edit `charts/qubership-logging-operator/README.md.gotmpl` and chart metadata or values, then run `make docs`; Helm
  documentation generation owns `charts/qubership-logging-operator/README.md`.
- The top-level reconciler starts failed-reconcile backoff at one second, doubles it after each failure, and resets it
  after success. It tracks per-component status through `StatusUpdater`, auto-detects the container runtime, and falls
  back to `containerd`.
- Robot Framework integration tests under `test/robot-tests/` require a Kind cluster and external components. Their
  executable workflow is `.github/workflows/integration-tests.yaml`; there is no repository Make target for that
  environment.

## Done when

- Focused tests for the affected controller or package pass.
- `make fmt`, `make test`, and applicable static checks pass, or the final report names each check that could not run.
- API changes include reviewed output from `make generate` and `make docs`, including the operator-chart CRD,
  `docs/api.md`, `docs/crds/`, and the standalone CRD chart.
- Integration-sensitive changes are covered by the matching GitHub Actions workflow or reported as awaiting CI.
- The final response lists commands that ran and commands that were unavailable.

## Context routing

- Before changing logging topology, HA behavior, or external outputs, read `docs/architecture.md` for the maintained
  component and data-flow model.
- Before changing integration scenarios, read `.github/workflows/integration-tests.yaml` for the required Kind,
  registry, OpenSearch, VictoriaLogs, and Robot Framework setup.
- Before changing chart behavior, read `charts/qubership-logging-operator/README.md.gotmpl` and the relevant page under
  `docs/`; generated chart documentation is not the source of truth.
