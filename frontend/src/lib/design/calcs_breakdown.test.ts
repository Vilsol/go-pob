// @vitest-environment jsdom
// CalcBreakdownControl.lua:174-209
import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, within } from '@testing-library/svelte';
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
