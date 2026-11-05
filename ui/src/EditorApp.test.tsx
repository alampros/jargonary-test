import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EditorApp from './EditorApp'
import { parseDocument } from './editorDocument'
import { editorYamlFixture } from './test/editorYamlFixture'

vi.mock('@mdxeditor/editor', () => {
  const plugin = () => ({})
  return {
    MDXEditor: ({ markdown, onChange }: { markdown: string; onChange: (text: string, initial: boolean) => void }) => (
      <textarea
        aria-label="Markdown definition"
        defaultValue={markdown}
        onChange={(event) => onChange(event.target.value, false)}
      />
    ),
    BlockTypeSelect: () => null,
    BoldItalicUnderlineToggles: () => null,
    CreateLink: () => null,
    ListsToggle: () => null,
    UndoRedo: () => null,
    headingsPlugin: plugin,
    linkDialogPlugin: plugin,
    linkPlugin: plugin,
    listsPlugin: plugin,
    markdownShortcutPlugin: plugin,
    quotePlugin: plugin,
    thematicBreakPlugin: plugin,
    toolbarPlugin: plugin,
  }
})
vi.mock('@uiw/react-codemirror', () => ({
  EditorView: { lineWrapping: {}, contentAttributes: { of: () => ({}) } },
  default: ({
    value,
    onChange,
    'aria-label': label,
  }: {
    value: string
    onChange: (value: string) => void
    'aria-label': string
  }) => <textarea aria-label={label} value={value} onChange={(event) => onChange(event.target.value)} />,
}))

