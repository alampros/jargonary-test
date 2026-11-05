import { readFileSync } from 'node:fs'
import Ajv from 'ajv/dist/2020'
import standaloneCode from 'ajv/dist/standalone'
import type { Plugin } from 'vite'

// Compile only the bundled schema in Node, so browser validation needs no eval.
// The same plugin runs in Vite development, production builds, and Vitest.
export function editorValidatorPlugin(): Plugin {
  const id = 'virtual:glossary-validator'
  const resolved = `\0${id}`
  const schemaPath = new URL('../docs-site/public/schema/v0.1.2/glossary.schema.json', import.meta.url)
  return {
    name: 'glossary-validator',
    resolveId(source) {
      if (source === id) return resolved
    },
    load(source) {
      if (source !== resolved) return
      this.addWatchFile(schemaPath.pathname)
      const schema = JSON.parse(readFileSync(schemaPath, 'utf8'))
      const ajv = new Ajv({ allErrors: true, code: { source: true, esm: true } })
      const code = standaloneCode(ajv, ajv.compile(schema))
      // Ajv's standalone output uses a CJS runtime helper even in ESM mode.
      return `export const partsOfSpeech = ${JSON.stringify(schema.$defs.Sense.properties.partOfSpeech.enum)};\nconst ucs2length = (value) => Array.from(value).length;\n${code.replaceAll('require("ajv/dist/runtime/ucs2length").default', 'ucs2length')}`
    },
  }
}
