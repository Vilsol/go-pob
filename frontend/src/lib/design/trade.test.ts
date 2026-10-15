import { describe, expect, it } from 'vitest';
import { tradeCategory, tradeDefences, tradeSearchUrl } from './trade';

const query = (url: string) => JSON.parse(decodeURIComponent(url.split('?q=')[1])) as { query: Record<string, unknown>; sort: unknown };
const o = { realm: 'pc', league: 'Standard', listed: 'securable', baseType: false };

// CompareBuySimilar.lua:44-203, TradeHelpers.lua:290-325
describe('Buy similar', () => {
  it('searches a unique by its name and base, Foulborn and a trailing number stripped', () => {
    const url = tradeSearchUrl(
      { rarity: 'UNIQUE', title: 'Foulborn Kaom’s Heart 2', name: '', baseName: 'Glorious Plate', type: 'Body Armour' },
      'Body Armour',
      o
    );
    expect(url.startsWith('https://www.pathofexile.com/trade/search/Standard?q=')).toBe(true);
    expect(query(url)).toEqual({
      query: {
        status: { option: 'securable' },
        stats: [{ type: 'and', filters: [] }],
        name: 'Kaom’s Heart',
        type: 'Glorious Plate',
        filters: { misc_filters: { filters: { foulborn_item: { option: 'true' } } } }
      },
      sort: { price: 'asc' }
    });
  });
  it('searches anything else by category, base when asked and item level, on the realm and league given', () => {
    const url = tradeSearchUrl({ rarity: 'RARE', title: 'Doom Grip', name: 'Doom Grip', baseName: 'Imperial Bow', type: 'Bow' }, 'Weapon 1', {
      realm: 'sony',
      league: 'Mercenaries HC',
      listed: 'any',
      baseType: true,
      ilvlMin: 82
    });
    expect(url.startsWith('https://www.pathofexile.com/trade/search/sony/Mercenaries%20HC?q=')).toBe(true);
    expect(query(url).query).toEqual({
      status: { option: 'any' },
      stats: [{ type: 'and', filters: [] }],
      type: 'Imperial Bow',
      filters: { type_filters: { filters: { category: { option: 'weapon.bow' } } }, misc_filters: { filters: { ilvl: { min: 82 } } } }
    });
  });
  it('maps slots to trade categories', () => {
    expect(tradeCategory('Weapon 2', 'Two Handed Claw')).toBe('weapon.twomelee');
    expect(tradeCategory('Weapon 1', 'Sceptre')).toBe('weapon.sceptre');
    expect(tradeCategory('Weapon 1', 'Rune Dagger')).toBe('weapon');
    expect(tradeCategory('Ring 2', 'Ring')).toBe('accessory.ring');
    expect(tradeCategory('Abyssal Socket 1', 'Jewel')).toBe('jewel.abyss');
    expect(tradeCategory('Flask 3', 'Flask')).toBe('flask');
    expect(tradeCategory(undefined, 'Ring')).toBeUndefined();
  });

  // CompareBuySimilar.lua:118-137, 310-329
  it('lists an armour piece’s defences and searches the ticked ones as armour filters', () => {
    const item = { rarity: 'RARE', title: 'Doom Shell', name: 'Doom Shell', baseName: 'Vaal Regalia', type: 'Body Armour' };
    const defences = tradeDefences(item, { Armour: 0, Evasion: 312.6, EnergyShield: 450, Ward: 0, BlockChance: 3 });
    expect(defences).toEqual([
      { label: 'Evasion', value: 312.6, tradeKey: 'ev' },
      { label: 'Energy Shield', value: 450, tradeKey: 'es' }
    ]);
    expect(tradeDefences({ ...item, rarity: 'UNIQUE' }, { EnergyShield: 450 })).toEqual([]);
    const url = tradeSearchUrl(item, 'Body Armour', { ...o, defences: { es: { min: 450 }, ev: {} } });
    expect((query(url).query.filters as Record<string, unknown>).armour_filters).toEqual({ filters: { es: { min: 450 } } });
  });
});
