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
