import { expect, test } from 'vitest';

import { parseLog, plan, units } from './shard.mjs';

const t = (file, ...titles) => ({ file, titles });

test('a test with › in a title is listed under the titles before it, with its siblings', () => {
  const u = units([t('a.spec.ts', 'layout', 'Compare › Tree'), t('a.spec.ts', 'layout', 'Compare › Config'), t('a.spec.ts', 'other')]);
  expect([...u]).toEqual([
    ['a.spec.ts › other', ['a.spec.ts › other']],
    ['a.spec.ts › layout', ['a.spec.ts › layout › Compare › Tree', 'a.spec.ts › layout › Compare › Config']]
  ]);
});

test('a line that would also select another test takes that test with it', () => {
  const u = units([t('a.spec.ts', 'x', 'y'), t('a.spec.ts', 'x'), t('a.spec.ts', 'xy')]);
  expect(u.get('a.spec.ts › x')).toEqual(['a.spec.ts › x', 'a.spec.ts › x › y']);
  expect(u.get('a.spec.ts › xy')).toEqual(['a.spec.ts › xy']);
  expect(u.size).toBe(2);
});

test('the longest tests are spread first, and an unrecorded test counts as the median', () => {
  const lines = new Map(['a', 'b', 'c', 'd', 'e'].map((k) => [k, [k]]));
  const { shards, unknown } = plan(lines, { a: 10, b: 6, c: 5, d: 1 }, 2);
  // e is unknown: median of 1, 5, 6, 10 is 6
  expect(unknown).toEqual(['e']);
  expect(shards.map((s) => [s.lines, s.seconds])).toEqual([
    [['a', 'c'], 15],
    [['b', 'e', 'd'], 13]
  ]);
});

test('the log gives each result its duration, without the tags the reporter prints, and a skip counts as 0', () => {
  const log = [
    'E2E (shard 4/4)\tUNKNOWN STEP\t2026-10-23T20:59:59.3Z   \x1b[32m✓\x1b[39m   2 e2e/p.spec.ts:9:3 › build (x) — trap (1.5m)',
    'E2E (shard 1/4)\tUNKNOWN STEP\t2026-10-23T21:00:00.0Z   ✘  69 e2e/f.spec.ts:20:1 › font scaling (0339) @slow @smoke (812ms)',
    'E2E (shard 1/4)\tUNKNOWN STEP\t2026-10-23T21:00:00.0Z   -  29 e2e/w.spec.ts:159:1 › workers › at 1, 2 and 4',
    'E2E (shard 1/4)\tUNKNOWN STEP\t2026-10-23T21:00:00.0Z \x1b[32m✓\x1b[39m 4 modules transformed.'
  ].join('\n');
  expect(parseLog(log)).toEqual({
    'p.spec.ts › build (x) — trap': 90,
    'f.spec.ts › font scaling (0339)': 0.812,
    'w.spec.ts › workers › at 1, 2 and 4': 0
  });
});

test("a serial file's tests stay on one line, so one shard runs them in order", () => {
  const u = units([t('s.spec.ts', 'first'), t('s.spec.ts', 'second'), t('a.spec.ts', 'other')], new Set(['s.spec.ts']));
  expect([...u]).toEqual([
    ['s.spec.ts', ['s.spec.ts › first', 's.spec.ts › second']],
    ['a.spec.ts › other', ['a.spec.ts › other']]
  ]);
});
