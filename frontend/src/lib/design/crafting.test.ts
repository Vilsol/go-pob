import { describe, expect, it } from 'vitest';
import { enchantSkills, finitePowers, maxOf, notableColour, notableMatches, sortByValue, sortNotables, type AnointNotable } from './crafting';

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
