import { ArrowClockwiseIcon, CaretDownIcon } from '@phosphor-icons/react'
import { useEffect, useRef, useState } from 'react'
import { keyedItems } from './keyedItems'
import type { AdminPage, Scrape } from './types'

const tabs = ['sources', 'configuration', 'diagnostics'] as const
type Tab = (typeof tabs)[number]
function timestamp(value?: string) {
  if (!value || value.startsWith('0001-')) return 'Not yet'
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'medium' }).format(new Date(value))
}
function state(scrape?: Scrape) {
  if (scrape?.running) return 'Running'
  if (scrape?.failures) return 'Retrying'
  if (!scrape || scrape.lastSuccess.startsWith('0001-')) return 'Waiting'
  return 'Healthy'
}

export default function AdminApp({ data }: { data: AdminPage }) {
  useEffect(() => {
    document.documentElement.dataset.jargonaryReady = 'admin'
    window.dispatchEvent(new CustomEvent('jargonary:ready', { detail: { mode: 'admin' } }))
  }, [])
  const [tab, setTab] = useState<Tab>(() =>
    tabs.includes(location.hash.slice(1) as Tab) ? (location.hash.slice(1) as Tab) : 'sources',
  )
  const [scrapes, setScrapes] = useState(data.scrapes)
  const [diagnostics, setDiagnostics] = useState(data.diagnostics)
  const [error, setError] = useState(false)
  const tablist = useRef<HTMLDivElement>(null)
  const configured = (data.config.sources || []) as {
    id: string
    url?: string
    file?: string
    format: string
    scrape_interval: string
  }[]
  useEffect(() => {
    const abort = new AbortController()
    let busy = false
    async function refresh() {
      if (busy) return
      busy = true
      try {
        const [scrapesResponse, diagnosticsResponse] = await Promise.all([
          fetch('/api/v1/scrapes', { signal: abort.signal }),
          fetch('/api/v1/diagnostics', { signal: abort.signal }),
        ])
        if (!scrapesResponse.ok || !diagnosticsResponse.ok) throw new Error('unavailable')
        setScrapes(await scrapesResponse.json())
        setDiagnostics(await diagnosticsResponse.json())
        setError(false)
      } catch {
        if (!abort.signal.aborted) setError(true)
      } finally {
        busy = false
      }
    }
    const timer = setInterval(refresh, 15000)
    return () => {
      clearInterval(timer)
      abort.abort()
    }
  }, [])
  function select(value: Tab) {
    setTab(value)
    history.replaceState(null, '', `#${value}`)
  }
  return (
    <>
      <a className="skip-link" href="#operator-content">
        Skip to operator content
      </a>
      <header className="masthead">
        <a className="wordmark" href="/">
          {data.siteName || 'Jargonary'}
        </a>
        <span className="masthead-divider" />
        <span className="masthead-label">Operator console</span>
        <span className="read-only">Read-only</span>
      </header>
      <main className="operator-main" id="operator-content" tabIndex={-1}>
        <div className="operator-heading">
          <div>
            <p className="eyebrow">Behind the dictionary</p>
            <h1>Source &amp; system health</h1>
          </div>
          <button type="button" className="text-button" onClick={() => location.reload()}>
            <ArrowClockwiseIcon size={22} aria-hidden />
            Refresh
          </button>
        </div>
        <div
          className="operator-tabs"
          role="tablist"
          aria-label="Operator views"
          ref={tablist}
          onKeyDown={(event) => {
            const current = tabs.indexOf(tab)
            let next = current
            if (event.key === 'ArrowRight') next = (current + 1) % tabs.length
            else if (event.key === 'ArrowLeft') next = (current + tabs.length - 1) % tabs.length
            else if (event.key === 'Home') next = 0
            else if (event.key === 'End') next = tabs.length - 1
            else return
            event.preventDefault()
            select(tabs[next])
            tablist.current?.querySelectorAll<HTMLButtonElement>('button')[next].focus()
          }}
        >
          {tabs.map((value) => (
            <button
              type="button"
              key={value}
              id={`tab-${value}`}
              role="tab"
              aria-selected={tab === value}
              tabIndex={tab === value ? 0 : -1}
              aria-controls={`panel-${value}`}
              onClick={() => select(value)}
            >
              {value === 'sources' ? 'Sources' : value === 'configuration' ? 'Resolved configuration' : 'Diagnostics'}
              {value === 'diagnostics' && diagnostics.length > 0 ? ` (${diagnostics.length})` : ''}
            </button>
          ))}
        </div>
        {error && (
          <p className="refresh-error" role="status">
            Could not refresh. Showing the last loaded diagnostics.
          </p>
        )}
        <section className="operator-panel" role="tabpanel" id={`panel-${tab}`} aria-labelledby={`tab-${tab}`}>
          {tab === 'sources' && (
            <>
              <p className="muted">
                {configured.length} configured {configured.length === 1 ? 'source' : 'sources'}. Status refreshes every
                15 seconds. Failed scrapes keep the last successful dictionary.
              </p>
              {!configured.length && (
                <div className="empty-state">
                  <h2>No company sources configured</h2>
                  <p>The bundled project glossary is available separately.</p>
                </div>
              )}
              {configured.map((source) => {
                const scrape = scrapes.find((value) => value.source === source.id)
                const label = state(scrape)
                return (
                  <details className="source-detail" key={source.id}>
                    <summary>
                      <div>
                        <h2>{source.id}</h2>
                        <p className="source-location">{source.url || source.file}</p>
                      </div>
                      <span className="source-summary-state">
                        <span className={`source-state state-${label.toLowerCase()}`}>{label}</span>
                        <CaretDownIcon className="source-chevron" size={20} aria-hidden />
                      </span>
                    </summary>
                    <dl className="source-facts">
                      <div>
                        <dt>Format</dt>
                        <dd>{source.format.toUpperCase()}</dd>
                      </div>
                      <div>
                        <dt>Scrape interval</dt>
                        <dd>{source.scrape_interval}</dd>
                      </div>
                      <div>
                        <dt>Accepted entries</dt>
                        <dd>{scrape?.entries ?? 0}</dd>
                      </div>
                      <div>
                        <dt>Consecutive failures</dt>
                        <dd>{scrape?.failures ?? 0}</dd>
                      </div>
                      <div>
                        <dt>Last attempt</dt>
                        <dd>{timestamp(scrape?.lastAttempt)}</dd>
                      </div>
                      <div>
                        <dt>Last successful check</dt>
                        <dd>{timestamp(scrape?.lastSuccess)}</dd>
                      </div>
                      <div>
                        <dt>Last content change</dt>
                        <dd>{timestamp(scrape?.lastChange)}</dd>
                      </div>
                      <div>
                        <dt>Next attempt</dt>
                        <dd>{timestamp(scrape?.nextAttempt)}</dd>
                      </div>
                    </dl>
                    <h3>Recent scrapes</h3>
                    {!scrape?.recent?.length ? (
                      <p className="muted">No completed attempts yet.</p>
                    ) : (
                      <ol className="scrape-history">
                        {[...scrape.recent].reverse().map((outcome) => (
                          <li key={outcome.at}>
                            <div>
                              <time dateTime={outcome.at}>{timestamp(outcome.at)}</time>
                              <span className={`outcome outcome-${outcome.result}`}>{outcome.result}</span>
                              <span className="muted">{outcome.durationMs} ms</span>
                            </div>
                            {outcome.detail && <p>{outcome.detail}</p>}
                          </li>
                        ))}
                      </ol>
                    )}
                  </details>
                )
              })}
            </>
          )}
          {tab === 'configuration' && (
            <>
              <h2>Resolved configuration</h2>
              <p className="muted">
                Settings resolve at startup. Secrets are represented by references; credential values and URL query
                strings are omitted. Unlisted setting origins use built-in defaults.
              </p>
              <textarea
                className="configuration-json"
                aria-label="Resolved configuration JSON"
                rows={24}
                readOnly
                value={JSON.stringify(data.config, null, 2)}
              />
            </>
          )}
          {tab === 'diagnostics' && (
            <>
              <h2>Vocabulary diagnostics</h2>
              {!diagnostics.length ? (
                <p>No unresolved relationships or replacement cycles.</p>
              ) : (
                <ul className="diagnostics-list">
                  {keyedItems(diagnostics, (item) =>
                    JSON.stringify([item.source, item.entry, item.sense, item.kind]),
                  ).map(({ item, key }) => (
                    <li key={key}>
                      <strong>{item.kind}</strong>
                      <span>
                        {item.source} / {item.entry}
                        {item.sense ? ` / ${item.sense}` : ''}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
        </section>
      </main>
    </>
  )
}
