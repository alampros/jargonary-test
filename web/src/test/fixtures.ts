import type { AdminPage, PublicPage } from '../types'

export function dictionaryPage(overrides: Partial<PublicPage> = {}): PublicPage {
  return {
    mode: 'public',
    siteName: 'Acme',
    base: '',
    query: 'component',
    source: '',
    letter: 'C',
    sources: [{ id: 'engineering', name: 'Engineering glossary', stale: false, letters: ['C', 'D', '#'] }],
    entries: [],
    notFound: false,
    entryURL: '/definitions/engineering/component',
    entry: {
      source: 'engineering',
      id: 'component',
      term: 'Component',
      pronunciation: 'kuhm-POH-nuhnt',
      deprecated: false,
      aliases: [{ text: 'Asset component', sense: 'asset' }],
      senses: [
        {
          source: 'engineering',
          entryId: 'component',
          term: 'Component',
          deprecated: false,
          sense: {
            id: 'asset',
            partOfSpeech: 'noun',
            context: 'Asset management',
            definition: 'A constituent part of an asset system.',
          },
        },
        {
          source: 'engineering',
          entryId: 'component',
          term: 'Component',
          deprecated: true,
          sense: {
            id: 'web',
            context: 'Frontend engineering',
            definition: 'A reusable web interface element.',
            deprecationReason: 'Use the new interface vocabulary.',
          },
        },
      ],
    },
    relationships: {
      'sense:web': [
        {
          type: 'deprecatedFor',
          title: 'Interface element',
          href: '/definitions/frontend/interface-element',
        },
      ],
    },
    ...overrides,
  }
}

export function operatorPage(): AdminPage {
  return {
    mode: 'admin',
    siteName: 'Acme',
    config: {
      sources: [
        {
          id: 'engineering',
          file: '/terms/engineering.yaml',
          format: 'yaml',
          scrape_interval: '5m',
        },
      ],
      exports: [{ id: 'public', auth: { shared_secret: { env: 'PUBLIC_EXPORT_SECRET' } } }],
    },
    scrapes: [
      {
        source: 'engineering',
        running: false,
        failures: 0,
        entries: 2,
        lastAttempt: '2026-11-03T12:00:00Z',
        lastSuccess: '2026-11-03T12:00:00Z',
        lastChange: '2026-11-03T11:00:00Z',
        nextAttempt: '2026-11-03T12:05:00Z',
        recent: [{ at: '2026-11-03T12:00:00Z', result: 'unchanged', durationMs: 3 }],
      },
    ],
    diagnostics: [{ source: 'engineering', entry: 'component', sense: 'web', kind: 'unresolved relationship' }],
  }
}
