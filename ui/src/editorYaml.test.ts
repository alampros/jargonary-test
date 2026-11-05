import { describe, expect, it } from 'vitest'
import { glossaryFormat, parseDocument, serializeDocument } from './editorDocument'
import { editorYamlFixture } from './test/editorYamlFixture'

describe('native YAML glossary editing', () => {
  it.each(['team.yaml', 'team.yml', 'TEAM.YAML'])('recognizes %s as YAML', (name) => {
    expect(glossaryFormat(name)).toBe('yaml')
  })
  it('rejects unsupported file types', () => {
    expect(() => glossaryFormat('team.csv')).toThrow('JSON or YAML')
  })
  it('reads YAML and keeps untouched source byte-for-byte', () => {
    const document = parseDocument(editorYamlFixture, 'yaml')
    expect(document.entries[0].senses[0].definition).toContain('**stable**')
    expect(serializeDocument(document, 'yaml', editorYamlFixture)).toBe(editorYamlFixture)
  })
  it('writes YAML while preserving comments, quotes, block scalars, and unchanged aliases', () => {
    const document = parseDocument(editorYamlFixture, 'yaml')
    document.entries[0].term = 'Public API'
    document.entries[0].senses[0].definition = 'A **revised** interface.\n\nDescribe requests clearly.'
    const saved = serializeDocument(document, 'yaml', editorYamlFixture)
    expect(saved).toContain('# Owned by the platform team')
    expect(saved).toContain('version: "1" # Keep the format version quoted')
    expect(saved).toContain("term: 'Public API' # Preferred name")
    expect(saved).toContain('definition: |-')
    expect(saved).toContain('*shared_context')
    expect(parseDocument(saved, 'yaml')).toEqual(document)
    expect(() => JSON.parse(saved)).toThrow()
  })
  it('keeps independent meanings when an anchored value changes', () => {
    const document = parseDocument(editorYamlFixture, 'yaml')
    document.entries[0].senses[0].context = 'Platform engineering'
    const saved = serializeDocument(document, 'yaml', editorYamlFixture)
    expect(parseDocument(saved, 'yaml')).toEqual(document)
    expect(parseDocument(saved, 'yaml').entries[0].senses[1].context).toBe('Software development')
  })
  it('can remove the meaning that originally owned a shared anchor', () => {
    const document = parseDocument(editorYamlFixture, 'yaml')
    document.entries[0].senses.splice(0, 1)
    document.entries[0].aliases = []
    const saved = serializeDocument(document, 'yaml', editorYamlFixture)
    expect(parseDocument(saved, 'yaml')).toEqual(document)
    expect(saved).not.toContain('*shared_context')
  })
  it('preserves the correct entry comments when deleting another entry', () => {
    const source = `version: "1"\nname: Team\nentries:\n  - id: one # First term\n    term: One\n    senses: [{id: meaning, definition: First.}]\n  - id: two # Second term\n    term: Two\n    senses: [{id: meaning, definition: Second.}]\n`
    const document = parseDocument(source, 'yaml')
    document.entries.shift()
    const saved = serializeDocument(document, 'yaml', source)
    expect(saved).toContain('# Second term')
    expect(saved).not.toContain('# First term')
    expect(parseDocument(saved, 'yaml')).toEqual(document)
  })
  it('retains a raw YAML comment edit through a later form edit', () => {
    const raw = editorYamlFixture.replace('# Preferred name', '# Reviewed locally')
    const document = parseDocument(raw, 'yaml')
    document.entries[0].term = 'Reviewed API'
    expect(serializeDocument(document, 'yaml', raw)).toContain("term: 'Reviewed API' # Reviewed locally")
  })
  it('writes an empty glossary and newly added values as YAML', () => {
    const document = { version: '1' as const, name: 'New glossary', entries: [] }
    const saved = serializeDocument(document, 'yaml')
    expect(saved).toContain('entries: []')
    expect(parseDocument(saved, 'yaml')).toEqual(document)
  })
  it.each([
    'version: "1"\nname: Team\nname: Other\nentries: []',
    'version: "1"\nname: Team\nentries: []\n---\nname: Other',
    'version: "1"\nname: Team\nentries: [',
    'version: "1"\nname: Team\nentries: []\nunknown: true',
    'version: 1\nname: Team\nentries: []',
    'version: "1"\nname: Team\nentries: *missing',
    'version: "1"\nname: !custom Team\nentries: []',
  ])('rejects invalid YAML without dropping data: %s', (source) => {
    expect(() => parseDocument(source, 'yaml')).toThrow()
  })
})
