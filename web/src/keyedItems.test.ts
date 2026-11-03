import { expect, it } from 'vitest'
import { keyedItems } from './keyedItems'

it('retains duplicate items and stable identities when unrelated content is inserted', () => {
  const before = keyedItems(['alpha', 'beta', 'alpha'], (value) => value)
  const after = keyedItems(['new', 'alpha', 'beta', 'alpha'], (value) => value)
  expect(new Set(before.map((value) => value.key)).size).toBe(3)
  expect(after.slice(1).map((value) => value.key)).toEqual(before.map((value) => value.key))
  expect(before.map((value) => value.item)).toEqual(['alpha', 'beta', 'alpha'])
})
