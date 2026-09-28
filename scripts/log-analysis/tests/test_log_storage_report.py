from __future__ import annotations

import argparse
import sys
import unittest
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPT_DIR))

import clients  # noqa: E402
import log_storage_report as report  # noqa: E402


class ArgumentParsingTest(unittest.TestCase):
    def test_duration_seconds_accepts_valid_duration(self) -> None:
        self.assertEqual(report.duration_seconds("30m"), 1800)
        self.assertEqual(report.duration_seconds("1h"), 3600)
        self.assertEqual(report.duration_seconds("0s", allow_zero=True), 0)

    def test_duration_seconds_rejects_zero_by_default(self) -> None:
        with self.assertRaises(argparse.ArgumentTypeError):
            report.duration_seconds("0s")

    def test_positive_size_kb_accepts_units(self) -> None:
        self.assertEqual(report.positive_size_kb("512"), 512)
        self.assertEqual(report.positive_size_kb("1MB"), 1024)
        self.assertEqual(report.positive_size_kb("1GB"), 1024 * 1024)
        self.assertEqual(report.positive_size_kb("1B"), 1)

    def test_positive_size_kb_rejects_zero(self) -> None:
        with self.assertRaises(argparse.ArgumentTypeError):
            report.positive_size_kb("0")


class FieldValidationTest(unittest.TestCase):
    def test_victorialogs_field_name_quotes_complex_supported_fields(self) -> None:
        self.assertEqual(clients.field_name("container"), "container")
        self.assertEqual(clients.field_name("user.username"), '"user.username"')

    def test_graylog_field_name_rejects_complex_fields(self) -> None:
        self.assertEqual(clients.graylog_field_name("container"), "container")
        with self.assertRaises(ValueError):
            clients.graylog_field_name("user.username")


class ReportTransformTest(unittest.TestCase):
    def test_convert_byte_fields_to_kb_renames_and_converts_columns(self) -> None:
        source = {
            "columns": {"rows": ["namespace", "sum_gl2_accounted_message_size"]},
            "rows": [["app", 2048]],
        }

        converted = report.convert_byte_fields_to_kb(source)

        self.assertEqual(converted["columns"]["rows"], ["namespace", "sum_gl2_accounted_message_size_kb"])
        self.assertEqual(converted["rows"], [["app", 2]])

    def test_detected_too_many_fields_uses_schema_quality_section(self) -> None:
        source = {
            "logs": {
                "schema_quality": {
                    "columns": {"top_by_max_fields": ["namespace", "container", "max_parse_field_count"]},
                    "top_by_max_fields": [["app", "service-a", 25]],
                }
            }
        }

        problem = report.detected_too_many_fields(source, 20)

        self.assertIsNotNone(problem)
        self.assertEqual(problem["problem"], "Too many parsed fields")
        self.assertEqual(problem["evidence"][0]["source"], "service-a")

    def test_detected_too_many_fields_uses_the_per_record_section(self) -> None:
        source = {
            "backend_type": "victorialogs",
            "logs": {
                "schema_quality": {
                    "columns": {"top_by_fields_per_record": ["namespace", "container", "avg_parsed_fields"]},
                    "top_by_fields_per_record": [["app", "service-a", 42.5]],
                }
            },
        }

        problem = report.detected_too_many_fields(source, 20)

        self.assertIsNotNone(problem)
        self.assertEqual(problem["problem"], "Too many parsed fields")
        self.assertEqual(problem["evidence"][0]["avg_parsed_fields"], 42)
        self.assertIn("payload fields per record", problem["description"])

    def test_detected_too_many_fields_ignores_counts_within_threshold(self) -> None:
        source = {
            "logs": {
                "schema_quality": {
                    "columns": {"top_by_fields_per_record": ["namespace", "container", "avg_parsed_fields"]},
                    "top_by_fields_per_record": [["app", "service-a", 20]],
                }
            }
        }

        self.assertIsNone(report.detected_too_many_fields(source, 20))


class PayloadFieldStatsTest(unittest.TestCase):
    @staticmethod
    def _client(min_field_share_percent: float = 0.0) -> clients.VictoriaLogsClient:
        return clients.VictoriaLogsClient(
            clients.HttpClient("http://logs.invalid"),
            "_time:1d",
            "container",
            10,
            min_field_share_percent=min_field_share_percent,
        )

    def test_composite_source_names_survive_the_result_lookup(self) -> None:
        # LogsQL needs app.name quoted, while the query result keys it unquoted.
        client = clients.VictoriaLogsClient(
            clients.HttpClient("http://logs.invalid"), "_time:1d", "app.name", 10
        )

        self.assertEqual(client.source_field, '"app.name"')
        self.assertEqual(client.source_field_key, "app.name")
        self.assertIn('"app.name":="billing"', client.schema_quality_field_names_query("ns", "billing"))

    def test_field_names_query_keeps_the_hit_counts(self) -> None:
        # payload_field_stats divides the hits by the record count, so a projection that
        # keeps only the name would make every average zero.
        query = self._client().schema_quality_field_names_query("ns", "svc")

        self.assertIn("| field_names", query)
        self.assertNotIn("| fields ", query)

    def test_stats_are_zero_when_the_query_drops_the_hits(self) -> None:
        rows = [{"name": "logger"}, {"name": "request_id"}]

        distinct, per_record = self._client().payload_field_stats(rows, 1000)

        self.assertEqual(distinct, 2)
        self.assertEqual(per_record, 0.0)

    def test_skips_pipeline_fields(self) -> None:
        rows = [
            {"name": "_time", "hits": "10"},
            {"name": "namespace", "hits": "10"},
            {"name": "labels.app", "hits": "10"},
            {"name": "parse_status", "hits": "10"},
            {"name": "logger", "hits": "10"},
            {"name": "request_id", "hits": "5"},
        ]

        distinct, per_record = self._client().payload_field_stats(rows, 10)

        self.assertEqual(distinct, 2)
        self.assertEqual(per_record, 1.5)

    def test_rare_names_count_toward_distinct_but_barely_toward_the_average(self) -> None:
        rows = [{"name": "banner_art", "hits": "1"}]

        distinct, per_record = self._client().payload_field_stats(rows, 1000)

        self.assertEqual(distinct, 1)
        self.assertEqual(per_record, 0.0)

    def test_min_share_drops_rare_names(self) -> None:
        rows = [{"name": "logger", "hits": "900"}, {"name": "banner_art", "hits": "1"}]

        distinct, per_record = self._client(min_field_share_percent=5).payload_field_stats(rows, 1000)

        self.assertEqual(distinct, 1)
        self.assertEqual(per_record, 0.9)

    def test_average_is_zero_without_a_record_count(self) -> None:
        rows = [{"name": "logger", "hits": "900"}]

        distinct, per_record = self._client().payload_field_stats(rows, 0)

        self.assertEqual(distinct, 1)
        self.assertEqual(per_record, 0.0)


if __name__ == "__main__":
    unittest.main()
