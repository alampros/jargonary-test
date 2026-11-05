import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { PublicApp } from './PublicApp'
import { dictionaryPage } from './test/fixtures'

function sectionFor(name: RegExp) {
  const section = screen.getByRole('heading', { name }).closest('section')
  if (!section) throw new Error('Meaning must have its own semantic section')
  return within(section)
}

describe('dictionary', () => {
  it('hides the source for a single-source term while keeping its contexts', () => {
    render(
      <PublicApp
        data={dictionaryPage({
          entry: undefined,
          query: '',
          sources: [
            { id: 'engineering', name: 'Engineering glossary', stale: false, letters: ['C', 'D', '#'] },
            { id: 'web', name: 'Web Development', stale: false, letters: ['C', 'D', '#'] },
          ],
          entries: [
            {
              id: 'component',
              source: 'engineering',
              term: 'Component',
              href: '/definitions/engineering/component',
              contexts: ['Asset management', 'Frontend engineering'],
              pronunciation: 'kuhm-POH-nuhnt',
              partsOfSpeech: ['noun', 'noun'],
              definitions: ['An asset part.', 'A web element.'],
              deprecated: false,
            },
          ],
        })}
      />,
    )
    expect(screen.getByText('An asset part.')).toBeVisible()
    expect(screen.getByText('A web element.')).toBeVisible()
    expect(screen.getByText('kuhm-POH-nuhnt')).toBeVisible()
    expect(screen.getAllByText('noun')).toHaveLength(2)
    const row = within(screen.getByRole('article'))
    expect(row.queryByRole('link', { name: 'Engineering glossary' })).not.toBeInTheDocument()
    expect(row.getByText('Asset management')).toBeVisible()
    expect(row.getByText('Frontend engineering')).toBeVisible()
  })

  it.each(['', 'API'])('groups terms across sources for browse/search (%s) without losing links or labels', (query) => {
    render(
      <PublicApp
        data={dictionaryPage({
          entry: undefined,
          query,
          sources: [
            { id: 'engineering', name: 'Engineering', stale: false, letters: ['C', 'D', '#'] },
            { id: 'web', name: 'Web Development', stale: false, letters: ['C', 'D', '#'] },
          ],
          entries: [
            {
              id: 'api',
              source: 'engineering',
              term: 'API',
              termKey: 'api',
              href: '/definitions/engineering/api',
              contexts: ['Petroleum engineering'],
              definitions: ['An industry standards organization.'],
              deprecated: true,
            },
            {
              id: 'component',
              source: 'web',
              term: 'Component',
              termKey: 'component',
              href: '/definitions/web/component',
              contexts: ['Frontend'],
              definitions: ['A reusable UI element.'],
              deprecated: false,
            },
            {
              id: 'api',
              source: 'web',
              term: 'api',
              termKey: 'api',
              href: '/definitions/web/api',
              contexts: ['Web development'],
              definitions: ['A software interface.'],
              deprecated: false,
            },
          ],
        })}
      />,
    )
    expect(screen.getAllByRole('heading', { name: 'API', level: 2 })).toHaveLength(1)
    expect(screen.getAllByRole('article')).toHaveLength(2)
    const component = within(screen.getByRole('heading', { name: 'Component', level: 2 }).closest('article')!)
    expect(component.queryByRole('link', { name: 'Web Development' })).not.toBeInTheDocument()
    const api = within(screen.getByRole('heading', { name: 'API', level: 2 }).closest('article')!)
    expect(api.getByRole('link', { name: 'API' })).toHaveAttribute('href', '/definitions/API')
    const engineering = api.getByRole('link', { name: 'Engineering' })
    const web = api.getByRole('link', { name: 'Web Development' })
    expect(engineering).toHaveAttribute('href', '/definitions/engineering/api')
    expect(web).toHaveAttribute('href', '/definitions/web/api')
    const engineeringMeaning = within(engineering.closest('.term-meaning')!)
    const webMeaning = within(web.closest('.term-meaning')!)
    expect(engineeringMeaning.getByText('An industry standards organization.')).toBeVisible()
    expect(engineeringMeaning.getByText('· Petroleum engineering')).toBeVisible()
    expect(engineeringMeaning.getByText('Deprecated')).toBeVisible()
    expect(webMeaning.getByText('A software interface.')).toBeVisible()
    expect(webMeaning.getByText('· Web development')).toBeVisible()
    expect(webMeaning.queryByText('Deprecated')).not.toBeInTheDocument()
  })

  it('shows every source on a term page while keeping aliases and relationships scoped to each entry', () => {
    const data = dictionaryPage()
    const first = data.entry!
    const second = {
      ...first,
      source: 'other',
      id: 'component',
      aliases: [{ text: 'Other alias', sense: 'asset' }],
      senses: [
        {
          ...first.senses[0],
          source: 'other',
          entryId: 'component',
          sense: { id: 'asset', context: 'Other team', definition: 'Another meaning.' },
        },
      ],
    }
    render(
      <PublicApp
        data={dictionaryPage({
          termEntries: [first, second],
          termURL: '/definitions/Component',
          entryURL: '/definitions/Component',
          relationships: {
            'other/component/sense:asset': [
              { type: 'relatedTo', title: 'Other relationship', href: '/definitions/other/related' },
            ],
          },
        })}
      />,
    )
    const original = sectionFor(/Asset management/)
    const other = sectionFor(/Other team/)
    expect(original.getByText('Asset component')).toBeVisible()
    expect(original.queryByText('Other alias')).not.toBeInTheDocument()
    expect(other.getByText('Other alias')).toBeVisible()
    expect(other.queryByText('Asset component')).not.toBeInTheDocument()
    expect(other.getByRole('link', { name: 'Other relationship' })).toBeVisible()
    expect(original.queryByRole('link', { name: 'Other relationship' })).not.toBeInTheDocument()
    expect(screen.getByRole('heading', { name: /3\.\s*Other team/ })).toBeVisible()
  })

  it.each([false, true])('keeps the term URL when changing sources, including an empty selection (%s)', (notFound) => {
    const submit = vi.spyOn(HTMLFormElement.prototype, 'requestSubmit').mockImplementation(() => {})
    const data = dictionaryPage({
      source: 'engineering',
      termURL: '/glossary/definitions/Component?source=engineering',
    })
    render(
      <PublicApp data={{ ...data, termPage: true, termEntries: notFound ? undefined : [data.entry!], notFound }} />,
    )
    const select = screen.getByRole('combobox', { name: 'Dictionary source' }) as HTMLSelectElement
    fireEvent.change(select, { target: { value: '' } })
    expect(submit).toHaveBeenCalledOnce()
    expect(select.form).toHaveAttribute('action', '/glossary/definitions/Component')
    expect(select.form).toHaveAttribute('method', 'get')
    expect(new FormData(select.form!).get('source')).toBe('')
  })

  it('shows pronunciation and meaning-specific part of speech on a definition page', () => {
    render(<PublicApp data={dictionaryPage()} />)
    expect(screen.getByText('kuhm-POH-nuhnt')).toBeVisible()
    expect(sectionFor(/Asset management/).getByText('noun')).toBeVisible()
    expect(sectionFor(/Frontend engineering/).queryByText('noun')).not.toBeInTheDocument()
  })

  it('keeps aliases and deprecation attached to the correct meaning', () => {
    render(<PublicApp data={dictionaryPage()} />)
    const asset = sectionFor(/Asset management/)
    const web = sectionFor(/Frontend engineering/)
    expect(asset.getByText('Asset component')).toBeVisible()
    expect(web.queryByText('Asset component')).not.toBeInTheDocument()
    expect(asset.queryByText('Deprecated', { exact: true })).not.toBeInTheDocument()
    expect(web.getByText('Deprecated', { exact: true })).toBeVisible()
    expect(web.getByRole('link', { name: 'Interface element' })).toHaveAttribute(
      'href',
      '/definitions/frontend/interface-element',
    )
  })

  it('supports arrow keys and Home/End on the alphabet without changing the selected filter', () => {
    render(
      <PublicApp
        data={dictionaryPage({
          sources: [{ id: 'engineering', name: 'Engineering', stale: false, letters: ['C', 'F', '0-9'] }],
        })}
      />,
    )
    const nav = within(screen.getByRole('navigation', { name: 'Browse by first letter' }))
    const selected = nav.getByRole('link', { name: 'Terms beginning with C' })
    selected.focus()
    fireEvent.keyDown(selected, { key: 'ArrowRight' })
    expect(nav.getByRole('button', { name: 'Terms beginning with D' })).toBeDisabled()
    const next = nav.getByRole('link', { name: 'Terms beginning with F' })
    expect(next).toHaveFocus()
    expect(selected).toHaveAttribute('aria-current', 'page')
    fireEvent.keyDown(next, { key: 'Home' })
    const all = nav.getByRole('link', { name: 'Browse all terms' })
    expect(all).toHaveFocus()
    fireEvent.keyDown(all, { key: 'End' })
    expect(nav.getByRole('link', { name: 'Terms beginning with 0-9' })).toHaveFocus()
    fireEvent.keyDown(document.activeElement!, { key: 'ArrowRight' })
    expect(all).toHaveFocus()
  })

  it('combines cached source letter sets and masks them by selected source, independently of search results', () => {
    const sources = [
      { id: 'engineering', name: 'Engineering', stale: false, letters: ['A', 'C'] },
      { id: 'web', name: 'Web', stale: false, letters: ['B', 'C', '#'] },
    ]
    const { rerender } = render(<PublicApp data={dictionaryPage({ entry: undefined, sources, query: 'no results' })} />)
    const nav = () => within(screen.getByRole('navigation', { name: 'Browse by first letter' }))
    expect(nav().getByRole('link', { name: 'Terms beginning with A' })).toHaveAttribute('href', '/?letter=A')
    expect(nav().getByRole('link', { name: 'Terms beginning with B' })).toBeVisible()
    rerender(<PublicApp data={dictionaryPage({ entry: undefined, sources, source: 'engineering' })} />)
    expect(nav().getByRole('button', { name: 'Terms beginning with B' })).toBeDisabled()
    expect(nav().getByRole('link', { name: 'Terms beginning with A' })).toHaveAttribute(
      'href',
      '/?source=engineering&letter=A',
    )
    rerender(<PublicApp data={dictionaryPage({ entry: undefined, sources: [], letter: 'Z' })} />)
    expect(nav().getAllByRole('link')).toHaveLength(1)
    expect(nav().getByRole('link', { name: 'Browse all terms' })).toHaveAttribute('tabindex', '0')
    expect(nav().getAllByRole('button')).toHaveLength(28)
  })

  it('copies the canonical definition URL rather than the current search URL', async () => {
    history.replaceState(null, '', '/?q=component&source=engineering')
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    render(<PublicApp data={dictionaryPage()} />)
    fireEvent.click(screen.getByRole('button', { name: 'Copy link' }))
    expect(await screen.findByRole('button', { name: 'Copied' })).toBeVisible()
    expect(writeText).toHaveBeenCalledWith('http://localhost:8080/definitions/engineering/component')
  })

  it('provides a selectable URL if clipboard permission is denied', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
    })
    render(<PublicApp data={dictionaryPage()} />)
    fireEvent.click(screen.getByRole('button', { name: 'Copy link' }))
    const fallback = await screen.findByRole('textbox', { name: 'Definition link' })
    expect(fallback).toHaveValue('http://localhost:8080/definitions/engineering/component')
    await waitFor(() => expect(fallback).toHaveFocus())
    expect(screen.queryByRole('button', { name: 'Copied' })).not.toBeInTheDocument()
  })

  it('submits the search text and source together as a native GET form', () => {
    const submit = vi.spyOn(HTMLFormElement.prototype, 'requestSubmit').mockImplementation(() => {})
    render(<PublicApp data={dictionaryPage({ base: '/glossary' })} />)
    fireEvent.change(screen.getByRole('combobox', { name: 'Dictionary source' }), {
      target: { value: 'engineering' },
    })
    expect(submit).toHaveBeenCalledOnce()
    const form = screen.getByRole('form', { name: 'Dictionary search' })
    expect(form).toHaveAttribute('action', '/glossary/')
    expect(form).toHaveAttribute('method', 'get')
    expect(form).toBeInstanceOf(HTMLFormElement)
    if (!(form instanceof HTMLFormElement)) throw new Error('Search must be a form')
    expect(Object.fromEntries(new FormData(form))).toMatchObject({
      q: 'component',
      source: 'engineering',
    })
  })

  it('distinguishes an empty letter from a missing definition', () => {
    const { rerender } = render(<PublicApp data={dictionaryPage({ entry: undefined, query: '', letter: 'Z' })} />)
    expect(screen.getByRole('heading', { name: 'No matching terms' })).toBeVisible()
    rerender(<PublicApp data={dictionaryPage({ entry: undefined, notFound: true })} />)
    expect(screen.getByRole('heading', { name: 'Definition not found' })).toBeVisible()
    expect(screen.queryByRole('heading', { name: 'No matching terms' })).not.toBeInTheDocument()
  })
})
