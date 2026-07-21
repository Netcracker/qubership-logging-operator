# Migrate call sites — qubership-dbaas (stage 2)

**Repository:** `qubership-dbaas` (monorepo — worktree assigned by operator)

**Prerequisite:** JSON logging envelope is **already enabled** for both components (Quarkus JSON / Go
`LOG_FORMAT`, Helm). Do not redo stage-1 config unless broken.

**Runtimes:** Java/Quarkus (`dbaas-aggregator`), Go/Kubebuilder (`dbaas-operator`)

## Ask

Migrate **call sites** in both runtime components so variable data is in structured fields:

1. Repo-root discovery — list **both** `dbaas-aggregator` and `dbaas-operator` in the coverage ledger.
2. Java: migrate `{}` parameterized SLF4J logging; semantic field names; preserve throwables.
3. Go: migrate production `log.*f` to structured fields; zero active formatted calls in production packages.
4. Inventory logged preformatted-message sites (`log.error(msg)`, etc.) and ask or mark blocked — do not silently skip.
5. Run applicable builds/tests; document blocked credentials explicitly.
6. Report completion gates with before/after counts.

Do not commit (eval / draft). Do not mark complete after only one component or a sample edit.
