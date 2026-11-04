import { CaretDownIcon, CheckIcon, LinkSimpleIcon, MagnifyingGlassIcon } from '@phosphor-icons/react'
import type { KeyboardEvent } from 'react'
import { useEffect, useId, useRef, useState } from 'react'
import { keyedItems } from './keyedItems'
import type { PublicPage, Relation } from './types'

const letters = ['All', ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', '0-9', '#']
const labels: Record<string, string> = {
  abbreviationOf: 'Abbreviation of',
  abbreviatedBy: 'Abbreviated as',
  deprecatedFor: 'Use instead',
  supersedes: 'Formerly',
  relatedTo: 'Related term',
}

function queryURL(data: PublicPage, values: Record<string, string>) {
  const params = new URLSearchParams()
  for (const [key, value] of Object.entries(values)) if (value) params.set(key, value)
  return `${data.base}/${params.size ? `?${params.toString()}` : ''}`
}

function Alphabet({ data }: { data: PublicPage }) {
  const rail = useRef<HTMLElement>(null)
  const current = data.letter || 'All'
  useEffect(() => {
    const nav = rail.current
    const active = [...(nav?.querySelectorAll<HTMLAnchorElement>('a') || [])].find(
      (link) => link.dataset.letter === current,
    )
    if (!nav || !active) return
    const reveal = () => {
      if (nav.scrollWidth <= nav.clientWidth) return
      // Scroll the strip, never the document, so selecting Z doesn't move the page sideways.
      nav.scrollLeft = Math.max(
        0,
        nav.scrollLeft +
          active.getBoundingClientRect().left -
          nav.getBoundingClientRect().left -
          nav.clientWidth / 2 +
          active.clientWidth / 2,
      )
    }
    reveal()
    const observer = new ResizeObserver(reveal)
    observer.observe(nav)
    return () => observer.disconnect()
  }, [current])
  function onKeyDown(event: KeyboardEvent<HTMLElement>) {
    const anchors = [...(rail.current?.querySelectorAll<HTMLAnchorElement>('a') || [])]
    const index = anchors.indexOf(event.target as HTMLAnchorElement)
    if (index < 0) return
    let next = index
    if (['ArrowRight', 'ArrowDown'].includes(event.key)) next = (index + 1) % anchors.length
    else if (['ArrowLeft', 'ArrowUp'].includes(event.key)) next = (index - 1 + anchors.length) % anchors.length
    else if (event.key === 'Home') next = 0
    else if (event.key === 'End') next = anchors.length - 1
    else return
    event.preventDefault()
    anchors[next].focus({ preventScroll: true })
    anchors[next].scrollIntoView({ block: 'nearest', inline: 'nearest', behavior: 'auto' })
  }
  return (
    <nav className="alphabet" aria-label="Browse by first letter" ref={rail} onKeyDown={onKeyDown}>
      <span className="sr-only">Use arrow keys to move between letters, then Enter to browse.</span>
      {letters.map((letter) => (
        <a
          key={letter}
          data-letter={letter}
          className="alphabet-link"
          href={queryURL(data, { source: data.source, letter: letter === 'All' ? '' : letter })}
          aria-label={
            letter === 'All'
              ? 'Browse all terms'
              : letter === '#'
                ? 'Terms beginning with other characters'
                : `Terms beginning with ${letter}`
          }
          aria-current={current === letter ? 'page' : undefined}
          tabIndex={current === letter ? 0 : -1}
        >
          {letter === '0-9' ? '0–9' : letter}
        </a>
      ))}
    </nav>
  )
}

function CopyLink({ path }: { path: string }) {
  const [state, setState] = useState<'ready' | 'copied' | 'manual'>('ready')
  const input = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (state === 'manual') {
      input.current?.focus()
      input.current?.select()
    }
  }, [state])
  useEffect(() => {
    if (state !== 'copied') return
    const timer = setTimeout(() => setState('ready'), 2500)
    return () => clearTimeout(timer)
  }, [state])
  const href = new URL(path, window.location.href).href
  async function copy() {
    try {
      await navigator.clipboard.writeText(href)
      setState('copied')
    } catch {
      setState('manual')
    }
  }
  return (
    <div className="copy-control">
      <button className="text-button copy-button" type="button" onClick={copy}>
        {state === 'copied' ? <CheckIcon size={22} aria-hidden /> : <LinkSimpleIcon size={23} aria-hidden />}
        {state === 'copied' ? 'Copied' : 'Copy link'}
      </button>
      <span className="sr-only" role="status">
        {state === 'copied'
          ? 'Definition link copied to clipboard.'
          : state === 'manual'
            ? 'Select and copy the definition link below.'
            : ''}
      </span>
      {state === 'manual' && (
        <input ref={input} className="copy-fallback" aria-label="Definition link" value={href} readOnly />
      )}
    </div>
  )
}

