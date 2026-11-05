import assert from 'node:assert/strict'
import test from 'node:test'
import remarkLinks, { siteLink } from './remark-links.mjs'

test('documentation and schema links respect the deployment base', () => {
  for (const base of ['/jargonary/', '/']) {
    const prefix = base.replace(/\/$/, '')
    assert.equal(siteLink('reference.md#configuration', base), `${prefix}/reference/#configuration`)
    assert.equal(siteLink('index.md', base), `${prefix}/`)
    assert.equal(siteLink('../schema/v1/glossary.schema.json', base), `${prefix}/schema/v1/glossary.schema.json`)
  }
})

test('repository files and directories point to GitHub', () => {
  assert.equal(
    siteLink('../examples/config.yaml', '/jargonary'),
    'https://github.com/alampros/jargonary/blob/main/examples/config.yaml',
  )
  assert.equal(
    siteLink('../examples/federation/', '/jargonary'),
    'https://github.com/alampros/jargonary/tree/main/examples/federation/',
  )
})

test('external links and local anchors stay intact', () => {
  for (const link of ['https://example.com/', 'http://localhost:8080/', '#configuration', 'mailto:a@example.com']) {
    assert.equal(siteLink(link, '/jargonary'), link)
  }
})

test('rewrites Markdown links and references without touching code examples', () => {
  const tree = {
    type: 'root',
    children: [
      { type: 'link', url: 'reference.md', children: [] },
      { type: 'definition', url: 'development.md' },
      { type: 'code', value: '[reference](reference.md)' },
    ],
  }
  remarkLinks({ base: '/jargonary' })(tree)
  assert.equal(tree.children[0].url, '/jargonary/reference/')
  assert.equal(tree.children[1].url, '/jargonary/development/')
  assert.equal(tree.children[2].value, '[reference](reference.md)')
})
