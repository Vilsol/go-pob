// @vitest-environment jsdom
// CalcBreakdownControl.lua:174-209
import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, within, fireEvent } from '@testing-library/svelte';
import CalcsBreakdown from './CalcsBreakdown.svelte';

afterEach(cleanup);

describe('CalcsBreakdown tables', () => {
  it('draws each table with its columns and rows', () => {
    const { getByRole } = render(CalcsBreakdown, {
      id: 'bd',
      title: 'Mana Reserved',
      pinned: false,
      onunpin: () => {},
      parts: [
        {
          text: '37.5%',
          bd: {
            lines: [],
            tables: [
              {
                label: 'Reservations',
                columns: [
                  { label: 'Skill', key: 'skillName' },
                  { label: 'Reservation', key: 'total' }
                ],
                rows: [{ skillName: 'Hatred', total: '37.5%' }]
              }
            ]
          }
        }
      ]
    });
    const table = getByRole('table', { name: 'Reservations' });
    expect(
      within(table)
        .getAllByRole('columnheader')
        .map((h) => h.textContent)
    ).toEqual(['Skill', 'Reservation']);
    expect(
      within(table)
        .getAllByRole('cell')
        .map((c) => c.textContent)
    ).toEqual(['Hatred', '37.5%']);
  });
});

// CalcBreakdownControl.lua:271-498
describe('CalcsBreakdown modifier tables', () => {
  it('draws each section entry as PoB columns it', () => {
    const row = (Value: string, Name: string, Source: string, SourceName: string) => ({
      Value,
      Name,
      Flags: 'Projectile',
      Tags: 'Condition: Full Life',
      Source,
      SourceName,
      Totals: [`Total from ${Source}:`, '8% increased']
    });
    const { getByRole } = render(CalcsBreakdown, {
      id: 'bd',
      title: 'Damage',
      pinned: false,
      onunpin: () => {},
      parts: [
        {
          text: '8%',
          mods: ['Damage INC'],
          modTables: [
            { label: 'Player modifiers', rows: [row('4', '', 'Tree', 'Projectile Damage')] },
            { label: 'Elemental', rows: [row('6', 'Elemental Damage', 'Tree', 'Elemental Damage')] }
          ]
        }
      ]
    });
    const player = getByRole('table', { name: 'Player modifiers' });
    expect(
      within(player)
        .getAllByRole('columnheader')
        .map((h) => h.textContent)
    ).toEqual(['Value', 'Skill types', 'Notes', 'Source', 'Source Name']);
    expect(
      within(player)
        .getAllByRole('cell')
        .map((c) => c.textContent)
    ).toEqual(['4', 'Projectile', 'Condition: Full Life', 'Tree', 'Projectile Damage']);
    expect(within(player).getAllByRole('cell')[3].title).toBe('Total from Tree:\n8% increased');
    expect(
      within(getByRole('table', { name: 'Elemental' }))
        .getAllByRole('columnheader')
        .map((h) => h.textContent)
    ).toContain('Stat');
  });
});

describe('CalcsBreakdown width', () => {
  it('follows a drag and the arrow keys, and comes back at that width', async () => {
    const props = { id: 'bd', pinned: false, onunpin: () => {}, parts: [] };
    const first = render(CalcsBreakdown, props);
    const aside = first.getByRole('complementary', { name: 'Breakdown' });
    const handle = first.getByRole('separator', { name: 'Resize breakdown' });
    const width = () => parseFloat(aside.style.width);
    const w0 = width();
    expect(w0).toBeGreaterThan(0);
    await fireEvent.keyDown(handle, { key: 'ArrowLeft' });
    expect(width()).toBe(w0 + 16);
    if (!('PointerEvent' in window)) Object.assign(window, { PointerEvent: MouseEvent });
    await fireEvent.pointerDown(handle, { clientX: 500, pointerId: 1, button: 0 });
    await fireEvent.pointerMove(handle, { clientX: 400, pointerId: 1 });
    await fireEvent.pointerUp(handle, { clientX: 400, pointerId: 1 });
    expect(width()).toBe(w0 + 116);
    expect(handle.getAttribute('aria-valuenow')).toBe(String(w0 + 116));
    first.unmount();
    const again = render(CalcsBreakdown, props);
    expect(parseFloat(again.getByRole('complementary', { name: 'Breakdown' }).style.width)).toBe(w0 + 116);
  });
});

describe('CalcsBreakdown reading', () => {
  it('splits the title and lays the calculation out as steps', () => {
    const { getByRole, container } = render(CalcsBreakdown, {
      id: 'bd',
      title: 'Life · Total',
      pinned: true,
      onunpin: () => {},
      parts: [{ text: '2,301', bd: { lines: ['1180 (base)', 'x 1.95 (increased/reduced)', '= 2301', 'Next breakpoint: 6% increased'] } }]
    });
    const h = getByRole('heading', { name: 'Life · Total' });
    expect(h.textContent).toBe('Life · Total');
    expect([...h.querySelectorAll('span:not(.sr-only)')].map((s) => s.textContent)).toEqual(['Life', 'Total']);
    const steps = [...container.querySelectorAll('[data-step]')].map((s) => [...s.children].map((c) => c.textContent));
    expect(steps).toEqual([
      ['', '1180', '(base)'],
      ['×', '1.95', '(increased/reduced)'],
      ['=', '2301', '']
    ]);
    expect(within(getByRole('group', { name: 'Calculation' })).getByText('Next breakpoint: 6% increased')).toBeTruthy();
  });
});