function Relationships({ items }: { items?: Relation[] }) {
  if (!items?.length) return null
  const groups = new Map<string, Relation[]>()
  for (const item of items) groups.set(item.type, [...(groups.get(item.type) || []), item])
  return (
    <div className="relationships">
      {[...groups].map(([type, links]) => (
        <p key={type}>
          <span className="muted">{labels[type] || type}: </span>
          {links.map((link, index) => (
            <span key={link.href}>
              {index > 0 ? ', ' : ''}
              <a href={link.href}>{link.title}</a>
            </span>
          ))}
        </p>
      ))}
    </div>
  )
}

export function PublicApp({ data }: { data: PublicPage }) {
  useEffect(() => {
    document.documentElement.dataset.jargonaryReady = 'public'
    window.dispatchEvent(new CustomEvent('jargonary:ready', { detail: { mode: 'public' } }))
  }, [])
  const isTermPage = !!data.termPage || !!data.termEntries
  const sourceFormId = useId()
  const form = useRef<HTMLFormElement>(null)
  const search = useRef<HTMLInputElement>(null)
  const [submitting, setSubmitting] = useState(false)
  const sourceName = (id: string) => data.sources.find((source) => source.id === id)?.name || id
  const termGroups = new Map<string, PublicPage['entries']>()
  for (const entry of data.entries) {
    const key = entry.termKey ?? entry.term.trim().normalize('NFC').toLowerCase()
    const group = termGroups.get(key) ?? []
    group.push(entry)
    termGroups.set(key, group)
  }
  useEffect(() => {
    function shortcut(event: globalThis.KeyboardEvent) {
      const target = event.target as HTMLElement
      if (
        event.key === '/' &&
        !event.metaKey &&
        !event.ctrlKey &&
        !event.altKey &&
        !['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) &&
        !target.isContentEditable
      ) {
        event.preventDefault()
        search.current?.focus()
      }
    }
    const restored = () => setSubmitting(false)
    document.addEventListener('keydown', shortcut)
    window.addEventListener('pageshow', restored)
    return () => {
      document.removeEventListener('keydown', shortcut)
      window.removeEventListener('pageshow', restored)
    }
  }, [])
  return (
    <>
      <a className="skip-link" href="#main-content">
        Skip to definitions
      </a>
      <header className="masthead">
        <a className="wordmark" href={`${data.base}/`}>
          {data.siteName || 'Jargonary'}
        </a>
        <span className="masthead-divider" />
        <span className="masthead-label">Shared glossary</span>
      </header>
      <div className="dictionary-layout">
        <search className="search-container">
          {isTermPage && data.termURL && (
            <form id={sourceFormId} action={data.termURL.split('?')[0]} method="get" hidden />
          )}
          <form
            className="search-tools"
            action={`${data.base}/`}
            method="get"
            aria-label="Dictionary search"
            ref={form}
            onSubmit={() => setSubmitting(true)}
          >
            <div className="search-field">
              <MagnifyingGlassIcon size={26} aria-hidden />
              <input
                ref={search}
                name="q"
                type="search"
                aria-label="Search terms"
                placeholder="Search terms and abbreviations"
                maxLength={256}
                defaultValue={data.query}
              />
              <button type="submit" disabled={submitting}>
                {submitting ? 'Searching…' : 'Search'}
              </button>
            </div>
            <div className="source-filter">
              <select
                name="source"
                aria-label="Dictionary source"
                defaultValue={data.source}
                form={isTermPage ? sourceFormId : undefined}
                onChange={(event) => event.currentTarget.form?.requestSubmit()}
              >
                <option value="">All sources</option>
                {data.sources.map((source) => (
                  <option key={source.id} value={source.id}>
                    {source.name}
                  </option>
                ))}
              </select>
              <CaretDownIcon size={18} aria-hidden />
            </div>
            {isTermPage && <input type="hidden" name="source" value={data.source} />}
            {!data.query && data.letter && <input type="hidden" name="letter" value={data.letter} />}
            <span role="status" className="sr-only">
              {submitting ? 'Loading dictionary results.' : ''}
            </span>
          </form>
        </search>
        <Alphabet data={data} />
        <main id="main-content" className="dictionary-content" tabIndex={-1}>
          {data.notFound ? (
            <div className="empty-state">
              <h1>Definition not found</h1>
              <p>This definition may have been removed or its source may no longer be available.</p>
              <a href={`${data.base}/`}>Browse the glossary</a>
            </div>
          ) : data.entry ? (
            <article className="definition">
              <div className="entry-heading">
                <h1>{data.entry.term}</h1>
                {data.entryURL && <CopyLink path={data.entryURL} />}
              </div>
              {!isTermPage && data.termURL && (
                <p className="source-line">
                  <a href={data.termURL}>All definitions of {data.entry.term}</a>
                </p>
              )}
              {(data.termEntries ?? [data.entry]).map((entry, entryIndex, entries) => {
                const prefix = data.termEntries ? `${entry.source}/${entry.id}/` : ''
                const offset = entries.slice(0, entryIndex).reduce((count, item) => count + item.senses.length, 0)
                return (
                  <div key={`${entry.source}/${entry.id}`}>
                    {entry.pronunciation && <p className="pronunciation">{entry.pronunciation}</p>}
                    {!!entry.aliases?.some((alias) => !alias.sense) && (
                      <p className="entry-aliases">
                        <span className="muted">Also known as: </span>
                        {entry.aliases
                          .filter((alias) => !alias.sense)
                          .map((alias) => alias.text)
                          .join(', ')}
                      </p>
                    )}
                    {entry.senses.map((view, index) => {
                      const aliases = entry.aliases?.filter((alias) => alias.sense === view.sense.id) || []
                      return (
                        <section
                          className="sense"
                          key={`${view.source}/${view.entryId}/${view.sense.id}`}
                          id={`meaning-${entry.source}/${entry.id}/${view.sense.id}`}
                        >
                          <h2>
                            <span className="sense-number">{offset + index + 1}.</span>
                            <span>{view.sense.context || 'Definition'}</span>
                          </h2>
                          <div className="sense-body">
                            {view.deprecated && (
                              <div className="deprecation">
                                <strong>Deprecated</strong>
                                <span>
                                  {view.sense.deprecationReason ||
                                    entry.deprecationReason ||
                                    'This term is no longer recommended.'}
                                </span>
                              </div>
                            )}
                            <p className="definition-text">
                              {view.sense.partOfSpeech && <em className="part-of-speech">{view.sense.partOfSpeech}</em>}
                              {view.sense.definition}
                            </p>
                            {keyedItems(view.sense.examples || [], (example) => example).map(
                              ({ item: example, key }) => (
                                <blockquote className="usage-example" key={key}>
                                  {example}
                                </blockquote>
                              ),
                            )}
                            {!!aliases.length && (
                              <p className="alias-line">
                                <span className="muted">{aliases.length === 1 ? 'Alias' : 'Aliases'}: </span>
                                {aliases.map((alias) => alias.text).join(', ')}
                              </p>
                            )}
                            <p className="source-line">
                              <span className="muted">Source: </span>
                              <a href={queryURL(data, { source: view.source })}>{sourceName(view.source)}</a>
                            </p>
                            {data.sources.find((source) => source.id === view.source)?.stale && (
                              <p className="source-note">Showing the last successful version.</p>
                            )}
                            <Relationships items={data.relationships[`${prefix}sense:${view.sense.id}`]} />
                            {!!view.sense.links?.length && (
                              <p className="references">
                                <span className="muted">Learn more: </span>
                                {view.sense.links.map((link, i) => (
                                  <span key={link.url}>
                                    {i > 0 ? ', ' : ''}
                                    <a href={link.url}>{link.label || link.url}</a>
                                  </span>
                                ))}
                              </p>
                            )}
                          </div>
                        </section>
                      )
                    })}
                    <Relationships items={data.relationships[`${prefix}entry`]} />
                  </div>
                )
              })}
            </article>
          ) : (
            <section className="browse-section">
              <div className="browse-heading">
                <h1>
                  {data.query
                    ? 'Search results'
                    : data.letter
                      ? data.letter === '0-9'
                        ? 'Numbers'
                        : data.letter === '#'
                          ? 'Other characters'
                          : data.letter
                      : 'Shared vocabulary'}
                </h1>
                {data.query && <p className="muted">For “{data.query}”</p>}
              </div>
              {!data.entries.length ? (
                <div className="empty-state">
                  <h2>{data.query || data.letter ? 'No matching terms' : 'A place for your shared vocabulary'}</h2>
                  <p>
                    {data.query
                      ? 'Try another term, an abbreviation, or a different source.'
                      : data.letter
                        ? 'There are no terms under this letter in the selected sources.'
                        : 'When your team adds its first terms, you’ll find them here.'}
                  </p>
                  {(data.query || data.letter) && (
                    <a href={queryURL(data, { source: data.source })}>Browse all terms</a>
                  )}
                </div>
              ) : (
                <div className="term-list">
                  {[...termGroups].map(([key, entries]) => {
                    const showSources = new Set(entries.map((entry) => entry.source)).size > 1
                    return (
                      <article className="term-row" key={key}>
                        <div className="term-row-heading">
                          <h2>
                            <a
                              href={
                                entries[0].termHref ??
                                `${data.base}/definitions/${encodeURIComponent(entries[0].term)}${data.source ? `?source=${encodeURIComponent(data.source)}` : ''}`
                              }
                            >
                              {entries[0].term}
                            </a>
                          </h2>
                        </div>
                        {entries.map((entry) => (
                          <div className="term-source-entry" key={`${entry.source}/${entry.id}`}>
                            {entry.pronunciation && <p className="pronunciation">{entry.pronunciation}</p>}
                            <div className="term-definitions">
                              {keyedItems(entry.definitions, (definition) => definition).map(
                                ({ item: definition, key }, index) => (
                                  <div className="term-meaning" key={key}>
                                    <p className="meaning-attribution">
                                      {showSources && <a href={entry.href}>{sourceName(entry.source)}</a>}
                                      {entry.contexts[index] && (
                                        <span>
                                          {showSources ? ' · ' : ''}
                                          {entry.contexts[index]}
                                        </span>
                                      )}
                                      {entry.deprecated && <span className="deprecated-label">Deprecated</span>}
                                    </p>
                                    <p>
                                      {entry.partsOfSpeech?.[index] && (
                                        <em className="part-of-speech">{entry.partsOfSpeech[index]}</em>
                                      )}
                                      {definition}
                                    </p>
                                  </div>
                                ),
                              )}
                            </div>
                          </div>
                        ))}
                      </article>
                    )
                  })}
                </div>
              )}
              {(data.previous || data.next) && (
                <nav aria-label="Result pages" className="pagination">
                  {data.previous && <a href={data.previous}>Previous</a>}
                  {data.next && <a href={data.next}>Next</a>}
                </nav>
              )}
            </section>
          )}
        </main>
      </div>
    </>
  )
}
