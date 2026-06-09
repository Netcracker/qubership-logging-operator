"""Self-contained HTML rendering for log storage reports."""

from __future__ import annotations

import json
import re
from pathlib import Path
from typing import Any
from xml.sax.saxutils import escape


MARKDOWN_LINK_PATTERN = re.compile(r"\[([^\]]+)]\((https://[^)]+)\)")
INLINE_CODE_PATTERN = re.compile(r"`([^`]+)`")
URL_PATTERN = re.compile(r"https://[^\s]+")


def html_escape(value: Any) -> str:
    return escape(str(value), {'"': "&quot;", "'": "&#x27;"})


def render_inline_markup(value: str) -> str:
    result = []
    position = 0
    for match in INLINE_CODE_PATTERN.finditer(value):
        result.append(render_links_and_markdown_links(value[position : match.start()]))
        result.append(f"<code>{html_escape(match.group(1))}</code>")
        position = match.end()
    result.append(render_links_and_markdown_links(value[position:]))
    return "".join(result)


def render_links_and_markdown_links(value: str) -> str:
    result = []
    position = 0
    for match in MARKDOWN_LINK_PATTERN.finditer(value):
        result.append(render_links(value[position : match.start()]))
        label = html_escape(match.group(1))
        url = html_escape(match.group(2))
        result.append(f'<a href="{url}">{label}</a>')
        position = match.end()
    result.append(render_links(value[position:]))
    return "".join(result)


def render_links(value: str) -> str:
    result = []
    position = 0
    for match in URL_PATTERN.finditer(value):
        url = match.group(0).rstrip(".,)")
        trailing = match.group(0)[len(url) :]
        result.append(html_escape(value[position : match.start()]))
        escaped_url = html_escape(url)
        result.append(f'<a href="{escaped_url}">{escaped_url}</a>')
        result.append(html_escape(trailing))
        position = match.end()
    result.append(html_escape(value[position:]))
    return "".join(result)


def format_number(value: int | float) -> str:
    if isinstance(value, bool):
        return str(value)
    if isinstance(value, int):
        return f"{value:,}".replace(",", " ")
    if value.is_integer():
        return f"{int(value):,}".replace(",", " ")
    if abs(value) < 0.01:
        return f"{value:,.6f}".rstrip("0").rstrip(".").replace(",", " ")
    return f"{value:,.2f}".replace(",", " ")


def format_adaptive_size_from_kb(value: Any) -> str:
    return format_adaptive_size(value, "KB")


def format_adaptive_size_from_gb(value: Any) -> str:
    return format_adaptive_size(value, "GB")


def format_adaptive_size(value: Any, base_unit: str) -> str:
    if isinstance(value, bool):
        return html_escape(value)
    try:
        number = float(value)
    except (TypeError, ValueError):
        return format_cell(value)
    units = ("KB", "MB", "GB", "TB", "PB")
    unit_index = units.index(base_unit)
    while abs(number) >= 1024 and unit_index < len(units) - 1:
        number /= 1024
        unit_index += 1
    while 0 < abs(number) < 1 and unit_index > 0:
        number *= 1024
        unit_index -= 1
    return f"{format_number(number)} {units[unit_index]}"


def is_victorialogs_size_column(path: tuple[str, ...], header: str) -> bool:
    if not header.endswith(("_kb", "_gb")):
        return False
    if path[:2] in {
        ("storage", "victorialogs_disk_usage"),
        ("storage", "victorialogs_data_size"),
    }:
        return True
    if path[:2] == ("storage", "victorialogs_block_stats"):
        return True
    if path and path[0] == "logs" and header in {
        "sum_message_size_kb",
        "max_message_size_kb",
    }:
        return True
    return False


def strip_size_suffix(header: str) -> str:
    if header.endswith("_kb"):
        return header.removesuffix("_kb")
    if header.endswith("_gb"):
        return header.removesuffix("_gb")
    return header


def display_header(path: tuple[str, ...], header: str) -> str:
    if path in {("index_stats", "global"), ("index_stats", "index_sets")} and header == "size_kb":
        return "size"
    if is_victorialogs_size_column(path, header):
        return strip_size_suffix(header)
    if header == "sum_gl2_accounted_message_size_kb":
        return "SUM_MESSAGE_SIZE"
    if header in {"max_gl2_accounted_message_size_kb", "max_message_size_kb"}:
        return "MAX_MESSAGE_SIZE"
    if header in {"debug_trace_logs_count", "error_logs_count", "total_logs_count"}:
        return header.upper()
    return header


