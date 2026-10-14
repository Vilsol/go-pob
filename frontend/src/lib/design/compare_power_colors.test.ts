// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, within } from '@testing-library/svelte';
import ComparePower from './ComparePower.svelte';
import CompareSummary from './CompareSummary.svelte';
import type { PowerRow } from './fixtures';

afterEach(cleanup);

// CompareTab.lua:3062-3072
const rows: PowerRow[] = [
  { cat: 'config', what: '# of ^x7070FFMana ^7spent Recently  (8000 -> (not set))', detail: 'General', tone: 'text', impact: [-5, 0, 0] }
];
const mana = (el: Element) => [...el.querySelectorAll<HTMLElement>('[data-pob-color]')].find((e) => e.textContent === 'Mana ');

describe('the power report config rows', () => {
  it("draws the label's colour codes instead of printing them", () => {
    const { getByRole } = render(ComparePower, { power: rows, metric: 'dps', a: 100, b: 95, onmetric: () => {}, onapply: () => {} });
    const row = within(getByRole('table', { name: 'Power report rows' })).getAllByRole('row')[1];
    expect(row.textContent).not.toMatch(/\^x|\^7/);
    expect(row.textContent).toContain('# of Mana spent Recently  (8000 -> (not set))');
    expect(mana(row)?.style.getPropertyValue('--tone')).toBeTruthy();
  });

  it('draws them in the Summary top 5 too', () => {
    const { getByRole } = render(CompareSummary, { groups: [], power: rows, metric: 'dps', better: false, onmetric: () => {}, onpower: () => {} });
    const top = getByRole('complementary', { name: 'Biggest differences' });
    expect(top.textContent).not.toMatch(/\^x|\^7/);
    expect(mana(top)?.style.getPropertyValue('--tone')).toBeTruthy();
  });
});
