"""HTTP and backend clients for log storage reports."""

from __future__ import annotations

import base64
import json
import re
import socket
import ssl
from concurrent.futures import ThreadPoolExecutor, as_completed
from threading import BoundedSemaphore
from typing import Any
from urllib import error, parse, request

from storage import calculate_filesystem_usage, calculate_victorialogs_data_size, calculate_victorialogs_disk_usage


FIELD_PATTERN = re.compile(r"^[A-Za-z_][A-Za-z0-9_./-]*$")
SIMPLE_FIELD_PATTERN = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*$")
LOGSQL_STATS_BY_PATTERN = re.compile(r"\|\s*stats\s+by\s*\(([^)]*)\)", re.IGNORECASE)
LOGSQL_STATS_PATTERN = re.compile(r"\|\s*stats\b", re.IGNORECASE)
LOGSQL_ALIAS_PATTERN = re.compile(r"\bas\s+([A-Za-z_][A-Za-z0-9_]*)", re.IGNORECASE)
DEBUG_TRACE_FILTER = 'level:"debug" OR level:"trace"'
LEVEL_NAMES = {
    "0": "emerg",
    "1": "alert",
    "2": "crit",
    "3": "err",
    "4": "warning",
    "5": "notice",
    "6": "info",
    "7": "debug",
}
GRAYLOG_STREAM_TITLES = (
    "Default Stream",
    "System logs",
    "Audit logs",
    "Kubernetes events",
)
GRAYLOG_AUDIT_SYSTEM_STREAM_TITLES = ("Audit logs", "System logs")
GRAYLOG_SIZE_FIELD = "gl2_accounted_message_size"
GRAYLOG_TOTAL_GROUP_FIELD = "gl2_source_node"


class QueryError(RuntimeError):
    """Raised when an API request cannot be completed."""


def field_name(value: str) -> str:
    if not FIELD_PATTERN.fullmatch(value):
        raise ValueError(f"unsupported field name: {value}")
    if SIMPLE_FIELD_PATTERN.fullmatch(value):
        return value
    return json.dumps(value)


