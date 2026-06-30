#!/usr/bin/env python3
"""Collect storage and log distribution statistics from Graylog or VictoriaLogs."""

from __future__ import annotations

import argparse
import json
import math
import os
import re
import sys
from datetime import UTC, datetime, timedelta
from functools import lru_cache
from html_report import write_html_report
from pathlib import Path
from typing import Any

from clients import (
    GraylogIndexStatsClient,
    HttpClient,
    QueryError,
    VictoriaLogsClient,
    VictoriaMetricsClient,
    field_name,
)
from collectors import collect_log_report

DURATION_PATTERN = re.compile(r"^(?:0|[1-9][0-9]*)(?:s|m|h|d|w)$")
SIZE_PATTERN = re.compile(r"^([1-9][0-9]*)(?:\s*(b|kb|k|mb|m|gb|g))?$", re.IGNORECASE)
BYTES_IN_KB = 1024
BYTE_FIELDS_WITHOUT_SUFFIX = {
    "sum_gl2_accounted_message_size",
    "max_gl2_accounted_message_size",
}
SOURCE_COLUMN_IGNORED = {
    "namespace",
    "stream",
    "level",
    "level_name",
    "nodename",
    "count",
    "messages_count",
    "sum_gl2_accounted_message_size_kb",
    "max_gl2_accounted_message_size_kb",
    "sum_message_size_kb",
    "max_message_size_kb",
    "max_parse_field_count",
    "share_percent",
    "percent",
}


def positive_int(value: str) -> int:
    number = int(value)
    if number < 1:
        raise argparse.ArgumentTypeError("value must be greater than zero")
    return number


def positive_size_kb(value: str) -> int:
    match = SIZE_PATTERN.fullmatch(value.strip())
    if not match:
        raise argparse.ArgumentTypeError("size must look like 60, 60KB, 1MB, or 1GB")
    amount = int(match.group(1))
    unit = (match.group(2) or "kb").lower()
    if unit == "b":
        return max(1, math.ceil(amount / BYTES_IN_KB))
    if unit in {"kb", "k"}:
        return amount
    if unit in {"mb", "m"}:
        return amount * BYTES_IN_KB
    if unit in {"gb", "g"}:
        return amount * BYTES_IN_KB * BYTES_IN_KB
    raise argparse.ArgumentTypeError("size must look like 60, 60KB, 1MB, or 1GB")


def duration_seconds(value: str, *, allow_zero: bool = False) -> int:
    if not DURATION_PATTERN.fullmatch(value):
        raise argparse.ArgumentTypeError("time range must look like 30m, 1h, or 7d")
    amount = int(value[:-1])
    if amount == 0 and not allow_zero:
        raise argparse.ArgumentTypeError("duration must be greater than zero")
    multiplier = {"s": 1, "m": 60, "h": 3600, "d": 86400, "w": 604800}[value[-1]]
    return amount * multiplier


def env_value(name: str, default: str = "") -> str:
    value = os.getenv(name)
    if value is None:
        return default
    if value == "" and default:
        return default
    return value


def env_required(name: str) -> bool:
    return not env_value(name)


def env_bool(name: str, default: bool = False) -> bool:
    default_value = "true" if default else "false"
    return env_value(name, default_value).lower() in ("1", "true", "yes", "on")


def env_positive_int(name: str, default: str) -> int:
    return positive_int(env_value(name, default))


def env_positive_size_kb_with_fallback(name: str, fallback_name: str, default: str) -> int:
    return positive_size_kb(env_value(name, env_value(fallback_name, default)))


def percentage(value: str) -> float:
    number = float(value)
    if number < 0 or number > 100:
        raise argparse.ArgumentTypeError("percentage must be between 0 and 100")
    return number


def env_percentage(name: str, default: str) -> float:
    return percentage(env_value(name, default))


def comma_separated_fields(value: str) -> list[str]:
    fields = [field.strip() for field in value.split(",") if field.strip()]
    if not fields:
        raise argparse.ArgumentTypeError("at least one field must be provided")
    for field in fields:
        field_name(field)
    return fields


def output_path(value: str) -> str:
    if value == "-":
        raise argparse.ArgumentTypeError("output file path is required; stdout output is not supported")
    return value


