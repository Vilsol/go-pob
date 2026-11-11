// @vitest-environment jsdom
// ItemsTab.lua:166-174

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import ItemsBoard from './ItemsBoard.svelte';

afterEach(cleanup);

const props = (extra: string[]) => ({
  slots: {},
  flasks: {},
  calcFlasks: {},
  fitting: (slot: string) => (slot === 'Ring 3' ? [{ id: 7, label: 'Spare Ring', tone: 'rare' as const }] : []),
  jewels: [],
  jewelOptions: [],
  onedit: () => {},
  onflask: () => {},
  onequip: () => {},
  extra
});

describe('ItemsBoard conditional slots', () => {
  it('draws Ring 3 only when it shows', async () => {
    const { queryByRole, rerender, getByRole } = render(ItemsBoard, props([]));
    expect(queryByRole('region', { name: 'Other slots' })).toBeNull();
    await rerender(props(['Ring 3']));
    const region = getByRole('region', { name: 'Other slots' });
    expect(region.querySelector('[aria-label="Ring 3: Empty"]')).not.toBeNull();
  });
});

describe('ItemsBoard item names', () => {
  it('draws colour codes in a user-written name as colour, and leaves them out of the label', () => {
    const item = { name: '^xFF0000Red ^7Ring', base: 'Ruby Ring', rarity: 'rare' as const, sockets: '', implicits: [], explicits: [], flask: false, fits: [] };
    const { getByRole } = render(ItemsBoard, { ...props(['Ring 3']), slots: { 'Ring 3': { id: 1, item }, Helmet: { id: 2, item } } });
    expect(getByRole('button', { name: 'Helmet: Red Ring' })).toBeTruthy();
    const ring = getByRole('button', { name: 'Ring 3: Red Ring' });
    expect(ring.textContent).not.toContain('^');
    expect(ring.querySelector<HTMLElement>('[data-pob-color] [data-pob-color]')?.style.getPropertyValue('--tone')).toBe('#FF0000');
  });
});
