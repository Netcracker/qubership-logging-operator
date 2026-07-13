# Smoke validation (stage 1)

Run after **config** changes. Unit tests alone are insufficient.

## How to smoke

1. Find the repo’s documented way to run or deploy the changed component (README, `dev/`, `bootstrap/`, Makefile,
   workflows).
2. Run that path with `LOG_FORMAT=json` (or chart default `json`) when dual rollout applies.
3. Capture **one** stdout or pod log line — app startup, health check, or (for Nginx/Envoy) one line after an HTTP
   request through the proxy.
4. Confirm it parses as JSON:
   - **App logs:** `time`, `level`, `message` (or stack-mapped equivalents). Legacy text inside `message` is OK for
     stage 1.
   - **Access logs:** HTTP access fields per [schema.md](schema.md) and [enable-nginx-envoy.md](enable-nginx-envoy.md)
     (e.g. `time`, `status`, `method`, `path`).

If local run is not possible, use CI on the PR (workflow that builds and exercises the component) and cite the job plus
one log snippet.

## Record in report

| Field         | Example                                              |
| ------------- | ---------------------------------------------------- |
| Doc reference | `bootstrap/README.md` — `make install`               |
| Command       | exact command from repo docs                         |
| Result        | PASS — JSON envelope OK                              |
| Note          | Message may contain legacy bracket text (stage 1 OK) |
