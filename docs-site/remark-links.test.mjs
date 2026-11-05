import assert from 'node:assert/strict'
import test from 'node:test'
import remarkLinks from './remark-links.mjs'

test('public images resolve under the configured site base', () => {
  for (const base of ['/', '/jargonary']) {
    const image = {
      type: 'image',
      url: '../docs-site/public/distributed-glossary.svg',
    }
    const tree = { type: 'root', children: [image] }

    remarkLinks({ base })(tree)

    assert.equal(image.url, `${base.replace(/\/$/, '')}/distributed-glossary.svg`)
  }
})
