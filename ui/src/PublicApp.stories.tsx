import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { PublicApp } from './PublicApp'
import { dictionaryPage } from './test/fixtures'

const meta = {
  component: PublicApp,
  tags: ['ai-generated'],
  parameters: {
    docs: {
      description: {
        component: 'Full-page integration stories for the public glossary. See Definition for the isolated component.',
      },
    },
  },
  args: { data: dictionaryPage() },
} satisfies Meta<typeof PublicApp>

export default meta
type Story = StoryObj<typeof meta>

export const DefinitionPage: Story = {
  play: async ({ canvas, userEvent }) => {
    const selected = canvas.getByRole('link', { name: 'Terms beginning with C' })
    selected.focus()
    await userEvent.keyboard('{ArrowRight}')
    await expect(canvas.getByRole('link', { name: 'Terms beginning with D' })).toHaveFocus()
    await expect(selected).toHaveAttribute('aria-current', 'page')
    await expect(canvas.getByRole('link', { name: 'Interface element' })).toHaveAttribute(
      'href',
      '/definitions/frontend/interface-element',
    )
  },
}

export const MultipleSources: Story = {
  args: {
    data: dictionaryPage({
      entry: undefined,
      query: 'API',
      letter: '',
      sources: [
        { id: 'engineering', name: 'Engineering', stale: false, letters: ['A'] },
        { id: 'web', name: 'Web Development', stale: false, letters: ['A'] },
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
          deprecated: false,
        },
        {
          id: 'api',
          source: 'web',
          term: 'API',
          termKey: 'api',
          href: '/definitions/web/api',
          contexts: ['Web development'],
          definitions: ['A software interface.'],
          deprecated: false,
        },
      ],
    }),
  },
}

export const StaleSource: Story = {
  args: { data: dictionaryPage({ sources: [{ ...dictionaryPage().sources[0], stale: true }] }) },
}

export const EmptyResults: Story = {
  args: { data: dictionaryPage({ entry: undefined, query: '', letter: 'Z' }) },
}

export const MissingDefinition: Story = {
  args: { data: dictionaryPage({ entry: undefined, notFound: true }) },
}

export const CssCheck: Story = {
  play: async ({ canvas }) => {
    const selected = canvas.getByRole('link', { name: 'Terms beginning with C' })
    // styles.css sets the selected alphabet link background to --jg-accent (#164c29).
    await expect(getComputedStyle(selected).backgroundColor).toBe('rgb(22, 76, 41)')
  },
}
