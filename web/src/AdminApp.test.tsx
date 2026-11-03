import { act, fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import AdminApp from './AdminApp'
import { operatorPage } from './test/fixtures'

describe('operator console', () => {
  it('makes configuration and diagnostics accessible by keyboard', () => {
    render(<AdminApp data={operatorPage()} />)
    const sources = screen.getByRole('tab', { name: 'Sources' })
    sources.focus()
    fireEvent.keyDown(sources, { key: 'ArrowRight' })
    const configuration = screen.getByRole('tab', { name: 'Resolved configuration' })
    expect(configuration).toHaveFocus()
    expect(configuration).toHaveAttribute('aria-selected', 'true')
    const output = screen.getByRole('textbox', { name: 'Resolved configuration JSON' })
    expect(output).toHaveAttribute('readonly')
    if (!(output instanceof HTMLTextAreaElement)) throw new Error('Configuration must be selectable text')
    expect(output.value).toContain('PUBLIC_EXPORT_SECRET')
    fireEvent.keyDown(configuration, { key: 'ArrowRight' })
    expect(screen.getByRole('tab', { name: 'Diagnostics (1)' })).toHaveFocus()
    expect(screen.getByText('unresolved relationship')).toBeVisible()
    expect(screen.getByText('engineering / component / web')).toBeVisible()
  })

  it('retains the last loaded source state when polling fails', async () => {
    vi.useFakeTimers()
    const fetch = vi.fn().mockRejectedValue(new Error('offline'))
    vi.stubGlobal('fetch', fetch)
    render(<AdminApp data={operatorPage()} />)
    expect(screen.getByText('Healthy')).toBeVisible()
    await act(async () => {
      await vi.advanceTimersByTimeAsync(15000)
    })
    expect(fetch).toHaveBeenCalledTimes(2)
    expect(screen.getByText('Healthy')).toBeVisible()
    expect(screen.getByRole('status')).toHaveTextContent('Could not refresh. Showing the last loaded diagnostics.')
  })
})