def parser() -> argparse.ArgumentParser:
    result = argparse.ArgumentParser(description=__doc__)
    result.add_argument(
        "--backend-type",
        default=env_value("BACKEND_TYPE"),
        required=env_required("BACKEND_TYPE"),
        choices=("graylog", "victorialogs"),
        help="Graylog or VictoriaLogs backend type. Env: BACKEND_TYPE.",
    )
    result.add_argument(
        "--backend-url",
        default=env_value("BACKEND_URL"),
        required=env_required("BACKEND_URL"),
        help="Graylog or VictoriaLogs base URL. Env: BACKEND_URL.",
    )
    result.add_argument("--graylog-user", default=env_value("GRAYLOG_USER"), help="Graylog API username.")
    result.add_argument("--graylog-pass", default=env_value("GRAYLOG_PASS"), help="Graylog API password.")
    result.add_argument("--vl-user", default=env_value("VL_USER"), help="VictoriaLogs API username.")
    result.add_argument("--vl-pass", default=env_value("VL_PASS"), help="VictoriaLogs API password.")
    result.add_argument(
        "--victoriametrics-url",
        default=env_value("VICTORIAMETRICS_URL"),
        required=env_required("VICTORIAMETRICS_URL"),
        help="VictoriaMetrics base URL used to retrieve storage metrics. Env: VICTORIAMETRICS_URL.",
    )
    result.add_argument("--vm-user", default=env_value("VM_USER"), help="VictoriaMetrics API username.")
    result.add_argument("--vm-pass", default=env_value("VM_PASS"), help="VictoriaMetrics API password.")
    result.add_argument(
        "--time-range",
        default=env_value("TIME_RANGE"),
        required=env_required("TIME_RANGE"),
        metavar="DURATION",
        help="Log analysis lookback, for example 1h or 24h. Env: TIME_RANGE.",
    )
    result.add_argument(
        "--time-offset",
        default=env_value("TIME_OFFSET", "0s"),
        metavar="DURATION",
        help=(
            "Shift the analysis window into the past, for example 2h means "
            "analyze [now-2h-time-range, now-2h]. Env: TIME_OFFSET."
        ),
    )
    result.add_argument(
        "--source-field",
        default=env_value("SOURCE_FIELD", "container"),
        help="Field used as a log source in top lists. Default: container.",
    )
    result.add_argument(
        "--system-source-fields",
        type=comma_separated_fields,
        default=comma_separated_fields(env_value("SYSTEM_SOURCE_FIELDS", "nodename")),
        help="Comma-separated fields used for system log top lists. Default: nodename.",
    )
    result.add_argument(
        "--audit-source-fields",
        type=comma_separated_fields,
        default=comma_separated_fields(env_value("AUDIT_SOURCE_FIELDS", "namespace,container")),
        help="Comma-separated fields used for audit log top lists. Default: namespace,container.",
    )
    result.add_argument("--top-limit", type=positive_int, default=env_positive_int("TOP_LIMIT", "50"))
    result.add_argument(
        "--include-vl-block-stats",
        action="store_true",
        default=env_bool("INCLUDE_VL_BLOCK_STATS"),
        help=(
            "Add VictoriaLogs diagnostic block_stats queries for top streams and fields by disk usage. "
            "Env: INCLUDE_VL_BLOCK_STATS."
        ),
    )
    result.add_argument(
        "--include-detailed-levels",
        action="store_true",
        default=env_bool("INCLUDE_DETAILED_LEVELS"),
        help=(
            "VictoriaLogs only: add per-level top-source queries. "
            "Disabled by default for faster reports. Env: INCLUDE_DETAILED_LEVELS."
        ),
    )
    result.add_argument(
        "--parallel-queries",
        dest="parallel_queries",
        action=argparse.BooleanOptionalAction,
        default=env_bool("PARALLEL_QUERIES", True),
        help=(
            "Run independent Graylog report queries in parallel. "
            "Use --no-parallel-queries or PARALLEL_QUERIES=false for conservative backend load. "
            "Env: PARALLEL_QUERIES."
        ),
    )
    result.add_argument(
        "--graylog-query-workers",
        type=positive_int,
        default=env_positive_int("GRAYLOG_QUERY_WORKERS", "4"),
        help=(
            "Maximum concurrent Graylog HTTP requests when parallel queries are enabled. "
            "Default: 4. Env: GRAYLOG_QUERY_WORKERS."
        ),
    )
    result.add_argument(
        "--fields-count-threshold",
        type=positive_int,
        default=env_positive_int("FIELDS_COUNT_THRESHOLD", "20"),
        help="Minimum parse_field_count considered suspicious. Default: 20.",
    )
    result.add_argument(
        "--graylog-large-record-threshold-kb",
        type=positive_size_kb,
        default=env_positive_size_kb_with_fallback(
            "GRAYLOG_LARGE_RECORD_THRESHOLD_KB",
            "LARGE_MESSAGE_THRESHOLD_KB",
            "60",
        ),
        help=(
            "Graylog gl2_accounted_message_size threshold for Large records. "
            "Accepts plain KB or units like 60KB, 1MB. Default: 60 KB. "
            "Env: GRAYLOG_LARGE_RECORD_THRESHOLD_KB."
        ),
    )
    result.add_argument(
        "--vl-large-message-threshold-kb",
        type=positive_size_kb,
        default=env_positive_size_kb_with_fallback(
            "VL_LARGE_MESSAGE_THRESHOLD_KB",
            "LARGE_MESSAGE_THRESHOLD_KB",
            "60",
        ),
        help=(
            "VictoriaLogs _msg size threshold for Large messages. "
            "Accepts plain KB or units like 60KB, 1MB. Default: 60 KB. "
            "Env: VL_LARGE_MESSAGE_THRESHOLD_KB."
        ),
    )
    result.add_argument(
        "--large-message-threshold-kb",
        type=positive_size_kb,
        default=None,
        help=(
            "Deprecated alias. Sets both Graylog record and VictoriaLogs _msg thresholds "
            "when used as a CLI argument. Prefer --graylog-large-record-threshold-kb "
            "and --vl-large-message-threshold-kb."
        ),
    )
    result.add_argument(
        "--error-level-percent-threshold",
        type=percentage,
        default=env_percentage("ERROR_LEVEL_PERCENT_THRESHOLD", "10"),
        help="Warn when error-level logs exceed this share of total logs count. Default: 10.",
    )
    result.add_argument(
        "--single-source-percent-threshold",
        type=percentage,
        default=env_percentage("SINGLE_SOURCE_PERCENT_THRESHOLD", "20"),
        help="Warn when one container source exceeds this share of all counted logs. Default: 20.",
    )
    result.add_argument(
        "--filesystem-selector",
        default=env_value("FILESYSTEM_SELECTOR"),
        help='PromQL label matchers for filesystem metrics, e.g. mountpoint="/data",instance="node:9100".',
    )
    result.add_argument(
        "--output",
        type=output_path,
        default=env_value("OUTPUT"),
        required=env_required("OUTPUT"),
        help="Required JSON output file path. Env: OUTPUT.",
    )
    result.add_argument(
        "--html-output",
        default=env_value("HTML_OUTPUT"),
        help="Optional self-contained HTML report output path. Env: HTML_OUTPUT.",
    )
    result.add_argument("--insecure-skip-verify", action="store_true", default=env_bool("INSECURE_SKIP_VERIFY"))
    result.add_argument(
        "--dry-run",
        action="store_true",
        default=env_bool("DRY_RUN"),
        help="Render backend queries without making API requests.",
    )
    return result


