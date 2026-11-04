import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import { Alphabet } from './Alphabet'

const meta = {
  component: Alphabet,
  parameters: { layout: 'padded' },
  decorators: [
    (Story) => (
      <div className="dictionary-layout" style={{ marginTop: 0 }}>
        <Story />
      </div>
    ),
  ],
  args: {
    sources: [
      { id: 'engineering', letters: ['A', 'C', 'F', '0-9'] },
      { id: 'web', letters: ['B', 'C', 'Z', '#'] },
    ],
  },
} satisfies Meta<typeof Alphabet>

export default meta
type Story = StoryObj<typeof meta>

export const AllTerms: Story = {
  play: async ({ canvas }) => {
    const all = canvas.getByRole('link', { name: 'Browse all terms' })
    await expect(all).toHaveAttribute('aria-current', 'page')
    await expect(all).toHaveAttribute('href', '/')
    await expect(all).toHaveAttribute('tabindex', '0')
    await expect(canvas.getByRole('link', { name: 'Terms beginning with B' })).toBeVisible()
    await expect(canvas.getByRole('button', { name: 'Terms beginning with D' })).toBeDisabled()
  },
}

export const SelectedLetter: Story = {
  args: { letter: 'C' },
  play: async ({ canvas, userEvent }) => {
    const selected = canvas.getByRole('link', { name: 'Terms beginning with C' })
    selected.focus()
    await userEvent.keyboard('{ArrowDown}')
    const next = canvas.getByRole('link', { name: 'Terms beginning with F' })
    await expect(next).toHaveFocus()
    await expect(selected).toHaveAttribute('aria-current', 'page')
    await expect(selected).toHaveAttribute('tabindex', '0')
    await expect(next).not.toHaveAttribute('aria-current')
    await userEvent.keyboard('{ArrowUp}')
    await expect(selected).toHaveFocus()
    await userEvent.keyboard('{Home}')
    const all = canvas.getByRole('link', { name: 'Browse all terms' })
    await expect(all).toHaveFocus()
    await userEvent.keyboard('{ArrowLeft}')
    await expect(canvas.getByRole('link', { name: 'Terms beginning with other characters' })).toHaveFocus()
    await userEvent.keyboard('{ArrowRight}')
    await expect(all).toHaveFocus()
    await userEvent.keyboard('{End}')
    await expect(canvas.getByRole('link', { name: 'Terms beginning with other characters' })).toHaveFocus()
  },
}

export const SelectedSource: Story = {
  args: { base: '/glossary', source: 'engineering', letter: 'C' },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('button', { name: 'Terms beginning with B' })).toBeDisabled()
    await expect(canvas.getByRole('link', { name: 'Terms beginning with A' })).toHaveAttribute(
      'href',
      '/glossary/?source=engineering&letter=A',
    )
    await expect(canvas.getByRole('link', { name: 'Browse all terms' })).toHaveAttribute(
      'href',
      '/glossary/?source=engineering',
    )
  },
}

export const EmptyGlossary: Story = {
  args: { sources: [], letter: 'Z' },
  play: async ({ canvas }) => {
    await expect(canvas.getAllByRole('link')).toHaveLength(1)
    await expect(canvas.getByRole('link', { name: 'Browse all terms' })).toHaveAttribute('tabindex', '0')
    await expect(canvas.getAllByRole('button')).toHaveLength(28)
    await expect(canvas.getByRole('button', { name: 'Terms beginning with Z' })).toBeDisabled()
  },
}

export const Numbers: Story = {
  args: { letter: '0-9' },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('link', { name: 'Terms beginning with 0-9' })).toHaveAttribute('aria-current', 'page')
  },
}

export const OtherCharacters: Story = {
  args: { letter: '#' },
  play: async ({ canvas }) => {
    await expect(canvas.getByRole('link', { name: 'Terms beginning with other characters' })).toHaveAttribute(
      'href',
      '/?letter=%23',
    )
  },
}

export const NarrowScreen: Story = {
  args: { letter: 'Z' },
  parameters: {
    viewport: {
      options: {
        narrow: { name: 'Narrow screen', styles: { width: '375px', height: '667px' } },
      },
    },
  },
  globals: { viewport: { value: 'narrow', isRotated: false } },
  play: async ({ canvas }) => {
    const nav = canvas.getByRole('navigation', { name: 'Browse by first letter' })
    const selected = canvas.getByRole('link', { name: 'Terms beginning with Z' })
    await expect(selected).toHaveAttribute('aria-current', 'page')
    await expect(getComputedStyle(nav).flexDirection).toBe('row')
    await expect(selected.getBoundingClientRect().left).toBeGreaterThanOrEqual(nav.getBoundingClientRect().left)
    await expect(selected.getBoundingClientRect().right).toBeLessThanOrEqual(nav.getBoundingClientRect().right)
    await expect(document.documentElement.scrollWidth).toBe(document.documentElement.clientWidth)
  },
}
