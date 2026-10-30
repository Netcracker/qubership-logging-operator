import os
import tempfile
import unittest
from pathlib import Path

import tags_exclusion


class TagsExclusionTest(unittest.TestCase):
    def test_excludes_archiving_plugin_without_secret_dir(self):
        self.assertEqual(
            ["archiving-plugin"],
            tags_exclusion.get_excluded_tags({"EXTERNAL_GRAYLOG_SERVER": "true"}),
        )

    def test_excludes_archiving_plugin_for_internal_graylog(self):
        with tempfile.TemporaryDirectory() as secrets_dir:
            Path(secrets_dir, "ssh-key").write_text("key\n", encoding="utf-8")
            Path(secrets_dir, "vm-user").write_text("user\n", encoding="utf-8")

            self.assertEqual(
                ["archiving-plugin"],
                tags_exclusion.get_excluded_tags(
                    {
                        "EXTERNAL_GRAYLOG_SERVER": "false",
                        "INTEGRATION_TESTS_SECRETS_DIR": secrets_dir,
                    }
                ),
            )

    def test_keeps_archiving_plugin_when_required_files_exist(self):
        with tempfile.TemporaryDirectory() as secrets_dir:
            Path(secrets_dir, "ssh-key").write_text("key\n", encoding="utf-8")
            Path(secrets_dir, "vm-user").write_text("user\n", encoding="utf-8")

            self.assertEqual(
                [],
                tags_exclusion.get_excluded_tags(
                    {
                        "EXTERNAL_GRAYLOG_SERVER": "true",
                        "INTEGRATION_TESTS_SECRETS_DIR": secrets_dir,
                    }
                ),
            )

    def test_excludes_archiving_plugin_when_required_file_is_empty(self):
        with tempfile.TemporaryDirectory() as secrets_dir:
            Path(secrets_dir, "ssh-key").write_text("key\n", encoding="utf-8")
            Path(secrets_dir, "vm-user").write_text(os.linesep, encoding="utf-8")

            self.assertEqual(
                ["archiving-plugin"],
                tags_exclusion.get_excluded_tags(
                    {
                        "EXTERNAL_GRAYLOG_SERVER": "true",
                        "INTEGRATION_TESTS_SECRETS_DIR": secrets_dir,
                    }
                ),
            )


if __name__ == "__main__":
    unittest.main()
