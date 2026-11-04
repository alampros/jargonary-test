import type { KeyboardEvent } from 'react'
import { useEffect, useRef } from 'react'

const letters = ['All', ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', '0-9', '#']

export interface AlphabetProps {
  base?: string
  source?: string
  letter?: string
  sources: { id: string; letters: string[] }[]
}

export function Alphabet({ base = '', source = '', letter = '', sources }: AlphabetProps) {
  const rail = useRef<HTMLElement>(null)
  const current = letter || 'All'
  const enabled = new Set([
    'All',
    ...sources.filter((item) => !source || item.id === source).flatMap((source) => source.letters),
  ])
  const tabStop = enabled.has(current) ? current : 'All'
  useEffect(() => {
    const nav = rail.current
    const active = [...(nav?.querySelectorAll<HTMLAnchorElement>('a[href]') || [])].find(
      (link) => link.dataset.letter === tabStop,
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
  }, [tabStop])
  function onKeyDown(event: KeyboardEvent<HTMLElement>) {
    const anchors = [...(rail.current?.querySelectorAll<HTMLAnchorElement>('a[href]') || [])]
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
      {letters.map((letter) => {
        const props = {
          'data-letter': letter,
          className: 'alphabet-link',
          'aria-label':
            letter === 'All'
              ? 'Browse all terms'
              : letter === '#'
                ? 'Terms beginning with other characters'
                : `Terms beginning with ${letter}`,
          'aria-current': current === letter ? ('page' as const) : undefined,
        }
        const label = letter === '0-9' ? '0–9' : letter
        const params = new URLSearchParams()
        if (source) params.set('source', source)
        if (letter !== 'All') params.set('letter', letter)
        return enabled.has(letter) ? (
          <a
            key={letter}
            {...props}
            href={`${base}/${params.size ? `?${params.toString()}` : ''}`}
            tabIndex={tabStop === letter ? 0 : -1}
          >
            {label}
          </a>
        ) : (
          <button key={letter} {...props} type="button" disabled aria-disabled="true">
            {label}
          </button>
        )
      })}
    </nav>
  )
}
