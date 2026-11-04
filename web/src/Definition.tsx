import { CheckIcon, LinkSimpleIcon } from '@phosphor-icons/react'
import { useEffect, useRef, useState } from 'react'
import { keyedItems } from './keyedItems'
import type { Entry, PublicPage, Relation } from './types'

export interface DefinitionProps {
  entry: Entry
  sources: PublicPage['sources']
  relationships: Record<string, Relation[]>
  base?: string
  entryURL?: string
  termURL?: string
  termEntries?: Entry[]
  termPage?: boolean
}

const labels: Record<string, string> = {
  abbreviationOf: 'Abbreviation of',
  abbreviatedBy: 'Abbreviated as',
  deprecatedFor: 'Use instead',
  supersedes: 'Formerly',
  relatedTo: 'Related term',
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

export function Definition({
  entry,
  sources,
  relationships,
  base = '',
  entryURL,
  termURL,
  termEntries,
  termPage,
}: DefinitionProps) {
  const isTermPage = !!termPage || !!termEntries
  const sourceName = (id: string) => sources.find((source) => source.id === id)?.name || id
  return (
    <article className="definition">
      <div className="entry-heading">
        <h1>{entry.term}</h1>
        {entryURL && <CopyLink path={entryURL} />}
      </div>
      {!isTermPage && termURL && (
        <p className="source-line">
          <a href={termURL}>All definitions of {entry.term}</a>
        </p>
      )}
      {(termEntries ?? [entry]).map((entry, entryIndex, entries) => {
        const prefix = termEntries ? `${entry.source}/${entry.id}/` : ''
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
                    {keyedItems(view.sense.examples || [], (example) => example).map(({ item: example, key }) => (
                      <blockquote className="usage-example" key={key}>
                        {example}
                      </blockquote>
                    ))}
                    {!!aliases.length && (
                      <p className="alias-line">
                        <span className="muted">{aliases.length === 1 ? 'Alias' : 'Aliases'}: </span>
                        {aliases.map((alias) => alias.text).join(', ')}
                      </p>
                    )}
                    <p className="source-line">
                      <span className="muted">Source: </span>
                      <a href={`${base}/?${new URLSearchParams({ source: view.source })}`}>{sourceName(view.source)}</a>
                    </p>
                    {sources.find((source) => source.id === view.source)?.stale && (
                      <p className="source-note">Showing the last successful version.</p>
                    )}
                    <Relationships items={relationships[`${prefix}sense:${view.sense.id}`]} />
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
            <Relationships items={relationships[`${prefix}entry`]} />
          </div>
        )
      })}
    </article>
  )
}
