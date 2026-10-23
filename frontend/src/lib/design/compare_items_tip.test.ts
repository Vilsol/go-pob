// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import CompareItems from './CompareItems.svelte';
import { setArtResolver } from './item_art';

afterEach(() => {
  cleanup();
  setArtResolver(undefined);
});

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

  it("an item's art that finishes loading after its tooltip closed reads nothing from the closed tooltip", async () => {
    setArtResolver((qs) => Promise.resolve(qs.map(() => 'Art/2DItems/Helmets/Crown')));
    const errors: unknown[] = [];
    const onerror = (e: ErrorEvent) => errors.push(e.error ?? e.message);
    window.addEventListener('error', onerror);
    try {
      const { getByText, container } = render(CompareItems, {
        items: rows,
        itemtip: (_side: 'a' | 'b', name: string) => Promise.resolve({ influence: [], title: [name], lines: [{ text: 'Quality: +20%', sep: false }] })
      });
      const name = getByText('Their Crown');
      await fireEvent.mouseEnter(name);
      const doc = container.ownerDocument;
      await waitFor(() => expect(doc.querySelector('#compare-item-tip img')).not.toBeNull());
      const art = doc.querySelector('#compare-item-tip img') as HTMLImageElement;
      await fireEvent.mouseLeave(name);
      await waitFor(() => expect(doc.querySelector('#compare-item-tip')).toBeNull());
      art.dispatchEvent(new Event('load'));
      expect(errors).toEqual([]);
    } finally {
      window.removeEventListener('error', onerror);
    }
  });
});
