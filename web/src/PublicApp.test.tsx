import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { PublicApp } from './PublicApp'
import { dictionaryPage } from './test/fixtures'

function sectionFor(name: RegExp) {
  const section = screen.getByRole('heading', { name }).closest('section')
  if (!section) throw new Error('Meaning must have its own semantic section')
  return within(section)
}

describe('dictionary', () => {
  it('renders dictionary previews and exposes source/context through accessible icon tooltips', () => {
    render(
      <PublicApp
        data={dictionaryPage({
          entry: undefined,
          query: '',
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
    const row = screen.getByRole('article')
    expect(within(row).queryByText('Engineering glossary')).not.toBeInTheDocument()
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    const source = screen.getByRole('button', { name: 'Source: Engineering glossary' })
    fireEvent.focus(source)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Source: Engineering glossary')
    expect(source).toHaveAttribute('aria-describedby', screen.getByRole('tooltip').id)
    fireEvent.keyDown(source, { key: 'Escape' })
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    fireEvent.blur(source)
    fireEvent.click(screen.getByRole('button', { name: 'Context: Asset management · Frontend engineering' }))
    expect(screen.getByRole('tooltip')).toHaveTextContent('Context: Asset management · Frontend engineering')
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
    render(<PublicApp data={dictionaryPage()} />)
    const nav = within(screen.getByRole('navigation', { name: 'Browse by first letter' }))
    const selected = nav.getByRole('link', { name: 'Terms beginning with C' })
    selected.focus()
    fireEvent.keyDown(selected, { key: 'ArrowRight' })
    const next = nav.getByRole('link', { name: 'Terms beginning with D' })
    expect(next).toHaveFocus()
    expect(selected).toHaveAttribute('aria-current', 'page')
    fireEvent.keyDown(next, { key: 'Home' })
    const all = nav.getByRole('link', { name: 'Browse all terms' })
    expect(all).toHaveFocus()
    fireEvent.keyDown(all, { key: 'End' })
    expect(nav.getByRole('link', { name: 'Terms beginning with other characters' })).toHaveFocus()
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
    expect(fallback).toHaveFocus()
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
