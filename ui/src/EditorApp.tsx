import { json } from '@codemirror/lang-json'
import { yaml } from '@codemirror/lang-yaml'
import {
  BlockTypeSelect,
  BoldItalicUnderlineToggles,
  CreateLink,
  headingsPlugin,
  ListsToggle,
  linkDialogPlugin,
  linkPlugin,
  listsPlugin,
  MDXEditor,
  markdownShortcutPlugin,
  quotePlugin,
  thematicBreakPlugin,
  toolbarPlugin,
  UndoRedo,
} from '@mdxeditor/editor'
import {
  ArrowLeftIcon,
  ArrowSquareOutIcon,
  BookOpenIcon,
  BracketsCurlyIcon,
  CheckCircleIcon,
  DownloadSimpleIcon,
  FileArrowUpIcon,
  FilePlusIcon,
  FileTextIcon,
  FloppyDiskIcon,
  FolderOpenIcon,
  HardDriveIcon,
  InfoIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  ShieldCheckIcon,
  TrashIcon,
  XIcon,
} from '@phosphor-icons/react'
import CodeMirror, { EditorView } from '@uiw/react-codemirror'
import { type DragEvent, type RefObject, useCallback, useEffect, useId, useRef, useState } from 'react'
import '@mdxeditor/editor/style.css'
import './editor.css'
import {
  type EditorDocument,
  type EditorEntry,
  type EditorSense,
  type FilePickerWindow,
  type GlossaryFormat,
  glossaryFormat,
  type LocalFileHandle,
  parseDocument,
  partsOfSpeech,
  serializeDocument,
  writeLocalFile,
} from './editorDocument'

function Field({
  label,
  value,
  onChange,
  inputRef,
}: {
  label: string
  value: string
  onChange: (value: string) => void
  inputRef?: RefObject<HTMLInputElement | null>
}) {
  const id = useId()
  return (
    <label className="editor-field" htmlFor={id}>
      {label}
      <input ref={inputRef} id={id} value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  )
}