def header_tooltip(header: str) -> str:
    if header == "sum_message_size_kb":
        return "Calculated as sum_len(_msg). This is the _msg field size, not the full parsed log record size."
    if header == "max_message_size_kb":
        return "Calculated from len(_msg). This is the _msg field size, not the full parsed log record size."
    if header == "sum_gl2_accounted_message_size_kb":
        return "Calculated from Graylog gl2_accounted_message_size."
    if header == "max_gl2_accounted_message_size_kb":
        return "Calculated from max(gl2_accounted_message_size)."
    return ""


def render_header_cell(path: tuple[str, ...], header: str) -> str:
    title = header_tooltip(header)
    title_attr = f' title="{html_escape(title)}"' if title else ""
    return f'<th class="{column_class(header)}"{title_attr}>{html_escape(display_header(path, header))}</th>'


def column_class(header: str) -> str:
    safe_header = re.sub(r"[^a-z0-9]+", "-", header.lower()).strip("-")
    return f"col-{safe_header}" if safe_header else "col-value"


def table_class(path: tuple[str, ...]) -> str:
    classes = ["data-table"]
    if path[:2] == ("logs", "log_patterns"):
        classes.append("log-patterns-table")
    return " ".join(classes)


def format_table_cell(value: Any, header: str | None = None, path: tuple[str, ...] = ()) -> str:
    if (
        path in {("index_stats", "global"), ("index_stats", "index_sets")}
        and header == "size_kb"
    ) or header in {"sum_gl2_accounted_message_size_kb", "max_gl2_accounted_message_size_kb", "max_message_size_kb"}:
        return format_adaptive_size_from_kb(value)
    if header and is_victorialogs_size_column(path, header):
        if header.endswith("_gb"):
            return format_adaptive_size_from_gb(value)
        return format_adaptive_size_from_kb(value)
    return format_cell(value)


def format_cell(value: Any) -> str:
    if isinstance(value, int) and not isinstance(value, bool):
        return format_number(value)
    if isinstance(value, float):
        return format_number(value)
    if isinstance(value, (dict, list)):
        return "<code>" + html_escape(json.dumps(value, ensure_ascii=True)) + "</code>"
    return html_escape(value)


def section_status(data: Any) -> str:
    if isinstance(data, dict):
        if "error" in data:
            return "error"
        if data.get("status") == "skipped":
            return "skipped"
    return ""


def display_title(path: tuple[str, ...]) -> str:
    name = path[-1] if path else ""
    parent = path[-2] if len(path) > 1 else ""
    grandparent = path[-3] if len(path) > 2 else ""
    exact = {
        ("index_stats",): "Graylog Index Stats",
        ("index_stats", "global"): "Global Index Stats",
        ("index_stats", "index_sets"): "Index Sets Stats",
        ("storage", "victorialogs_data_size"): "VictoriaLogs Data Size",
        ("logs", "graylog_streams"): "Graylog Streams",
        ("logs", "audit_system_without_namespace_container"): "Audit/System Logs Without Namespace / Container",
        ("logs", "graylog_streams", "total_by_stream"): "Stream Totals",
        ("logs", "graylog_streams", "by_stream_and_namespace"): "Stream And Namespace Totals",
        ("logs", "graylog_streams", "top_default_stream_sources"): "Top Default Stream Sources",
        ("logs", "graylog_streams", "audit_system_by_stream_and_nodename"): "Audit/System Logs By Node",
        ("logs", "graylog_streams", "audit_system_by_stream_and_source"): "Audit/System Workload Logs",
        ("logs", "graylog_streams", "k8s_events_by_object"): "Kubernetes Events",
        ("logs", "levels", "total_by_level"): "Level Totals",
        ("logs", "levels", "top_namespaces_by_level"): "Top Namespaces By Level",
        ("logs", "levels", "top_sources_by_level"): "Top Sources By Level",
        ("logs", "levels", "top_nodes_without_namespace_source_by_level"): "Top Non-Container Logs By Level",
        ("logs", "audit_system_without_namespace_container", "total_by_stream"): "Audit/System Stream Totals",
        ("logs", "audit_system_without_namespace_container", "by_stream_and_nodename"): "Audit/System Nodes",
        ("logs", "audit_system_without_namespace_container", "by_stream_and_level"): "Audit/System Levels",
        ("logs", "debug_trace"): "Debug / Trace Logs",
        ("logs", "namespace_logs"): "Namespace Logs",
        ("logs", "k8s_events"): "Kubernetes Events",
        ("logs", "unattributed_logs"): "System/Audit Non-Container Logs",
        ("logs", "categories"): "Log Categories",
        ("logs", "namespace_logs", "top_namespaces_by_count"): "Top Namespaces By Log Count",
        ("logs", "namespace_logs", "top_by_count"): "Top Sources By Log Count",
        ("logs", "unattributed_logs", "top_nodes_by_count"): "Top Non-Container Log Sources",
        ("logs", "levels", "top_by_level_and_source"): "Top Log Sources By Level",
        ("logs", "levels", "top_non_container_by_level_and_node"): "Top Non-Container Sources By Level",
        ("logs", "debug_trace", "top_by_count"): "Top Debug/Trace Sources",
        ("logs", "log_patterns"): "Log Patterns",
        ("logs", "log_patterns", "top_patterns_by_count"): "Top Log Patterns By Count",
        ("logs", "k8s_events", "top_by_count"): "Top Kubernetes Event Sources",
        ("logs", "message_size", "top_by_max_message_size"): "Top Sources By Max Message Size",
        ("logs", "schema_quality", "top_by_max_fields"): "Top Sources By Max Parse Field Count",
        ("logs", "large_messages"): "Large Records",
        ("logs", "large_messages", "top_by_max_message_size"): "Top Sources By Max Record Size",
        ("storage", "victorialogs_block_stats"): "VictoriaLogs Block Stats",
        ("storage", "victorialogs_block_stats", "top_streams_by_disk_usage"): "Top Streams By Disk Usage",
        ("storage", "victorialogs_block_stats", "top_fields_by_disk_usage"): "Top Fields By Disk Usage",
    }
    if path in exact:
        return exact[path]
    if grandparent == "categories" and name == "top_by_count":
        return "Top Sources By Count"
    if parent in {"emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"} and name == "top_by_count":
        return f"Top {parent.title()} Sources"
    if grandparent == "graylog_streams" and name == "total":
        return "Total Logs In Stream"
    if grandparent == "graylog_streams" and name == "top_nodes_by_count":
        return "Top Nodes In Stream"
    if grandparent == "graylog_streams" and name == "top_namespaces_by_count":
        return "Top Namespaces In Stream"
    if grandparent == "graylog_streams" and name == "top_sources_by_count":
        return "Top Sources In Stream"
    if grandparent == "graylog_streams" and name == "top_levels_by_count":
        return "Top Levels In Stream"
    if name == "top_by_count":
        return "Top Sources By Count"
    return name.replace("_", " ").title()


