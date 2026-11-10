// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, within, waitFor } from '@testing-library/svelte';
import BuySimilarDialog from './BuySimilarDialog.svelte';
import type { TradeOptions } from './trade';

afterEach(cleanup);

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
}

// CompareBuySimilar.lua:436-444, 118-137
describe('Buy Similar defences', () => {
  it('searches the ticked defence from its minimum', async () => {
    const urls: string[] = [];
    const { getByRole } = render(BuySimilarDialog, {
      open: true,
      item: { rarity: 'RARE', title: 'Doom Shell', name: 'Doom Shell', baseName: 'Vaal Regalia', type: 'Body Armour' },
      slot: 'Body Armour',
      defences: [
        { label: 'Evasion', value: 312.6, tradeKey: 'ev' },
        { label: 'Energy Shield', value: 450, tradeKey: 'es' }
      ],
      onsearch: (u: string) => urls.push(u)
    });
    expect((getByRole('textbox', { name: 'Evasion min' }) as HTMLInputElement).value).toBe('312');
    await fireEvent.click(getByRole('checkbox', { name: 'Energy Shield' }));
    await fireEvent.click(getByRole('button', { name: 'Search' }));
    const q = JSON.parse(decodeURIComponent(urls[0].split('?q=')[1])) as { query: { filters: Record<string, unknown> } };
    expect(q.query.filters.armour_filters).toEqual({ filters: { es: { min: 450 } } });
  });
});

// CompareBuySimilar.lua:134-175, 459-516 (PoB-PoE2)
describe('Buy Similar mods', () => {
  it('sends the checked rows with their boxes, and a row without trade ids cannot be checked', async () => {
    const sent: unknown[] = [];
    const { getByRole } = render(BuySimilarDialog, {
      open: true,
      item: { rarity: 'RARE', title: '', name: 'Doom Shell', baseName: 'Vaal Cuirass', type: 'Body Armour' },
      slot: 'Body Armour',
      mods: [
        { type: 'explicit', lines: ['^x8888FF+90 to maximum Life'], canSearch: true, hasBoxes: true, min: '90' },
        { type: 'explicit', lines: ['^8Grants Skill: Level 20 Fireball'], canSearch: false, hasBoxes: false, min: '' },
        { type: 'explicit', lines: ['^x8888FF20% reduced Light Radius'], canSearch: true, hasBoxes: true, min: '20' }
      ],
      url: (o: TradeOptions) => {
        sent.push(o.mods);
        return Promise.resolve('https://www.pathofexile.com/trade2/search/poe2/Standard/x');
      },
      onsearch: () => {}
    });
    const mods = within(getByRole('group', { name: 'Mods' }));
    expect(mods.getByRole<HTMLInputElement>('checkbox', { name: 'Grants Skill: Level 20 Fireball' }).disabled).toBe(true);
    expect(mods.getByRole<HTMLInputElement>('textbox', { name: '+90 to maximum Life min' }).value).toBe('90');
    await fireEvent.click(mods.getByRole('checkbox', { name: '20% reduced Light Radius' }));
    await fireEvent.input(mods.getByRole('textbox', { name: '20% reduced Light Radius max' }), { target: { value: '-2.5' } });
    await fireEvent.click(getByRole('button', { name: 'Search' }));
    await waitFor(() => expect(sent).toHaveLength(1));
    expect(sent[0]).toEqual([
      { checked: false, min: 90, max: undefined },
      { checked: false, min: undefined, max: undefined },
      { checked: true, min: 20, max: -2.5 }
    ]);
  });
});
