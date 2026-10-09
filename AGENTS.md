# Repository agent instructions

## Commands

- Run `go test ./api/...` as well as `make unit-test` when you change `api/`. `api/` is a separate Go module in
  `go.work`, and `make unit-test` covers only the root module.
- Check changes to `charts/qubership-logging-operator/` with `helm lint charts/qubership-logging-operator -f <file>`
  for the `*-values.yaml` files in `docs/examples/*/`. CI lints the chart with each of those files as values, so a
  values or schema change can break an example.
- Run `scripts/chart-tests/test-victorialogs-rendering.sh` after changing `charts/qubership-victorialogs/`.

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
