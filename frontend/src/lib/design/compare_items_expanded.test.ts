// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor, within } from '@testing-library/svelte';
import CompareItems from './CompareItems.svelte';

afterEach(cleanup);

const rows = [
  {
    slot: 'Helmet',
    a: { name: 'Mine Hood', rarity: 'rare' as const, line: '+50 to maximum Life' },
    b: { name: 'Their Crown', rarity: 'unique' as const, line: '+1 to Level of all Minion Skill Gems' }
  },
  { slot: 'Boots', a: { name: 'Same Boots', rarity: 'rare' as const, line: '' }, b: { name: 'Same Boots', rarity: 'rare' as const, line: '' } }
];
const jewels = [
  {
    slot: 'Jewel 26725',
    label: 'Socket #1',
    node: 26725,
    a: { name: 'Mine Eye', rarity: 'rare' as const, line: '' },
    b: null,
    aWarn: false,
    bWarn: false
  },
  {
    slot: 'Jewel 33631',
    label: 'Socket',
    node: 33631,
    a: null,
    b: { name: 'Their Eye', rarity: 'magic' as const, line: '' },
    aWarn: false,
    bWarn: true
  }
];

describe('Compare › Items expanded mode and jewels', () => {
  // CompareTab.lua:766-773, 3816-3843
  it('Expanded mode shows both cards with their diff markers', async () => {
    const asked: string[] = [];
    const { getByRole, findByRole } = render(CompareItems, {
      items: rows,
      jewels,
      onitem: () => {},
      cards: (slot: string) => (
        asked.push(slot),
        Promise.resolve([
          ['^xFFFF77Mine Hood, Hubris Circlet', '^x33FF77+ ^x8888FF+50 to maximum Life'],
          ['^xAF6025Their Crown, Bone Helmet', '^xDD0022< ^x8888FF+20 to maximum Life']
        ] as [string[], string[]])
      )
    });
    await fireEvent.click(getByRole('checkbox', { name: 'Expanded mode' }));
    const helmet = await findByRole('group', { name: 'Helmet' });
    await waitFor(() => expect(within(helmet).getAllByRole('list')).toHaveLength(2));
    const [a, b] = within(helmet).getAllByRole('list');
    expect([...a.querySelectorAll('li')].map((l) => l.textContent)).toEqual(['Mine Hood, Hubris Circlet', '+ +50 to maximum Life']);
    expect([...b.querySelectorAll('li')].map((l) => l.textContent)).toEqual(['Their Crown, Bone Helmet', '< +20 to maximum Life']);
    // TradeHelpers.lua:359-372
    expect(helmet.textContent).toContain('(different)');
    expect((await findByRole('group', { name: 'Boots' })).textContent).toContain('(match)');
    expect(within(helmet).getByRole('button', { name: 'Equip' })).toBeTruthy();
    expect(asked).toContain('Helmet');
  });

  // CompareTab.lua:3926-3940
  it('lists the jewel sockets with the missing-node warning', () => {
    const { getByRole } = render(CompareItems, { items: rows, jewels, onitem: () => {} });
    const section = getByRole('rowgroup', { name: 'Jewels' });
    const rowsShown = within(section)
      .getAllByRole('row')
      .filter((r) => r.querySelector('[role=cell]'));
    expect(rowsShown.map((r) => r.querySelector('[role=cell]')?.textContent)).toEqual(['Socket #1', 'Socket']);
    expect(rowsShown[1].textContent).toContain('(tree missing allocated node)');
    expect(rowsShown[0].textContent).not.toContain('(tree missing allocated node)');
  });

  // CompareTab.lua:3857-3872
  it('hovering a jewel socket label asks for the tree around it', async () => {
    const asked: number[] = [];
    const { getByText } = render(CompareItems, {
      items: rows,
      jewels,
      socketview: (node: number) => (asked.push(node), Promise.resolve(undefined))
    });
    await fireEvent.mouseEnter(getByText('Socket #1'));
    await waitFor(() => expect(asked).toEqual([26725]));
  });
});