def write_report(report: dict[str, Any], output: str) -> None:
    rendered = json.dumps(report, ensure_ascii=True, indent=2) + "\n"
    Path(output).write_text(rendered, encoding="utf-8")


def utc_iso(value: datetime) -> str:
    return value.isoformat(timespec="milliseconds").replace("+00:00", "Z")


def time_window(time_range: str, time_offset: str) -> tuple[int, int, str, str]:
    range_seconds = duration_seconds(time_range)
    offset_seconds = duration_seconds(time_offset, allow_zero=True)
    now = datetime.now(UTC)
    window_to = now - timedelta(seconds=offset_seconds)
    window_from = window_to - timedelta(seconds=range_seconds)
    return range_seconds, offset_seconds, utc_iso(window_from), utc_iso(window_to)


def graylog_timerange(range_seconds: int, offset_seconds: int, time_from: str, time_to: str) -> dict[str, Any]:
    if offset_seconds == 0:
        return {"type": "relative", "range": range_seconds}
    return {"type": "absolute", "from": time_from, "to": time_to}


def victorialogs_time_filter(time_range: str, time_offset: str) -> str:
    if duration_seconds(time_offset, allow_zero=True) == 0:
        return f"_time:{time_range}"
    return f"_time:{time_range} offset {time_offset}"


