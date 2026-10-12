import { describe, expect, it } from 'vitest';
import { modRowsEqual } from './compare_mods';

// CompareTab.lua:4617-4633
describe('modRowsEqual', () => {
  const row = (Key: string, Value: number) => ({ Key, Value, Display: '', SourceType: '', SourceName: '', ModName: '' });
  it('compares the rows as a multiset of key and value', () => {
    expect(modRowsEqual([row('x', 1), row('y', 2), row('x', 1)], [row('x', 1), row('x', 1), row('y', 2)])).toBe(true);
    expect(modRowsEqual([row('x', 1), row('y', 2)], [row('x', 1), row('y', 3)])).toBe(false);
    expect(modRowsEqual([row('x', 1), row('x', 1)], [row('x', 1), row('y', 1)])).toBe(false);
    expect(modRowsEqual([row('x', 1)], [])).toBe(false);
    expect(modRowsEqual([], [])).toBe(true);
  });
});