def table_description(path: tuple[str, ...]) -> str:
    name = path[-1] if path else ""
    parent = path[-2] if len(path) > 1 else ""
    grandparent = path[-3] if len(path) > 2 else ""

    exact = {
        ("storage", "filesystem_usage"): (
            "Filesystem usage from node_filesystem* metrics for the selected devices or mountpoints. "
            "Use --filesystem-selector to target Graylog, OpenSearch, or VictoriaLogs storage paths. "
            "Labels are reduced to fstype, device, and mountpoint; sizes are shown in GB."
        ),
        ("storage", "victorialogs_disk_usage"): (
            "VictoriaLogs own disk capacity metrics."
        ),
        ("storage", "victorialogs_data_size"): (
            "VictoriaLogs data size from `vl_uncompressed_data_size_bytes` and `vl_compressed_data_size_bytes`. "
            "The `total` row sums all data types. `storage/inmemory` is recently ingested data still held in "
            "memory; `storage/small` is small on-disk parts created after flushing inmemory data; `storage/big` "
            "is merged larger on-disk parts used for long-term storage. These values describe VictoriaLogs "
            "internal storage parts and are not expected to match `sum_message_size_kb` from LogsQL queries."
        ),
        ("index_stats",): (
            "Graylog index statistics retrieved from Graylog API. Size values use adaptive units."
        ),
        ("index_stats", "global"): (
            "Global Graylog managed indices statistics from /api/system/indices/index_sets/stats."
        ),
        ("index_stats", "index_sets"): (
            "Index set level statistics from /api/system/indices/index_sets?stats=true, when Graylog exposes them."
        ),
        ("logs", "graylog_streams"): (
            "Graylog stream-based storage impact view. Tables show message count and sum(gl2_accounted_message_size) in KB."
        ),
        ("logs", "graylog_streams", "total_by_stream"): (
            "Total number of records and summed gl2_accounted_message_size in KB for each Graylog stream."
        ),
        ("logs", "graylog_streams", "by_stream_and_namespace"): (
            "Graylog stream totals split by namespace. This shows which namespaces contribute most inside each stream."
        ),
        ("logs", "graylog_streams", "top_default_stream_sources"): (
            "Top namespace/container sources in Default Stream, ranked by summed gl2_accounted_message_size in KB."
        ),
        ("logs", "graylog_streams", "audit_system_by_stream_and_nodename"): (
            "Audit and system stream records grouped by node name. "
            "Use this table to find nodes that produce the largest audit/system log volume."
        ),
        ("logs", "graylog_streams", "audit_system_by_stream_and_source"): (
            "Audit and system stream records that still have Kubernetes namespace and container fields. "
            "Use this table to find workloads, for example MongoDB or OpenSearch, that produce audit logs."
        ),
        ("logs", "graylog_streams", "k8s_events_by_object"): (
            "Kubernetes events stream split by namespace and involved object."
        ),
        ("logs", "namespace_logs"): (
            "Namespace-scoped workload logs, excluding Kubernetes events. "
            "Use this section to find noisy namespaces and containers."
        ),
        ("logs", "k8s_events"): (
            "Kubernetes event logs only. Events are reported separately from namespace workload logs because "
            "they do not have a container source and need object-based grouping."
        ),
        ("logs", "categories"): (
            "Logs grouped by normalized `log_category`, excluding Kubernetes events. "
            "Current categories are `container`, `audit`, and `system`."
        ),
        ("logs", "levels"): (
            "Severity-level distribution, excluding Kubernetes events. Container and non-container drill-downs "
            "are shown separately."
        ),
        ("logs", "debug_trace"): (
            "Debug and trace logs, excluding Kubernetes events. These are usually first candidates for log-level tuning."
        ),
        ("logs", "log_patterns"): (
            "Most frequent normalized `_msg` patterns for namespace/container logs, excluding Kubernetes events. "
            "Patterns are calculated at query time and are not stored back into VictoriaLogs."
        ),
        ("logs", "log_patterns", "top_patterns_by_count"): (
            "Top normalized message patterns by count. The query copies `_msg` to `message_pattern`, then applies "
            "`collapse_nums prettify` to replace values such as numbers, UUIDs, IPs, and timestamps with placeholders."
        ),
        ("logs", "message_size"): (
            "Top message-size views for non-event logs. Source top lists are limited to namespace/container records. "
            "For VictoriaLogs, message size means the size of `_msg` only, not the whole parsed log record."
        ),
        ("logs", "levels", "top_namespaces_by_level"): (
            "Top namespaces for each level, ranked by number of records."
        ),
        ("logs", "levels", "top_sources_by_level"): (
            "Top namespace/container sources for each level, ranked by summed gl2_accounted_message_size in KB."
        ),
        ("logs", "levels", "top_nodes_without_namespace_source_by_level"): (
            "Logs grouped by level and nodename when namespace and container field are absent. "
            "This helps inspect host-level, system, or audit logs that are not tied to a Kubernetes workload."
        ),
        ("logs", "audit_system_without_namespace_container"): (
            "Diagnostic view for audit and system stream records that do not have namespace and container. "
            "These are usually host-level or non-Kubernetes logs, not all audit/system logs."
        ),
        ("logs", "audit_system_without_namespace_container", "total_by_stream"): (
            "Audit and system stream totals for host-level/non-Kubernetes records without namespace and container."
        ),
        ("logs", "audit_system_without_namespace_container", "by_stream_and_nodename"): (
            "Audit and system records without namespace/container split by nodename."
        ),
        ("logs", "audit_system_without_namespace_container", "by_stream_and_level"): (
            "Audit and system records without namespace/container split by level."
        ),
        ("logs", "namespace_logs", "top_namespaces_by_count"): (
            "Namespaces that produced the highest number of log records in the selected time range. "
            "This view excludes Kubernetes events. It shows namespace-level totals only; container-level "
            "details are shown in Top Sources By Log Count. Rows are sorted by `messages_count`; a source with "
            "more records can still have lower `sum_message_size_kb` if its `_msg` values are shorter."
        ),
        ("logs", "namespace_logs", "total"): (
            "Total namespace-scoped log count for the selected time range, excluding Kubernetes events. "
            "`sum_message_size_kb` is calculated as `sum_len(_msg)` over matched log rows, so it is a logical "
            "message payload size for the selected window, not VictoriaLogs physical storage usage."
        ),
        ("logs", "namespace_logs", "top_by_count"): (
            "Most active log sources grouped by namespace and source field, usually container. "
            "Use it to find noisy workloads inside noisy namespaces. Records without these fields "
            "are not shown in this drill-down table. `sum_message_size_kb` is calculated from `_msg` only, "
            "not from the whole parsed log record."
        ),
        ("logs", "unattributed_logs", "total"): (
            "Total logs without namespace and container fields, excluding Kubernetes events. "
            "These records are usually system, audit, host-level, or otherwise non-container logs. "
            "A zero total means no matching records were found for this section."
        ),
        ("logs", "unattributed_logs", "top_nodes_by_count"): (
            "Nodes that produced the most logs without namespace and container fields. "
            "Use this to find noisy system, audit, host-level, or otherwise non-container log sources. "
            "If the section total is zero, this grouped table has no node rows to display."
        ),
        ("logs", "levels", "total_by_level"): (
            "Distribution of log records by severity level. Size values are shown in KB when the table includes them."
        ),
        ("logs", "levels", "top_by_level_and_source"): (
            "Noisiest container sources grouped by severity level, namespace, and source field. "
            "Useful for finding who emits most warnings, errors, debug logs, and so on."
        ),
        ("logs", "levels", "top_non_container_by_level_and_node"): (
            "Noisiest non-container sources grouped by severity level and nodename. "
            "This excludes Kubernetes events and logs tied to namespace/container workloads."
        ),
        ("logs", "debug_trace", "top_by_count"): (
            "Sources that emit the most debug or trace logs. These are usually the first candidates "
            "for log-level tuning."
        ),
        ("logs", "k8s_events", "top_by_count"): (
            "Kubernetes event sources grouped by namespace and involved object. "
            "Container is not expected for these records."
        ),
        ("logs", "message_size", "top_by_max_message_size"): (
            "Sources with the largest single message size."
        ),
        ("logs", "schema_quality"): (
            "Schema quality view based on `parse_field_count`. Use it to find sources that produce records "
            "with too many parsed fields. This field is produced by the Fluent Bit pipeline; Fluentd or older "
            "logging versions may not have it, so this section can be empty even when logs exist."
        ),
        ("logs", "schema_quality", "top_by_max_fields"): (
            "Sources ranked by the highest observed `parse_field_count` value. If the collector does not add "
            "`parse_field_count`, no rows are expected."
        ),
        ("logs", "large_messages"): (
            "Sources ranked by the largest observed `gl2_accounted_message_size` value."
        ),
        ("logs", "large_messages", "top_by_max_message_size"): (
            "Sources with the largest single accounted record size. "
            "This is based on `max(gl2_accounted_message_size)` and helps find individual oversized records."
        ),
        ("storage", "victorialogs_block_stats"): (
            "Optional VictoriaLogs physical storage diagnostics from `block_stats`. "
            "VictoriaLogs stores values for every log field in separate compressed data blocks. "
            "See [VictoriaLogs storage overview](https://docs.victoriametrics.com/victorialogs/faq/#how-does-victorialogs-work) "
            "and [block_stats](https://docs.victoriametrics.com/victorialogs/logsql/#block_stats-pipe)."
        ),
        ("storage", "victorialogs_block_stats", "top_streams_by_disk_usage"): (
            "VictoriaLogs diagnostic view of streams that occupy the most disk space according to `block_stats`. "
            "`values` is the stored values size for the stream, `bloom` is bloom-filter overhead, and "
            "`total` is `values + bloom`. Sizes use adaptive units. Bloom filters are auxiliary data used "
            "to skip data blocks that do not contain searched terms."
        ),
        ("storage", "victorialogs_block_stats", "top_fields_by_disk_usage"): (
            "VictoriaLogs diagnostic view of fields that occupy the most disk space. "
            "`values` is stored values size, `bloom` is bloom-filter overhead, and `total` is their sum. "
            "Sizes use adaptive units. Bloom filters are auxiliary data used to skip data blocks that do not "
            "contain searched terms."
        ),
    }
    if path in exact:
        return exact[path]
    if grandparent == "categories" and name == "total":
        return "Total number of log records in this category for the selected time range."
    if grandparent == "categories" and name == "top_by_count":
        return "Noisiest sources inside this log category, ranked by number of records."
    if parent in {"emerg", "alert", "crit", "err", "warning", "notice", "info", "debug"} and name == "top_by_count":
        return f"Noisiest sources for {parent} level logs."
    if parent == "graylog_streams":
        return (
            "Graylog stream-based view. This uses Graylog routing streams and works even when records "
            "do not have log_category."
        )
    if grandparent == "graylog_streams" and name == "total":
        return "Total number of records routed to this Graylog stream in the selected time range."
    if grandparent == "graylog_streams" and name == "top_nodes_by_count":
        return "Nodes that produced the most records in this Graylog stream."
    if grandparent == "graylog_streams" and name == "top_namespaces_by_count":
        return "Namespaces that produced the most records in this Graylog stream, when namespace is present."
    if grandparent == "graylog_streams" and name == "top_sources_by_count":
        return "Namespace and source-field breakdown for this Graylog stream, when both fields are present."
    if grandparent == "graylog_streams" and name == "top_levels_by_count":
        return "Severity distribution for records routed to this Graylog stream."
    if name == "top_by_count":
        return "Top rows ranked by number of log records."
    if name.startswith("top_"):
        return "Top rows ranked by the metric shown in the last column."
    return ""


