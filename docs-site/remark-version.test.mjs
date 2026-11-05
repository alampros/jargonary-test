import assert from 'node:assert/strict'
import test from 'node:test'
import { createMarkdownProcessor } from '@astrojs/markdown-remark'
import remarkLinks from './remark-links.mjs'
import remarkVersion, { activeVersion } from './remark-version.mjs'

test('manifest versions include the URL prefix and reject missing versions', () => {
  assert.equal(activeVersion({ '.': '0.0.0' }), 'v0.0.0')
  assert.equal(activeVersion({ '.': '1.2.3-rc.1' }), 'v1.2.3-rc.1')
  for (const manifest of [{}, { '.': null }, { '.': 'v1.2.3' }, { '.': '../other' }]) {
    assert.throws(() => activeVersion(manifest), /Release manifest/)
  }
})

test('Markdown prose, links, inline code, and code fences use the active version', async () => {
  const markdown = [
    '# Release __JARGONARY_VERSION__',
    '',
    '[Schema for __JARGONARY_VERSION__](../docs-site/public/schema/__JARGONARY_VERSION__/config.schema.json)',
    '',
    '`go tool task schema VERSION=__JARGONARY_VERSION__`',
    '',
    '```yaml',
    '# yaml-language-server: $schema=https://alampros.github.io/jargonary/schema/__JARGONARY_VERSION__/glossary.schema.json',
    'version: "1"',
    '```',
  ].join('\n')
  for (const version of ['v0.0.0', 'v1.2.3', 'v2.0.0-rc.1']) {
    const processor = await createMarkdownProcessor({
      remarkPlugins: [
        [remarkVersion, { version }],
        [remarkLinks, { base: '/jargonary' }],
      ],
      syntaxHighlight: false,
      smartypants: false,
    })
    const { code } = await processor.render(markdown)
    assert.ok(!code.includes('__JARGONARY_VERSION__'))
    assert.ok(code.includes(`Release ${version}`))
    assert.ok(code.includes(`href="/jargonary/schema/${version}/config.schema.json"`))
    assert.ok(code.includes(`go tool task schema VERSION=${version}`))
    assert.ok(code.includes(`/schema/${version}/glossary.schema.json`))
    assert.ok(code.includes('version: "1"'))
  }
})
