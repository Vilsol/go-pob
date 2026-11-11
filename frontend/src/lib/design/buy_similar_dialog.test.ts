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
  it('searches the checked rows with their boxes within the click, and a row without trade ids cannot be checked', async () => {
    const opened: string[] = [];
    const { getByRole } = render(BuySimilarDialog, {
      open: true,
      item: { rarity: 'RARE', title: '', name: 'Doom Shell', baseName: 'Vaal Cuirass', type: 'Body Armour' },
      slot: 'Body Armour',
      mods: [
        { type: 'explicit', lines: ['^x8888FF+90 to maximum Life'], canSearch: true, hasBoxes: true, min: '90' },
        { type: 'explicit', lines: ['^8Grants Skill: Level 20 Fireball'], canSearch: false, hasBoxes: false, min: '' },
        { type: 'explicit', lines: ['^x8888FF20% reduced Light Radius'], canSearch: true, hasBoxes: true, min: '20' }
      ],
      // the URL is built as the selections change, so the click opens it without waiting
      url: (o: TradeOptions) => Promise.resolve(`https://www.pathofexile.com/trade2/search/poe2/Standard/${JSON.stringify(o.mods)}`),
      onsearch: (u: string) => opened.push(u)
    });
    const mods = within(getByRole('group', { name: 'Mods' }));
    expect(mods.getByRole<HTMLInputElement>('checkbox', { name: 'Grants Skill: Level 20 Fireball' }).disabled).toBe(true);
    expect(mods.getByRole<HTMLInputElement>('textbox', { name: '+90 to maximum Life min' }).value).toBe('90');
    await fireEvent.click(mods.getByRole('checkbox', { name: '20% reduced Light Radius' }));
    await fireEvent.input(mods.getByRole('textbox', { name: '20% reduced Light Radius max' }), { target: { value: '-2.5' } });
    const search = getByRole('button', { name: 'Search' }) as HTMLButtonElement;
    await waitFor(() => expect(search.disabled).toBe(false));
    await fireEvent.click(search);
    expect(opened).toHaveLength(1);
    expect(JSON.parse(opened[0].slice(opened[0].lastIndexOf('/') + 1))).toEqual([
      { checked: false, min: 90 },
      { checked: false },
      { checked: true, min: 20, max: -2.5 }
    ]);
  });

  it('keeps Search disabled while the URL for the current selections is being built', async () => {
    let resolve: (u: string) => void = () => {};
    const opened: string[] = [];
    const { getByRole } = render(BuySimilarDialog, {
      open: true,
      item: { rarity: 'RARE', title: '', name: 'Doom Shell', baseName: 'Vaal Cuirass', type: 'Body Armour' },
      url: () => new Promise<string>((r) => (resolve = r)),
      onsearch: (u: string) => opened.push(u)
    });
    const search = getByRole('button', { name: 'Search' }) as HTMLButtonElement;
    expect(search.disabled).toBe(true);
    resolve('https://www.pathofexile.com/trade/search/Standard?q=x');
    await waitFor(() => expect(search.disabled).toBe(false));
    await fireEvent.click(search);
    expect(opened).toEqual(['https://www.pathofexile.com/trade/search/Standard?q=x']);
  });
});
