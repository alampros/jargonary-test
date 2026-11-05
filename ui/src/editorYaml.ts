import { type Document, isAlias, isMap, isNode, isScalar, isSeq, type Node, parseDocument } from 'yaml'

// Keep the YAML syntax tree so form edits retain comments, key order, quoting,
// and block scalars instead of rewriting a YAML file through JSON.
export function readYaml(source: string): Document {
  const document = parseDocument(source, { uniqueKeys: true, merge: true })
  const problem = document.errors[0] ?? document.warnings[0]
  if (problem) throw new Error(problem.message)
  return document
}

function equal(left: unknown, right: unknown): boolean {
  if (Object.is(left, right)) return true
  if (Array.isArray(left) && Array.isArray(right))
    return left.length === right.length && left.every((item, index) => equal(item, right[index]))
  if (record(left) && record(right)) {
    const keys = Object.keys(left).filter((key) => left[key] !== undefined)
    return (
      keys.length === Object.keys(right).filter((key) => right[key] !== undefined).length &&
      keys.every((key) => Object.hasOwn(right, key) && equal(left[key], right[key]))
    )
  }
  return false
}

function nodeValue(node: unknown, document: Document): unknown {
  return isNode(node) ? node.toJS(document, { maxAliasCount: 100 }) : node
}

function matches(node: unknown, value: unknown, document: Document) {
  try {
    return equal(nodeValue(node, document), value)
  } catch {
    // An edited/deleted anchor may invalidate a formerly valid alias. Replace
    // it with its intended value rather than retaining a dangling reference.
    return false
  }
}

function record(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function itemID(value: unknown): unknown {
  return isMap(value) ? value.get('id') : record(value) ? value.id : undefined
}

function patchNode(node: unknown, value: unknown, document: Document): Node {
  if (isNode(node) && matches(node, value, document)) return node
  if (isScalar(node) && (value === null || typeof value !== 'object')) {
    node.value = value
    return node
  }
  if (isMap(node) && record(value)) {
    // Editing a merged mapping materializes inherited values. Explicit fields
    // retain their nodes and comments; the resulting file is still YAML.
    node.items = node.items.filter((pair) => {
      const key = isScalar(pair.key) ? pair.key.value : pair.key
      return typeof key === 'string' && Object.hasOwn(value, key) && value[key] !== undefined
    })
    for (const [key, next] of Object.entries(value)) {
      if (next !== undefined) node.set(key, patchNode(node.get(key, true), next, document))
    }
    return node
  }
  if (isSeq(node) && Array.isArray(value)) {
    const previous = [...node.items]
    const used = new Set<unknown>()
    const desiredIDs = new Set<unknown>(value.map(itemID).filter((id) => id !== undefined))
    node.items = value.map((next, index) => {
      const id = itemID(next)
      let candidate = previous.find(
        (item) => !used.has(item) && (id !== undefined ? itemID(item) === id : matches(item, next, document)),
      )
      if (!candidate && !used.has(previous[index]) && !desiredIDs.has(itemID(previous[index])))
        candidate = previous[index]
      used.add(candidate)
      return patchNode(candidate, next, document)
    })
    return node
  }
  const replacement = document.createNode(value)
  if (isNode(node)) {
    replacement.comment = node.comment
    replacement.commentBefore = node.commentBefore
    replacement.spaceBefore = node.spaceBefore
    // A changed alias becomes independent, so editing one meaning cannot
    // accidentally change another meaning that shared the same YAML anchor.
    if (!isAlias(node) && 'anchor' in node && 'anchor' in replacement) replacement.anchor = node.anchor
  }
  return replacement
}

export function writeYaml(value: unknown, source?: string) {
  const document = readYaml(source ?? '')
  if (source && equal(document.toJS({ maxAliasCount: 100 }), value)) return source
  document.contents = patchNode(document.contents, value, document)
  // Sequence edits may remove or move anchors after their aliases were visited.
  // Reconcile against the completed tree before writing any file contents.
  document.contents = patchNode(document.contents, value, document)
  const result = document.toString({ lineWidth: 0 })
  if (!equal(readYaml(result).toJS({ maxAliasCount: 100 }), value)) {
    throw new Error('Unable to preserve the glossary values in YAML. Use the raw YAML view to review the document.')
  }
  return result
}
