// ItemListControl.lua:198-258
import { describe, expect, it } from 'vitest';
import { itemUsage, type UsageContext } from './item_usage';

const ctx: UsageContext = {
  sets: [
    { id: '1', title: 'Default', slots: [{ slot: 'Helmet', item: 1 }] },
    {
      id: '2',
      title: 'Bossing',
      slots: [
        { slot: 'Helmet', item: 2 },
        { slot: 'Belt Abyssal Socket 1', item: 5 }
      ]
    }
  ],
  activeSet: '1',
  specs: [
    { title: 'Leveling', sockets: [{ node: 10, item: 3 }], alloc: new Set([10]) },
    { title: 'Maps', sockets: [{ node: 11, item: 4 }], alloc: new Set([11]) }
  ],
  activeSpec: 1
};

describe('itemUsage', () => {
  it('names nothing for an item the active set holds', () => {
    expect(itemUsage(1, { jewel: false, abyss: false }, ctx)).toBe('');
  });
  it("names another set that holds it (Used in '<set>')", () => {
    expect(itemUsage(2, { jewel: false, abyss: false }, ctx)).toBe("Used in 'Bossing'");
  });
  it("names another tree that sockets a jewel, nothing for the active tree's", () => {
    expect(itemUsage(3, { jewel: true, abyss: false }, ctx)).toBe("Used in 'Leveling'");
    expect(itemUsage(4, { jewel: true, abyss: false }, ctx)).toBe('');
  });
  it('names the set of an abyss jewel socketed outside the active set', () => {
    expect(itemUsage(5, { jewel: true, abyss: true }, ctx)).toBe("Used in 'Bossing'");
  });
  it('calls everything else Unused', () => {
    expect(itemUsage(9, { jewel: false, abyss: false }, ctx)).toBe('Unused');
  });
});
