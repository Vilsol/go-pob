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

  it('shows a load failure as an alert in the error colour, and a note in the better colour', async () => {
    const props = { kind: 'UNIQUE' as const, filter: defaultItemDBFilter(), sort: '', rows: [], onadd: noop, onopen: noop, oncopy: noop };
    const { rerender } = render(ItemDB, { ...props, error: 'The item list could not be loaded: boom' });
    await tick();
    const alert = screen.getByRole('alert');
    expect(alert.textContent).toBe('The item list could not be loaded: boom');
    expect(alert.classList.contains('text-error')).toBe(true);
    expect(alert.classList.contains('text-better')).toBe(false);
    await rerender({ ...props, error: '', note: 'Added Item 1' });
    expect(screen.queryByRole('alert')).toBeNull();
    expect(screen.getByRole('status').classList.contains('text-better')).toBe(true);
  });
});
