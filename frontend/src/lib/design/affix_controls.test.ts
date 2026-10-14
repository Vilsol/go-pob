// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte';
import AffixControls from './AffixControls.svelte';
import type { AffixControls as AffixData, CraftApi } from './crafting';

afterEach(cleanup);

const series = (label: string) => ({ label, tiers: 1, haveRange: false, tooltip: [], mid: [] });
const affixes = (base: string): AffixData => ({
  sorting: false,
  controls: [{ type: 'Prefix', series: [series('None'), series('+# to maximum Life')], sel: 1, sliderShown: false, sliderVal: 0.5 }],
  base,
  bases: ['Hubris Circlet', 'Royal Burgonet', 'Lion Pelt']
});

describe('AffixControls', () => {
  it('lists the bases of the type on the item’s own; choosing one rebases, and the dropped affixes show on the new base', async () => {
    const rebase = vi.fn((b: string) => Promise.resolve(b === 'Royal Burgonet' ? ['(26-30)% increased Energy Shield'] : []));
    const craft = { sortStats: () => Promise.resolve([]), rebase } as unknown as CraftApi;
    const view = render(AffixControls, { affixes: affixes('Hubris Circlet'), craft });
    const box = screen.getByRole('combobox', { name: 'Item base' });
    expect(box.textContent).toBe('Hubris Circlet');
    await fireEvent.click(box);
    expect(
      within(screen.getByRole('listbox', { name: 'Item base' }))
        .getAllByRole('option')
        .map((o) => o.textContent?.trim())
    ).toEqual(['Hubris Circlet', 'Royal Burgonet', 'Lion Pelt']);
    await fireEvent.click(screen.getByRole('option', { name: 'Royal Burgonet' }));
    expect(rebase).toHaveBeenCalledWith('Royal Burgonet');
    await view.rerender({ affixes: affixes('Royal Burgonet'), craft });
    const note = await screen.findByRole('status', { name: 'Dropped affixes' });
    expect(note.textContent).toContain('Royal Burgonet cannot roll');
    expect(note.textContent).toContain('(26-30)% increased Energy Shield');
    expect(screen.getByRole('combobox', { name: 'Item base' }).textContent).toBe('Royal Burgonet');
  });

  it('has no Base dropdown when the item has no other base', () => {
    const craft = { sortStats: () => Promise.resolve([]) } as unknown as CraftApi;
    render(AffixControls, { affixes: { ...affixes('Hubris Circlet'), bases: ['Hubris Circlet'] }, craft });
    expect(screen.queryByRole('combobox', { name: 'Item base' })).toBeNull();
  });
});
