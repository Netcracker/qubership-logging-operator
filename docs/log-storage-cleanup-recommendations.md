# Log Storage Cleanup Recommendations

This document lists common actions after running the log storage analysis report.

## Free Disk Space

- Reduce retention for noisy indices or streams if historical logs are no longer required.
- Delete old indices only after confirming the retention policy and support requirements.
- For Graylog/OpenSearch, check index rotation and retention settings before manual deletion.
- For VictoriaLogs, reduce retention or remove old data according to the deployment-specific maintenance procedure.
- Increase disk capacity only after checking whether the growth is caused by a few noisy sources.

## Reduce Log Volume

- Lower log level for services producing many `debug` or `trace` records.
- Fix loops that repeatedly emit the same warning or error.
- Reduce periodic health-check or reconciliation logs when they do not add operational value.
- Move verbose audit/debug logs to a dedicated backend or shorter-retention storage if they are still required.

## Reduce Log Size

- Avoid logging full payloads, documents, binary data, PDFs, certificates, tokens, or large JSON bodies.
- Truncate oversized fields before logs reach the storage backend.
- Replace large repeated payloads with IDs, hashes, object names, request IDs, or short summaries.
- Keep stack traces useful but avoid logging the same full stack trace at high frequency.

## Reject Or Drop Unwanted Logs

- Prefer fixing the application first when logs are useless or too verbose.
- Use Fluent Bit or Fluentd filters to drop known noisy records before they reach Graylog, OpenSearch, or VictoriaLogs.
- Drop rules should be narrow: match namespace, container, logger, level, and a stable message pattern where possible.
- Avoid broad drops such as dropping all warnings or all logs from a namespace unless explicitly approved.
- Track dropped-log rules as configuration changes and document why they were added.

## Improve Routing And Retention

- Route system, audit, container, and Kubernetes event logs into separate categories or streams.
- Apply shorter retention to high-volume low-value categories.
- Keep audit logs according to compliance requirements even if they are noisy.
- For Kubernetes events, consider separate retention because event bursts can be useful during incident analysis but may not need long storage.

## Validate After Changes

- Re-run the log storage report for the same time range after changes.
- Compare total logs, top namespaces, top sources, debug/trace sources, and storage usage.
- Check dashboards and alerts to make sure important operational signals were not removed.
- Keep a rollback path for any drop/reject rule.
