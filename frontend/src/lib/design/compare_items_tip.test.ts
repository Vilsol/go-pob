// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import CompareItems from './CompareItems.svelte';

afterEach(cleanup);

const rows = [
  {
    slot: 'Helmet',
    a: { name: 'Mine Hood', rarity: 'rare' as const, line: '+50 to maximum Life' },
    b: { name: 'Their Crown', rarity: 'unique' as const, line: '+1 to Level of all Minion Skill Gems' },
    impact: [{ lines: [{ label: 'Life', value: '+12', better: true }] }]
  }
];

describe('Compare › Items tooltips', () => {
  // CompareTab.lua:3954-3963
  it("hovering an item shows its build's tooltip of it", async () => {
    const asked: string[] = [];
    const { getByText, container } = render(CompareItems, {
      items: rows,
      itemtip: (side: 'a' | 'b', name: string) => (
        asked.push(`${side}:${name}`), Promise.resolve({ influence: [], title: [name], lines: [{ text: 'Quality: +20%', sep: false }] })
      )
    });
    await fireEvent.mouseEnter(getByText('Their Crown'));
    await waitFor(() => expect(container.ownerDocument.body.textContent).toContain('Quality: +20%'));
    expect(asked).toEqual(['b:Their Crown']);
  });

  // CompareTab.lua:3965-4010
  it('hovering Equip shows what equipping the item gives', async () => {
    const { getByRole, container } = render(CompareItems, { items: rows, onitem: () => {} });
    await fireEvent.mouseEnter(getByRole('button', { name: 'Equip' }));
    const tip = container.ownerDocument.querySelector('#compare-equip-tip');
    expect([...(tip?.querySelectorAll('[data-line]') ?? [])].map((l) => l.textContent)).toEqual([
      'Equipping this item in Helmet will give you:',
      '(replacing Mine Hood)',
      'Life +12'
    ]);
  });
});