function PartOfSpeech({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const id = useId()
  return (
    <label className="editor-field" htmlFor={id}>
      Part of speech
      <select id={id} value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">Not specified</option>
        {partsOfSpeech.map((part) => (
          <option key={part} value={part}>
            {part.charAt(0).toUpperCase() + part.slice(1)}
          </option>
        ))}
      </select>
    </label>
  )
}

function Lines({
  label,
  values,
  onChange,
}: {
  label: string
  values?: string[] | null
  onChange: (values: string[]) => void
}) {
  const id = useId()
  return (
    <label className="editor-field" htmlFor={id}>
      {label} <span className="muted">(one per line)</span>
      <textarea
        id={id}
        value={(values ?? []).join('\n')}
        onChange={(event) => onChange(event.target.value === '' ? [] : event.target.value.split('\n'))}
      />
    </label>
  )
}

function Deprecation({
  value,
  onChange,
}: {
  value: { deprecated?: boolean; deprecationReason?: string }
  onChange: (patch: { deprecated?: boolean; deprecationReason?: string }) => void
}) {
  return (
    <div className="editor-deprecation">
      <label>
        <input
          type="checkbox"
          checked={value.deprecated ?? false}
          onChange={(event) => onChange({ deprecated: event.target.checked })}
        />{' '}
        Deprecated
      </label>
      {value.deprecated && (
        <Field
          label="Deprecation reason"
          value={value.deprecationReason ?? ''}
          onChange={(deprecationReason) => onChange({ deprecationReason })}
        />
      )}
    </div>
  )
}

const identities = new WeakMap<object, string>()
function localKey(value: object) {
  let key = identities.get(value)
  if (!key) {
    key = crypto.randomUUID()
    identities.set(value, key)
  }
  return key
}
function patchObject<T extends object>(value: T, patch: Partial<T>): T {
  const updated = { ...value, ...patch }
  identities.set(updated, localKey(value))
  return updated
}

const jsonExtensions = [
  json(),
  EditorView.lineWrapping,
  EditorView.contentAttributes.of({ 'aria-label': 'Glossary JSON' }),
]

const yamlExtensions = [
  yaml(),
  EditorView.lineWrapping,
  EditorView.contentAttributes.of({ 'aria-label': 'Glossary YAML' }),
]

const plugins = () => [
  headingsPlugin(),
  listsPlugin(),
  quotePlugin(),
  thematicBreakPlugin(),
  linkPlugin(),
  linkDialogPlugin(),
  markdownShortcutPlugin(),
  toolbarPlugin({
    toolbarContents: () => (
      <>
        <UndoRedo />
        <BoldItalicUnderlineToggles />
        <BlockTypeSelect />
        <ListsToggle />
        <CreateLink />
      </>
    ),
  }),
]

export default function EditorApp({ base = '' }: { siteName: string; base?: string }) {
  const supported = window.isSecureContext && typeof (window as FilePickerWindow).showOpenFilePicker === 'function'
  const [document, setDocument] = useState<EditorDocument | null>(null)
  const [raw, setRaw] = useState('')
  const [rawMode, setRawMode] = useState(false)
  const [handle, setHandle] = useState<LocalFileHandle | null>(null)
  const [fileName, setFileName] = useState('')
  const [isNew, setIsNew] = useState(false)
  const [fileFormat, setFileFormat] = useState<GlossaryFormat>('json')
  const [formatSource, setFormatSource] = useState('')
  const formatLabel = fileFormat.toUpperCase()
  const [dragging, setDragging] = useState(false)
  const [original, setOriginal] = useState('')
  const [saved, setSaved] = useState('')
  const [selected, setSelected] = useState(0)
  const [filter, setFilter] = useState('')
  const [message, setMessage] = useState('')
  const [messageError, setMessageError] = useState(false)
  const [busy, setBusy] = useState(false)
  const busyRef = useRef(false)
  const termInput = useRef<HTMLInputElement>(null)
  const glossaryNameInput = useRef<HTMLInputElement>(null)
  const focusNewGlossary = useRef(false)
  const focusNewEntry = useRef(false)
  let contents = rawMode ? raw : ''
  let validation = ''
  if (document) {
    try {
      if (!rawMode) contents = serializeDocument(document, fileFormat, formatSource)
      parseDocument(contents, fileFormat)
    } catch (error) {
      validation = error instanceof Error ? error.message : 'Invalid glossary.'
    }
  }
  const dirty = !!document && contents !== saved
  useEffect(() => {
    if (focusNewGlossary.current) {
      glossaryNameInput.current?.focus()
      glossaryNameInput.current?.select()
      focusNewGlossary.current = false
    }
    if (!focusNewEntry.current) return
    termInput.current?.focus()
    termInput.current?.select()
    focusNewEntry.current = false
  })
  useEffect(() => {
    if (!dirty) return
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }
    window.addEventListener('beforeunload', warn)
    return () => window.removeEventListener('beforeunload', warn)
  }, [dirty])
  useEffect(() => {
    if (!message || messageError) return
    const timeout = setTimeout(() => setMessage(''), 5000)
    return () => clearTimeout(timeout)
  }, [message, messageError])
  useEffect(() => {
    documentElementReady()
  }, [])

  function newGlossary() {
    if (busyRef.current || (dirty && !window.confirm('Discard unsaved changes and start a new glossary?'))) return
    const fresh: EditorDocument = { version: '1', name: 'Untitled glossary', entries: [] }
    setDocument(fresh)
    setHandle(null)
    setIsNew(true)
    setFileName('')
    setOriginal('')
    setSaved('')
    setFormatSource('')
    setRaw(serializeDocument(fresh, fileFormat))
    setRawMode(false)
    setSelected(0)
    setFilter('')
    setMessage('')
    setMessageError(false)
    focusNewGlossary.current = true
  }
  function changeDraftFormat(format: GlossaryFormat) {
    if (!document || !isNew || rawMode || busyRef.current) return
    setFileFormat(format)
    setFormatSource('')
    setRaw(serializeDocument(document, format))
  }
  async function open() {
    if (busyRef.current || (dirty && !window.confirm('Discard unsaved changes and open another glossary?'))) return
    busyRef.current = true
    setBusy(true)
    try {
      const [file] =
        (await (window as FilePickerWindow).showOpenFilePicker?.({
          multiple: false,
          types: [
            {
              description: 'JSON or YAML glossary',
              accept: { 'application/json': ['.json'], 'application/yaml': ['.yaml', '.yml'] },
            },
          ],
        })) ?? []
      if (!file) return
      await loadFile(await file.getFile(), file)
    } catch (error) {
      report(error)
    } finally {
      busyRef.current = false
      setBusy(false)
    }
  }
  async function loadFile(file: File, fileHandle: LocalFileHandle | null) {
    const format = glossaryFormat(file.name)
    const text = await file.text()
    const parsed = parseDocument(text, format)
    setDocument(parsed)
    setRaw(text)
    setFileFormat(format)
    setFormatSource(text)
    setHandle(fileHandle)
    setIsNew(false)
    setFileName(file.name)
    setOriginal(text)
    setSaved(serializeDocument(parsed, format, text))
    setRawMode(false)
    setSelected(0)
    setFilter('')
    setMessageError(false)
    setMessage(
      fileHandle
        ? 'Opened locally. File contents stay in this browser.'
        : 'Opened locally. This browser did not provide a writable file handle. Use Download copy, or reopen with the file picker to save to the original.',
    )
  }
  async function dropFile(event: DragEvent<HTMLElement>) {
    event.preventDefault()
    setDragging(false)
    if (busyRef.current) return
    const items = Array.from(event.dataTransfer.items).filter((item) => item.kind === 'file')
    const files = Array.from(event.dataTransfer.files)
    if ((items.length || files.length) !== 1) {
      setMessageError(true)
      setMessage('Drop exactly one glossary file at a time.')
      return
    }
    if (dirty && !window.confirm('Discard unsaved changes and open another glossary?')) return
    busyRef.current = true
    setBusy(true)
    try {
      // Request the handle and fallback File in the drop event's synchronous tick.
      // The drag data store is unavailable after the first await.
      const item = items[0] as
        | (DataTransferItem & {
            getAsFileSystemHandle?: () => Promise<(LocalFileHandle & { kind: 'file' }) | { kind: 'directory' } | null>
          })
        | undefined
      const fallback = item?.getAsFile() ?? files[0]
      const pending = item?.getAsFileSystemHandle?.()
      const droppedHandle = await pending
      if (droppedHandle?.kind === 'directory') throw new Error('Drop a glossary file, not a folder.')
      if (droppedHandle) await loadFile(await droppedHandle.getFile(), droppedHandle)
      else if (fallback) await loadFile(fallback, null)
      else throw new Error('Could not read this file. Try Open glossary instead.')
    } catch (error) {
      report(error)
    } finally {
      busyRef.current = false
      setBusy(false)
    }
  }
  const report = useCallback((error: unknown) => {
    if (error instanceof DOMException && error.name === 'AbortError') return
    setMessageError(true)
    setMessage(error instanceof Error ? error.message : 'The file operation failed. Try downloading a copy.')
  }, [])
  const save = useCallback(async () => {
    if ((!handle && !isNew) || busyRef.current || validation) return
    busyRef.current = true
    setBusy(true)
    try {
      const parsed = parseDocument(contents, fileFormat)
      let target = handle
      let output = contents
      let outputFormat = fileFormat
      if (!target) {
        const picker = (window as FilePickerWindow).showSaveFilePicker
        if (!picker)
          throw new Error(
            'This browser cannot choose a new save location. Use Download copy to save your new glossary.',
          )
        target = await picker.call(window, {
          suggestedName: `untitled-glossary.${fileFormat}`,
          types: [
            {
              description: 'JSON or YAML glossary',
              accept: { 'application/json': ['.json'], 'application/yaml': ['.yaml', '.yml'] },
            },
          ],
        })
        outputFormat = glossaryFormat(target.name)
        if (outputFormat !== fileFormat) output = serializeDocument(parsed, outputFormat)
      } else {
        const disk = await (await target.getFile()).text()
        if (disk !== original)
          throw new Error(
            'The file changed on disk. Download your edits or reopen the file before saving to avoid overwriting external changes.',
          )
      }
      parseDocument(output, outputFormat)
      await writeLocalFile(target, output)
      setHandle(target)
      setIsNew(false)
      setFileName(target.name)
      if (outputFormat !== fileFormat) setDocument(parseDocument(output, outputFormat))
      setFileFormat(outputFormat)
      setRaw(output)
      setOriginal(output)
      setFormatSource(output)
      setSaved(output)
      setMessageError(false)
      setMessage(`Saved ${target.name} to disk.`)
    } catch (error) {
      report(error)
    } finally {
      busyRef.current = false
      setBusy(false)
    }
  }, [contents, fileFormat, handle, isNew, original, validation, report])

  useEffect(() => {
    const shortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
        event.preventDefault()
        void save()
      }
    }
    window.addEventListener('keydown', shortcut)
    return () => window.removeEventListener('keydown', shortcut)
  }, [save])
  function download() {
    if (validation) return
    const url = URL.createObjectURL(
      new Blob([contents], { type: fileFormat === 'yaml' ? 'application/yaml' : 'application/json' }),
    )
    const anchor = window.document.createElement('a')
    anchor.href = url
    anchor.download = fileName || `glossary.${fileFormat}`
    anchor.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
    setMessageError(false)
    setMessage(
      isNew
        ? 'Downloaded your new glossary. Reopen the downloaded file to enable saving directly to it.'
        : 'Downloaded a copy. The original file has not been saved.',
    )
  }
  function toggle() {
    if (!document) return
    if (rawMode) {
      try {
        setDocument(parseDocument(raw, fileFormat))
        setFormatSource(raw)
        setRawMode(false)
        setMessage('')
      } catch (error) {
        report(error)
      }
    } else {
      try {
        setRaw(serializeDocument(document, fileFormat, formatSource))
        setRawMode(true)
      } catch (error) {
        report(error)
      }
    }
  }
  function update(patch: Partial<EditorDocument>) {
    if (document) setDocument({ ...document, ...patch })
  }
  function updateEntry(patch: Partial<EditorEntry>) {
    if (document)
      update({
        entries: document.entries.map((entry, index) => (index === selected ? patchObject(entry, patch) : entry)),
      })
  }
  const entry = document?.entries[selected]
  function updateSense(index: number, patch: Partial<EditorSense>) {
    if (entry)
      updateEntry({ senses: entry.senses.map((sense, i) => (i === index ? patchObject(sense, patch) : sense)) })
  }
  const visibleEntries =
    document?.entries
      .map((item, index) => ({ item, index }))
      .filter(({ item }) => `${item.term} ${item.id}`.toLowerCase().includes(filter.toLowerCase())) ?? []

  function addEntry() {
    if (!document) return
    focusNewEntry.current = true
    update({
      entries: [
        ...document.entries,
        { id: `entry-${crypto.randomUUID()}`, term: 'New term', senses: [{ id: 'meaning', definition: '' }] },
      ],
    })
    setSelected(document.entries.length)
    setFilter('')
  }
  function addMeaning() {
    if (entry) updateEntry({ senses: [...entry.senses, { id: `sense-${crypto.randomUUID()}`, definition: '' }] })
  }
  return (
    <main
      className={`glossary-editor${document ? ' editor-has-file' : ''}`}
      aria-label="Drop a glossary or choose a file"
      onDragOver={(event) => {
        if (!supported || !event.dataTransfer.types.includes('Files')) return
        event.preventDefault()
        event.dataTransfer.dropEffect = busy ? 'none' : 'copy'
        setDragging(!busy)
      }}
      onDragLeave={(event) => {
        if (!(event.relatedTarget instanceof Node) || !event.currentTarget.contains(event.relatedTarget))
          setDragging(false)
      }}
      onDrop={(event) => {
        event.preventDefault()
        if (supported) void dropFile(event)
      }}
    >
      <header className="editor-appbar">
        <div className="editor-app-identity">
          <a className="editor-back" href={`${base}/`} aria-label="Back to glossary" title="Back to glossary">
            <ArrowLeftIcon size={18} aria-hidden />
          </a>
          <BookOpenIcon className="editor-brand-icon" size={23} aria-hidden />
          <h1>Glossary editor</h1>
          <span className="editor-local-badge" title="File contents stay in your browser. Nothing is uploaded.">
            <ShieldCheckIcon size={15} aria-hidden />
            Local only
          </span>
        </div>
        {document && (
          <div className="editor-file-actions">
            <button
              type="button"
              className="editor-button editor-new-button"
              aria-label="New glossary"
              title="New glossary"
              disabled={busy}
              onClick={newGlossary}
            >
              <PlusIcon size={17} aria-hidden />
              <span>New</span>
            </button>
            <button
              type="button"
              className="editor-button editor-open-button"
              aria-label="Open glossary"
              onClick={() => void open()}
              disabled={busy}
            >
              <FolderOpenIcon size={17} aria-hidden />
              <span>Open file</span>
            </button>
            <button type="button" className="editor-button" disabled={!!validation || busy} onClick={download}>
              <DownloadSimpleIcon size={17} aria-hidden />
              <span>Download copy</span>
            </button>
            <button
              type="button"
              className="editor-button editor-primary"
              disabled={busy || !!validation || !dirty || (!handle && !isNew)}
              onClick={() => void save()}
              title={
                isNew
                  ? 'Choose a location for your new glossary (Ctrl/Cmd+S)'
                  : !handle
                    ? 'Reopen with the file picker to save to the original file'
                    : 'Save to the original file (Ctrl/Cmd+S)'
              }
            >
              <FloppyDiskIcon size={17} aria-hidden />
              {busy ? 'Working…' : 'Save'}
            </button>
          </div>
        )}
      </header>
      {!supported ? (
        <section className="editor-welcome editor-unsupported">
          <HardDriveIcon size={36} aria-hidden />
          <h2>Local file editing is unavailable</h2>
          <p>
            This editor needs the File System Access API on HTTPS or localhost. Try a supported browser such as Chrome
            or Edge.
          </p>
          <a href={`${base}/`}>
            Return to glossary <ArrowSquareOutIcon size={14} aria-hidden />
          </a>
        </section>
      ) : !document ? (
        <div className="editor-start">
          <section className="editor-welcome" aria-labelledby="editor-welcome-title">
            <h2 id="editor-welcome-title">Create or open a glossary.</h2>
            <p>
              Edit a glossary document directly on your computer. <br className="editor-desktop-break" />
              Save it locally, host it somewhere and add it to Jargonary as a source.
            </p>
            <div className="editor-dropzone">
              <FileArrowUpIcon size={26} aria-hidden />
              <strong>Drop your glossary here</strong>
              <span>or choose a file from your computer</span>
              <button
                type="button"
                className="editor-button editor-primary"
                onClick={() => void open()}
                disabled={busy}
              >
                <FolderOpenIcon size={18} aria-hidden />
                {busy ? 'Opening…' : 'Open glossary'}
              </button>
              <small>One .json, .yaml, or .yml file at a time</small>
            </div>
            <div className="editor-start-new">
              <small>- or -</small>
              <p>Starting from scratch?</p>
              <button type="button" className="editor-button" onClick={newGlossary} disabled={busy}>
                <FilePlusIcon size={17} aria-hidden />
                New glossary
              </button>
            </div>
          </section>
        </div>
      ) : (
        <>
          <div className="editor-documentbar">
            <div className="editor-file-label">
              <FileTextIcon size={18} aria-hidden />
              <strong>{fileName || `Untitled.${fileFormat}`}</strong>
              <span className={`editor-save-state${dirty ? ' editor-unsaved' : ''}`}>
                {dirty ? 'Unsaved changes' : handle ? 'All changes saved' : 'Download only'}
              </span>
            </div>
            {isNew && (
              <label className="editor-draft-format">
                File format
                <select
                  aria-label="New glossary format"
                  value={fileFormat}
                  disabled={busy || rawMode}
                  onChange={(event) => changeDraftFormat(event.target.value as GlossaryFormat)}
                >
                  <option value="json">JSON</option>
                  <option value="yaml">YAML</option>
                </select>
              </label>
            )}
            <fieldset className="editor-view-toggle" aria-label="Editor view">
              <button
                type="button"
                aria-pressed={!rawMode}
                onClick={() => {
                  if (rawMode) toggle()
                }}
                disabled={busy}
              >
                <BookOpenIcon size={15} aria-hidden />
                Form
              </button>
              <label className={`editor-json-toggle${rawMode ? ' editor-view-active' : ''}`}>
                <input
                  className="sr-only"
                  type="checkbox"
                  role="switch"
                  aria-label={`Raw ${formatLabel}`}
                  aria-checked={rawMode}
                  checked={rawMode}
                  onChange={toggle}
                  disabled={busy}
                />
                <BracketsCurlyIcon size={15} aria-hidden />
                {formatLabel}
              </label>
            </fieldset>
          </div>
          {validation && (
            <div role="alert" className="editor-error">
              <InfoIcon size={18} aria-hidden />
              <div>
                <strong>Some fields need your attention before saving.</strong>
                <p>{validation}</p>
              </div>
            </div>
          )}
          {rawMode ? (
            <section className="editor-json-panel" aria-label={`Raw ${formatLabel} editor`}>
              <div className="editor-json-heading">
                <span>Complete glossary document</span>
                <span>
                  {formatLabel} · Format {document.version}
                </span>
              </div>
              <CodeMirror
                className="editor-code-input"
                value={raw}
                extensions={fileFormat === 'yaml' ? yamlExtensions : jsonExtensions}
                indentWithTab={false}
                onChange={setRaw}
                editable={!busy}
                aria-label={`Glossary ${formatLabel}`}
                height="100%"
              />
              <p className="editor-json-help">
                All fields are editable here. Fix validation errors before saving or returning to the form.
              </p>
            </section>
          ) : (
            <fieldset className="editor-workspace" disabled={busy}>
              <aside className="editor-sidebar">
                <div className="editor-glossary-settings">
                  <p className="editor-kicker">GLOSSARY</p>
                  <Field
                    inputRef={glossaryNameInput}
                    label="Glossary name"
                    value={document.name}
                    onChange={(name) => update({ name })}
                  />
                </div>
                <div className="editor-search">
                  <MagnifyingGlassIcon size={17} aria-hidden />
                  <input
                    aria-label="Find an entry"
                    placeholder="Find an entry…"
                    value={filter}
                    onChange={(event) => setFilter(event.target.value)}
                  />
                  {filter && (
                    <button type="button" aria-label="Clear search" onClick={() => setFilter('')}>
                      <XIcon size={14} aria-hidden />
                    </button>
                  )}
                </div>
                <div className="editor-list-heading">
                  <h2>
                    Entries <span>{document.entries.length}</span>
                  </h2>
                  <button
                    type="button"
                    className="editor-icon-button"
                    aria-label="Add entry"
                    title="Add entry"
                    onClick={addEntry}
                  >
                    <PlusIcon size={18} aria-hidden />
                  </button>
                </div>
                <nav className="editor-entry-list" aria-label="Glossary entries">
                  {visibleEntries.map(({ item, index }) => (
                    <button
                      type="button"
                      key={localKey(item)}
                      aria-label={`${item.term || 'Untitled term'}, ${item.senses.length} ${item.senses.length === 1 ? 'meaning' : 'meanings'}`}
                      aria-current={index === selected ? 'true' : undefined}
                      onClick={() => setSelected(index)}
                    >
                      <span className="editor-entry-list-title">
                        {item.term || 'Untitled term'}
                        {item.deprecated && (
                          <span className="editor-deprecated-mark" title="Deprecated">
                            Deprecated
                          </span>
                        )}
                      </span>
                      <small>
                        {item.senses.length} {item.senses.length === 1 ? 'meaning' : 'meanings'}
                      </small>
                    </button>
                  ))}
                  {visibleEntries.length === 0 && (
                    <div className="editor-no-results">
                      <p>{filter ? `No entries match “${filter}”.` : 'A fresh page for your team’s vocabulary.'}</p>
                      <button
                        type="button"
                        className="editor-button"
                        onClick={() => {
                          if (filter) setFilter('')
                          else addEntry()
                        }}
                      >
                        {filter ? 'Clear search' : 'Add your first entry'}
                      </button>
                    </div>
                  )}
                </nav>
                <div className="editor-sidebar-foot">
                  <HardDriveIcon size={15} aria-hidden />
                  <span>{isNew ? 'Local draft' : 'Local file'} · No uploads</span>
                </div>
              </aside>
              <section className="editor-entry" aria-label="Entry form">
                {entry ? (
                  <div className="editor-entry-content" key={localKey(entry)}>
                    <div className="editor-entry-heading">
                      <p className="editor-kicker">
                        ENTRY {String(selected + 1).padStart(2, '0')}{' '}
                        <span>/ {String(document.entries.length).padStart(2, '0')}</span>
                      </p>
                      <button
                        type="button"
                        className="editor-icon-button editor-destructive"
                        aria-label="Remove entry"
                        title="Remove entry"
                        onClick={() => {
                          if (
                            window.confirm(
                              'Remove this entry and all its senses? Relationship targets are not changed.',
                            )
                          ) {
                            update({ entries: document.entries.filter((_, i) => i !== selected) })
                            setSelected(Math.max(0, selected - 1))
                          }
                        }}
                      >
                        <TrashIcon size={18} aria-hidden />
                      </button>
                    </div>
                    <div className="editor-term">
                      <Field
                        inputRef={termInput}
                        label="Term"
                        value={entry.term}
                        onChange={(term) => updateEntry({ term })}
                      />
                    </div>
                    <details className="editor-details editor-entry-details">
                      <summary>
                        Entry details <span>Pronunciation, ID & status</span>
                      </summary>
                      <div className="editor-details-body">
                        <div className="editor-field-grid">
                          <Field
                            label="Pronunciation"
                            value={entry.pronunciation ?? ''}
                            onChange={(pronunciation) => updateEntry({ pronunciation })}
                          />
                          <Field label="Stable entry ID" value={entry.id} onChange={(id) => updateEntry({ id })} />
                        </div>
                        <p className="editor-field-help">
                          Keep IDs stable. Changing an ID does not update aliases or relationship targets.
                        </p>
                        <Deprecation value={entry} onChange={updateEntry} />
                      </div>
                    </details>
                    <div className="editor-meanings-heading">
                      <h2>
                        Meanings <span>{entry.senses.length}</span>
                      </h2>
                      <button type="button" className="editor-button editor-add-meaning" onClick={addMeaning}>
                        <PlusIcon size={16} aria-hidden />
                        Add meaning
                      </button>
                    </div>
                    {entry.senses.map((sense, index) => (
                      <section className="editor-sense" key={localKey(sense)} aria-label={`Sense ${index + 1}`}>
                        <div className="editor-sense-heading">
                          <h3>
                            <span aria-hidden>{String(index + 1).padStart(2, '0')}</span>Meaning {index + 1}
                          </h3>
                          <button
                            type="button"
                            className="editor-icon-button editor-destructive"
                            aria-label={`Remove meaning ${index + 1}`}
                            title={
                              entry.senses.length === 1 ? 'Every entry needs at least one meaning' : 'Remove meaning'
                            }
                            disabled={entry.senses.length === 1}
                            onClick={() => {
                              if (
                                window.confirm('Remove this meaning? Alias and relationship targets are not changed.')
                              )
                                updateEntry({ senses: entry.senses.filter((_, i) => i !== index) })
                            }}
                          >
                            <TrashIcon size={16} aria-hidden />
                          </button>
                        </div>
                        <div className="editor-field editor-definition">
                          <span className="editor-definition-label" id={`definition-label-${index}`}>
                            Definition <small>Markdown supported</small>
                          </span>
                          <MDXEditor
                            markdown={sense.definition}
                            readOnly={busy}
                            onChange={(definition, initial) => {
                              if (!initial) updateSense(index, { definition })
                            }}
                            plugins={plugins()}
                            contentEditableClassName="editor-markdown-content"
                            placeholder="Write a clear definition…"
                            translation={(key, fallback, values = {}) => {
                              if (key === 'contentArea.editableMarkdown') return `Definition for meaning ${index + 1}`
                              return Object.entries(values).reduce(
                                (text, [name, value]) => text.replaceAll(`{{${name}}}`, String(value)),
                                fallback,
                              )
                            }}
                            onError={({ error }) =>
                              report(
                                new Error(`Markdown editor: ${error}. Use Raw ${formatLabel} to edit this definition.`),
                              )
                            }
                          />
                        </div>
                        <div className="editor-field-grid editor-sense-context">
                          <Field
                            label="Context"
                            value={sense.context ?? ''}
                            onChange={(context) => updateSense(index, { context })}
                          />
                          <PartOfSpeech
                            value={sense.partOfSpeech ?? ''}
                            onChange={(partOfSpeech) => updateSense(index, { partOfSpeech: partOfSpeech || undefined })}
                          />
                        </div>
                        <details className="editor-details editor-usage-details">
                          <summary>
                            Usage & details{' '}
                            <span>
                              {[
                                sense.examples?.length
                                  ? `${sense.examples.length} ${sense.examples.length === 1 ? 'example' : 'examples'}`
                                  : '',
                                sense.tags?.length ? `${sense.tags.length} tags` : '',
                              ]
                                .filter(Boolean)
                                .join(' · ') || 'Examples, tags, ID & status'}
                            </span>
                          </summary>
                          <div className="editor-details-body">
                            <Lines
                              label="Examples"
                              values={sense.examples}
                              onChange={(examples) => updateSense(index, { examples })}
                            />
                            <Lines label="Tags" values={sense.tags} onChange={(tags) => updateSense(index, { tags })} />
                            <Field
                              label="Stable sense ID"
                              value={sense.id}
                              onChange={(id) => updateSense(index, { id })}
                            />
                            <Deprecation value={sense} onChange={(patch) => updateSense(index, patch)} />
                          </div>
                        </details>
                      </section>
                    ))}
                    <div className="editor-advanced-note">
                      <BracketsCurlyIcon size={18} aria-hidden />
                      <p>
                        Aliases, links, relationships, and export metadata are preserved.
                        <button type="button" onClick={toggle}>
                          Edit advanced fields in {formatLabel} <ArrowSquareOutIcon size={13} aria-hidden />
                        </button>
                      </p>
                    </div>
                  </div>
                ) : (
                  <div className="editor-no-entry">
                    <BookOpenIcon size={32} weight="duotone" aria-hidden />
                    <h2>Your next definition starts here.</h2>
                    <p>Add your first entry to this glossary. An empty glossary is also valid.</p>
                    <button type="button" className="editor-button editor-primary" onClick={addEntry}>
                      <PlusIcon size={17} aria-hidden />
                      Add entry
                    </button>
                  </div>
                )}
              </section>
            </fieldset>
          )}
          <footer className="editor-statusbar">
            <span
              title={
                isNew
                  ? 'This new glossary is in browser memory. Choose Save or Download copy to write it to disk.'
                  : 'Opened from your computer. No uploads. Your browser hides the full path.'
              }
            >
              <HardDriveIcon size={14} aria-hidden />
              <strong>{isNew ? 'Local draft' : 'Local file'}</strong>
              <span className="editor-status-path">
                {isNew ? 'Not saved to disk' : 'Your browser hides the full path'}
              </span>
            </span>
            <span className="editor-drop-hint">Drop a JSON or YAML file anywhere to open it</span>
            <span>
              Format {document.version}
              <span className="editor-shortcut">⌘ / Ctrl S to save</span>
            </span>
          </footer>
        </>
      )}
      {message && (
        <div role="status" className={`editor-notice${messageError ? ' editor-notice-error' : ''}`}>
          {messageError ? <InfoIcon size={19} aria-hidden /> : <CheckCircleIcon size={19} aria-hidden />}
          <p>{message}</p>
          <button
            type="button"
            className="editor-icon-button"
            aria-label="Dismiss notification"
            onClick={() => setMessage('')}
          >
            <XIcon size={16} aria-hidden />
          </button>
        </div>
      )}
      {dragging && (
        <div className="editor-drag-overlay">
          <FileArrowUpIcon size={40} aria-hidden />
          <strong>Drop to open this local glossary</strong>
          <span>Your current file stays open until the new file is validated.</span>
        </div>
      )}
    </main>
  )
}

function documentElementReady() {
  document.documentElement.dataset.jargonaryReady = 'editor'
  window.dispatchEvent(new CustomEvent('jargonary:ready', { detail: { mode: 'editor' } }))
}
