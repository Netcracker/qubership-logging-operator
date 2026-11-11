# Fluent Bit YAML configuration migration

Issue: [#674](https://github.com/Netcracker/qubership-logging-operator/issues/674). Date: November 10, 2026.

## Goal

Build the Fluent Bit configuration of all three agents (standard, HA forwarder, HA aggregator) in Go instead of Go
templates, write it as YAML, and give users structured replacements for the classic custom fields, with automatic
conversion of the classic ones.

## Decisions

- One configuration model with two writers, classic and YAML. Phases 1 and 2 ship in one release; the classic writer
  is an internal checkpoint for the equivalence tests.
- One shared processing chain for the standard agent and the aggregator. Intended differences are explicit options;
  accidental differences are resolved during the port and recorded in the expected results.
- `fluentbit.configFormat: yaml|classic`, default `yaml`. The classic writer and this field stay for one major release.
- Users get the converted Helm values from a ConfigMap. No converter binary or subcommand.
- Packages, kept for now and open to change:
  - `internal/fluentbitconf/`: model, `MarshalClassic`, `MarshalYAML`, `ParseClassic`. No Kubernetes dependency.
  - `internal/builder/fluentbitconfig/`: pure builders `Standard`, `Forwarder`, `Aggregator`.
  - `internal/defaults/`: Fluent Bit fallback values, matching `charts/qubership-logging-operator/values.yaml`.
- Out of scope: FluentD and Graylog templates, and splitting `LoggingService` into several CRs. The plugin type from
  phase 3 must be reusable as the spec of future pipeline CRs.

## Phase 0: spike

Answer the open Fluent Bit questions with `--dry-run` against `fluent/fluent-bit:5.1.0` before writing the YAML
writer. Record the answers in this plan.

- [ ] Can a YAML main file include a classic file, or point `service.parsers_file` at a classic parsers file?
- [ ] Does `rewrite_tag` accept a list under `rule:`?
- [ ] Do `modify`, `record_modifier`, and `nest` accept lists for their repeated keys in 5.1.0?
- [ ] In which order do `includes` merge into `pipeline.filters`?
- [ ] Is `hot_reload` on by default in 5.1.0, and does `configmap-reload` trigger a reload with a YAML file?
- [ ] Does an official classic-to-YAML converter exist (search the fluent/fluent-bit issues)?

## Phase 1: Go builders (release N, internal checkpoint)

### 1.1 Model and classic I/O

- [ ] Add `internal/fluentbitconf/` with `Config`, `Pipeline`, `Plugin`, `Properties`, `Parser`, `MultilineParser`,
  `EnvVar`. `Properties` keeps insertion order and allows repeated keys.
- [ ] Add `MarshalClassic`: sections, aligned key/value lines, repeated keys as repeated lines, multiline rules as
  `rule "<state>" "<regex>" "<next>"`.
- [ ] Add `ParseClassic`: sections, `key value` lines, repeated keys, comments, blank lines, `@SET`, multiline rules.
  Return an error for `@INCLUDE` unless the caller resolves it.
- [ ] Add a `Normalize` helper for comparisons: lowercase keys, trimmed values.
- [ ] Unit tests: round trip of every file in `internal/controllers/fluentbit*/**/*.conf` rendered for a sample CR.

### 1.2 Defaults

- [ ] Move the template fallbacks to `internal/defaults/`: `Flush` 5, `Log_Level` warn, `storage.backlog.mem_limit`
  5M, `Mem_Buf_Limit` 10M, `storage.total_limit_size` 1024Mb, DB journal mode and sync. List them by grepping
  `default` in the templates.
- [ ] Test that each default matches `values.yaml`.

### 1.3 Builders

- [ ] Add `internal/builder/fluentbitconfig/` with `Standard(cr, creds, d)`, `Forwarder(cr, d)`,
  `Aggregator(cr, creds, d)`, each returning `*fluentbitconf.Config` and the side files (Lua scripts,
  `loki-labels.json`).
- [ ] `service(cr, d)`: the `[SERVICE]` block, including the filesystem buffer settings.
- [ ] Inputs: `containerInputs` (containerd/CRI-O and Docker), `systemInputs` (three log types), `auditInputs`
  (audit, kube audit, kube-apiserver audit, OpenShift audit), `forwardInput` for the aggregator.
- [ ] Filters: `concatFilters`, `kubernetesFilters` (with the mock as a builder option), `rewriteTagFilters`,
  `processingChain(opts)` covering validate, generic parsers, post-generic, audit, unparsed counter, add-stream,
  non-supported levels, define-format, and HTTP routing.
- [ ] Custom slots: insert `customInputConf`, `customFilterConf`, `customOutputConf` at today's `@INCLUDE` positions.
  In phase 1 they are parsed with `ParseClassic`; a parse error fails the reconcile as the broken include does today.
- [ ] Outputs: Graylog, Loki, HTTP, OpenTelemetry, Prometheus log-to-metrics, forward with upstream for the forwarder.
  `extraParams` is parsed as classic key/value lines.
- [ ] Parsers: move the static definitions to an embedded YAML data file decoded into the model; set the two
  multiline regexes from the CR in Go.
- [ ] Lua scripts stay embedded files, copied unchanged.

### 1.4 Equivalence

- [ ] Test for each CR under `internal/builder/testdata/` and `test/fluent-pipeline/testdata/assets/render/`, and each
  mode: render the old templates, parse with `ParseClassic` (resolving `@INCLUDE` against the rendered files), and
  compare with the builder model after `Normalize`.
- [ ] For each difference between the standard and aggregator templates, decide: an option on `processingChain`, or a
  fix. Note each fix in this plan and in the expected results under `test/fluent-pipeline/testdata/`.

### 1.5 Switch the controllers

- [ ] `internal/controllers/fluentbit/manifest.go` and `fluentbit-forwarder-aggregator/manifest.go` call the
  builders and `MarshalClassic`. Secret and ConfigMap names stay.
- [ ] Run `make unit-test` and `make test-fluent-pipeline` for `fluentbit`, `fluentbit-ha`, `kube-metadata`, and
  `render`.
- [ ] Delete the Fluent Bit `.conf` templates and the equivalence tests. Keep the Lua scripts and the parser data file.
- [ ] Add golden files of the builder output per testdata CR (`internal/builder/buildertest/golden.go`).

## Phase 2: YAML output (same release as phase 1)

- [ ] Add `MarshalYAML` using `sigs.k8s.io/yaml` or `gopkg.in/yaml.v3`: flat dotted keys, lists for repeated keys,
  `rules` maps for multiline parsers, quoting by the library.
- [ ] Write `fluent-bit.yaml` as one file with `service`, `parsers`, `multiline_parsers`, and `pipeline`. User
  plugins go inline, not through `includes`.
- [ ] Add `Args: ["-c", "/fluent-bit/etc/fluent-bit.yaml"]` (or the classic path) to the Fluent Bit containers in
  `internal/builder/fluentbit/` and `internal/builder/forwarderaggregator/`. Update their golden files.
- [ ] Add `fluentbit.configFormat` (`yaml` default, `classic`) to `api/v1`, run `make generate`, `make update-crds`,
  `make -B docs/crds`, and `make docs/api.md`. Wire it in the Helm `loggingservice` template, `values.yaml`,
  `values.schema.json`, and `docs/installation-parameters.md`.
- [ ] Image check: with `configFormat: yaml` and an image tag older than 3.2, use classic and report it in the status.
  Treat tags that do not parse as new enough.
- [ ] Update `test/fluent-pipeline/run.sh` to run the YAML file, including `--dry-run`. Expected results stay
  unchanged. Keep one scenario on `classic` while the writer exists.
- [ ] Regenerate the golden files in YAML.
- [ ] Update `docs/fluentbit-log-pipeline.md` and `docs/troubleshooting.md` (edit the symlink target) where they
  show the rendered classic configuration.
- [ ] Run `helm lint` with each `docs/examples/*/*-values.yaml` and the scripts in `scripts/chart-tests/`.

## Phase 3: structured user fields and conversion

- [ ] Add a plugin type to `api/v1` (a map with a required `name`, unknown fields preserved) and the fields
  `customInputs`, `customFilters`, `customOutputs`, `customParsers`, `customMultilineParsers` under `fluentbit` and
  `fluentbit.aggregator`, and `extraProperties` on the Loki, HTTP, and OpenTelemetry outputs. Regenerate as in
  phase 2.
- [ ] Wire the fields in the Helm `loggingservice` template with `toYaml`; add them to `values.yaml`,
  `values.schema.json`, and `docs/installation-parameters.md`.
- [ ] Builders: use the new field when set, otherwise convert the classic field with `ParseClassic`. When both are
  set, use the new field and warn.
- [ ] Reporting: a `FluentbitClassicConfig` status condition and a Warning event per classic field in use; a conversion
  error names the field and line and leaves the running configuration unchanged.
- [ ] Write the converted fields, in Helm values form, to a ConfigMap (name to agree, for example
  `logging-fluentbit-converted-config`). Delete it when no classic field is set.
- [ ] Helm `NOTES.txt` warning when a classic field is set.
- [ ] Conversion tests: every classic example in the docs converts, and the result passes `--dry-run`.
- [ ] Rewrite the classic examples in `docs/user-guides/agents-pipeline-customization.md`,
  `docs/log-processing/configure-parsing.md`, `docs/integrations/{splunk,kafka,syslog}.md`, and
  `docs/installation-parameters.md`. Add a migration section to `docs/migration.md`.

## Phase 4: cleanup (next major release)

- [ ] Remove `MarshalClassic`, `configFormat`, and the image-tag fallback.
- [ ] Remove the classic fields and `extraParams` together with `ParseClassic`, or keep the conversion one more
  release if users still rely on it.

## Follow-up: format routing

Issues [#397](https://github.com/Netcracker/qubership-logging-operator/issues/397) and
[#407](https://github.com/Netcracker/qubership-logging-operator/issues/407). Do this after phase 1.5, in
`processingChain`, so the equivalence tests compare like with like. The expected results under
`test/fluent-pipeline/testdata/` stay unchanged.

- [ ] Measure the agent CPU per 1000 records on the `fluentbit` fixtures before and after, so the change is judged by
  numbers. The klog, Qubership, and level parsers, the logfmt check, and the audit expressions run on every container
  record; most are anchored and fail on the first characters.
- [ ] Make the fast path explicit (#407). A record parsed by the Kubernetes filter or the annotation parser skips the
  generic chain because `filter-validate.conf` renames `log` to `original_log`. Use a condition on `parse_status`
  instead, and keep the raw line until the cleanup step.
- [ ] Run only the parsers that can apply to a record (#397). Gate each generic parser through its own input key, as
  the logfmt parser does with `_logfmt_input`, rather than by `rewrite_tag`: re-emitted records run the `Match *`
  filters again.

## Risks

- Behavior drift during the port: covered by the equivalence tests and unchanged pipeline results.
- A conversion that loads but behaves differently: users can review the ConfigMap; `--dry-run` catches syntax and
  property names only.
- A bad user plugin that stops Fluent Bit on hot reload: the 5.x docs list errors that abort the reload after the old
  pipeline stopped. Consider a `--dry-run` init container later; not part of this plan.
