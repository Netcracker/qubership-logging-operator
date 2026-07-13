# Stage 1 — Nginx / Envoy access logs

Config and Helm only when the **target repo owns** nginx or Envoy access-log configuration (ConfigMap, sidecar
bootstrap, chart templates). **No** call-site migration — stage 2 does not apply to access logs.

Read [schema.md](schema.md) — access logs use the **access-log JSON contract**, not the app `time` / `level` /
`message` envelope.

## Discovery

Locate repo-owned config before editing:

- Nginx: `nginx.conf`, `*.conf` in charts, ConfigMap templates, `log_format` / `access_log` directives.
- Envoy: bootstrap YAML/JSON, Helm values, `access_log` blocks on `HttpConnectionManager`.
- Ingestion: FluentBit / logging-operator annotations on the pod (JSON vs nginx text parser).
- Dual rollout: repo may use a value/env flag for text vs JSON access format — mirror existing app `LOG_FORMAT` patterns
  when present.

Extend the repo’s existing format names and paths; do not copy another service’s field list blindly.

## Nginx

### Baseline

Many charts still use a **text** `log_format` (combined/custom) parsed by a dedicated FluentBit nginx parser.

### Enable JSON (config only)

1. Define a JSON `log_format` with `escape=json` (or the repo’s documented JSON access-log module).
2. Point `access_log` at stdout (`/dev/stdout`) or the path the chart already uses for container logs.
3. Keep a text format only when dual rollout requires it — wire the switch via existing Helm values or documented flags.
4. Align field names with the logging guide (*log-formats.md*) and what downstream queries expect (`time`, `status`,
   `method`, `path`, `request_time`, etc.).

Example shape (adapt field names to the target repo):

```nginx
log_format json_access escape=json '{'
  '"time":"$time_iso8601",'
  '"remote_addr":"$remote_addr",'
  '"method":"$request_method",'
  '"path":"$uri",'
  '"status":$status,'
  '"body_bytes_sent":$body_bytes_sent,'
  '"request_time":$request_time'
'}';

access_log /dev/stdout json_access;
```

### Helm

- Update ConfigMap / chart values that render `log_format` and `access_log`.
- If JSON access logs are emitted, ensure pod annotations use the **JSON** parser path — not the legacy nginx text
  parser.

## Envoy

### Baseline

Envoy often logs access lines as **text** or legacy formats unless `access_log` is explicitly JSON.

### Enable JSON (config only)

1. On each `HttpConnectionManager` (or repo-documented listener), set `access_log` to a **stdout** (or existing file)
   sink.
2. Use `typed_json` when the repo’s Envoy version and platform standard require it; otherwise `json_format` with
   documented field tokens.
3. Map tokens to stable JSON keys per logging guide / existing observability queries.

Example shape (adapt to target Envoy version and repo conventions):

```yaml
access_log:
  - name: envoy.access_loggers.stdout
    typed_config:
      "@type": type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog
      log_format:
        json_format:
          time: "%START_TIME%"
          method: "%REQ(:METHOD)%"
          path: "%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%"
          status: "%RESPONSE_CODE%"
          duration: "%DURATION%"
          upstream_cluster: "%UPSTREAM_CLUSTER%"
```

### Helm

- Patch bootstrap ConfigMap / values the same way other Envoy filters are managed in the chart.
- Confirm access-log stdout is collected by the same FluentBit tail as app logs.

## Pitfalls

- **Parser mismatch:** JSON access log + nginx text parser annotation → fix chart annotation in stage 1.
- **Platform-owned ingress:** Config lives outside the repo — record `blocked` unless the user scopes chart changes here.
- **Mixed pods:** Sidecar access logs and app logs share stdout — smoke must identify an access-log line (HTTP fields),
  not an app `level` line.