def render_explanation(path: tuple[str, ...]) -> str:
    description = table_description(path)
    if not description:
        return ""
    return (
        '<details class="explain"><summary>What this table shows</summary>'
        f"<p>{render_inline_markup(description)}</p></details>"
    )


def table_from_rows(rows: list[Any], headers: list[str] | None = None, path: tuple[str, ...] = ()) -> str:
    if not rows:
        return '<p class="muted">No matching rows for this grouped view.</p>'
    if all(isinstance(row, dict) for row in rows):
        if headers is None:
            headers = []
            for row in rows:
                for key in row:
                    if key not in headers:
                        headers.append(key)
        body = "\n".join(
            "<tr>"
            + "".join(
                f'<td class="{column_class(header)}">{format_table_cell(row.get(header, ""), header, path)}</td>'
                for header in headers
            )
            + "</tr>"
            for row in rows
        )
        header_html = "".join(render_header_cell(path, header) for header in headers)
        return f'<table class="{table_class(path)}"><thead><tr>{header_html}</tr></thead><tbody>{body}</tbody></table>'
    if all(isinstance(row, list) for row in rows):
        width = max(len(row) for row in rows)
        if headers is None:
            headers = [f"value_{index + 1}" for index in range(width)]
        if len(headers) < width:
            headers = [*headers, *(f"value_{index + 1}" for index in range(len(headers), width))]
        body = "\n".join(
            "<tr>"
            + "".join(
                f'<td class="{column_class(headers[index])}">'
                f"{format_table_cell(row[index], headers[index], path) if index < len(row) else ''}</td>"
                for index in range(width)
            )
            + "</tr>"
            for row in rows
        )
        header_html = "".join(render_header_cell(path, header) for header in headers)
        return f'<table class="{table_class(path)}"><thead><tr>{header_html}</tr></thead><tbody>{body}</tbody></table>'
    return "<pre>" + html_escape(json.dumps(rows, ensure_ascii=True, indent=2)) + "</pre>"