def kb_field_name(field: str) -> str:
    if field in BYTE_FIELDS_WITHOUT_SUFFIX:
        return f"{field}_kb"
    if field.endswith("_bytes"):
        return f"{field.removesuffix('_bytes')}_kb"
    return field


def is_byte_field(field: str) -> bool:
    return kb_field_name(field) != field


def bytes_to_kb(value: Any, *, ceil_value: bool = False) -> Any:
    if isinstance(value, str):
        try:
            number = float(value)
        except ValueError:
            return value
        converted = bytes_to_kb(number, ceil_value=ceil_value)
        return int(converted) if isinstance(converted, float) and converted.is_integer() else converted
    if not isinstance(value, (int, float)) or isinstance(value, bool):
        return value
    if ceil_value:
        return math.ceil(value / BYTES_IN_KB)
    result = round(value / BYTES_IN_KB, 6)
    if float(result).is_integer():
        return int(result)
    return result


def convert_row_byte_fields(row: Any, columns: list[str]) -> Any:
    if isinstance(row, dict):
        return convert_dict_byte_fields(row)
    if not isinstance(row, list):
        return convert_byte_fields_to_kb(row)
    converted = list(row)
    for index, column in enumerate(columns):
        if index < len(converted) and is_byte_field(column):
            converted[index] = bytes_to_kb(converted[index], ceil_value=column in BYTE_FIELDS_WITHOUT_SUFFIX)
    return converted


def convert_dict_byte_fields(row: dict[str, Any]) -> dict[str, Any]:
    converted: dict[str, Any] = {}
    for key, value in row.items():
        if isinstance(key, str) and is_byte_field(key):
            converted[kb_field_name(key)] = bytes_to_kb(value, ceil_value=key in BYTE_FIELDS_WITHOUT_SUFFIX)
        else:
            converted[key] = convert_byte_fields_to_kb(value)
    return converted


def convert_columns_to_kb(columns: Any) -> Any:
    if isinstance(columns, list):
        return [kb_field_name(column) if isinstance(column, str) else column for column in columns]
    if isinstance(columns, dict):
        return {name: convert_columns_to_kb(value) for name, value in columns.items()}
    return columns


def convert_byte_fields_to_kb(value: Any, row_columns: list[str] | None = None) -> Any:
    if isinstance(value, list):
        if row_columns:
            return [convert_row_byte_fields(row, row_columns) for row in value]
        return [convert_byte_fields_to_kb(item) for item in value]
    if not isinstance(value, dict):
        return value

    columns = value.get("columns")
    if isinstance(columns, dict):
        converted: dict[str, Any] = {"columns": convert_columns_to_kb(columns)}
        for key, child in value.items():
            if key == "columns":
                continue
            if key == "queries":
                converted[key] = child
                continue
            child_columns = columns.get(key)
            if isinstance(key, str) and is_byte_field(key):
                converted[kb_field_name(key)] = bytes_to_kb(child, ceil_value=key in BYTE_FIELDS_WITHOUT_SUFFIX)
            else:
                converted[key] = convert_byte_fields_to_kb(
                    child,
                    child_columns if isinstance(child_columns, list) else None,
                )
        return converted

    converted = {}
    for key, child in value.items():
        if key == "queries":
            converted[key] = child
        elif isinstance(key, str) and is_byte_field(key):
            converted[kb_field_name(key)] = bytes_to_kb(child, ceil_value=key in BYTE_FIELDS_WITHOUT_SUFFIX)
        else:
            converted[key] = convert_byte_fields_to_kb(child)
    return converted


