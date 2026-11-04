import type { Meta, StoryObj } from '@storybook/react-vite'
import { HttpResponse, http } from 'msw'
import { expect } from 'storybook/test'
import AdminApp from './AdminApp'
import { operatorPage } from './test/fixtures'

const meta = {
  component: AdminApp,
  tags: ['ai-generated'],
  args: { data: operatorPage() },
} satisfies Meta<typeof AdminApp>

export default meta
type Story = StoryObj<typeof meta>

export const Sources: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByText('engineering', { selector: 'h2' }))
    await expect(canvas.getByText('Accepted entries')).toBeVisible()
    await userEvent.click(canvas.getByRole('tab', { name: 'Resolved configuration' }))
    await expect(canvas.getByRole('tab', { name: 'Resolved configuration' })).toHaveAttribute('aria-selected', 'true')
    await expect(canvas.getByRole('textbox', { name: 'Resolved configuration JSON' })).toHaveValue(
      JSON.stringify(operatorPage().config, null, 2),
    )
    await userEvent.keyboard('{ArrowRight}')
    await expect(canvas.getByRole('tab', { name: 'Diagnostics (1)' })).toHaveFocus()
    await expect(canvas.getByText('unresolved relationship')).toBeVisible()
  },
}

const retrying = operatorPage()
retrying.scrapes[0] = {
  ...retrying.scrapes[0],
  failures: 3,
  recent: [
    {
      at: '2026-11-03T12:00:00Z',
      result: 'error',
      detail: 'Source unavailable; retaining the last successful glossary.',
      durationMs: 1000,
    },
  ],
}

export const RetryingSource: Story = {
  args: { data: retrying },
  beforeEach({ msw }) {
    msw.use(http.get('/api/v1/scrapes', () => HttpResponse.json(retrying.scrapes)))
  },
}

const empty = { ...operatorPage(), config: { sources: [] }, scrapes: [], diagnostics: [] }

export const NoSources: Story = {
  args: { data: empty },
  beforeEach({ msw }) {
    msw.use(
      http.get('/api/v1/scrapes', () => HttpResponse.json([])),
      http.get('/api/v1/diagnostics', () => HttpResponse.json([])),
    )
  },
}
