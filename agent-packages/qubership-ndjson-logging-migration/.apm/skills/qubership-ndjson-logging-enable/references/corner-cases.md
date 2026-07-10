# Corner cases (stage 1)

- **Dual format:** `LOG_FORMAT=text` must still produce valid legacy output; `json` must produce one JSON object per line.
- **Parser mismatch:** JSON stdout + text parser annotation breaks ingestion — fix chart/annotation in stage 1.
- **Pre-logger stdout:** `fmt.Printf` banners may stay plain text — record as `blocked` with reason.
- **GELF / file appenders:** Out of scope for stdout NDJSON stage 1.
- **Nginx / Envoy access logs:** Config-only JSON via [enable-nginx-envoy.md](enable-nginx-envoy.md); smoke one access-log line after an HTTP request — not app `log.*f` work.
- **Infra-only is valid for stage 1** — logger + Helm + smoke without touching call sites.
- **Do not claim stage 2 complete** after stage 1 — hand off app call-site work to `qubership-ndjson-logging-migrate`.
