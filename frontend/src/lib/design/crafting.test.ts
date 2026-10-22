import { describe, expect, it } from 'vitest';
import {
  corruptOptions,
  nameUnnamedItem,
  craftFitsSlot,
  enchantSkills,
  finitePowers,
  maxOf,
  notableColour,
  notableMatches,
  sortByValue,
  sortNotables,
  type AnointNotable
} from './crafting';

const n = (id: number, name: string, stats: string[] = []): AnointNotable => ({ id, name, stats, reminder: [], recipe: [] });

describe('enchant popup lists (ItemsTab.lua:2688-2814)', () => {
  it('lists the used skills unless All skills, and every skill when none is used', () => {
    const c = {
      haveSkills: true,
      skills: [
        { skill: 'A', sources: [] },
        { skill: 'B', sources: [] }
      ],
      used: ['B']
    };
    expect(enchantSkills(c, false)).toEqual(['B']);
    expect(enchantSkills(c, true)).toEqual(['A', 'B']);
    expect(enchantSkills({ ...c, used: [] }, false)).toEqual(['A', 'B']);
  });
  it('sorts by value, highest first, ties in the default order', () => {
    expect(sortByValue([1, 3, 3, 2], 4)).toEqual([1, 2, 3, 0]);
    expect(sortByValue(undefined, 3)).toEqual([0, 1, 2]);
  });
});

describe('notable list (NotableDBControl.lua)', () => {
  const list = [n(1, 'Zeal', ['+10 to maximum Life']), n(2, 'Acuity', ['10% increased Accuracy']), n(3, 'Bane', ['Curse|ish'])];
  it('searches names, modifiers or both, with "|" alternatives', () => {
    expect(list.filter((x) => notableMatches(x, 'LIFE', 'Anywhere')).map((x) => x.id)).toEqual([1]);
    expect(list.filter((x) => notableMatches(x, 'life', 'Names')).map((x) => x.id)).toEqual([]);
    expect(list.filter((x) => notableMatches(x, 'zeal|acu', 'Names')).map((x) => x.id)).toEqual([1, 2]);
    expect(list.filter((x) => notableMatches(x, '  ', 'Modifiers')).length).toBe(3);
  });
  it('sorts by power, then name', () => {
    expect(sortNotables(list).map((x) => x.name)).toEqual(['Acuity', 'Bane', 'Zeal']);
    expect(
      sortNotables(
        list,
        new Map([
          [1, 5],
          [3, 5]
        ])
      ).map((x) => x.name)
    ).toEqual(['Bane', 'Zeal', 'Acuity']);
  });
  it('makes an infinite power twice the largest finite one', () => {
    const p = finitePowers(
      new Map([
        [1, 4],
        [2, Infinity],
        [3, -1]
      ])
    );
    expect([...p.values()]).toEqual([4, 8, -1]);
    expect(maxOf(p)).toBe(8);
  });
  it('colours by power when sorting by a stat, else as crafted', () => {
    expect(notableColour(undefined, undefined, false)).toBe('^xB8DAF1');
    expect(notableColour(8, 8, true)).toBe('^xFF8080');
    expect(notableColour(0, 8, true)).toBe('^x808080');
    expect(notableColour(4, 8, true)).toBe('^xBF8080');
  });
});

describe('corrupt popup lists (ItemsTab.lua:3177-3218)', () => {
  it('leaves out every group another dropdown chose', () => {
    const m = (id: string, group: string) => ({ id, label: id, lines: [], group, modTags: [] });
    const list = [m('a1', 'A'), m('a2', 'A'), m('b1', 'B'), m('c1', 'C')];
    expect(corruptOptions(list, ['a2', '', 'c1']).map((x) => x.id)).toEqual(['b1']);
    expect(corruptOptions(list, [undefined]).map((x) => x.id)).toEqual(['a1', 'a2', 'b1', 'c1']);
  });
});

describe('craftFitsSlot', () => {
  const GROUPS = [
    'Amulet',
    'Belt',
    'Body Armour',
    'Bow',
    'Dagger: Rune',
    'Flask: Life',
    'Jewel',
    'Jewel: Abyss',
    'Quiver',
    'Ring',
    'Shield: Armour',
    'Staff: Warstaff',
    'Tincture',
    'Wand'
  ];
  const fits = (slot: string) => GROUPS.filter((g) => craftFitsSlot(slot, g));
  it('offers the types each slot takes', () => {
    expect(fits('Ring 2')).toEqual(['Ring']);
    expect(fits('Body Armour')).toEqual(['Body Armour']);
    expect(fits('Flask 3')).toEqual(['Flask: Life', 'Tincture']);
    expect(fits('Weapon 1')).toEqual(['Bow', 'Dagger: Rune', 'Staff: Warstaff', 'Wand']);
    expect(fits('Weapon 2 Swap')).toEqual(['Dagger: Rune', 'Quiver', 'Shield: Armour', 'Wand']);
    expect(fits('Belt Abyssal Socket 1')).toEqual(['Jewel: Abyss']);
  });
});

describe('nameUnnamedItem', () => {
  it('names an unnamed rare after its rarity and base, numbered past the names in use', () => {
    const rare = 'Rarity: RARE\nNew Item\nCobalt Jewel\nCrafted: true';
    expect(nameUnnamedItem(rare, [])).toBe('Rarity: RARE\nRare Cobalt Jewel\nCobalt Jewel\nCrafted: true');
    expect(nameUnnamedItem(rare, ['Rare Cobalt Jewel'])).toBe('Rarity: RARE\nRare Cobalt Jewel 2\nCobalt Jewel\nCrafted: true');
    expect(nameUnnamedItem(rare, ['Rare Cobalt Jewel', 'Rare Cobalt Jewel 2'])).toContain('\nRare Cobalt Jewel 3\n');
  });
  it('puts the rarity in front of a magic base and leaves normal items and chosen names alone', () => {
    expect(nameUnnamedItem('Rarity: MAGIC\nLeather Belt\nCrafted: true', ['Rare Leather Belt'])).toBe('Rarity: MAGIC\nMagic Leather Belt\nCrafted: true');
    expect(nameUnnamedItem('Rarity: NORMAL\nIron Hat\nQuality: 20', [])).toBe('Rarity: NORMAL\nIron Hat\nQuality: 20');
    expect(nameUnnamedItem('Rarity: RARE\nDoom Crown\nIron Hat', [])).toBe('Rarity: RARE\nDoom Crown\nIron Hat');
  });
});
