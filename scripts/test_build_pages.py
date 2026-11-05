"""Regression checks for canonical docs and schema release selection."""

import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("build_pages", Path(__file__).with_name("build-pages.py"))
pages = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pages)


def schema(version, title):
    return {"properties": {"version": {"enum": [version]}}, "title": title}


class PagesTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "schema").mkdir()
        (self.root / pages.SCHEMA_PATH).write_text(json.dumps(schema("2", "current")))
        self.output = self.root / "output"
        self.patch = patch.object(pages, "ROOT", self.root)
        self.patch.start()
        self.addCleanup(self.patch.stop)

    def fake_build(self, *args, **kwargs):
        self.output.mkdir()
        (self.output / "index.html").write_text("Built docs")

    def read_schema(self, folder=""):
        return json.loads((self.output / "schema" / folder / "glossary.schema.json").read_text())

    @patch.object(pages.subprocess, "run")
    def test_first_release_uses_current_schema_and_keeps_built_docs(self, run):
        run.side_effect = self.fake_build
        pages.build(self.output, "https://example.com/docs/", [])
        self.assertEqual(self.read_schema("v2")["title"], "current")
        self.assertEqual(self.read_schema()["$id"], "https://example.com/docs/schema/glossary.schema.json")
        self.assertTrue((self.output / "index.html").is_file())
        self.assertTrue((self.output / ".nojekyll").is_file())
        self.assertEqual(run.call_args.kwargs["env"]["PAGES_BASE_URL"], "https://example.com/docs/")

    @patch.object(pages.subprocess, "check_output")
    @patch.object(pages.subprocess, "run")
    def test_stable_releases_use_tagged_schemas_in_semver_order(self, run, git):
        run.side_effect = self.fake_build
        git.side_effect = lambda command, **kwargs: json.dumps(schema("1", command[-1].split(":")[0]))
        releases = [
            {"tag_name": tag, "draft": draft, "prerelease": prerelease}
            for tag, draft, prerelease in [
                ("v1.10.0", False, False), ("v1.2.0", False, False),
                ("v9.0.0", True, False), ("v2.0.0-rc.1", False, True),
            ]
        ]
        pages.build(self.output, "https://example.com", releases)
        self.assertEqual(self.read_schema("v1.2.0")["title"], "refs/tags/v1.2.0")
        self.assertEqual(self.read_schema("v1")["title"], "refs/tags/v1.10.0")
        self.assertEqual(self.read_schema()["title"], "refs/tags/v1.10.0")
        self.assertEqual(git.call_count, 2)
        self.assertFalse((self.output / "schema/v2").exists())

    def test_docs_are_generated_with_titles_and_canonical_edit_links(self):
        (self.root / "docs").mkdir()
        (self.root / "docs/guide.md").write_text("# A guide\n\nSome prose.\n")
        pages.prepare_docs()
        generated = (self.root / "docs-site/.content/guide.md").read_text()
        self.assertIn('title: "A guide"', generated)
        self.assertIn("/edit/main/docs/guide.md", generated)
        self.assertIn("Some prose.", generated)
        self.assertNotIn("# A guide", generated)
