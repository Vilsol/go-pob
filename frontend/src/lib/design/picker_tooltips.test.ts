// @vitest-environment jsdom
// ItemsTab.lua:818-930

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte';
import AffixControls from './AffixControls.svelte';
import AddModifierDialog from './AddModifierDialog.svelte';
import CraftItemDialog from './CraftItemDialog.svelte';
import type { AffixControls as Affixes, CraftApi } from './crafting';
import type { ImpactBlock, TipProps } from './items';

afterEach(cleanup);

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
}

const block = (line: string): ImpactBlock => ({
  header: '\nAdding this mod will give: ',
  sections: [{ lines: [{ label: line, value: '+1', better: true }] }]
});
const tip = (base: string): TipProps => ({ item: 'Test Crown', base, rarity: 'rare', mods: [`${base} implicit`], impactTitle: '', impact: [] });

const affixes: Affixes = {
  sorting: false,
  base: 'Iron Hat',
  bases: ['Iron Hat', 'Hubris Circlet'],
  controls: [
    {
      type: 'Prefix',
      sel: 0,
      sliderShown: false,
      sliderVal: 0.5,
      series: [
        { label: 'None', tiers: 0, haveRange: false, tooltip: [], mid: [] },
        {
          label: '+# to maximum Life',
          tiers: 3,
          haveRange: true,
          tooltip: ['^73 Tiers', '+(10-39) to maximum Life', 'Level: 1 to 30'],
          mid: ['+25 to maximum Life']
        }
      ]
    }
  ]
};

const craftApi = (over: Partial<CraftApi> = {}): CraftApi =>
  ({
    sortStats: () => Promise.resolve([]),
    modCompare: vi.fn((lines: string[]) => Promise.resolve([block(`from ${lines.join('|')}`)])),
    rebaseTip: vi.fn((base: string) => Promise.resolve(tip(base))),
    ...over
  }) as unknown as CraftApi;

const open = async (name: string) => {
  const box = screen.getByRole('combobox', { name });
  await fireEvent.click(box);
  return within(screen.getByRole('listbox', { name }));
};

describe('picker tooltips', () => {
  it("an affix entry's tiers and what adding its mid tier gives; None has none", async () => {
    const craft = craftApi();
    render(AffixControls, { affixes, craft });
    const list = await open('Affix 1');
    await fireEvent.mouseEnter(list.getByRole('option', { name: '+# to maximum Life' }));
    const t = await screen.findByRole('tooltip', { name: 'Affix tooltip' });
    expect(t.textContent).toContain('3 Tiers');
    expect(t.textContent).toContain('+(10-39) to maximum Life');
    await vi.waitFor(() => expect(t.textContent).toContain('from +25 to maximum Life'));
    expect(t.getAttribute('aria-busy')).toBe('false');
    expect(craft.modCompare).toHaveBeenCalledWith(['+25 to maximum Life']);

    await fireEvent.mouseEnter(list.getByRole('option', { name: 'None' }));
    expect(screen.queryByRole('tooltip', { name: 'Affix tooltip' })).toBeNull();
    await fireEvent.mouseEnter(list.getByRole('option', { name: '+# to maximum Life' }));
    await vi.waitFor(() => expect(screen.getByRole('tooltip', { name: 'Affix tooltip' }).textContent).toContain('from +25'));
    expect(craft.modCompare).toHaveBeenCalledTimes(1);
  });

  it("a base entry's item on that base", async () => {
    const craft = craftApi();
    render(AffixControls, { affixes, craft });
    const list = await open('Item base');
    await fireEvent.mouseEnter(list.getByRole('option', { name: 'Hubris Circlet' }));
    await vi.waitFor(() => expect(document.body.textContent).toContain('Hubris Circlet implicit'));
    expect(craft.rebaseTip).toHaveBeenLastCalledWith('Hubris Circlet');
  });

  it('the newest hover wins over a slower answer for an earlier one', async () => {
    let release: (b: ImpactBlock[]) => void = () => {};
    const craft = craftApi({
      modCompare: vi.fn((lines: string[]) => (lines[0].includes('Life') ? new Promise<ImpactBlock[]>((r) => (release = r)) : Promise.resolve([block('fast')])))
    });
    const two: Affixes = {
      ...affixes,
      controls: [
        {
          ...affixes.controls[0],
          series: [...affixes.controls[0].series, { label: '+# to maximum Mana', tiers: 1, haveRange: false, tooltip: ['Mana'], mid: ['+9 to maximum Mana'] }]
        }
      ]
    };
    render(AffixControls, { affixes: two, craft });
    const list = await open('Affix 1');
    await fireEvent.mouseEnter(list.getByRole('option', { name: '+# to maximum Life' }));
    await fireEvent.mouseEnter(list.getByRole('option', { name: '+# to maximum Mana' }));
    await vi.waitFor(() => expect(screen.getByRole('tooltip', { name: 'Affix tooltip' }).textContent).toContain('fast'));
    release([block('slow')]);
    await new Promise((r) => setTimeout(r, 0));
    expect(screen.getByRole('tooltip', { name: 'Affix tooltip' }).textContent).not.toContain('slow');
  });

  it("Add Modifier's hovered modifier: its lines and what adding it gives", async () => {
    const craft = craftApi({
      customModChoices: () =>
        Promise.resolve({
          sources: [{ id: 'BENCH', label: 'Crafting Bench' }],
          mods: { BENCH: [{ id: 'a', label: 'Life', lines: ['+(15-25) to maximum Life'], modTags: [] }] }
        }),
      customModText: () => Promise.resolve(undefined)
    });
    render(AddModifierDialog, { open: true, craft });
    const list = await (async () => {
      await vi.waitFor(() => screen.getByRole('combobox', { name: 'Modifier' }));
      return open('Modifier');
    })();
    await fireEvent.mouseEnter(list.getByRole('option', { name: 'Life' }));
    const t = await screen.findByRole('tooltip', { name: 'Modifier tooltip' });
    expect(t.textContent).toContain('+(15-25) to maximum Life');
    await vi.waitFor(() => expect(t.textContent).toContain('from +(15-25) to maximum Life'));
  });

  it("Craft Item's hovered base stands beside its entry, the chosen one below the fields", async () => {
    render(CraftItemDialog, { open: true, groups: [{ type: 'Helmet', bases: ['Iron Hat', 'Hubris Circlet'] }], preview: tip('Iron Hat'), oncreate: () => {} });
    const inline = () => screen.getByRole('group', { name: 'Base preview' });
    expect(inline().firstElementChild?.classList.contains('fixed')).toBe(false);
    const list = await open('Base');
    await fireEvent.mouseEnter(list.getByRole('option', { name: 'Hubris Circlet' }));
    await vi.waitFor(() => expect(inline().querySelector('.fixed')).not.toBeNull());
  });
});
