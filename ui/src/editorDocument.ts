import validate from 'virtual:glossary-validator'

export { partsOfSpeech } from 'virtual:glossary-validator'

import { jsonLanguage } from '@codemirror/lang-json'
import { readYaml, writeYaml } from './editorYaml'

export type GlossaryFormat = 'json' | 'yaml'

export function glossaryFormat(fileName: string): GlossaryFormat {
  if (/\.json$/i.test(fileName)) return 'json'
  if (/\.ya?ml$/i.test(fileName)) return 'yaml'
  throw new Error('Open a JSON or YAML glossary file (.json, .yaml, or .yml).')
}

export interface Relationship {
  type: 'abbreviationOf' | 'relatedTo' | 'deprecatedFor'
  target: { entry: string; source?: string; sense?: string }
}
interface Deprecation {
  deprecated?: boolean
  deprecationReason?: string
  relationships?: Relationship[] | null
}
export interface EditorSense extends Deprecation {
  id: string
  definition: string
  context?: string
  partOfSpeech?: string
  examples?: string[] | null
  tags?: string[] | null
  links?: { label: string; url: string }[] | null
}
export interface EditorEntry extends Deprecation {
  id: string
  term: string
  pronunciation?: string
  aliases?: { text: string; sense?: string }[] | null
  origin?: { source?: string; entry: string; sense?: string }
  senses: EditorSense[]
}
export interface EditorDocument {
  version: '1'
  name: string
  exported?: boolean
  entries: EditorEntry[]
}
const validID = (value: string) => /^[A-Za-z0-9][A-Za-z0-9._-]{0,199}$/.test(value)

export function parseDocument(raw: string, format: GlossaryFormat = 'json'): EditorDocument {
  const value: unknown = format === 'yaml' ? readYaml(raw).toJS({ maxAliasCount: 100 }) : JSON.parse(raw)
  const keys = new Map<number, Set<string>>()
  if (format === 'json')
    jsonLanguage.parser.parse(raw).iterate({
      enter(node) {
        if (node.name !== 'PropertyName') return
        const object = node.node.parent?.parent?.from ?? 0
        const name = JSON.parse(raw.slice(node.from, node.to)) as string
        const seen = keys.get(object) ?? new Set<string>()
        if (seen.has(name)) throw new Error(`Duplicate JSON field: ${name}`)
        seen.add(name)
        keys.set(object, seen)
      },
    })
  if (!validate(value)) {
    throw new Error(validate.errors?.map((error) => `${error.instancePath || '/'} ${error.message}`).join('; '))
  }
  if (!value.name.trim()) throw new Error('Glossary name must not be blank.')
  const entries = new Set<string>()
  for (const entry of value.entries) {
    if (entries.has(entry.id) || !entry.term.trim()) throw new Error('Entries need unique IDs and nonblank terms.')
    entries.add(entry.id)
    if (value.exported) {
      if (!entry.origin || !validID(entry.origin.source ?? '') || !validID(entry.origin.entry) || entry.origin.sense) {
        throw new Error(`Exported entry ${entry.id} requires valid origin source and entry IDs, without a sense.`)
      }
    } else if (entry.origin) throw new Error('Origin metadata is reserved for exported glossaries.')
    const senses = new Set<string>()
    for (const sense of entry.senses) {
      if (senses.has(sense.id) || !sense.definition.trim())
        throw new Error(`${entry.id}: senses need unique IDs and nonblank definitions.`)
      senses.add(sense.id)
      for (const link of sense.links ?? []) {
        const url = new URL(link.url)
        if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password)
          throw new Error(`${entry.id}: links must use HTTP(S) without credentials.`)
      }
    }
    for (const alias of entry.aliases ?? []) {
      if (!alias.text.trim() || (alias.sense && !senses.has(alias.sense)))
        throw new Error(`${entry.id}: aliases need text and an existing sense ID.`)
    }
    for (const owner of [entry, ...entry.senses]) {
      for (const relation of owner.relationships ?? []) {
        if (
          (relation.target.source && !validID(relation.target.source)) ||
          (relation.target.sense && !validID(relation.target.sense))
        )
          throw new Error(`${entry.id}: relationship target IDs are invalid.`)
      }
    }
  }
  return value
}

export function serializeDocument(document: EditorDocument, format: GlossaryFormat = 'json', source?: string) {
  if (format === 'yaml') return writeYaml(document, source)
  return `${JSON.stringify(document, null, 2)}\n`
}

// Keep browser file access separate from the glossary server APIs.
export interface LocalFileHandle {
  name: string
  getFile(): Promise<File>
  createWritable(): Promise<{ write(data: string): Promise<void>; close(): Promise<void>; abort(): Promise<void> }>
}
export type FilePickerWindow = Window & {
  showOpenFilePicker?: (options: {
    multiple: boolean
    types: { description: string; accept: Record<string, string[]> }[]
  }) => Promise<LocalFileHandle[]>
}
export async function writeLocalFile(handle: LocalFileHandle, contents: string) {
  const stream = await handle.createWritable()
  try {
    await stream.write(contents)
    await stream.close()
  } catch (error) {
    await stream.abort().catch(() => {})
    throw error
  }
}
