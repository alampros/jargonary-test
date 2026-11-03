// Preserve duplicate content without using positional React keys. Occurrence
// numbers are scoped to an item's identity, so unrelated insertions/reordering
// do not remount existing items.
export function keyedItems<T>(items: readonly T[], identify: (item: T) => string) {
  const occurrences = new Map<string, number>()
  return items.map((item) => {
    const identity = identify(item)
    const occurrence = occurrences.get(identity) || 0
    occurrences.set(identity, occurrence + 1)
    return { item, key: JSON.stringify([identity, occurrence]) }
  })
}
