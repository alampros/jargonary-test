import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { Definition } from './Definition'
import { dictionaryPage } from './test/fixtures'

const page = dictionaryPage()
const entry = page.entry!

const meta = {
  component: Definition,
  tags: ['ai-generated'],
  parameters: { layout: 'padded' },
  args: {
    entry,
    sources: page.sources,
    relationships: page.relationships,
    entryURL: page.entryURL,
  },
} satisfies Meta<typeof Definition>

export default meta
type Story = StoryObj<typeof meta>

export const MultipleMeanings: Story = {
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('link', { name: 'Interface element' })).toHaveAttribute(
      'href',
      '/definitions/frontend/interface-element',
    )
    await expect(canvas.getByText('Asset component')).toBeVisible()
    await expect(canvas.queryByRole('searchbox')).not.toBeInTheDocument()
    await expect(canvas.queryByRole('banner')).not.toBeInTheDocument()
    await expect(canvas.queryByRole('navigation')).not.toBeInTheDocument()
  },
}

export const SingleMeaning: Story = {
  args: {
    entry: { ...entry, senses: [entry.senses[0]] },
    relationships: {},
    termURL: '/definitions/Component',
  },
}

export const MultipleSources: Story = {
  args: {
    sources: [...page.sources, { id: 'web', name: 'Web Development', stale: false, letters: ['C'] }],
    termEntries: [
      { ...entry, senses: [entry.senses[0]] },
      {
        ...entry,
        source: 'web',
        aliases: [{ text: 'UI component', sense: 'asset' }],
        senses: [
          {
            ...entry.senses[0],
            source: 'web',
            sense: { id: 'asset', context: 'Frontend engineering', definition: 'A reusable web interface element.' },
          },
        ],
      },
    ],
    entryURL: '/definitions/Component',
    termURL: '/definitions/Component',
    relationships: {
      'web/component/sense:asset': [
        { type: 'relatedTo', title: 'Interface element', href: '/definitions/web/interface-element' },
      ],
    },
  },
  play: async ({ canvas }) => {
    const relationship = canvas.getByRole('link', { name: 'Interface element' })
    await expect(relationship.closest('section')).toHaveAttribute('id', 'meaning-web/component/asset')
    await expect(canvas.getByText('UI component').closest('section')).toHaveAttribute(
      'id',
      'meaning-web/component/asset',
    )
    await expect(canvas.getByText('Asset component').closest('section')).toHaveAttribute(
      'id',
      'meaning-engineering/component/asset',
    )
  },
}

export const StaleSource: Story = {
  args: { sources: [{ ...page.sources[0], stale: true }] },
}