def section_rows(report: dict[str, Any], path: tuple[str, ...]) -> tuple[list[Any], list[str]]:
    current: Any = report
    for key in path[:-1]:
        if not isinstance(current, dict):
            return [], []
        current = current.get(key)
    if not isinstance(current, dict):
        return [], []
    rows = current.get(path[-1], [])
    columns = current.get("columns", {}).get(path[-1], [])
    return (rows if isinstance(rows, list) else []), (columns if isinstance(columns, list) else [])


def row_value(row: Any, columns: list[str], *names: str) -> Any:
    if isinstance(row, dict):
        for name in names:
            if name in row:
                return row[name]
        return None
    if not isinstance(row, list):
        return None
    index_map = column_index_map(tuple(columns))
    for name in names:
        index = index_map.get(name)
        if index is not None:
            return row[index] if index < len(row) else None
    return None


@lru_cache(maxsize=128)
def column_index_map(columns: tuple[str, ...]) -> dict[str, int]:
    return {column: index for index, column in enumerate(columns)}


def number_value(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def source_column(columns: list[str]) -> str:
    return next((column for column in columns if column not in SOURCE_COLUMN_IGNORED), "")


def source_value(row: Any, columns: list[str]) -> Any:
    column = source_column(columns)
    return row_value(row, columns, column) if column else None


def total_workload_log_count(report: dict[str, Any]) -> float:
    if report.get("backend_type") == "graylog":
        rows, columns = section_rows(report, ("logs", "graylog_streams", "total_by_stream"))
        return sum(
            number_value(row_value(row, columns, "count", "messages_count"))
            for row in rows
            if row_value(row, columns, "stream") == "Default Stream"
        )
    rows, columns = section_rows(report, ("logs", "namespace_logs", "total"))
    return sum(number_value(row_value(row, columns, "count", "messages_count")) for row in rows)


def detected_large_graylog_messages(report: dict[str, Any], threshold_kb: int) -> dict[str, Any] | None:
    rows, columns = section_rows(report, ("logs", "large_messages", "top_by_max_message_size"))
    oversized = [
        row for row in rows
        if number_value(row_value(row, columns, "max_gl2_accounted_message_size_kb")) > threshold_kb
    ]
    if not oversized:
        return None
    evidence = [
        {
            "namespace": row_value(row, columns, "namespace"),
            "source": source_value(row, columns),
            "max_message_size_kb": row_value(row, columns, "max_gl2_accounted_message_size_kb"),
        }
        for row in oversized[:5]
    ]
    return {
        "problem": "Large records",
        "severity": "warning",
        "description": (
            "At least one source has max record size above "
            f"the configured threshold of {threshold_kb} KB."
        ),
        "evidence": evidence,
    }


def detected_large_victorialogs_messages(report: dict[str, Any], threshold_kb: int) -> dict[str, Any] | None:
    rows, columns = section_rows(report, ("logs", "message_size", "top_by_max_message_size"))
    oversized = [
        row for row in rows
        if number_value(row_value(row, columns, "max_message_size_kb")) > threshold_kb
    ]
    if not oversized:
        return None
    evidence = [
        {
            "namespace": row_value(row, columns, "namespace"),
            "source": source_value(row, columns),
            "max_message_size_kb": row_value(row, columns, "max_message_size_kb"),
        }
        for row in oversized[:5]
    ]
    return {
        "problem": "Large messages",
        "severity": "warning",
        "description": (
            "At least one source has max _msg size above "
            f"the configured threshold of {threshold_kb} KB."
        ),
        "evidence": evidence,
    }


def detected_error_level_share(report: dict[str, Any], threshold_percent: float) -> dict[str, Any] | None:
    rows, columns = section_rows(report, ("logs", "levels", "total_by_level"))
    total = sum(number_value(row_value(row, columns, "count", "messages_count")) for row in rows)
    if not total:
        return None
    error_count = 0.0
    for row in rows:
        level = str(row_value(row, columns, "level") or "").lower()
        level_name = str(row_value(row, columns, "level_name") or "").lower()
        if level in {"3", "err", "error"} or level_name in {"err", "error"}:
            error_count += number_value(row_value(row, columns, "count", "messages_count"))
    error_percent = round(error_count * 100 / total, 2)
    if error_percent <= threshold_percent:
        return None
    return {
        "problem": "High error-level log percentage",
        "severity": "warning",
        "description": (
            f"Error-level logs are {error_percent}% of total logs count, "
            f"above the configured threshold of {threshold_percent}%."
        ),
        "evidence": [
            {
                "error_logs_count": int(error_count),
                "total_logs_count": int(total),
                "percent": error_percent,
            }
        ],
    }


def detected_debug_trace_logs(report: dict[str, Any]) -> dict[str, Any] | None:
    rows, columns = section_rows(report, ("logs", "debug_trace", "total"))
    debug_count = sum(number_value(row_value(row, columns, "count", "messages_count")) for row in rows)
    if debug_count <= 0:
        return None
    source_rows, source_columns = section_rows(report, ("logs", "debug_trace", "top_by_count"))
    sources = [
        {
            "namespace": row_value(row, source_columns, "namespace"),
            "source": source_value(row, source_columns),
            "messages_count": int(number_value(row_value(row, source_columns, "count", "messages_count"))),
        }
        for row in source_rows[:5]
    ]
    return {
        "problem": "Debug/trace logs found",
        "severity": "warning",
        "description": "Debug or trace logs are present. Consider lowering log verbosity for listed sources.",
        "evidence": {
            "summary": [{"debug_trace_logs_count": int(debug_count)}],
            "top_sources": sources,
        },
    }


def detected_noisy_container_source(report: dict[str, Any], threshold_percent: float) -> dict[str, Any] | None:
    total = total_workload_log_count(report)
    if not total:
        return None
    if report.get("backend_type") == "graylog":
        rows, columns = section_rows(report, ("logs", "graylog_streams", "top_default_stream_sources"))
    else:
        rows, columns = section_rows(report, ("logs", "namespace_logs", "top_by_count"))
    noisy = []
    for row in rows:
        messages_count = number_value(row_value(row, columns, "count", "messages_count"))
        share = round(messages_count * 100 / total, 2)
        if share > threshold_percent:
            noisy.append(
                {
                    "namespace": row_value(row, columns, "namespace"),
                    "source": source_value(row, columns),
                    "messages_count": int(messages_count),
                    "share_percent": share,
                }
            )
    if not noisy:
        return None
    return {
        "problem": "Noisy container source",
        "severity": "warning",
        "description": (
            "At least one container source produces more than "
            f"{threshold_percent}% of all counted workload logs."
        ),
        "evidence": noisy[:5],
    }


def detected_too_many_fields(report: dict[str, Any], threshold: int) -> dict[str, Any] | None:
    rows, columns = section_rows(report, ("logs", "schema_quality", "top_by_max_fields"))
    suspicious = [
        row for row in rows
        if number_value(row_value(row, columns, "max_parse_field_count")) > threshold
    ]
    if not suspicious:
        return None

    evidence = [
        {
            "namespace": row_value(row, columns, "namespace"),
            "source": source_value(row, columns),
            "max_parse_field_count": int(number_value(row_value(row, columns, "max_parse_field_count"))),
        }
        for row in suspicious[:5]
    ]
    return {
        "problem": "Too many parsed fields",
        "severity": "warning",
        "description": (
            "At least one source has max parse_field_count above "
            f"the configured threshold of {threshold}."
        ),
        "evidence": evidence,
    }


def detect_problems(report: dict[str, Any], args: argparse.Namespace) -> list[dict[str, Any]]:
    checks = [
        detected_large_graylog_messages(report, args.graylog_large_record_threshold_kb)
        if report.get("backend_type") == "graylog"
        else None,
        detected_large_victorialogs_messages(report, args.vl_large_message_threshold_kb)
        if report.get("backend_type") == "victorialogs"
        else None,
        detected_error_level_share(report, args.error_level_percent_threshold),
        detected_debug_trace_logs(report),
        detected_noisy_container_source(report, args.single_source_percent_threshold),
        detected_too_many_fields(report, args.fields_count_threshold),
    ]
    return [problem for problem in checks if problem]


def graylog_index_stats_report(args: argparse.Namespace, client: HttpClient | None) -> dict[str, Any] | None:
    if args.backend_type != "graylog":
        return None
    if client is None:
        raise ValueError("Graylog HTTP client is required")
    return GraylogIndexStatsClient(client).report(dry_run=args.dry_run)


def graylog_http_client(args: argparse.Namespace) -> HttpClient | None:
    if args.backend_type != "graylog":
        return None
    return HttpClient(
        args.backend_url,
        user=args.graylog_user,
        password=args.graylog_pass,
        insecure_skip_verify=args.insecure_skip_verify,
        extra_headers={"X-Requested-By": "log-storage-report"},
    )


def victorialogs_client(args: argparse.Namespace, time_filter: str) -> VictoriaLogsClient | None:
    if args.backend_type != "victorialogs":
        return None
    return VictoriaLogsClient(
        HttpClient(
            args.backend_url,
            user=args.vl_user,
            password=args.vl_pass,
            insecure_skip_verify=args.insecure_skip_verify,
        ),
        time_filter,
        args.source_field,
        args.top_limit,
        parallel_queries=args.parallel_queries,
    )


def victorialogs_block_stats_report(
    args: argparse.Namespace,
    backend: VictoriaLogsClient | None,
) -> dict[str, Any] | None:
    if args.backend_type != "victorialogs" or not args.include_vl_block_stats:
        return None
    if backend is None:
        raise ValueError("VictoriaLogs client is required")
    return backend.execute_set(backend.block_stats_queries(), dry_run=args.dry_run)


def main() -> int:
    args = parser().parse_args()
    try:
        args.output = output_path(args.output)
    except argparse.ArgumentTypeError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    if args.large_message_threshold_kb is not None:
        args.graylog_large_record_threshold_kb = args.large_message_threshold_kb
        args.vl_large_message_threshold_kb = args.large_message_threshold_kb
    if args.backend_type == "graylog" and args.include_detailed_levels:
        print(
            "error: --include-detailed-levels is supported only for VictoriaLogs; "
            "Graylog reports include the aggregated levels section by default.",
            file=sys.stderr,
        )
        return 1
    try:
        range_seconds, offset_seconds, window_from, window_to = time_window(args.time_range, args.time_offset)
        vl_time_filter = victorialogs_time_filter(args.time_range, args.time_offset)
        graylog_time = graylog_timerange(range_seconds, offset_seconds, window_from, window_to)
        vm_client = VictoriaMetricsClient(
            HttpClient(
                args.victoriametrics_url,
                user=args.vm_user,
                password=args.vm_pass,
                insecure_skip_verify=args.insecure_skip_verify,
            ),
            args.filesystem_selector,
            window_to,
            parallel_queries=args.parallel_queries,
        )
        graylog_client = graylog_http_client(args)
        vl_client = victorialogs_client(args, vl_time_filter)
        report = {
            "generated_at": utc_iso(datetime.now(UTC)),
            "backend_type": args.backend_type,
            "backend_url": args.backend_url,
            "time_range": args.time_range,
            "time_offset": args.time_offset,
            "time_from": window_from,
            "time_to": window_to,
            "problem_thresholds": {
                "graylog_large_record_threshold_kb": args.graylog_large_record_threshold_kb,
                "vl_large_message_threshold_kb": args.vl_large_message_threshold_kb,
                "error_level_percent_threshold": args.error_level_percent_threshold,
                "single_source_percent_threshold": args.single_source_percent_threshold,
                "fields_count_threshold": args.fields_count_threshold,
            },
            "storage": vm_client.storage_report(args.backend_type, dry_run=args.dry_run),
        }
        index_stats = graylog_index_stats_report(args, graylog_client)
        if index_stats is not None:
            report["index_stats"] = index_stats
        block_stats = victorialogs_block_stats_report(args, vl_client)
        if block_stats is not None:
            report["storage"]["victorialogs_block_stats"] = block_stats
        report["logs"] = collect_log_report(args, vl_time_filter, graylog_time, graylog_client, vl_client)
        report = convert_byte_fields_to_kb(report)
        report["detected_problems"] = detect_problems(report, args) if not args.dry_run else []
        write_report(report, args.output)
        if args.html_output:
            write_html_report(report, args.html_output)
        return 0
    except (QueryError, ValueError, argparse.ArgumentTypeError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
