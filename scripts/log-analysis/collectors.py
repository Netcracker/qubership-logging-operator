"""Report collection orchestration."""

from __future__ import annotations

import argparse
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
    "k8s_events": {
        "graylog": "kind:KubernetesEvent",
        "victorialogs": "kind:KubernetesEvent",
        "source_fields": ("namespace", "involvedObjectKind", "involvedObjectName"),
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
    report = {
        "graylog_streams": backend.stream_storage_report(dry_run=args.dry_run),
        "levels": backend.level_storage_report(dry_run=args.dry_run),
        "debug_trace": backend.debug_trace_report(dry_run=args.dry_run),
        "large_messages": backend.large_message_report(dry_run=args.dry_run),
        "schema_quality": backend.schema_quality_report(dry_run=args.dry_run),
    }
    audit_system = backend.audit_system_without_namespace_container_report(dry_run=args.dry_run)
    if args.dry_run or report_has_result_rows(audit_system):
        report["audit_system_without_namespace_container"] = audit_system
    return report


def report_has_result_rows(report: dict[str, Any]) -> bool:
    return any(
        isinstance(value, list) and bool(value)
        for key, value in report.items()
        if key not in ("columns", "queries")
    )


def collect_log_report(args: argparse.Namespace, time_filter: str, graylog_timerange: dict[str, Any]) -> dict[str, Any]:
    if args.backend_type == "graylog":
        client = HttpClient(
            args.backend_url,
            user=args.graylog_user,
            password=args.graylog_pass,
            insecure_skip_verify=args.insecure_skip_verify,
            extra_headers={"X-Requested-By": "log-storage-report"},
        )
        backend: GraylogClient | VictoriaLogsClient = GraylogClient(
            client,
            graylog_timerange,
            args.source_field,
            args.top_limit,
        )
        return collect_graylog_log_report(backend, args)
    else:
        client = HttpClient(
            args.backend_url,
            user=args.vl_user,
            password=args.vl_pass,
            insecure_skip_verify=args.insecure_skip_verify,
        )
        backend = VictoriaLogsClient(client, time_filter, args.source_field, args.top_limit)
    categories = {
        category: backend.execute_set(
            backend.category_queries(
                filters[args.backend_type],
                category_source_fields(filters, args),
            ),
            dry_run=args.dry_run,
        )
        for category, filters in CATEGORIES.items()
        if category != "k8s_events"
    }
    levels = backend.execute_set(backend.levels_overview_queries(), dry_run=args.dry_run)
    schema_quality = backend.execute_set(backend.schema_quality_queries(), dry_run=args.dry_run)
    return {
        "namespace_logs": backend.execute_set(backend.source_activity_queries(), dry_run=args.dry_run),
        "k8s_events": backend.execute_set(backend.k8s_events_queries(), dry_run=args.dry_run),
        "unattributed_logs": backend.execute_set(backend.unattributed_logs_queries(), dry_run=args.dry_run),
        "levels": levels,
        "debug_trace": backend.execute_set(backend.debug_trace_queries(), dry_run=args.dry_run),
        "log_patterns": backend.execute_set(backend.log_patterns_queries(), dry_run=args.dry_run),
        "message_size": backend.execute_set(backend.message_size_queries(), dry_run=args.dry_run),
        "schema_quality": schema_quality,
        "categories": categories,
        **(
            {
                "detailed_levels": {
                    level: backend.execute_set(backend.level_queries(level), dry_run=args.dry_run)
                    for level in LEVELS
                }
            }
            if args.include_detailed_levels
            else {}
        ),
    }
