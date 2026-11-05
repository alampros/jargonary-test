"""Build Starlight documentation and publish versioned schemas in one Pages artifact."""

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]
SCHEMA_PATH = "schema/glossary.schema.json"


def prepare_docs():
    """Keep docs/ canonical; generate Starlight metadata without duplicating prose."""
    content = ROOT / "docs-site/.content"
    shutil.rmtree(content, ignore_errors=True)
    content.mkdir(parents=True)
    for source in sorted((ROOT / "docs").glob("*.md")):
        heading, body = source.read_text().split("\n", 1)
        if not heading.startswith("# "):
            raise ValueError(f"{source} must start with a level-one title")
        metadata = (
            "---\n"
            f"title: {json.dumps(heading[2:])}\n"
            f"editUrl: https://github.com/alampros/jargonary/edit/main/docs/{source.name}\n"
            "---\n"
        )
        (content / source.name).write_text(metadata + body)
    (content / "404.md").write_text(
        '---\ntitle: Page not found\neditUrl: false\npagefind: false\n---\n'
        'This page could not be found. [Return to the overview](index.md) '
        'or use search to find a guide.\n'
    )


def build(output, base_url, releases):
    # Always assemble the entire site: Pages replaces the previous deployment.
    shutil.rmtree(output, ignore_errors=True)
    subprocess.run(
        ["npm", "run", "build", "--", "--outDir", str(output)],
        cwd=ROOT / "docs-site",
        env=dict(os.environ, PAGES_BASE_URL=base_url, ASTRO_TELEMETRY_DISABLED="1"),
        check=True,
    )
    schemas = output / "schema"
    schemas.mkdir(parents=True, exist_ok=True)

    def publish(schema, directory):
        destination = schemas / directory / "glossary.schema.json"
        destination.parent.mkdir(parents=True, exist_ok=True)
        relative_url = destination.relative_to(output).as_posix()
        document = dict(schema, **{"$id": f"{base_url.rstrip('/')}/{relative_url}"})
        destination.write_text(json.dumps(document, indent=2) + "\n")

    def format_version(schema):
        root = schema
        if "$ref" in schema:
            root = schema["$defs"][schema["$ref"].removeprefix("#/$defs/")]
        versions = root["properties"]["version"]["enum"]
        if len(versions) != 1 or not re.fullmatch(r"[0-9]+", versions[0]):
            raise ValueError("Expected a single numeric dictionary format version")
        return versions[0]

    versions = []
    for release in releases:
        tag = release["tag_name"]
        match = re.fullmatch(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", tag)
        if not release["draft"] and not release["prerelease"] and match:
            versions.append((tuple(map(int, match.groups())), tag))

    latest = json.loads((ROOT / SCHEMA_PATH).read_text())
    if not versions:
        publish(latest, f"v{format_version(latest)}")
    for _, tag in sorted(versions):
        # Read each release's committed schema, never the current schema under an old tag.
        latest = json.loads(subprocess.check_output(
            ["git", "show", f"refs/tags/{tag}:{SCHEMA_PATH}"], cwd=ROOT, text=True
        ))
        publish(latest, tag)
        publish(latest, f"v{format_version(latest)}")
    publish(latest, "")
    (output / ".nojekyll").touch()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="https://alampros.github.io/jargonary")
    parser.add_argument("--releases", type=Path)
    parser.add_argument("--prepare-only", action="store_true")
    args = parser.parse_args()
    if args.prepare_only:
        prepare_docs()
        return
    output = ROOT / ".cache/pages"
    releases = json.loads(args.releases.read_text()) if args.releases else []
    build(output, args.base_url, releases)


if __name__ == "__main__":
    main()
