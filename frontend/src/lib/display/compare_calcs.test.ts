// CompareTab.lua:4692-4740
import { describe, expect, it } from 'vitest';
import type { Outputs } from '$lib/custom_types';
import type { CalcSection } from '$lib/calcs/calc_sections';
import { compareCalcsView, modEntries, modKey, modTipLines } from './compare_calcs';

const SECTIONS = [
  [
    1,
    'Attack',
    1,
    '^xE07030',
    [{ defaultCollapsed: false, label: 'Attack', data: { flag: 'attack', 1: { label: 'Speed', 0: { format: '{2:output:Speed}' } } } }]
  ],
  [
    1,
    'Life',
    2,
    '^x7070FF',
    [
      {
        defaultCollapsed: false,
        label: 'Life',
        data: {
          extra: '{0:output:Life}',
          1: { label: 'Total', 0: { format: '{0:output:Life}', 0: { breakdown: 'Life' } } },
          2: { label: 'Regen', 0: { format: '{1:output:LifeRegen}' } },
          3: { label: 'Totems', flag: 'totem', 0: { format: '{0:output:TotemLife}' } },
          4: { label: 'Base from Gear', 0: { format: '{0:mod:1}', 0: { modName: 'Life', modType: 'BASE' } } },
          5: { 0: { format: 'no label' } },
          6: { label: 'No format', 0: { 0: { format: '{0:output:Life}' } } }
        }
      },
      { defaultCollapsed: false, label: 'Mana', data: { extra: '{0:output:Mana}', 1: { label: 'Regen', 0: { format: '{1:output:ManaRegen}' } } } }
    ]
  ]
] as unknown as CalcSection[];

const out = (o: Record<string, number>, flags: Record<string, boolean>, lines: string[], gear: number): Outputs =>
  ({
    Output: o,
    SkillFlags: flags,
    Breakdown: { Life: { Lines: lines } },
    Calcs: { 'Life:0:3:0:0': gear }
  }) as unknown as Outputs;

const A = out({ Speed: 1.5, Life: 1000, LifeRegen: 10, Mana: 50, ManaRegen: 2 }, { attack: false }, ['x', 'y'], 40);
const B = out({ Speed: 2, Life: 1000, LifeRegen: 12, TotemLife: 300, Mana: 60, ManaRegen: 2 }, { attack: true, totem: true }, ['x', 'z'], 40);

describe('compareCalcsView', () => {
  it('shows every row either build shows, A and B formatted from their own outputs', () => {
    const v = compareCalcsView(SECTIONS, A, B, false);
    expect(v.map((s) => s.id)).toEqual(['Life']);
    const [life, mana] = v[0].blocks;
    expect(life.label).toBe('Life');
    expect(life.extra).toEqual({ a: '1,000', b: '1,000' });
    expect(life.rows).toEqual([
      { label: 'Total', a: '1,000', b: '1,000', tip: ['Total', 'A:', '  x', '  y', 'B:', '  x', '  z'] },
      { label: 'Regen', a: '10', b: '12' },
      { label: 'Totems', a: '0', b: '300' },
      {
        label: 'Base from Gear',
        a: '40',
        b: '40',
        mods: [{ label: '', sd: { ModName: ['Life'], Multi: false, ModType: 'BASE', Cfg: '', ModSource: '', Enemy: false, Actor: '' } }]
      }
    ]);
    expect(mana.rows).toEqual([{ label: 'Regen', a: '2', b: '2' }]);
  });

  // CompareCalcsHelpers.lua:185-210
  it("gives a row with a breakdown both builds' lines", () => {
    const total = compareCalcsView(SECTIONS, A, B, false, 'Mine', 'Theirs')[0].blocks[0].rows[0];
    expect(total.tip).toEqual(['Total', 'Mine:', '  x', '  y', 'Theirs:', '  x', '  z']);
  });

  it('with only differences, drops rows both show alike and sub-sections left empty whose extra matches', () => {
    const v = compareCalcsView(SECTIONS, A, B, true);
    expect(v[0].blocks.map((b) => b.label)).toEqual(['Life', 'Mana']);
    expect(v[0].blocks[0].rows.map((r) => r.label)).toEqual(['Total', 'Regen', 'Totems', 'Base from Gear']);
    expect(v[0].blocks[1].rows).toEqual([]);
    const same = compareCalcsView(SECTIONS, A, out({ Speed: 1.5, Life: 1000, LifeRegen: 10, Mana: 50, ManaRegen: 2 }, {}, ['x', 'y'], 40), true);
    expect(same[0].blocks.map((b) => b.rows.map((r) => r.label))).toEqual([['Base from Gear']]);
  });

  // CompareTab.lua:4617-4633
  it('drops a modifier row whose entries both builds tabulate alike', () => {
    const sds = modEntries(SECTIONS);
    expect(sds.map((sd) => sd.ModName)).toEqual([['Life']]);
    const twin = out({ Speed: 1.5, Life: 1000, LifeRegen: 10, Mana: 50, ManaRegen: 2 }, {}, ['x', 'y'], 40);
    const view = (eq: boolean) => compareCalcsView(SECTIONS, A, twin, true, 'A', 'B', new Map([[modKey(sds[0]), eq]]));
    expect(view(true)).toEqual([]);
    expect(view(false)[0].blocks.map((b) => b.rows.map((r) => r.label))).toEqual([['Base from Gear']]);
  });
});

// CompareCalcsHelpers.lua:212-310
describe('modTipLines', () => {
  const row = (Key: string, Value: number, SourceName = Key) => ({ Key, Value, Display: String(Value), SourceType: 'Item', SourceName, ModName: '' });
  it('lists common rows with both values when they differ, then each side only, largest first', () => {
    const lines = modTipLines('', [row('x', 5), row('y', 2), row('p', 1), row('q', 3)], [row('y', 4), row('x', 5), row('c', 7)], 'Mine', 'Theirs');
    expect(lines).toEqual([
      '^7Player modifiers:',
      '^x808080  Common:',
      '    ^75          ^7Item   ^7x',
      '    ^#33FF772    ^7/^#FF99224     ^7Item   ^7y',
      '^#33FF77  Mine only:',
      '    ^73          ^7Item   ^7q',
      '    ^71          ^7Item   ^7p',
      '^#FF9922  Theirs only:',
      '    ^77          ^7Item   ^7c'
    ]);
  });
  it('has no lines when neither build has a row', () => {
    expect(modTipLines('Sources', [], [], 'A', 'B')).toEqual([]);
  });
});