const contents = JSON.stringify({
  version: '1',
  name: 'Team',
  entries: [
    {
      id: 'api',
      term: 'API',
      aliases: [{ text: 'Interface' }],
      senses: [{ id: 'software', definition: '**Interface**' }],
    },
  ],
})
function fileHandle() {
  const stream = {
    write: vi.fn().mockResolvedValue(undefined),
    close: vi.fn().mockResolvedValue(undefined),
    abort: vi.fn(),
  }
  const file = {
    name: 'team.json',
    getFile: vi.fn().mockResolvedValue({ name: 'team.json', text: async () => contents }),
    createWritable: vi.fn().mockResolvedValue(stream),
  }
  vi.stubGlobal('showOpenFilePicker', vi.fn().mockResolvedValue([file]))
  return { file, stream }
}
async function open() {
  fireEvent.click(screen.getByRole('button', { name: 'Open glossary' }))
  await screen.findByLabelText('Term')
}
beforeEach(() => {
  vi.stubGlobal('isSecureContext', true)
})
describe('glossary editor', () => {
  it('explains unsupported file APIs', () => {
    vi.stubGlobal('showOpenFilePicker', undefined)
    render(<EditorApp siteName="Acme" />)
    expect(screen.getByText('Local file editing is unavailable')).toBeVisible()
    expect(screen.queryByRole('button', { name: 'Open glossary' })).not.toBeInTheDocument()
  })
  it('edits locally and saves Markdown to the same handle without network requests', async () => {
    const { stream } = fileHandle()
    const fetch = vi.fn()
    vi.stubGlobal('fetch', fetch)
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Updated API' } })
    fireEvent.change(screen.getByLabelText('Markdown definition'), { target: { value: '**Updated** definition' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(stream.close).toHaveBeenCalledOnce())
    const saved = JSON.parse(stream.write.mock.calls[0][0])
    expect(saved.entries[0]).toMatchObject({
      term: 'Updated API',
      aliases: [{ text: 'Interface' }],
      senses: [{ definition: '**Updated** definition' }],
    })
    expect(fetch).not.toHaveBeenCalled()
    expect(await screen.findByText('Saved team.json to disk.')).toBeVisible()
  })
  it('opens a dropped file and saves through its original handle', async () => {
    const { file, stream } = fileHandle()
    const getAsFileSystemHandle = vi.fn().mockResolvedValue({ ...file, kind: 'file' })
    render(<EditorApp siteName="Acme" />)
    fireEvent.drop(screen.getByRole('main', { name: 'Drop a glossary or choose a file' }), {
      dataTransfer: { items: [{ kind: 'file', getAsFile: () => null, getAsFileSystemHandle }], files: [] },
    })
    // Handle access must start synchronously while the browser drop data is readable.
    expect(getAsFileSystemHandle).toHaveBeenCalledOnce()
    await screen.findByLabelText('Term')
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Dropped API' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(stream.close).toHaveBeenCalledOnce())
    expect(JSON.parse(stream.write.mock.calls[0][0]).entries[0].term).toBe('Dropped API')
  })
  it('opens drops without handles as download-only files', async () => {
    fileHandle()
    render(<EditorApp siteName="Acme" />)
    fireEvent.drop(screen.getByRole('main', { name: 'Drop a glossary or choose a file' }), {
      dataTransfer: {
        items: [{ kind: 'file', getAsFile: () => ({ name: 'dropped.json', text: async () => contents }) }],
        files: [],
      },
    })
    await screen.findByLabelText('Term')
    expect(screen.getByText(/This browser did not provide a writable file handle/)).toBeVisible()
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Edited' } })
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Download copy' })).toBeEnabled()
    expect(screen.getByText(/dropped.json/)).toBeVisible()
  })
  it('rejects folders and multiple files without replacing the current glossary', async () => {
    fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    const target = screen.getByRole('main', { name: 'Drop a glossary or choose a file' })
    fireEvent.drop(target, { dataTransfer: { items: [], files: [{}, {}] } })
    expect(screen.getByText('Drop exactly one glossary file at a time.')).toBeVisible()
    fireEvent.drop(target, {
      dataTransfer: {
        items: [{ kind: 'file', getAsFile: () => null, getAsFileSystemHandle: async () => ({ kind: 'directory' }) }],
        files: [],
      },
    })
    expect(await screen.findByText('Drop a glossary file, not a folder.')).toBeVisible()
    expect(screen.getByLabelText('Term')).toHaveValue('API')
  })
  it('preserves unsaved edits when replacing a file is cancelled', async () => {
    fileHandle()
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Unsaved edit' } })
    const getAsFileSystemHandle = vi.fn()
    fireEvent.drop(screen.getByRole('main', { name: 'Drop a glossary or choose a file' }), {
      dataTransfer: { items: [{ kind: 'file', getAsFileSystemHandle }], files: [] },
    })
    expect(window.confirm).toHaveBeenCalled()
    expect(getAsFileSystemHandle).not.toHaveBeenCalled()
    expect(screen.getByLabelText('Term')).toHaveValue('Unsaved edit')
  })
  it('retains the current glossary when a dropped file is invalid', async () => {
    fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.drop(screen.getByRole('main', { name: 'Drop a glossary or choose a file' }), {
      dataTransfer: { items: [], files: [{ name: 'invalid.json', text: async () => '{bad' }] },
    })
    await waitFor(() => expect(screen.getByRole('status')).not.toHaveTextContent('Opened locally.'))
    expect(screen.getByLabelText('Term')).toHaveValue('API')
  })
  it('offers a useful empty-search state and restores the entry list', async () => {
    fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.change(screen.getByLabelText('Find an entry'), { target: { value: 'missing' } })
    expect(screen.getByText('No entries match “missing”.')).toBeVisible()
    fireEvent.click(screen.getAllByRole('button', { name: 'Clear search' })[1])
    expect(screen.getByRole('button', { name: 'API, 1 meaning' })).toBeVisible()
    expect(screen.getByLabelText('Term')).toHaveValue('API')
  })
  it('focuses the new term and does not insert placeholder content in the document', async () => {
    fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.click(screen.getByRole('button', { name: 'Add entry' }))
    expect(screen.getByLabelText('Term')).toHaveFocus()
    expect(screen.getByLabelText('Term')).toHaveValue('New term')
    expect(screen.getByLabelText('Markdown definition')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })
  it('uses a fixed part-of-speech list and saves the selected value', async () => {
    const { stream } = fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    const select = screen.getByRole('combobox', { name: 'Part of speech' })
    expect(select).toHaveValue('')
    expect(Array.from(select.querySelectorAll('option')).map((option) => option.value)).toEqual([
      '',
      'noun',
      'pronoun',
      'verb',
      'adjective',
      'adverb',
      'preposition',
      'conjunction',
      'interjection',
      'determiner',
    ])
    fireEvent.change(select, { target: { value: 'noun' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(stream.close).toHaveBeenCalledOnce())
    expect(JSON.parse(stream.write.mock.calls[0][0]).entries[0].senses[0].partOfSpeech).toBe('noun')
  })
  it('rejects a file with a part of speech outside the fixed list', async () => {
    const { file } = fileHandle()
    const imported = JSON.parse(contents)
    imported.entries[0].senses[0].partOfSpeech = 'noun phrase'
    file.getFile.mockResolvedValue({ name: 'team.json', text: async () => JSON.stringify(imported) })
    render(<EditorApp siteName="Acme" />)
    fireEvent.click(screen.getByRole('button', { name: 'Open glossary' }))
    expect(await screen.findByRole('status')).toHaveTextContent('must be equal to one of the allowed values')
    expect(screen.queryByLabelText('Term')).not.toBeInTheDocument()
  })
  it('omits the part-of-speech field when Not specified is selected', async () => {
    const { stream } = fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    const select = screen.getByRole('combobox', { name: 'Part of speech' })
    fireEvent.change(select, { target: { value: 'noun' } })
    fireEvent.change(select, { target: { value: '' } })
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Updated API' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(stream.close).toHaveBeenCalledOnce())
    expect(JSON.parse(stream.write.mock.calls[0][0]).entries[0].senses[0]).not.toHaveProperty('partOfSpeech')
  })
  it.each(['platform.yaml', 'platform.yml'])('opens and saves %s as YAML with its comments', async (name) => {
    const { file, stream } = fileHandle()
    file.name = name
    file.getFile.mockResolvedValue({ name, text: async () => editorYamlFixture })
    render(<EditorApp siteName="Acme" />)
    await open()
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Raw YAML' })).not.toBeChecked()
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'YAML API' } })
    fireEvent.change(screen.getAllByLabelText('Markdown definition')[0], {
      target: { value: '**Edited** definition.\n\nStill Markdown.' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(stream.close).toHaveBeenCalledOnce())
    const saved = stream.write.mock.calls[0][0] as string
    expect(saved).toContain('# Owned by the platform team')
    expect(saved).toContain("term: 'YAML API' # Preferred name")
    expect(parseDocument(saved, 'yaml').entries[0].term).toBe('YAML API')
    expect(parseDocument(saved, 'yaml').entries[0].senses[0].definition).toBe(
      '**Edited** definition.\n\nStill Markdown.',
    )
    expect(() => JSON.parse(saved)).toThrow()
  })
  it('preserves raw YAML comments through form edits and prevents invalid YAML saves', async () => {
    const { file, stream } = fileHandle()
    file.name = 'platform.yaml'
    file.getFile.mockResolvedValue({ name: file.name, text: async () => editorYamlFixture })
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.click(screen.getByRole('switch', { name: 'Raw YAML' }))
    fireEvent.change(screen.getByLabelText('Glossary YAML'), { target: { value: 'entries: [' } })
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
    fireEvent.click(screen.getByRole('switch', { name: 'Raw YAML' }))
    expect(screen.getByLabelText('Glossary YAML')).toHaveValue('entries: [')
    fireEvent.change(screen.getByLabelText('Glossary YAML'), {
      target: { value: editorYamlFixture.replace('# Preferred name', '# Edited in raw YAML') },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Form' }))
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Revised API' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(stream.close).toHaveBeenCalledOnce())
    expect(stream.write.mock.calls[0][0]).toContain("term: 'Revised API' # Edited in raw YAML")
  })
  it('opens a dropped YAML file with save-to-original support', async () => {
    const { file, stream } = fileHandle()
    file.name = 'dropped.yml'
    file.getFile.mockResolvedValue({ name: file.name, text: async () => editorYamlFixture })
    render(<EditorApp siteName="Acme" />)
    fireEvent.drop(screen.getByRole('main', { name: 'Drop a glossary or choose a file' }), {
      dataTransfer: {
        items: [
          { kind: 'file', getAsFile: () => null, getAsFileSystemHandle: async () => ({ ...file, kind: 'file' }) },
        ],
        files: [],
      },
    })
    await screen.findByLabelText('Term')
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Dropped YAML API' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(stream.close).toHaveBeenCalledOnce())
    expect(parseDocument(stream.write.mock.calls[0][0], 'yaml').entries[0].term).toBe('Dropped YAML API')
  })
  it('downloads YAML with the original extension and YAML media type', async () => {
    const { file } = fileHandle()
    file.name = 'platform.yml'
    file.getFile.mockResolvedValue({ name: file.name, text: async () => editorYamlFixture })
    const createURL = vi.fn().mockReturnValue('blob:local-yaml')
    const revokeURL = vi.fn()
    vi.stubGlobal(
      'URL',
      class extends URL {
        static createObjectURL = createURL
        static revokeObjectURL = revokeURL
      },
    )
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      expect(this.download).toBe('platform.yml')
    })
    render(<EditorApp siteName="Acme" />)
    await open()
    vi.useFakeTimers()
    fireEvent.click(screen.getByRole('button', { name: 'Download copy' }))
    expect(createURL.mock.calls[0][0].type).toBe('application/yaml')
    expect(click).toHaveBeenCalledOnce()
    vi.runAllTimers()
    expect(revokeURL).toHaveBeenCalledWith('blob:local-yaml')
  })
  it('keeps invalid raw drafts and prevents saving until fixed', async () => {
    fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.click(screen.getByRole('switch'))
    fireEvent.change(screen.getByLabelText('Glossary JSON'), { target: { value: '{broken' } })
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
    fireEvent.click(screen.getByRole('switch'))
    expect(screen.getByLabelText('Glossary JSON')).toHaveValue('{broken')
    fireEvent.change(screen.getByLabelText('Glossary JSON'), { target: { value: contents.replace('API', 'Changed') } })
    fireEvent.click(screen.getByRole('switch'))
    expect(screen.getByLabelText('Term')).toHaveValue('Changed')
  })
  it('blocks overwriting external changes and retains unsaved edits', async () => {
    const { file, stream } = fileHandle()
    render(<EditorApp siteName="Acme" />)
    await open()
    fireEvent.change(screen.getByLabelText('Term'), { target: { value: 'Local edit' } })
    file.getFile.mockResolvedValue({ text: async () => 'external edit' })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText(/The file changed on disk/)).toBeVisible()
    expect(stream.write).not.toHaveBeenCalled()
    expect(screen.getByLabelText('Term')).toHaveValue('Local edit')
  })
})
