declare module 'virtual:glossary-validator' {
  import type { ValidateFunction } from 'ajv'
  import type { EditorDocument } from './editorDocument'

  const validate: ValidateFunction<EditorDocument>
  export const partsOfSpeech: readonly string[]
  export default validate
}
