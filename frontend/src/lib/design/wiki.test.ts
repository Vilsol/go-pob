import { describe, expect, it } from 'vitest';
import { gemWikiName, itemWikiName, wikiUrl } from './wiki';

// ItemTools.lua:380-405
describe('itemLib.wiki', () => {
  it('names a unique or relic by its title, anything else by its base', () => {
    expect(itemWikiName('UNIQUE', "Kaom's Heart", 'Glorious Plate')).toBe("Kaom's Heart");
    expect(itemWikiName('relic', 'Doryani', 'Ruby Ring')).toBe('Doryani');
    expect(itemWikiName('RARE', 'Doom Grip', 'Titan Gauntlets')).toBe('Titan Gauntlets');
  });
  it('names a support gem with " Support"', () => {
    expect(gemWikiName('Added Fire Damage', true)).toBe('Added Fire Damage Support');
    expect(gemWikiName('Fireball', false)).toBe('Fireball');
    expect(gemWikiName('Added Fire Damage Support', true)).toBe('Added Fire Damage Support');
  });
  it('routes spaces to underscores and nothing else', () => {
    expect(wikiUrl("Kaom's Heart")).toBe("https://www.poewiki.net/wiki/Kaom's_Heart");
  });
});
