// CompareBuySimilar.lua:22-203, TradeHelpers.lua:290-325

export const REALMS: [string, string][] = [
  ['PC', 'pc'],
  ['PS4', 'sony'],
  ['Xbox', 'xbox']
];

export const LISTED: [string, string][] = [
  ['Instant Buyout', 'securable'],
  ['Instant Buyout & In Person', 'available'],
  ['In Person (Online)', 'online'],
  ['Any', 'any']
];

export function tradeCategory(slotName: string | undefined, itemType: string | undefined): string | undefined {
  if (!slotName) return undefined;
  if (/^Weapon \d/.test(slotName)) {
    if (!itemType) return 'weapon.one';
    const exact: Record<string, string> = {
      Shield: 'armour.shield',
      Quiver: 'armour.quiver',
      Bow: 'weapon.bow',
      Staff: 'weapon.staff',
      'Two Handed Sword': 'weapon.twosword',
      'Two Handed Axe': 'weapon.twoaxe',
      'Two Handed Mace': 'weapon.twomace',
      'Fishing Rod': 'weapon.rod',
      'One Handed Sword': 'weapon.onesword',
      'One Handed Axe': 'weapon.oneaxe',
      'One Handed Mace': 'weapon.onemace',
      Sceptre: 'weapon.onemace',
      Wand: 'weapon.wand',
      Dagger: 'weapon.dagger',
      Claw: 'weapon.claw'
    };
    if (exact[itemType]) return exact[itemType];
    if (itemType.includes('Two Handed')) return 'weapon.twomelee';
    if (itemType.includes('One Handed')) return 'weapon.one';
    return 'weapon';
  }
  const bySlot: Record<string, string> = {
    'Body Armour': 'armour.chest',
    Helmet: 'armour.helmet',
    Gloves: 'armour.gloves',
    Boots: 'armour.boots',
    Amulet: 'accessory.amulet',
    'Ring 1': 'accessory.ring',
    'Ring 2': 'accessory.ring',
    'Ring 3': 'accessory.ring',
    Belt: 'accessory.belt'
  };
  if (bySlot[slotName]) return bySlot[slotName];
  if (slotName.includes('Abyssal')) return 'jewel.abyss';
  if (slotName.includes('Jewel')) return 'jewel';
  if (slotName.includes('Flask')) return 'flask';
  return undefined;
}

export type TradeItem = { rarity: string; title: string; name: string; baseName: string; type: string };
export type TradeDefence = { label: string; value: number; tradeKey: string };
export type TradeOptions = {
  realm: string;
  league: string;
  listed: string;
  baseType: boolean;
  ilvlMin?: number;
  ilvlMax?: number;
  defences?: Record<string, { min?: number; max?: number }>;
};

export const isTradeUnique = (item: TradeItem) => item.rarity === 'UNIQUE' || item.rarity === 'RELIC';

/* CompareBuySimilar.lua:310-329 */
export const tradeDefences = (item: TradeItem, armourData: Record<string, number> | undefined): TradeDefence[] => {
  if (isTradeUnique(item) || !armourData) return [];
  const defences = [
    { key: 'Armour', label: 'Armour', tradeKey: 'ar' },
    { key: 'Evasion', label: 'Evasion', tradeKey: 'ev' },
    { key: 'EnergyShield', label: 'Energy Shield', tradeKey: 'es' },
    { key: 'Ward', label: 'Ward', tradeKey: 'ward' }
  ];
  return defences.filter((d) => (armourData[d.key] ?? 0) > 0).map((d) => ({ label: d.label, value: armourData[d.key], tradeKey: d.tradeKey }));
};

// Common.lua:962-967
const urlEncode = (s: string) =>
  Array.from(new TextEncoder().encode(s), (b) => {
    const c = String.fromCharCode(b);
    return /[A-Za-z0-9_\-.~]/.test(c) ? c : `%${b.toString(16).toUpperCase().padStart(2, '0')}`;
  }).join('');

export function tradeSearchUrl(item: TradeItem, slotName: string | undefined, o: TradeOptions): string {
  const query: Record<string, unknown> = { status: { option: o.listed }, stats: [{ type: 'and', filters: [] }] };
  const filters: Record<string, unknown> = {};
  if (isTradeUnique(item)) {
    let tradeName = (item.title || item.name).replace(/^Foulborn\s+/, '').replace(/\s+$/, '');
    tradeName = /^(.*?)\s+\d+$/.exec(tradeName)?.[1] ?? tradeName;
    query.name = tradeName;
    query.type = item.baseName;
    if (/^Foulborn\s/.test(item.title || item.name)) filters.misc_filters = { filters: { foulborn_item: { option: 'true' } } };
  } else {
    const category = tradeCategory(slotName, item.type);
    if (category) filters.type_filters = { filters: { category: { option: category } } };
    if (o.baseType) query.type = item.baseName;
    if (o.ilvlMin !== undefined || o.ilvlMax !== undefined) {
      const ilvl: Record<string, number> = {};
      if (o.ilvlMin !== undefined) ilvl.min = o.ilvlMin;
      if (o.ilvlMax !== undefined) ilvl.max = o.ilvlMax;
      filters.misc_filters = { filters: { ilvl } };
    }
    // CompareBuySimilar.lua:118-137
    const armour: Record<string, { min?: number; max?: number }> = {};
    for (const [key, r] of Object.entries(o.defences ?? {})) {
      if (r.min !== undefined || r.max !== undefined) armour[key] = { ...(r.min !== undefined && { min: r.min }), ...(r.max !== undefined && { max: r.max }) };
    }
    if (Object.keys(armour).length) filters.armour_filters = { filters: armour };
  }
  if (Object.keys(filters).length) query.filters = filters;
  let url = 'https://www.pathofexile.com/trade/search';
  if (o.realm && o.realm !== 'pc') url += `/${o.realm}`;
  const league = (o.league.trim() || 'Standard').replace(/[^\w\-.~]/g, (c) => urlEncode(c));
  return `${url}/${league}?q=${urlEncode(JSON.stringify({ query, sort: { price: 'asc' } }))}`;
}
