import type { calculator } from '$lib/types';

// src/Classes/CompareTab.lua:4617-4633
export const modRowsEqual = (pRows: calculator.CompareModRow[], cRows: calculator.CompareModRow[]) => {
  if (pRows.length !== cRows.length) return false;
  const counts = new Map<string, number>();
  for (const r of pRows) counts.set(`${r.Key}|${r.Value}`, (counts.get(`${r.Key}|${r.Value}`) ?? 0) + 1);
  for (const r of cRows) {
    const key = `${r.Key}|${r.Value}`;
    const count = counts.get(key);
    if (!count) return false;
    counts.set(key, count - 1);
  }
  return true;
};