def render_value_block(
    name: str,
    value: Any,
    headers: list[str] | None = None,
    path: tuple[str, ...] = (),
) -> str:
    status = section_status(value)
    current_path = (*path, name)
    title = html_escape(display_title(current_path))
    if isinstance(value, dict) and "error" in value:
        return f'<article class="panel error"><h3>{title}</h3><pre>{html_escape(value["error"])}</pre></article>'
    if isinstance(value, dict) and value.get("status") == "skipped":
        panel = (
            f'<article class="panel skipped"><h3>{title}</h3>'
            f'<p>{html_escape(value.get("reason", "Skipped."))}</p></article>'
        )
        if path == ("logs",):
            return f'<section class="group"><h2>{title}</h2>{panel}</section>'
        return panel
    if isinstance(value, dict) and "value" in value and len(value) == 1:
        return (
            f'<article class="metric"><span>{title}</span>{render_explanation(current_path)}'
            f'<strong>{format_cell(value["value"])}</strong></article>'
        )
    if isinstance(value, list):
        return (
            f'<article class="panel {status}"><h3>{title}</h3>'
            f"{render_explanation(current_path)}{table_from_rows(value, headers, current_path)}</article>"
        )
    if isinstance(value, dict) and "queries" in value and len(value) == 1:
        return (
            f'<article class="panel skipped"><h3>{title}</h3>'
            '<p>Dry-run mode: query rendered, backend was not called.</p>'
            f'<details><summary>Queries</summary><pre>{html_escape(json.dumps(value["queries"], ensure_ascii=True, indent=2))}</pre></details>'
            "</article>"
        )
    if isinstance(value, dict):
        column_map = value.get("columns", {})
        children = "".join(
            render_value_block(
                child_name,
                child_value,
                column_map.get(child_name) if isinstance(column_map, dict) else None,
                current_path,
            )
            for child_name, child_value in value.items()
            if child_name not in ("queries", "columns")
        )
        queries = ""
        if "queries" in value:
            queries = (
                f'<details><summary>Queries for {title}</summary>'
                f'<pre>{html_escape(json.dumps(value["queries"], ensure_ascii=True, indent=2))}</pre></details>'
            )
        return f'<section class="group"><h2>{title}</h2>{render_explanation(current_path)}{children}{queries}</section>'
    return f'<article class="panel"><h3>{title}</h3><p>{format_cell(value)}</p></article>'