class HttpClient:
    def __init__(
        self,
        base_url: str,
        *,
        user: str = "",
        password: str = "",
        insecure_skip_verify: bool = False,
        extra_headers: dict[str, str] | None = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.headers = dict(extra_headers or {})
        if user or password:
            credentials = base64.b64encode(f"{user}:{password}".encode("utf-8")).decode("ascii")
            self.headers["Authorization"] = f"Basic {credentials}"
        self.context = ssl._create_unverified_context() if insecure_skip_verify else None  # noqa: SLF001

    def call(
        self,
        method: str,
        path: str,
        *,
        query: dict[str, str] | None = None,
        json_body: dict[str, Any] | None = None,
        form_body: dict[str, str] | None = None,
    ) -> Any:
        url = self.base_url + path
        if query:
            url += "?" + parse.urlencode(query)
        headers = dict(self.headers)
        data = None
        if json_body is not None:
            data = json.dumps(json_body).encode("utf-8")
            headers["Content-Type"] = "application/json"
        elif form_body is not None:
            data = parse.urlencode(form_body).encode("utf-8")
            headers["Content-Type"] = "application/x-www-form-urlencoded"
        req = request.Request(url, data=data, headers=headers, method=method)
        try:
            with request.urlopen(req, context=self.context, timeout=60) as response:
                body = response.read().decode("utf-8")
        except error.HTTPError as exc:
            body = exc.read().decode("utf-8", errors="replace")
            raise QueryError(f"{method} {path} returned HTTP {exc.code}: {body}") from exc
        except error.URLError as exc:
            raise QueryError(f"cannot connect to {self.base_url}: {exc.reason}") from exc
        except (TimeoutError, socket.timeout) as exc:
            raise QueryError(f"{method} {path} timed out while connecting to {self.base_url}") from exc
        return body


class VictoriaMetricsClient:
    def __init__(
        self,
        client: HttpClient,
        filesystem_selector: str,
        query_time: str,
        *,
        parallel_queries: bool = True,
        max_parallel_queries: int = 4,
    ) -> None:
        self.client = client
        self.selector = "{" + filesystem_selector + "}" if filesystem_selector else ""
        self.query_time = query_time
        self.parallel_queries = parallel_queries
        self.max_parallel_queries = max_parallel_queries if parallel_queries else 1
        self.query_semaphore = BoundedSemaphore(self.max_parallel_queries)

    def query(self, expression: str) -> list[dict[str, Any]]:
        with self.query_semaphore:
            raw = self.client.call("GET", "/api/v1/query", query={"query": expression, "time": self.query_time})
        response = json.loads(raw)
        if response.get("status") != "success":
            raise QueryError(f"VictoriaMetrics query failed for {expression}: {response}")
        return response.get("data", {}).get("result", [])

    def storage_report(self, backend_type: str, *, dry_run: bool) -> dict[str, Any]:
        expressions: dict[str, str] = {}
        if backend_type == "graylog" and self.selector:
            expressions.update(
                {
                    metric: f"{metric}{self.selector}"
                    for metric in (
                        "node_filesystem_size_bytes",
                        "node_filesystem_free_bytes",
                        "node_filesystem_avail_bytes",
                    )
                }
            )
        if backend_type == "victorialogs":
            expressions["vl_total_disk_space_bytes"] = "vl_total_disk_space_bytes"
            expressions["vl_free_disk_space_bytes"] = "vl_free_disk_space_bytes"
            expressions["vl_uncompressed_data_size_total"] = "sum(vl_uncompressed_data_size_bytes)"
            expressions["vl_compressed_data_size_total"] = "sum(vl_compressed_data_size_bytes)"
            expressions["vl_uncompressed_data_size_by_type"] = "sum(vl_uncompressed_data_size_bytes) by (type)"
            expressions["vl_compressed_data_size_by_type"] = "sum(vl_compressed_data_size_bytes) by (type)"
        if dry_run:
            report: dict[str, Any] = {"query_time": self.query_time, "queries": expressions}
            if backend_type == "graylog" and not self.selector:
                report["filesystem_usage"] = skipped_filesystem_usage()
            return report
        results = self.query_expressions(expressions)
        report = {}
        if backend_type == "graylog" and self.selector:
            report["filesystem_usage"] = calculate_filesystem_usage(results)
        elif backend_type == "graylog":
            report["filesystem_usage"] = skipped_filesystem_usage()
        if backend_type == "victorialogs":
            report["victorialogs_disk_usage"] = calculate_victorialogs_disk_usage(results)
            report["victorialogs_data_size"] = calculate_victorialogs_data_size(results)
        return report

    def query_expressions(self, expressions: dict[str, str]) -> dict[str, Any]:
        if not expressions:
            return {}
        if not self.parallel_queries:
            return {name: self.safe_query(expression) for name, expression in expressions.items()}
        max_workers = min(len(expressions), self.max_parallel_queries)
        results: dict[str, Any] = {}
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = {executor.submit(self.safe_query, expression): name for name, expression in expressions.items()}
            for future in as_completed(futures):
                results[futures[future]] = future.result()
        return results

    def safe_query(self, expression: str) -> Any:
        try:
            return self.query(expression)
        except QueryError as exc:
            return {"error": str(exc)}


def skipped_filesystem_usage() -> dict[str, str]:
    return {
        "status": "skipped",
        "reason": (
            "FILESYSTEM_SELECTOR is not set; node_filesystem* metrics are skipped "
            "to avoid reporting unrelated disks."
        ),
    }


class GraylogIndexStatsClient:
    def __init__(self, client: HttpClient) -> None:
        self.client = client

    @staticmethod
    def index_stats_row(index_set: dict[str, Any], stats_by_id: dict[str, Any]) -> list[Any]:
        stats = stats_by_id.get(index_set.get("id"), {})
        if not isinstance(stats, dict):
            stats = {}
        size_bytes = stats.get("size", 0)
        return [
            index_set.get("title", ""),
            index_set.get("index_prefix", ""),
            stats.get("indices", 0),
            stats.get("documents", 0),
            size_bytes,
        ]

    def global_stats(self) -> list[list[Any]]:
        raw = self.client.call("GET", "/api/system/indices/index_sets/stats")
        response = json.loads(raw)
        return [[response.get("indices", 0), response.get("documents", 0), response.get("size", 0)]]

    def index_sets_stats(self) -> list[list[Any]]:
        raw = self.client.call(
            "GET",
            "/api/system/indices/index_sets",
            query={"skip": "0", "limit": "0", "stats": "true"},
        )
        response = json.loads(raw)
        index_sets = response.get("index_sets", [])
        stats_by_id = response.get("stats", {})
        if not isinstance(stats_by_id, dict):
            stats_by_id = {}
        return [
            self.index_stats_row(index_set, stats_by_id)
            for index_set in index_sets
            if isinstance(index_set, dict)
        ]

    def report(self, *, dry_run: bool) -> dict[str, Any]:
        queries = {
            "global": {
                "method": "GET",
                "path": "/api/system/indices/index_sets/stats",
            },
            "index_sets": {
                "method": "GET",
                "path": "/api/system/indices/index_sets",
                "query": {"skip": "0", "limit": "0", "stats": "true"},
            },
        }
        report: dict[str, Any] = {
            "queries": queries,
            "columns": {
                "global": [
                    "indices",
                    "documents",
                    "size_bytes",
                ],
                "index_sets": [
                    "title",
                    "index_prefix",
                    "indices",
                    "documents",
                    "size_bytes",
                ],
            },
        }
        if dry_run:
            return report
        try:
            report["global"] = self.global_stats()
        except (QueryError, json.JSONDecodeError) as exc:
            report["global"] = {"error": str(exc)}
        try:
            report["index_sets"] = self.index_sets_stats()
        except (QueryError, json.JSONDecodeError) as exc:
            report["index_sets"] = {"error": str(exc)}
        return report


class VictoriaLogsClient:
    def __init__(
        self,
        client: HttpClient,
        time_filter: str,
        source_field: str,
        top_limit: int,
        *,
        parallel_queries: bool = True,
        max_parallel_queries: int = 4,
    ) -> None:
        self.client = client
        self.time_filter = time_filter
        self.source_field = field_name(source_field)
        self.top_limit = top_limit
        self.parallel_queries = parallel_queries
        self.max_parallel_queries = max_parallel_queries if parallel_queries else 1
        self.query_semaphore = BoundedSemaphore(self.max_parallel_queries)

    def query(self, expression: str) -> list[dict[str, Any]]:
        with self.query_semaphore:
            raw = self.client.call("POST", "/select/logsql/query", form_body={"query": expression})
        return [json.loads(line) for line in raw.splitlines() if line.strip()]

    def category_queries(self, log_filter: str, source_fields: list[str]) -> dict[str, str]:
        prefix = f"{self.time_filter} {log_filter}"
        source_group = ", ".join(field_name(field) for field in source_fields)
        return {
            "total": f"{prefix} | stats count() as messages_count, sum_len(_msg) as sum_message_size_bytes",
            "top_by_count": (
                f"{prefix} | stats by ({source_group}) count() as messages_count,"
                " sum_len(_msg) as sum_message_size_bytes"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
        }

    def level_queries(self, level: str) -> dict[str, str]:
        prefix = f'{self.time_filter} NOT kind:KubernetesEvent level:"{level}"'
        return {
            "total": f"{prefix} | stats count() as messages_count, sum_len(_msg) as sum_message_size_bytes",
            "top_by_count": (
                f"{prefix} | stats by (namespace, {self.source_field}) count() as messages_count,"
                " sum_len(_msg) as sum_message_size_bytes"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
        }

    def levels_overview_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} NOT kind:KubernetesEvent"
        return {
            "total_by_level": (
                f"{prefix} | stats by (level) count() as messages_count"
                " | sort by (messages_count desc)"
            ),
            "top_namespaces_by_level": (
                f"{prefix} namespace:* | stats by (level, namespace) count() as messages_count"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
            "top_by_level_and_source": (
                f"{prefix} | stats by (level, namespace, {self.source_field}) count() as messages_count"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
            "top_non_container_by_level_and_node": (
                f"{prefix} NOT namespace:* NOT {self.source_field}:* nodename:*"
                " | stats by (level, nodename) count() as messages_count"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
        }

    def source_activity_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} NOT kind:KubernetesEvent namespace:*"
        return {
            "total": f"{prefix} | stats count() as messages_count, sum_len(_msg) as sum_message_size_bytes",
            "top_namespaces_by_count": (
                f"{prefix} | stats by (namespace) count() as messages_count,"
                " sum_len(_msg) as sum_message_size_bytes"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
            "top_by_count": (
                f"{prefix} {self.source_field}:*"
                f" | stats by (namespace, {self.source_field}) count() as messages_count,"
                " sum_len(_msg) as sum_message_size_bytes"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
        }

    def unattributed_logs_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} NOT namespace:* NOT container:* NOT kind:KubernetesEvent"
        return {
            "total": f"{prefix} | stats count() as messages_count, sum_len(_msg) as sum_message_size_bytes",
            "top_nodes_by_count": (
                f"{prefix} nodename:* | stats by (nodename) count() as messages_count,"
                " sum_len(_msg) as sum_message_size_bytes"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
        }

    def debug_trace_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} NOT kind:KubernetesEvent ({DEBUG_TRACE_FILTER})"
        return {
            "total": f"{prefix} | stats count() as messages_count",
            "top_by_count": (
                f"{prefix} | stats by (namespace, {self.source_field}) count() as messages_count"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
        }

    def message_size_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} NOT kind:KubernetesEvent"
        return {
            "top_by_max_message_size": (
                f"{prefix} namespace:* {self.source_field}:* | len(_msg) as message_size_bytes"
                f" | stats by (namespace, {self.source_field}) max(message_size_bytes) as max_message_size_bytes"
                f" | sort by (max_message_size_bytes desc) | limit {self.top_limit}"
            ),
        }

    def log_patterns_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} NOT kind:KubernetesEvent namespace:* {self.source_field}:*"
        pattern_pipe = " | copy _msg as message_pattern | collapse_nums at message_pattern prettify"
        return {
            "top_patterns_by_count": (
                f"{prefix}{pattern_pipe}"
                f" | stats by (namespace, {self.source_field}, message_pattern) count() as messages_count"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            )
        }

    def k8s_events_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} kind:KubernetesEvent"
        return {
            "total": f"{prefix} | stats count() as messages_count, sum_len(_msg) as sum_message_size_bytes",
            "top_by_count": (
                f"{prefix} | stats by (namespace, involvedObjectKind, involvedObjectName)"
                " count() as messages_count, sum_len(_msg) as sum_message_size_bytes"
                f" | sort by (messages_count desc) | limit {self.top_limit}"
            ),
        }

    def block_stats_queries(self) -> dict[str, str]:
        prefix = self.time_filter
        return {
            "top_streams_by_disk_usage": (
                f"{prefix} | block_stats"
                " | stats by (_stream) sum(values_bytes) as values_bytes,"
                " sum(bloom_bytes) as bloom_bytes"
                " | math (values_bytes+bloom_bytes) as total_bytes"
                f" | first {self.top_limit} (total_bytes desc)"
            ),
            "top_fields_by_disk_usage": (
                f"{prefix} | block_stats"
                " | stats by (field) sum(values_bytes) as values_bytes,"
                " sum(bloom_bytes) as bloom_bytes,"
                " sum(rows) as rows"
                " | math (values_bytes+bloom_bytes) as total_bytes"
                f" | first {self.top_limit} (total_bytes desc)"
            ),
        }

    def schema_quality_queries(self) -> dict[str, str]:
        prefix = f"{self.time_filter} NOT kind:KubernetesEvent parse_field_count:*"
        return {
            "top_by_max_fields": (
                f"{prefix} | stats by (namespace, {self.source_field})"
                " max(parse_field_count) as max_parse_field_count"
                f" | sort by (max_parse_field_count desc) | limit {self.top_limit}"
            ),
        }

    def execute_set(self, queries: dict[str, str], *, dry_run: bool) -> dict[str, Any]:
        report: dict[str, Any] = {
            "queries": queries,
            "columns": {
                name: infer_logsql_columns(expression)
                for name, expression in queries.items()
            },
        }
        if dry_run:
            return report
        report.update(self.query_set(queries))
        return report

    def query_set(self, queries: dict[str, str]) -> dict[str, Any]:
        if not queries:
            return {}
        if not self.parallel_queries:
            return {name: self.safe_query(expression) for name, expression in queries.items()}
        max_workers = min(len(queries), self.max_parallel_queries)
        results: dict[str, Any] = {}
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = {executor.submit(self.safe_query, expression): name for name, expression in queries.items()}
            for future in as_completed(futures):
                results[futures[future]] = future.result()
        return results

    def safe_query(self, expression: str) -> Any:
        try:
            return self.query(expression)
        except QueryError as exc:
            return {"error": str(exc)}


def infer_logsql_columns(expression: str) -> list[str]:
    columns: list[str] = []
    stats_by_match = LOGSQL_STATS_BY_PATTERN.search(expression)
    if stats_by_match:
        columns.extend(
            unquote_logsql_field(field.strip())
            for field in stats_by_match.group(1).split(",")
            if field.strip()
        )
    stats_match = LOGSQL_STATS_PATTERN.search(expression)
    alias_source = expression[stats_match.start() :] if stats_match else expression
    for alias in LOGSQL_ALIAS_PATTERN.findall(alias_source):
        if alias not in columns:
            columns.append(alias)
    return columns


def unquote_logsql_field(field: str) -> str:
    if len(field) >= 2 and field[0] == '"' and field[-1] == '"':
        try:
            return json.loads(field)
        except json.JSONDecodeError:
            return field.strip('"')
    return field


def numeric_metric(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


class GraylogClient:
    def __init__(
        self,
        client: HttpClient,
        timerange: dict[str, Any],
        source_field: str,
        top_limit: int,
        parallel_queries: bool = True,
        max_parallel_queries: int = 4,
    ) -> None:
        self.client = client
        self.timerange = timerange
        self.source_field = source_field
        field_name(source_field)
        self.top_limit = top_limit
        self.parallel_queries = parallel_queries
        self.max_parallel_queries = max_parallel_queries if parallel_queries else 1
        self.query_semaphore = BoundedSemaphore(self.max_parallel_queries)

    def count_size_body(
        self,
        query: str,
        fields: list[str],
        streams: list[str] | None = None,
        *,
        limit: int | None = None,
        sort_by: str = "count",
    ) -> dict[str, Any]:
        count_metric = {"function": "count"}
        sum_metric = {"function": "sum", "field": GRAYLOG_SIZE_FIELD}
        if sort_by == "count":
            count_metric["sort"] = "desc"
        elif sort_by == "size":
            sum_metric["sort"] = "desc"
        body = {
            "query": query,
            "timerange": self.timerange,
            "group_by": [{"field": field, "limit": limit or self.top_limit} for field in fields],
            "metrics": [
                count_metric,
                sum_metric,
            ],
            "_columns": [*fields, "count", "sum_gl2_accounted_message_size"],
            "_result": "rows",
        }
        if streams:
            body["streams"] = streams
        return body

    def query(self, body: dict[str, Any]) -> list[Any]:
        request_body = {key: value for key, value in body.items() if not key.startswith("_")}
        with self.query_semaphore:
            raw = self.client.call("POST", "/api/search/aggregate", json_body=request_body)
        return json.loads(raw).get("datarows", [])

    def streams_by_title(self) -> dict[str, str]:
        with self.query_semaphore:
            raw = self.client.call("GET", "/api/streams")
        streams = json.loads(raw).get("streams", [])
        return {
            stream["title"]: stream["id"]
            for stream in streams
            if isinstance(stream, dict) and stream.get("title") and stream.get("id")
        }

    def count_size_rows(self, body: dict[str, Any]) -> list[Any]:
        return self.query(body)

    def count_size_total_body(self, query: str, streams: list[str] | None = None) -> dict[str, Any]:
        return self.count_size_body(query, [GRAYLOG_TOTAL_GROUP_FIELD], streams, limit=10000)

    def count_size_total(self, body: dict[str, Any]) -> list[Any]:
        total_count = 0
        total_size = 0
        for row in self.count_size_rows(body):
            if not isinstance(row, list) or len(row) < 2:
                continue
            total_count += numeric_metric(row[-2])
            total_size += numeric_metric(row[-1])
        return [total_count, total_size]

    def count_size_rows_by_name(self, bodies: dict[str, dict[str, Any]]) -> dict[str, list[Any]]:
        return self.parallel_query_by_name(bodies, self.count_size_rows)

    def count_size_totals_by_name(self, bodies: dict[str, dict[str, Any]]) -> dict[str, list[Any]]:
        return self.parallel_query_by_name(bodies, self.count_size_total)

    def parallel_query_by_name(
        self,
        bodies: dict[str, dict[str, Any]],
        query_function: Any,
    ) -> dict[str, list[Any]]:
        if not bodies:
            return {}
        if not self.parallel_queries:
            return {name: query_function(body) for name, body in bodies.items()}
        max_workers = min(len(bodies), self.max_parallel_queries)
        results: dict[str, list[Any]] = {}
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            futures = {executor.submit(query_function, body): name for name, body in bodies.items()}
            for future in as_completed(futures):
                results[futures[future]] = future.result()
        return results

    def stream_ids_for_report(self, dry_run: bool) -> dict[str, str]:
        if dry_run:
            return {title: f"<{title} stream id>" for title in GRAYLOG_STREAM_TITLES}
        streams_by_title = self.streams_by_title()
        return {
            title: stream_id
            for title in GRAYLOG_STREAM_TITLES
            if (stream_id := streams_by_title.get(title))
        }

    def sorted_limited_rows(
        self,
        rows: list[Any],
        columns: list[str] | None = None,
        sort_by: str | None = None,
    ) -> list[Any]:
        return sort_metric_rows(rows, columns, sort_by)[: self.top_limit]

    def stream_storage_report(self, *, dry_run: bool) -> dict[str, Any]:
        stream_ids = self.stream_ids_for_report(dry_run)
        queries: dict[str, Any] = {
            "total_by_stream": {
                title: self.count_size_total_body("*", [stream_id])
                for title, stream_id in stream_ids.items()
            },
            "by_stream_and_namespace": {
                title: self.count_size_body("namespace:*", ["namespace"], [stream_id])
                for title, stream_id in stream_ids.items()
            },
            "top_default_stream_sources": self.count_size_body(
                f"namespace:* AND {self.source_field}:*",
                ["namespace", self.source_field],
                [stream_ids["Default Stream"]],
            )
            if "Default Stream" in stream_ids
            else None,
            "audit_system_by_stream_and_nodename": {
                title: self.count_size_body("*", ["nodename"], [stream_id])
                for title, stream_id in stream_ids.items()
                if title in GRAYLOG_AUDIT_SYSTEM_STREAM_TITLES
            },
            "audit_system_by_stream_and_source": {
                title: self.count_size_body(
                    f"namespace:* AND {self.source_field}:*",
                    ["namespace", self.source_field],
                    [stream_id],
                )
                for title, stream_id in stream_ids.items()
                if title in GRAYLOG_AUDIT_SYSTEM_STREAM_TITLES
            },
            "k8s_events_by_object": self.count_size_body(
                "namespace:* AND involvedObjectKind:* AND involvedObjectName:*",
                ["namespace", "involvedObjectKind", "involvedObjectName"],
                [stream_ids["Kubernetes events"]],
            )
            if "Kubernetes events" in stream_ids
            else None,
        }
        report: dict[str, Any] = {
            "queries": queries,
            "columns": {
                "total_by_stream": ["stream", "count", "sum_gl2_accounted_message_size"],
                "by_stream_and_namespace": ["stream", "namespace", "count", "sum_gl2_accounted_message_size"],
                "top_default_stream_sources": [
                    "stream",
                    "namespace",
                    self.source_field,
                    "count",
                    "sum_gl2_accounted_message_size",
                ],
                "audit_system_by_stream_and_nodename": [
                    "stream",
                    "nodename",
                    "count",
                    "sum_gl2_accounted_message_size",
                ],
                "audit_system_by_stream_and_source": [
                    "stream",
                    "namespace",
                    self.source_field,
                    "count",
                    "sum_gl2_accounted_message_size",
                ],
                "k8s_events_by_object": [
                    "stream",
                    "namespace",
                    "involvedObjectKind",
                    "involvedObjectName",
                    "count",
                    "sum_gl2_accounted_message_size",
                ],
            },
        }
        if dry_run:
            return report
        try:
            total_rows = []
            for title, (count, size) in self.count_size_totals_by_name(queries["total_by_stream"]).items():
                total_rows.append([title, count, size])
            report["total_by_stream"] = self.sorted_limited_rows(
                total_rows,
                report["columns"]["total_by_stream"],
                "count",
            )
        except (QueryError, json.JSONDecodeError) as exc:
            report["total_by_stream"] = {"error": str(exc)}

        try:
            namespace_rows = []
            for title, rows in self.count_size_rows_by_name(queries["by_stream_and_namespace"]).items():
                namespace_rows.extend([title, *row] for row in rows)
            report["by_stream_and_namespace"] = self.sorted_limited_rows(
                namespace_rows,
                report["columns"]["by_stream_and_namespace"],
                "count",
            )
        except (QueryError, json.JSONDecodeError) as exc:
            report["by_stream_and_namespace"] = {"error": str(exc)}

        try:
            if queries["top_default_stream_sources"]:
                default_stream_rows = [
                    ["Default Stream", *row] for row in self.count_size_rows(queries["top_default_stream_sources"])
                ]
                report["top_default_stream_sources"] = self.sorted_limited_rows(
                    default_stream_rows,
                    report["columns"]["top_default_stream_sources"],
                    "count",
                )
        except (QueryError, json.JSONDecodeError) as exc:
            report["top_default_stream_sources"] = {"error": str(exc)}

        try:
            nodename_rows = []
            for title, rows in self.count_size_rows_by_name(queries["audit_system_by_stream_and_nodename"]).items():
                nodename_rows.extend([title, *row] for row in rows)
            if nodename_rows:
                report["audit_system_by_stream_and_nodename"] = self.sorted_limited_rows(
                    nodename_rows,
                    report["columns"]["audit_system_by_stream_and_nodename"],
                    "count",
                )
        except (QueryError, json.JSONDecodeError) as exc:
            report["audit_system_by_stream_and_nodename"] = {"error": str(exc)}

        try:
            audit_source_rows = []
            for title, rows in self.count_size_rows_by_name(queries["audit_system_by_stream_and_source"]).items():
                audit_source_rows.extend([title, *row] for row in rows)
            if audit_source_rows:
                report["audit_system_by_stream_and_source"] = self.sorted_limited_rows(
                    audit_source_rows,
                    report["columns"]["audit_system_by_stream_and_source"],
                    "count",
                )
        except (QueryError, json.JSONDecodeError) as exc:
            report["audit_system_by_stream_and_source"] = {"error": str(exc)}

        try:
            if queries["k8s_events_by_object"]:
                k8s_event_rows = [
                    ["Kubernetes events", *row] for row in self.count_size_rows(queries["k8s_events_by_object"])
                ]
                if k8s_event_rows:
                    report["k8s_events_by_object"] = self.sorted_limited_rows(
                        k8s_event_rows,
                        report["columns"]["k8s_events_by_object"],
                        "count",
                    )
        except (QueryError, json.JSONDecodeError) as exc:
            report["k8s_events_by_object"] = {"error": str(exc)}
        return report

    def level_storage_report(self, *, dry_run: bool) -> dict[str, Any]:
        base_query = "NOT kind:KubernetesEvent AND level:*"
        queries = {
            "total_by_level": self.count_size_body(base_query, ["level"]),
            "top_namespaces_by_level": self.count_size_body(f"{base_query} AND namespace:*", ["level", "namespace"]),
            "top_sources_by_level": self.count_size_body(
                f"{base_query} AND namespace:* AND {self.source_field}:*",
                ["level", "namespace", self.source_field],
            ),
            "top_nodes_without_namespace_source_by_level": self.count_size_body(
                f"{base_query} AND NOT namespace:* AND NOT {self.source_field}:* AND nodename:*",
                ["level", "nodename"],
            ),
        }
        report: dict[str, Any] = {
            "queries": queries,
            "columns": {
                "total_by_level": ["level", "count", "sum_gl2_accounted_message_size"],
                "top_namespaces_by_level": ["level", "namespace", "count", "sum_gl2_accounted_message_size"],
                "top_sources_by_level": [
                    "level",
                    "namespace",
                    self.source_field,
                    "count",
                    "sum_gl2_accounted_message_size",
                ],
                "top_nodes_without_namespace_source_by_level": [
                    "level",
                    "nodename",
                    "count",
                    "sum_gl2_accounted_message_size",
                ],
            },
        }
        if dry_run:
            return report
        for name, body in queries.items():
            try:
                rows = self.count_size_rows(body)
                rows, columns = enrich_level_rows(rows, report["columns"][name])
                report["columns"][name] = columns or report["columns"][name]
                report[name] = self.sorted_limited_rows(rows, report["columns"][name], "count")
            except (QueryError, json.JSONDecodeError) as exc:
                report[name] = {"error": str(exc)}
        return report

    def debug_trace_report(self, *, dry_run: bool) -> dict[str, Any]:
        debug_trace_query = "NOT kind:KubernetesEvent AND (level:7 OR level:debug OR level:trace)"
        queries = {
            "total": self.count_size_total_body(debug_trace_query),
            "top_by_count": self.count_size_body(
                f"{debug_trace_query} AND namespace:* AND {self.source_field}:*",
                ["namespace", self.source_field],
            ),
        }
        report: dict[str, Any] = {
            "queries": queries,
            "columns": {
                "total": ["count", "sum_gl2_accounted_message_size"],
                "top_by_count": ["namespace", self.source_field, "count", "sum_gl2_accounted_message_size"],
            },
        }
        if dry_run:
            return report
        try:
            count, size = self.count_size_total(queries["total"])
            report["total"] = [[count, size]]
        except (QueryError, json.JSONDecodeError) as exc:
            report["total"] = {"error": str(exc)}
        try:
            report["top_by_count"] = self.sorted_limited_rows(
                self.count_size_rows(queries["top_by_count"]),
                report["columns"]["top_by_count"],
                "count",
            )
        except (QueryError, json.JSONDecodeError) as exc:
            report["top_by_count"] = {"error": str(exc)}
        return report

    def large_message_report(self, *, dry_run: bool) -> dict[str, Any]:
        query = f"NOT kind:KubernetesEvent AND namespace:* AND {self.source_field}:* AND {GRAYLOG_SIZE_FIELD}:*"
        body = {
            "query": query,
            "timerange": self.timerange,
            "group_by": [
                {"field": "namespace", "limit": self.top_limit},
                {"field": self.source_field, "limit": self.top_limit},
            ],
            "metrics": [
                {"function": "count"},
                {"function": "max", "field": GRAYLOG_SIZE_FIELD, "sort": "desc"},
            ],
            "_columns": ["namespace", self.source_field, "count", "max_gl2_accounted_message_size"],
            "_result": "rows",
        }
        report: dict[str, Any] = {
            "queries": {"top_by_max_message_size": body},
            "columns": {"top_by_max_message_size": body["_columns"]},
        }
        if dry_run:
            return report
        try:
            report["top_by_max_message_size"] = self.sorted_limited_rows(
                self.query(body),
                report["columns"]["top_by_max_message_size"],
                "max_gl2_accounted_message_size",
            )
        except (QueryError, json.JSONDecodeError) as exc:
            report["top_by_max_message_size"] = {"error": str(exc)}
        return report

    def schema_quality_report(self, *, dry_run: bool) -> dict[str, Any]:
        body = {
            "query": f"NOT kind:KubernetesEvent AND parse_field_count:* AND namespace:* AND {self.source_field}:*",
            "timerange": self.timerange,
            "group_by": [
                {"field": "namespace", "limit": self.top_limit},
                {"field": self.source_field, "limit": self.top_limit},
            ],
            "metrics": [
                {"function": "max", "field": "parse_field_count", "sort": "desc"},
            ],
            "_columns": ["namespace", self.source_field, "max_parse_field_count"],
            "_result": "rows",
        }
        report: dict[str, Any] = {
            "queries": {"top_by_max_fields": body},
            "columns": {"top_by_max_fields": body["_columns"]},
        }
        if dry_run:
            return report
        try:
            report["top_by_max_fields"] = self.sorted_limited_rows(
                self.query(body),
                report["columns"]["top_by_max_fields"],
                "max_parse_field_count",
            )
        except (QueryError, json.JSONDecodeError) as exc:
            report["top_by_max_fields"] = {"error": str(exc)}
        return report

    def audit_system_without_namespace_container_report(self, *, dry_run: bool) -> dict[str, Any]:
        stream_ids = self.stream_ids_for_report(dry_run)
        stream_ids = {
            title: stream_id
            for title, stream_id in stream_ids.items()
            if title in GRAYLOG_AUDIT_SYSTEM_STREAM_TITLES
        }
        base_query = "NOT namespace:* AND NOT container:*"
        queries: dict[str, Any] = {
            "total_by_stream": {
                title: self.count_size_total_body(base_query, [stream_id])
                for title, stream_id in stream_ids.items()
            },
            "by_stream_and_nodename": {
                title: self.count_size_body(f"({base_query}) AND nodename:*", ["nodename"], [stream_id])
                for title, stream_id in stream_ids.items()
            },
            "by_stream_and_level": {
                title: self.count_size_body(f"({base_query}) AND level:*", ["level"], [stream_id])
                for title, stream_id in stream_ids.items()
            },
        }
        report: dict[str, Any] = {
            "queries": queries,
            "columns": {
                "total_by_stream": ["stream", "count", "sum_gl2_accounted_message_size"],
                "by_stream_and_nodename": ["stream", "nodename", "count", "sum_gl2_accounted_message_size"],
                "by_stream_and_level": ["stream", "level", "count", "sum_gl2_accounted_message_size"],
            },
        }
        if dry_run:
            return report
        try:
            total_rows = []
            for title, (count, size) in self.count_size_totals_by_name(queries["total_by_stream"]).items():
                if count or size:
                    total_rows.append([title, count, size])
            if total_rows:
                report["total_by_stream"] = self.sorted_limited_rows(
                    total_rows,
                    report["columns"]["total_by_stream"],
                    "count",
                )
        except (QueryError, json.JSONDecodeError) as exc:
            report["total_by_stream"] = {"error": str(exc)}

        try:
            nodename_rows = []
            for title, rows in self.count_size_rows_by_name(queries["by_stream_and_nodename"]).items():
                nodename_rows.extend([title, *row] for row in rows)
            if nodename_rows:
                report["by_stream_and_nodename"] = self.sorted_limited_rows(
                    nodename_rows,
                    report["columns"]["by_stream_and_nodename"],
                    "count",
                )
        except (QueryError, json.JSONDecodeError) as exc:
            report["by_stream_and_nodename"] = {"error": str(exc)}

        try:
            level_rows = []
            for title, rows in self.count_size_rows_by_name(queries["by_stream_and_level"]).items():
                level_rows.extend([title, *row] for row in rows)
            level_rows, columns = enrich_level_rows(level_rows, report["columns"]["by_stream_and_level"])
            report["columns"]["by_stream_and_level"] = columns or report["columns"]["by_stream_and_level"]
            if level_rows:
                report["by_stream_and_level"] = self.sorted_limited_rows(
                    level_rows,
                    report["columns"]["by_stream_and_level"],
                    "count",
                )
        except (QueryError, json.JSONDecodeError) as exc:
            report["by_stream_and_level"] = {"error": str(exc)}
        return report


def enrich_level_rows(rows: list[Any], columns: list[str] | None) -> tuple[list[Any], list[str] | None]:
    if not columns or "level" not in columns:
        return rows, columns
    level_index = columns.index("level")
    enriched_columns = [*columns[: level_index + 1], "level_name", *columns[level_index + 1 :]]
    enriched_rows: list[Any] = []
    for row in rows:
        if not isinstance(row, list) or level_index >= len(row):
            enriched_rows.append(row)
            continue
        level_value = str(row[level_index])
        enriched_rows.append(
            [
                *row[: level_index + 1],
                LEVEL_NAMES.get(level_value, level_value),
                *row[level_index + 1 :],
            ]
        )
    return enriched_rows, enriched_columns


def sort_metric_rows(rows: list[Any], columns: list[str] | None = None, sort_by: str | None = None) -> list[Any]:
    sort_index = columns.index(sort_by) if columns and sort_by in columns else None

    def metric_value(row: Any) -> float:
        if not isinstance(row, list) or not row:
            return float("-inf")
        index = sort_index if sort_index is not None else -1
        try:
            return float(row[index])
        except (TypeError, ValueError):
            return float("-inf")

    return sorted(rows, key=metric_value, reverse=True)
