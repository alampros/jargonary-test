"""Check local links and fragments in the assembled GitHub Pages artifact."""

from html.parser import HTMLParser
import os
from pathlib import Path
from urllib.parse import unquote, urljoin, urlsplit

ROOT = Path(__file__).resolve().parents[1]


class Page(HTMLParser):
    def __init__(self, content):
        super().__init__(convert_charrefs=True)
        self.ids = set()
        self.links = []
        self.feed(content)

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if "id" in attrs:
            self.ids.add(attrs["id"])
        if tag == "a" and attrs.get("href"):
            self.links.append(attrs["href"])


def check(output, base_url):
    base = urlsplit(base_url)
    prefix = base.path.rstrip("/") + "/"
    pages = {p: Page(p.read_text()) for p in output.rglob("*.html")}
    errors = []
    if "starlight__search" not in pages[output / "index.html"].ids:
        errors.append("Search UI is missing: build with NODE_ENV=production")
    if not (output / "pagefind/pagefind.js").is_file():
        errors.append("Pagefind search index is missing")
    for file, page in pages.items():
        relative = file.relative_to(output).as_posix()
        page_url = base_url.rstrip("/") + "/" + relative.removesuffix("index.html")
        for href in page.links:
            target = urlsplit(urljoin(page_url, href))
            if target.netloc != base.netloc or target.scheme not in ("http", "https"):
                continue
            pathname = unquote(target.path)
            if not pathname.startswith(prefix):
                errors.append(f"{relative}: link escapes deployment base: {href}")
                continue
            destination = output / pathname[len(prefix):]
            if destination.is_dir():
                destination /= "index.html"
            if not destination.is_file():
                errors.append(f"{relative}: missing target: {href}")
            elif target.fragment and destination in pages:
                if unquote(target.fragment) not in pages[destination].ids:
                    errors.append(f"{relative}: missing fragment: {href}")
    if errors:
        raise ValueError("\n".join(errors))
    print(f"Checked local links and fragments in {len(pages)} HTML pages")


if __name__ == "__main__":
    check(
        ROOT / ".cache/pages",
        os.environ.get("PAGES_BASE_URL", "https://alampros.github.io/jargonary"),
    )