def render_summary(report: dict[str, Any]) -> str:
    items = {
        "Backend": report.get("backend_type", ""),
        "Window from": report.get("time_from", ""),
        "Window to": report.get("time_to", ""),
        "Generated at": report.get("generated_at", ""),
    }
    return "".join(
        f'<article class="metric"><span>{html_escape(name)}</span><strong>{html_escape(value)}</strong></article>'
        for name, value in items.items()
    )


def render_detected_problems(problems: Any) -> str:
    if not isinstance(problems, list):
        return ""
    if not problems:
        return (
            '<section class="group problems ok"><h2>Detected Problems</h2>'
            "<p>No detected problems for configured thresholds.</p></section>"
        )
    cards = []
    for problem in problems:
        if not isinstance(problem, dict):
            continue
        title = html_escape(problem.get("problem", "Detected problem"))
        severity = html_escape(problem.get("severity", "warning"))
        description = html_escape(problem.get("description", ""))
        evidence = problem.get("evidence", [])
        evidence_html = render_problem_evidence(evidence)
        cards.append(
            f'<article class="panel problem {severity}"><div class="problem-head">'
            f"<h3>{title}</h3><span>{severity}</span></div>"
            f"<p>{description}</p>{evidence_html}</article>"
        )
    return f'<section class="group problems"><h2>Detected Problems</h2>{"".join(cards)}</section>'


