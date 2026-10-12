// GemSelectControl.lua:154-253
import { describe, expect, it } from 'vitest';
import { searchGems, populateGems, type SearchGem } from './gem_search';

const gem = (name: string, tags: string[], over: Partial<SearchGem> = {}): SearchGem => ({
  name: over.name ?? name,
  searchName: name,
  tags,
  support: tags.includes('support'),
  levelReq: 1,
  ...over
});

const all: SearchGem[] = [
  gem('Fireball', ['area', 'fire', 'grants_active_skill', 'intelligence', 'projectile', 'spell']),
  gem('Fire Trap', ['area', 'fire', 'grants_active_skill', 'dexterity', 'trap', 'spell', 'duration']),
  gem('Cold to Fire', ['support', 'cold', 'fire', 'strength'], { name: 'Cold to Fire Support' }),
  gem('Anger', ['aura', 'fire', 'grants_active_skill', 'intelligence', 'spell', 'area']),
  gem('Trap', ['support', 'trap', 'dexterity', 'duration'], { name: 'Trap Support' }),
  gem('Flame Dash', ['fire', 'grants_active_skill', 'dexterity', 'movement', 'spell', 'travel', 'blink']),
  gem('Empower', ['support', 'exceptional', 'strength'], { name: 'Empower Support', levelReq: 1 }),
  gem('Enlighten', ['support', 'exceptional', 'intelligence'], { name: 'Enlighten Support', levelReq: 1 }),
  gem('Ice Nova', ['cold', 'area', 'spell', 'grants_active_skill', 'intelligence', 'nova'], { levelReq: 12 })
];

const names = (list: SearchGem[]) => list.map((g) => g.searchName);

const PIN: [string, string][] = [
  ['fire', 'Anger|Cold to Fire|Fire Trap|Fireball|Flame Dash'],
  ['fire trap', 'Fire Trap'],
  ['CtF', 'Cold to Fire'],
  ['trap', 'Fire Trap|Trap'],
  [':fire:-aura:active', 'Fire Trap|Fireball|Flame Dash'],
  ['f:dex', 'Fire Trap|Flame Dash'],
  [':aoe:int', 'Anger|Fireball|Ice Nova'],
  ['aura', 'Anger'],
  ['str', 'Cold to Fire|Empower'],
  ['', 'Anger|Cold to Fire|Empower|Enlighten|Fire Trap|Fireball|Flame Dash|Ice Nova|Trap'],
  ['zzz', ''],
  ['cold', 'Cold to Fire|Ice Nova'],
  ['ice n', 'Ice Nova'],
  ['f.r', 'Fire Trap|Fireball|Cold to Fire'],
  ['e', 'Empower|Enlighten|Anger|Cold to Fire|Fire Trap|Fireball|Flame Dash|Ice Nova']
];

describe('searchGems', () => {
  it.each(PIN)('%j lists what the pin lists', (buf, want) => {
    expect(names(searchGems(all, buf)).join('|')).toBe(want);
  });

  it('sorts each tier by the DPS key first when one is given', () => {
    const dps: Record<string, number> = { 'Cold to Fire Support': 9, Fireball: 1 };
    expect(names(searchGems(all, 'fire', (g) => dps[g.name])).join('|')).toBe('Cold to Fire|Fireball|Anger|Fire Trap|Flame Dash');
  });
});

describe('searchGems with the sort cache (SortGemList, GemSelectControl.lua:364-384)', () => {
  const dps: Record<string, number> = { Fireball: 10, 'Fire Trap': 30, 'Cold to Fire': 5, Anger: 50, Trap: 20, 'Flame Dash': 10 };
  const can = new Set(['Cold to Fire', 'Trap']);
  it('lists the gems that can support first, then by DPS, then by name', () => {
    const got = searchGems(
      all,
      '',
      (g) => dps[g.searchName],
      (g) => can.has(g.searchName)
    ).map((g) => g.searchName);
    expect(got.slice(0, 2)).toEqual(['Trap', 'Cold to Fire']);
    expect(got.slice(2, 4)).toEqual(['Anger', 'Fire Trap']);
  });
  it('orders by canSupport even with the DPS sort off', () => {
    const got = searchGems(all, 'fire', undefined, (g) => can.has(g.searchName)).map((g) => g.searchName);
    expect(got.indexOf('Cold to Fire')).toBeLessThan(got.indexOf('Fire Trap'));
  });
});

describe('populateGems', () => {
  it('gates by character level only in the characterLevel mode', () => {
    expect(names(populateGems(all, { showSupport: 'ALL', matchLevel: true, characterLevel: 10, showLegacy: false }))).not.toContain('Ice Nova');
    expect(names(populateGems(all, { showSupport: 'ALL', matchLevel: false, characterLevel: 10, showLegacy: false }))).toContain('Ice Nova');
  });

  it('keeps normal or exceptional supports by the support-type option', () => {
    const normal = names(populateGems(all, { showSupport: 'NORMAL', matchLevel: false, characterLevel: 100, showLegacy: false }));
    expect(normal).toContain('Trap');
    expect(normal).not.toContain('Empower');
    const exceptional = names(populateGems(all, { showSupport: 'EXCEPTIONAL', matchLevel: false, characterLevel: 100, showLegacy: false }));
    expect(exceptional).toContain('Empower');
    expect(exceptional).not.toContain('Trap');
    // GemSelectControl.lua:124-130
    expect(exceptional).not.toContain('Fireball');
  });

  // GemSelectControl.lua:114-151
  const legacy: SearchGem[] = [
    gem('Trap', ['support', 'trap', 'dexterity']),
    gem('Hidden', ['support', 'strength'], { hideFromGemList: true }),
    gem('Awakened Trap', ['support', 'awakened', 'trap', 'dexterity'], { legacy: true, legacyAwakened: true }),
    gem('Item Quantity', ['support', 'strength'], { legacy: true }),
    gem('Empower', ['support', 'exceptional', 'strength'])
  ];
  const listed = (showSupport: string, showLegacy: boolean) =>
    names(populateGems(legacy, { showSupport, matchLevel: false, characterLevel: 100, showLegacy }))
      .sort()
      .join('|');
  it.each([
    ['ALL', false, 'Empower|Trap'],
    ['ALL', true, 'Awakened Trap|Empower|Item Quantity|Trap'],
    ['NORMAL', false, 'Trap'],
    ['NORMAL', true, 'Item Quantity|Trap'],
    ['EXCEPTIONAL', false, 'Empower'],
    ['EXCEPTIONAL', true, 'Awakened Trap|Empower']
  ] as [string, boolean, string][])('%s with showLegacyGems %s lists %s', (showSupport, showLegacy, want) => {
    expect(listed(showSupport, showLegacy)).toBe(want);
  });
});
