// @vitest-environment jsdom
// Main.lua:1032-1035
// ItemsTab.lua:1025-1089

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import DisplayItemControls from './DisplayItemControls.svelte';
import { allAffixSliders, type DisplayEdit, type DisplayInfo, type DisplayMod } from './items';

afterEach(() => {
  cleanup();
  allAffixSliders.set(true);
});

const mod = (line: string, hasRange: boolean, range = 0.5): DisplayMod => ({
  line,
  kind: 'explicit',
  hasRange,
  range,
  disabled: false,
  removable: false,
  removal: ''
});
const info: DisplayInfo = {
  quality: 0,
  hasQuality: false,
  variants: [],
  variant: 0,
  altVariants: [],
  sockets: '',
  socketLimit: 0,
  defaultColor: 'W',
  canInfluence: false,
  influence: [],
  showsCatalyst: false,
  catalyst: 0,
  catalystQuality: 0,
  mods: [mod('+(80-100) to maximum Life', true, 0.25), mod('Cannot be Frozen', false), mod('(20-30)% increased Attack Speed', true, 0.75)]
};

describe('DisplayItemControls affix sliders', () => {
  it('on: one roll slider per ranged line', () => {
    const { getAllByRole, queryByRole } = render(DisplayItemControls, { info, onedit: () => {} });
    expect(getAllByRole('slider').map((s) => s.getAttribute('aria-label'))).toEqual([
      'Roll of +(80-100) to maximum Life',
      'Roll of (20-30)% increased Attack Speed'
    ]);
    expect(queryByRole('combobox', { name: 'Modifier range' })).toBeNull();
  });

  it('off: a menu of the ranged lines, one slider for the picked one, its edit names that line', async () => {
    allAffixSliders.set(false);
    const edits: DisplayEdit[] = [];
    const { getAllByRole, getByRole } = render(DisplayItemControls, { info, onedit: (e: DisplayEdit) => edits.push(e) });
    const pick = getByRole('combobox', { name: 'Modifier range' }) as HTMLSelectElement;
    expect([...pick.options].map((o) => o.textContent)).toEqual(['+(80-100) to maximum Life', '(20-30)% increased Attack Speed']);
    const sliders = getAllByRole('slider') as HTMLInputElement[];
    expect(sliders.length).toBe(1);
    expect(sliders[0].value).toBe('0.25');
    await fireEvent.change(pick, { target: { value: '2' } });
    const slider = getByRole('slider', { name: 'Roll of (20-30)% increased Attack Speed' }) as HTMLInputElement;
    expect(slider.value).toBe('0.75');
    await fireEvent.change(slider, { target: { value: '0.5' } });
    expect(edits).toEqual([{ kind: 'range', index: 2, float: 0.5 }]);
  });
});