def render_problem_evidence(evidence: Any) -> str:
    if isinstance(evidence, list) and evidence:
        return f'<details open><summary>Details</summary>{table_from_rows(evidence)}</details>'
    if not isinstance(evidence, dict):
        return ""
    sections = []
    for name, rows in evidence.items():
        if not isinstance(rows, list) or not rows:
            continue
        sections.append(
            f'<h4>{html_escape(name.replace("_", " ").title())}</h4>'
            f"{table_from_rows(rows)}"
        )
    if not sections:
        return ""
    return f'<details open><summary>Details</summary>{"".join(sections)}</details>'


def render_notes(notes: Any) -> str:
    if not isinstance(notes, list) or not notes:
        return ""
    items = "".join(f"<li>{render_inline_markup(str(note))}</li>" for note in notes)
    return f'<section class="group notes"><h2>Notes</h2><ul>{items}</ul></section>'


def report_notes(report: dict[str, Any], notes: Any) -> list[str]:
    result = list(notes) if isinstance(notes, list) else []
    if report.get("backend_type") == "victorialogs":
        result.append(
            "`sum_message_size` and message-size tables are based on `sum_len(_msg)`. "
            "They measure only the `_msg` field, not the whole parsed log record. Logs with many parsed fields "
            "can take more storage space than logs with larger `_msg` but fewer fields, so use this metric as "
            "message payload size, not as full storage impact."
        )
    return result


def render_recommendations_link() -> str:
    return (
        '<section class="group recommendations">'
        "<h2>Next Steps</h2>"
        "<p>Use the cleanup recommendations when deciding how to free disk space, reduce noisy logs, "
        "or reject unwanted records before they reach the storage backend.</p>"
        '<p><a href="../../docs/log-storage-cleanup-recommendations.md">'
        "Log Storage Cleanup Recommendations</a></p>"
        "</section>"
    )


