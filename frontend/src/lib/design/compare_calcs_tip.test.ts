// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';
import CompareCalcs from './CompareCalcs.svelte';
import type { CompareCalcSection } from '$lib/display/compare_calcs';

afterEach(cleanup);

// CompareCalcsHelpers.lua:167-313
describe('the Compare › Calcs row tooltip', () => {
  const sd = { ModName: ['Life'], Multi: false, ModType: 'BASE', Cfg: '', ModSource: '', Enemy: false, Actor: '' };
  const sections: CompareCalcSection[] = [
    {
      id: 'Life',
      color: '',
      blocks: [{ label: 'Life', rows: [{ label: 'Base from Gear', a: '40', b: '50', tip: ['Base from Gear', 'Mine:', '  x'], mods: [{ label: '', sd }] }] }]
    }
  ];
  const row = (Key: string, Value: number) => ({ Key, Value, Display: String(Value), SourceType: 'Item', SourceName: Key, ModName: '' });

  it("lists the breakdown and both builds' modifier rows", async () => {
    const asked: number[] = [];
    const { container, getByRole } = render(CompareCalcs, {
      sections,
      onlyDifferences: false,
      nameA: 'Mine',
      nameB: 'Theirs',
      modrows: (side: 0 | 1) => (asked.push(side), Promise.resolve(side === 0 ? [row('Helm', 40)] : [row('Boots', 50)]))
    });
    await fireEvent.mouseEnter(getByRole('row', { name: /Base from Gear/ }));
    await waitFor(() => expect(container.ownerDocument.querySelector('[role="tooltip"]')?.textContent).toContain('Theirs only:'));
    const lines = [...container.ownerDocument.querySelectorAll('[role="tooltip"] [data-line]')].map((l) => l.textContent?.trim());
    expect(lines).toEqual([
      'Base from Gear',
      'Mine:',
      'x',
      'Player modifiers:',
      'Mine only:',
      '40         Item   Helm',
      'Theirs only:',
      '50         Item   Boots'
    ]);
    expect(asked.sort()).toEqual([0, 1]);
  });
});
