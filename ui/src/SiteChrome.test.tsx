import { render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import AdminApp from './AdminApp'
import { PublicApp } from './PublicApp'
import { dictionaryPage, operatorPage } from './test/fixtures'

describe('site navigation', () => {
  it.each(['public', 'operator'])('provides project resources in the %s header and footer', (mode) => {
    render(
      mode === 'public' ? (
        <PublicApp data={dictionaryPage({ base: '/glossary' })} />
      ) : (
        <AdminApp data={operatorPage()} />
      ),
    )
    const header = within(screen.getByRole('banner'))
    expect(header.getByRole('link', { name: 'Acme' })).toHaveAttribute('href', mode === 'public' ? '/glossary/' : '/')
    expect(header.getByText(mode === 'public' ? 'Shared glossary' : 'Operator console')).toBeVisible()
    if (mode === 'operator') expect(header.getByText('Read-only')).toBeVisible()
    for (const landmark of [screen.getByRole('banner'), screen.getByRole('contentinfo')]) {
      const resources = within(landmark)
      expect(resources.getByRole('link', { name: 'Docs' })).toHaveAttribute(
        'href',
        'https://alampros.github.io/jargonary/',
      )
      expect(resources.getByRole('link', { name: 'GitHub' })).toHaveAttribute(
        'href',
        'https://github.com/alampros/jargonary',
      )
    }
  })
})