def render_html_report(report: dict[str, Any]) -> str:
    logs = report.get("logs", {})
    notes = report_notes(report, logs.get("notes") if isinstance(logs, dict) else [])
    log_sections = {
        key: value
        for key, value in logs.items()
        if isinstance(logs, dict) and key != "notes"
    }
    body = "\n".join(
        (
            '<section class="hero">',
            "<div>",
            "<p>Log Storage Analysis</p>",
            "<h1>Storage and Log Noise Report</h1>",
            f'<span class="subtle">{html_escape(report.get("backend_url", ""))}</span>',
            "</div>",
            "</section>",
            f'<section class="summary">{render_summary(report)}</section>',
            render_detected_problems(report.get("detected_problems", [])),
            render_value_block("storage", report.get("storage", {})),
            render_value_block("index_stats", report["index_stats"]) if "index_stats" in report else "",
            "".join(render_value_block(name, value, path=("logs",)) for name, value in log_sections.items()),
            render_notes(notes),
            render_recommendations_link(),
            '<details class="raw"><summary>Raw JSON report</summary>',
            f'<pre>{html_escape(json.dumps(report, ensure_ascii=True, indent=2))}</pre></details>',
        )
    )
    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Log Storage Analysis Report</title>
  <style>
    :root {{
      --bg: #f5f1e8;
      --ink: #1d2528;
      --muted: #687276;
      --card: #fffaf0;
      --line: #dfd5c3;
      --accent: #b85c38;
      --accent-2: #264653;
      --error: #b42318;
      --skip: #8a5a00;
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      background:
        radial-gradient(circle at top left, rgba(184, 92, 56, .18), transparent 30rem),
        linear-gradient(135deg, #f5f1e8 0%, #ece2d0 100%);
      color: var(--ink);
      font: 15px/1.5 "Aptos", "Segoe UI", sans-serif;
    }}
    main {{ width: min(1180px, calc(100% - 32px)); margin: 0 auto; padding: 32px 0 56px; }}
    .hero {{
      border: 1px solid var(--line);
      border-radius: 28px;
      padding: 32px;
      background: linear-gradient(135deg, rgba(38, 70, 83, .95), rgba(38, 70, 83, .78));
      color: #fffaf0;
      box-shadow: 0 24px 70px rgba(38, 70, 83, .18);
    }}
    .hero p {{ margin: 0 0 6px; letter-spacing: .12em; text-transform: uppercase; color: #f2c9a8; }}
    .hero h1 {{ margin: 0; font-size: clamp(32px, 6vw, 58px); line-height: .95; }}
    .subtle {{ display: block; margin-top: 14px; color: rgba(255, 250, 240, .72); word-break: break-all; }}
    .summary {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(190px, 1fr)); gap: 14px; margin: 18px 0; }}
    .metric, .panel, .group, .raw {{
      border: 1px solid var(--line);
      border-radius: 20px;
      background: rgba(255, 250, 240, .82);
      box-shadow: 0 16px 38px rgba(38, 70, 83, .08);
    }}
    .metric {{ padding: 18px; }}
    .metric span {{ display: block; color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .08em; }}
    .metric strong {{ display: block; margin-top: 8px; font-size: 22px; word-break: break-word; }}
    .group {{ margin-top: 18px; padding: 18px; }}
    .group h2 {{ margin: 0 0 14px; font-size: 24px; color: var(--accent-2); }}
    .panel {{ margin: 12px 0; padding: 16px; overflow-x: auto; }}
    .panel h3 {{ margin: 0 0 12px; font-size: 17px; }}
    .error {{ border-color: rgba(180, 35, 24, .45); background: #fff4f2; }}
    .error h3 {{ color: var(--error); }}
    .skipped {{ border-color: rgba(138, 90, 0, .35); background: #fff7df; }}
    .skipped h3 {{ color: var(--skip); }}
    .problems {{
      border-color: rgba(180, 35, 24, .32);
      background: linear-gradient(135deg, rgba(255, 244, 242, .96), rgba(255, 250, 240, .84));
    }}
    .problems.ok {{
      border-color: rgba(38, 70, 83, .18);
      background: rgba(255, 250, 240, .82);
    }}
    .problem {{
      border-color: rgba(180, 35, 24, .42);
      border-left: 7px solid var(--error);
      background: #fff4f2;
      box-shadow: 0 18px 42px rgba(180, 35, 24, .1);
    }}
    .problem-head {{ display: flex; align-items: center; justify-content: space-between; gap: 12px; }}
    .problem-head h3 {{ color: var(--error); margin: 0; }}
    .problem-head span {{
      display: inline-block;
      padding: 3px 9px;
      border-radius: 999px;
      background: rgba(180, 35, 24, .12);
      color: var(--error);
      font-size: 11px;
      font-weight: 800;
      letter-spacing: .08em;
      text-transform: uppercase;
    }}
    .problem h4 {{
      margin: 14px 0 8px;
      color: var(--accent-2);
      font-size: 13px;
      letter-spacing: .06em;
      text-transform: uppercase;
    }}
    table {{ width: 100%; border-collapse: collapse; min-width: 520px; }}
    .log-patterns-table {{
      min-width: 0;
      table-layout: fixed;
    }}
    .log-patterns-table .col-message-pattern {{
      width: 52%;
      max-width: 48rem;
      overflow-wrap: anywhere;
      word-break: break-word;
    }}
    th, td {{ padding: 9px 10px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }}
    th {{ color: var(--accent-2); font-size: 12px; text-transform: uppercase; letter-spacing: .06em; background: rgba(38, 70, 83, .06); }}
    tr:hover td {{ background: rgba(184, 92, 56, .06); }}
    code, pre {{
      font-family: "Cascadia Code", "SFMono-Regular", Consolas, monospace;
      font-size: 12px;
      white-space: pre-wrap;
      word-break: break-word;
    }}
    .explain code, .notes code {{
      display: inline-block;
      padding: 1px 6px;
      border: 1px solid rgba(38, 70, 83, .18);
      border-radius: 7px;
      background: rgba(38, 70, 83, .09);
      color: var(--accent-2);
      font-weight: 800;
      white-space: normal;
    }}
    pre {{ margin: 0; }}
    details {{ margin-top: 12px; }}
    summary {{ cursor: pointer; color: var(--accent); font-weight: 700; }}
    .raw {{ margin-top: 18px; padding: 18px; }}
    .recommendations a {{ color: var(--accent); font-weight: 800; }}
    .muted {{ color: var(--muted); }}
    ul {{ margin: 0; padding-left: 22px; }}
    @media (max-width: 640px) {{
      main {{ width: min(100% - 20px, 1180px); padding-top: 16px; }}
      .hero {{ padding: 22px; border-radius: 22px; }}
      .group {{ padding: 12px; }}
    }}
  </style>
</head>
<body>
  <main>
    {body}
  </main>
</body>
</html>
"""


def write_html_report(report: dict[str, Any], output: str) -> None:
    Path(output).write_text(render_html_report(report), encoding="utf-8")
