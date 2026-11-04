export interface Alias {
  text: string
  sense?: string
}
export interface Sense {
  id: string
  definition: string
  partOfSpeech?: string
  context?: string
  examples?: string[]
  deprecated?: boolean
  deprecationReason?: string
  links?: { label: string; url: string }[]
}
export interface SenseView {
  source: string
  entryId: string
  term: string
  pronunciation?: string
  sense: Sense
  deprecated: boolean
}
export interface Entry {
  source: string
  id: string
  term: string
  pronunciation?: string
  aliases?: Alias[]
  deprecated: boolean
  deprecationReason?: string
  senses: SenseView[]
}
export interface Relation {
  type: string
  title: string
  href: string
}
export interface PublicPage {
  mode: 'public'
  siteName: string
  base: string
  query: string
  source: string
  letter: string
  sources: { id: string; name: string; stale: boolean; lastChecked?: string }[]
  entries: {
    source: string
    id: string
    term: string
    pronunciation?: string
    partsOfSpeech?: string[]
    contexts: string[]
    definitions: string[]
    deprecated: boolean
    href: string
  }[]
  entry?: Entry
  entryURL?: string
  relationships: Record<string, Relation[]>
  notFound: boolean
  next?: string
  previous?: string
}
export interface Scrape {
  source: string
  running: boolean
  lastAttempt: string
  lastSuccess: string
  lastChange: string
  nextAttempt: string
  failures: number
  entries: number
  recent: { at: string; result: string; detail?: string; durationMs: number }[]
}
export interface AdminPage {
  mode: 'admin'
  siteName: string
  config: Record<string, unknown>
  scrapes: Scrape[]
  diagnostics: { source: string; entry: string; sense?: string; kind: string }[]
}
