// @vitest-environment jsdom

import { describe, expect, it, afterEach } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import DisplayItemControls from './DisplayItemControls.svelte';
import type { DisplayEdit, DisplayInfo, DisplayMod } from './items';
import type { CraftApi } from './crafting';

afterEach(cleanup);

const mod = (line: string, section: string, kind = 'magic', range?: number): DisplayMod => ({
  line,
  kind,
  section,
  hasRange: range !== undefined,
  range: range ?? 0,
  disabled: false,
  removable: kind === 'crafted',
  removal: kind === 'crafted' ? 'Crafted' : ''
});
const series = (label: string) => ({ label, tiers: 3, haveRange: true, tooltip: [], mid: [] });

const info: DisplayInfo = {
  quality: 20,
  hasQuality: true,
  variants: ['Life', 'Mana'],
  variant: 1,
  altVariants: [2],
  sockets: 'R-G B',
  socketLimit: 4,
  defaultColor: 'R',
  canInfluence: true,
  influence: ['Hunter'],
  showsCatalyst: true,
  catalyst: 2,
  catalystQuality: 10,
  enchants: 1,
  anoints: 1,
  anointed: [],
  corruptible: true,
  addImplicit: true,
  addCustom: true,
  addCrucible: true,
  foulborn: [
    { mutable: false, mutated: false },
    { mutable: false, mutated: false },
    { mutable: true, mutated: false },
    { mutable: false, mutated: false }
  ],
  cluster: { skills: [{ id: 'a', label: 'Skill A' }], skill: 'a', nodeCount: 8, min: 8, max: 12 },
  affixes: {
    sorting: true,
    base: 'Carnal Armour',
    bases: ['Carnal Armour', 'Sadist Garb'],
    controls: [
      { type: 'Prefix', series: [series('None'), series('+# to maximum Life')], sel: 1, sliderShown: true, sliderVal: 0.5 },
      { type: 'Suffix', series: [series('None'), series('+#% to Cold Resistance')], sel: 0, sliderShown: false, sliderVal: 0.5 }
    ]
  },
  mods: [
    mod('10% increased maximum Life', 'enchant', 'enchant'),
    mod('+18 to maximum Mana', 'implicit', 'magic', 0.4),
    mod('+94 to maximum Life', 'explicit', 'magic', 0.7),
    mod('+15% to Lightning Resistance', 'explicit', 'crafted', 1)
  ]
};
const craft = {
  sortStats: () => Promise.resolve([]),
  affixRollTooltip: () => Promise.resolve([])
} as unknown as CraftApi;

const controls = async (root: HTMLElement) => {
  const tabs = within(root).queryAllByRole('tab');
  const seen = new Set<string>();
  const take = () => {
    for (const role of ['button', 'combobox', 'slider', 'checkbox', 'spinbutton'])
      for (const el of within(root).queryAllByRole(role)) seen.add(`${role}: ${el.getAttribute('aria-label') ?? el.textContent?.trim()}`);
  };
  take();
  for (const t of tabs) {
    await fireEvent.click(t);
    take();
  }
  for (const t of tabs) seen.delete(`button: ${t.textContent?.trim()}`);
  return [...seen].sort();
};

const TODAY = [
  'button: +15% to Lightning Resistance',
  'button: +18 to maximum Mana',
  'button: +94 to maximum Life',
  'button: 10% increased maximum Life',
  'button: Add Crucible mod...',
  'button: Add Implicit...',
  'button: Add a socket',
  'button: Add modifier...',
  'button: Anoint...',
  'button: Change Enchantment...',
  'button: Corrupt...',
  'button: Remove +15% to Lightning Resistance',
  'checkbox: Foulborn: +94 to maximum Life',
  'checkbox: Link sockets 1 and 2',
  'checkbox: Link sockets 2 and 3',
  'combobox: Affix 1',
  'combobox: Affix 2',
  'combobox: Alt variant',
  'combobox: Catalyst',
  'combobox: Cluster skill',
  'combobox: Influence 1',
  'combobox: Influence 2',
  'combobox: Item base',
  'combobox: Modifier sorting',
  'combobox: Socket 1 colour',
  'combobox: Socket 2 colour',
  'combobox: Socket 3 colour',
  'combobox: Variant',
  'slider: Added Passives',
  'slider: Roll of +15% to Lightning Resistance',
  'slider: Roll of +18 to maximum Mana',
  'slider: Roll of +94 to maximum Life',
  'slider: Roll of affix 1',
  'spinbutton: Catalyst quality',
  'spinbutton: Quality'
];

describe('crafting card layouts', () => {
  for (const layout of ['stepped', 'tabs'] as const) {
    it(`${layout}: every control the card had, and the same edits`, async () => {
      const edits: DisplayEdit[] = [];
      const { container } = render(DisplayItemControls, { info, craft, layout, onedit: (e: DisplayEdit) => edits.push(e) });
      expect(await controls(container)).toEqual(TODAY);
      const c = within(container);
      if (layout === 'stepped') expect(c.getAllByRole('heading', { level: 3 }).map((h) => h.textContent)).toEqual(['Base', 'Implicits', 'Affixes']);
      else expect(c.getAllByRole('tab').map((t) => t.textContent?.trim())).toEqual(['Base', 'Implicits2', 'Affixes2']);

      for (const t of c.queryAllByRole('tab')) {
        await fireEvent.click(t);
        for (const line of info.mods.map((m) => m.line)) {
          const b = c.queryByRole('button', { name: line });
          if (b) await fireEvent.click(b);
          const s = c.queryByRole('slider', { name: `Roll of ${line}` });
          if (s) await fireEvent.change(s, { target: { value: '0.25' } });
        }
      }
      if (layout === 'stepped')
        for (const line of info.mods.map((m) => m.line)) {
          await fireEvent.click(c.getByRole('button', { name: line }));
          const s = c.queryByRole('slider', { name: `Roll of ${line}` });
          if (s) await fireEvent.change(s, { target: { value: '0.25' } });
        }
      expect(edits).toEqual([
        { kind: 'disable', index: 0 },
        { kind: 'disable', index: 1 },
        { kind: 'range', index: 1, float: 0.25 },
        { kind: 'disable', index: 2 },
        { kind: 'range', index: 2, float: 0.25 },
        { kind: 'disable', index: 3 },
        { kind: 'range', index: 3, float: 0.25 }
      ]);
    });
  }
});
