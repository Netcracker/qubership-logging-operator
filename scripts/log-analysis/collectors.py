"""Report collection orchestration."""

from __future__ import annotations

import argparse
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import Any

from clients import GraylogClient, HttpClient, VictoriaLogsClient, field_name


CATEGORIES = {
    "system": {
        "graylog": "log_category:system AND NOT kind:KubernetesEvent",
        "victorialogs": "log_category:system NOT kind:KubernetesEvent",
        "source_fields": ("{system_source_fields}",),
    },
    "audit": {
        "graylog": "(log_category:audit OR nc_audit_label:true) AND NOT kind:KubernetesEvent",
        "victorialogs": "(log_category:audit OR nc_audit_label:true) NOT kind:KubernetesEvent",
        "source_fields": ("{audit_source_fields}",),
    },
    "container": {
        "graylog": "log_category:container AND NOT kind:KubernetesEvent AND NOT nc_audit_label:true",
        "victorialogs": "log_category:container NOT kind:KubernetesEvent NOT nc_audit_label:true",
        "source_fields": ("namespace", "{source_field}"),
    },
}
# The pipeline normalizes ERROR to "err"; keep it in the report even though it
# is often described to users as "error".
LEVELS = ("emerg", "alert", "crit", "err", "warning", "notice", "info", "debug")


def category_source_fields(filters: dict[str, Any], args: argparse.Namespace) -> list[str]:
    fields: list[str] = []
    placeholders = {
        "{source_field}": [args.source_field],
        "{system_source_fields}": args.system_source_fields,
        "{audit_source_fields}": args.audit_source_fields,
    }
    for field in filters["source_fields"]:
        fields.extend(placeholders.get(field, [field]))
    for field in fields:
        field_name(field)
    return fields


def collect_graylog_log_report(backend: GraylogClient, args: argparse.Namespace) -> dict[str, Any]:
    tasks = {
        "graylog_streams": lambda: backend.stream_storage_report(dry_run=args.dry_run),
        "levels": lambda: backend.level_storage_report(dry_run=args.dry_run),
        "debug_trace": lambda: backend.debug_trace_report(dry_run=args.dry_run),
        "large_messages": lambda: backend.large_message_report(dry_run=args.dry_run),
        "schema_quality": lambda: backend.schema_quality_report(dry_run=args.dry_run),
        "audit_system_without_namespace_container": lambda: backend.audit_system_without_namespace_container_report(
            dry_run=args.dry_run
        ),
    }

    results: dict[str, Any] = {}
    if args.parallel_queries:
        max_workers = min(len(tasks), args.graylog_query_workers)
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = {executor.submit(task): name for name, task in tasks.items()}
            for future in as_completed(futures):
                name = futures[future]
                try:
                    results[name] = future.result()
                except Exception as exc:  # noqa: BLE001
                    results[name] = {"error": str(exc)}
    else:
        for name, task in tasks.items():
            try:
                results[name] = task()
            except Exception as exc:  # noqa: BLE001
                results[name] = {"error": str(exc)}

    report: dict[str, Any] = {}
    for name in tasks:
        if name == "audit_system_without_namespace_container":
            if args.dry_run or report_has_visible_content(results[name]):
                report[name] = results[name]
            continue
        report[name] = results[name]
    return report


def collect_victorialogs_log_report(backend: VictoriaLogsClient, args: argparse.Namespace) -> dict[str, Any]:
    category_fields = {
        category: category_source_fields(filters, args)
        for category, filters in CATEGORIES.items()
    }
    tasks = {
        "namespace_logs": lambda: backend.execute_set(backend.source_activity_queries(), dry_run=args.dry_run),
        "k8s_events": lambda: backend.execute_set(backend.k8s_events_queries(), dry_run=args.dry_run),
        "unattributed_logs": lambda: backend.execute_set(backend.unattributed_logs_queries(), dry_run=args.dry_run),
        "levels": lambda: backend.execute_set(backend.levels_overview_queries(), dry_run=args.dry_run),
        "debug_trace": lambda: backend.execute_set(backend.debug_trace_queries(), dry_run=args.dry_run),
        "log_patterns": lambda: backend.execute_set(backend.log_patterns_queries(), dry_run=args.dry_run),
        "message_size": lambda: backend.execute_set(backend.message_size_queries(), dry_run=args.dry_run),
        "schema_quality": lambda: backend.execute_set(backend.schema_quality_queries(), dry_run=args.dry_run),
        "categories": lambda: {
            category: backend.execute_set(
                backend.category_queries(filters["victorialogs"], category_fields[category]),
                dry_run=args.dry_run,
            )
            for category, filters in CATEGORIES.items()
        },
    }
    if args.include_detailed_levels:
        tasks["detailed_levels"] = lambda: {
            level: backend.execute_set(backend.level_queries(level), dry_run=args.dry_run)
            for level in LEVELS
        }
    if not args.parallel_queries:
        return {name: task() for name, task in tasks.items()}
    results: dict[str, Any] = {}
    max_workers = min(len(tasks), backend.max_parallel_queries)
    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {executor.submit(task): name for name, task in tasks.items()}
        for future in as_completed(futures):
            name = futures[future]
            try:
                results[name] = future.result()
            except Exception as exc:  # noqa: BLE001
                results[name] = {"error": str(exc)}
    return {name: results[name] for name in tasks}


def report_has_visible_content(report: dict[str, Any]) -> bool:
    return any(
        (isinstance(value, list) and bool(value))
        or (isinstance(value, dict) and "error" in value)
        or key == "error"
        for key, value in report.items()
        if key not in ("columns", "queries")
    )


def collect_log_report(
    args: argparse.Namespace,
    time_filter: str,
    graylog_timerange: dict[str, Any],
    graylog_client: HttpClient | None = None,
    victorialogs_client: VictoriaLogsClient | None = None,
) -> dict[str, Any]:
    if args.backend_type == "graylog":
        backend: GraylogClient | VictoriaLogsClient = GraylogClient(
            require_client(graylog_client),
            graylog_timerange,
            args.source_field,
            args.top_limit,
            parallel_queries=args.parallel_queries,
            max_parallel_queries=args.graylog_query_workers,
        )
        return collect_graylog_log_report(backend, args)
    else:
        return collect_victorialogs_log_report(require_victorialogs_client(victorialogs_client), args)


def require_client(client: HttpClient | None) -> HttpClient:
    if client is None:
        raise ValueError("Graylog HTTP client is required")
    return client


def require_victorialogs_client(client: VictoriaLogsClient | None) -> VictoriaLogsClient:
    if client is None:
        raise ValueError("VictoriaLogs client is required")
    return client
