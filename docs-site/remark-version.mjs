export function activeVersion(manifest) {
  const version = manifest['.']
  if (typeof version !== 'string' || !/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version)) {
    throw new Error('Release manifest must contain a valid version for the root package')
  }
  return `v${version}`
}

// Replace before parsing so Markdown does not interpret the token's underscores.
export default function remarkVersion({ version }) {
  const parse = this.parser
  this.parser = (markdown, file) => parse(markdown.replaceAll('__JARGONARY_VERSION__', version), file)
}
