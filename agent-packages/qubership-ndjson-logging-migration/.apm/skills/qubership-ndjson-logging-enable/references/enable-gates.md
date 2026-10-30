# Stage 1 completion gates

Stage 1 is complete when the **JSON envelope** works — not when call sites are migrated.

## Gate order

1. **Build** — component compiles (or explicit `blocked` with error).
2. **Config** — `LOG_FORMAT` / encoder / Helm aligned per component.
3. **Smoke** — one realistic path emits valid NDJSON (app envelope or access-log JSON per [schema.md](schema.md)).

## Build and smoke

**Follow the target repo’s documentation** — do not invent build or smoke commands.

Discover from README, `dev/`, `bootstrap/`, `Makefile`, chart notes, and `.github/workflows/`:

- **Build** — run the documented local build or the same Maven/Go target CI uses (e.g. workflow `maven-command`,
  `make test-unit`, image build in bootstrap).
- **Smoke** — use the shallowest documented path that runs the changed component and emits stdout or pod logs (local
  deploy scripts, `dev/` workflows, documented startup, or a CI job that exercises the service).
- Record the **exact commands** (and doc links) in the migration report.
- When docs assume prerequisites (Maven `settings.xml`, kind cluster, VPN), note them; ask the user or cite **CI
  evidence** on the PR when the agent environment lacks them.
- Mark **blocked** only when neither the documented local path nor CI can validate — include the error and missing
  prerequisite.

**Maven 401 (GitHub Packages):** Usually missing or mismatched `~/.m2/settings.xml` `<server>` id vs POM — see
[maven-github-packages.md](maven-github-packages.md). Ask the user to fix auth before treating Java build as permanently
blocked.

Private Maven packages (401): after auth setup, re-run compile. Stage 1 can still be **partial** on components that
build while others remain blocked. A documented local deploy path may still need Maven auth to build images — same
prerequisite, not a reason to skip repo docs.

## Config checklist

- [ ] JSON formatter enabled (`quarkus-logging-json`, logrus JSON, zap JSON handler, etc.)
- [ ] `LOG_FORMAT=text|json` env + Helm default where dual-rollout per logging guide applies
- [ ] Field name mapping if encoder uses `msg`/`ts` → map to `message`/`time` in config when possible
- [ ] FluentBit/chart parser matches output (JSON for JSON access logs; nginx text parser only for text formats)

## Smoke

See [smoke-validation.md](smoke-validation.md). **Required** for stage 1 sign-off.

## Out of scope for stage 1

- GELF / non-stdout app appenders.
- Nginx/Envoy access logs when config is **not** owned by the target repo (platform ingress only).
