import { describe, expect, it, vi } from 'vitest'
import { type LocalFileHandle, parseDocument, partsOfSpeech, serializeDocument, writeLocalFile } from './editorDocument'

const glossary = {
  version: '1',
  name: 'Team',
  entries: [
    {
      id: 'api',
      term: 'API',
      aliases: [{ text: 'Interface', sense: 'software' }],
      relationships: [{ type: 'relatedTo', target: { entry: 'other' } }],
      senses: [
        {
          id: 'software',
          definition: '**An interface**',
          links: [{ label: 'Docs', url: 'https://example.com' }],
          tags: ['public'],
        },
      ],
    },
  ],
}

describe('local glossary documents', () => {
  it('round-trips Markdown and fields outside the basic form', () => {
    const parsed = parseDocument(JSON.stringify(glossary))
    parsed.entries[0].term = 'Updated API'
    expect(parseDocument(serializeDocument(parsed))).toEqual({
      ...glossary,
      entries: [{ ...glossary.entries[0], term: 'Updated API' }],
    })
  })
  it.each([
    '{"version":"1","name":"First","name":"Second","entries":[]}',
    '{"version":"1","name":"Team","entries":null}',
    JSON.stringify({ ...glossary, entries: [...glossary.entries, ...glossary.entries] }),
    JSON.stringify({ ...glossary, name: '  ' }),
    JSON.stringify({ ...glossary, unknown: true }),
    JSON.stringify({
      ...glossary,
      entries: [{ ...glossary.entries[0], aliases: [{ text: 'Bad', sense: 'missing' }] }],
    }),
    JSON.stringify({
      ...glossary,
      entries: [{ ...glossary.entries[0], senses: [{ id: 'software', definition: '   ' }] }],
    }),
    JSON.stringify({
      ...glossary,
      entries: [
        {
          ...glossary.entries[0],
          senses: [{ id: 'software', definition: 'Text', links: [{ label: 'Bad', url: 'javascript:alert(1)' }] }],
        },
      ],
    }),
    JSON.stringify({ ...glossary, exported: true }),
  ])('rejects invalid or ambiguous input %s', (input) => {
    expect(() => parseDocument(input)).toThrow()
  })
  it.each(partsOfSpeech)('accepts the schema part of speech %s', (partOfSpeech) => {
    const document = {
      ...glossary,
      entries: [{ ...glossary.entries[0], senses: [{ ...glossary.entries[0].senses[0], partOfSpeech }] }],
    }
    expect(parseDocument(JSON.stringify(document)).entries[0].senses[0].partOfSpeech).toBe(partOfSpeech)
  })
  it.each(['noun phrase', 'Noun', 'unknown', '', null])('rejects part of speech %s in raw JSON', (partOfSpeech) => {
    const document = {
      ...glossary,
      entries: [{ ...glossary.entries[0], senses: [{ ...glossary.entries[0].senses[0], partOfSpeech }] }],
    }
    expect(() => parseDocument(JSON.stringify(document))).toThrow()
  })
  it('allows an intentionally empty glossary and unresolved relationship targets', () => {
    expect(parseDocument('{"version":"1","name":"Team","entries":[]}').entries).toEqual([])
    expect(() => parseDocument(JSON.stringify(glossary))).not.toThrow()
  })
  it('commits to the original handle only after writing completes', async () => {
    const stream = {
      write: vi.fn().mockResolvedValue(undefined),
      close: vi.fn().mockResolvedValue(undefined),
      abort: vi.fn(),
    }
    const handle = { createWritable: vi.fn().mockResolvedValue(stream) } as unknown as LocalFileHandle
    await writeLocalFile(handle, 'updated JSON')
    expect(stream.write).toHaveBeenCalledWith('updated JSON')
    expect(stream.close).toHaveBeenCalledOnce()
    expect(stream.abort).not.toHaveBeenCalled()
  })
  it('aborts a failed write and reports the failure', async () => {
    const stream = {
      write: vi.fn().mockRejectedValue(new Error('Disk full')),
      close: vi.fn(),
      abort: vi.fn().mockResolvedValue(undefined),
    }
    const handle = { createWritable: vi.fn().mockResolvedValue(stream) } as unknown as LocalFileHandle
    await expect(writeLocalFile(handle, 'JSON')).rejects.toThrow('Disk full')
    expect(stream.abort).toHaveBeenCalledOnce()
    expect(stream.close).not.toHaveBeenCalled()
  })
})
