// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import BuySimilarDialog from './BuySimilarDialog.svelte';

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
