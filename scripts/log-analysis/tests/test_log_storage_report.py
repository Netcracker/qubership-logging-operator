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


if __name__ == "__main__":
    unittest.main()
