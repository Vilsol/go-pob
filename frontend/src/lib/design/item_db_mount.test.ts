// @vitest-environment jsdom

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import ItemDB from './ItemDB.svelte';
import { defaultItemDBFilter, type ItemDBRowPlain } from './item_db';

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  cleanup();
  vi.useRealTimers();
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted component threw').toEqual([]);
});

const row = (i: number): ItemDBRowPlain => ({
  name: `Item ${i}`,
  title: '',
  base: 'Leather Belt',
  rarity: 'UNIQUE',
  type: 'Belt',
  slot: 'Belt',
  level: 1,
  league: '',
  source: '',
  power: 0
});
const listed = () => screen.getByRole('table', { name: 'Unique list' }).querySelectorAll('button[data-pob-color]').length;
const noop = () => {};

describe('the item database list', () => {
  it('draws a long list a slice per task, so no one task holds the page, and ends with every row', async () => {
    const rows = Array.from({ length: 1234 }, (_, i) => row(i));
    render(ItemDB, { kind: 'UNIQUE', filter: defaultItemDBFilter(), sort: '', rows, onadd: noop, onopen: noop, oncopy: noop });
    await tick();
    const first = listed();
    expect(first).toBeGreaterThan(0);
    expect(first, 'the first task drew every row').toBeLessThan(rows.length);
    await vi.runAllTimersAsync();
    await tick();
    expect(listed()).toBe(rows.length);
    expect(screen.getByText('1234 items')).toBeTruthy();
  });
});
