import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect } from 'storybook/test'
import EditorApp from './EditorApp'
import type { FilePickerWindow } from './editorDocument'

const sample = {
  version: '1',
  name: 'Product & engineering',
  entries: [
    {
      id: 'api',
      term: 'API',
      aliases: [{ text: 'Application programming interface' }],
      senses: [
        {
          id: 'software',
          definition:
            'A defined way for software systems to **communicate** and exchange data.\n\nAn API describes what you can request, how to request it, and what to expect in return.',
          context: 'Software development',
          partOfSpeech: 'noun',
          examples: ['The mobile app uses our API to retrieve order details.'],
          tags: ['engineering', 'public'],
        },
      ],
    },
    {
      id: 'deployment',
      term: 'Deployment',
      senses: [
        { id: 'release', definition: 'The process of making a version of a service available in an environment.' },
      ],
    },
    {
      id: 'design-system',
      term: 'Design system',
      senses: [
        {
          id: 'product',
          definition: 'Shared components, patterns, and principles for building a consistent product experience.',
        },
      ],
    },
    {
      id: 'feature-flag',
      term: 'Feature flag',
      senses: [
        {
          id: 'release',
          definition: 'A switch that controls whether a feature is available without a new deployment.',
        },
      ],
    },
    {
      id: 'latency',
      term: 'Latency',
      senses: [{ id: 'network', definition: 'The time between a request and its response.' }],
    },
    {
      id: 'onboarding',
      term: 'Onboarding',
      senses: [{ id: 'product', definition: 'The experience that helps someone get started with a product.' }],
    },
    {
      id: 'release',
      term: 'Release',
      senses: [{ id: 'software', definition: 'A version of a product made available to its users.' }],
    },
    {
      id: 'service-level-objective',
      term: 'Service level objective',
      senses: [{ id: 'reliability', definition: 'A measurable target for the reliability of a service.' }],
    },
  ],
}
const meta = {
  component: EditorApp,
  parameters: { layout: 'fullscreen' },
  tags: ['ai-generated'],
  args: { siteName: 'Jargonary' },
  beforeEach: () => {
    const previous = (window as FilePickerWindow).showOpenFilePicker
    let text = JSON.stringify(sample, null, 2)
    ;(window as FilePickerWindow).showOpenFilePicker = async () => [
      {
        name: 'product-engineering.json',
        getFile: async () => new File([text], 'product-engineering.json', { type: 'application/json' }),
        createWritable: async () => ({
          write: async (value) => {
            text = value
          },
          close: async () => {},
          abort: async () => {},
        }),
      },
    ]
    return () => {
      ;(window as FilePickerWindow).showOpenFilePicker = previous
    }
  },
} satisfies Meta<typeof EditorApp>
export default meta
type Story = StoryObj<typeof meta>
export const Start: Story = {}
export const Editing: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Open JSON glossary' }))
    await expect(await canvas.findByLabelText('Term', { exact: true })).toHaveValue('API')
  },
}

export const EditingWorkflow: Story = {
  play: async ({ canvas, userEvent }) => {
    await userEvent.click(canvas.getByRole('button', { name: 'Open JSON glossary' }))
    const term = await canvas.findByLabelText('Term', { exact: true })
    await userEvent.clear(term)
    await userEvent.type(term, 'Application programming interface')
    await expect(term).toHaveFocus()
    const definition = canvas.getByRole('textbox', { name: 'Definition for meaning 1' })
    await userEvent.click(definition)
    await userEvent.type(definition, ' Additional context.', { skipClick: true })
    await expect(definition).toHaveTextContent('Additional context.')
    await userEvent.click(canvas.getByRole('button', { name: 'Save' }))
    await expect(canvas.getByRole('status')).toHaveTextContent('Saved product-engineering.json to disk.')
    await userEvent.type(canvas.getByLabelText('Find an entry'), 'Latency')
    await userEvent.click(canvas.getByRole('button', { name: 'Latency, 1 meaning' }))
    await expect(canvas.getByLabelText('Term', { exact: true })).toHaveValue('Latency')
    await userEvent.click(canvas.getByRole('switch', { name: 'Raw JSON' }))
    await expect(canvas.getByRole('region', { name: 'Raw JSON editor' })).toBeVisible()
    await userEvent.click(canvas.getByRole('button', { name: 'Form' }))
    await expect(canvas.getByLabelText('Term', { exact: true })).toHaveValue('Latency')
  },
}
