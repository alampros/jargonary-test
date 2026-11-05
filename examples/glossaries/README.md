# Local glossary editor fixtures

Open the public UI’s **Edit a local glossary** page (`/editor`), then open or drop one of these JSON or YAML files:

| File | What to try |
| --- | --- |
| `minimal.json` | Rename the single term, edit its definition, and save back to the file. |
| `rich.json` | Multiple meanings, Markdown headings/lists/quotes/links, examples, tags, aliases, relationships, deprecation, Unicode, and long text. Expand entry/usage details and switch between Form and JSON. |
| `many-entries.json` | 48 entries for searching, no-match results, selection, sidebar scrolling, and JSON scrolling. |
| `rich.yaml` | Native YAML editing and saving, comments, quoted values, multiline Markdown, and shared aliases. |
| `empty.json` | The empty-glossary state and adding the first entry. |

All are valid format v1 glossaries. Markdown stays in the existing `definition` field; the public glossary displays it as authored text.

These fixtures sit outside `examples/terms`, so the example server configuration does not ingest them automatically. Editor operations stay local. **Save** changes the selected fixture on disk; **Download copy** leaves the original unchanged.

To validate a fixture from the repository root:

```sh
go run ./cmd/jargonary -validate examples/glossaries/rich.json -format json
```
