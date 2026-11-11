// ItemTools.lua:380-409; PoB-PoE2's ItemTools.lua:460-490 is the same with poe2wiki.net

export const WIKI_KEY = 'F1';

export const itemWikiName = (rarity: string, title: string, baseName: string) =>
  rarity.toUpperCase() === 'UNIQUE' || rarity.toUpperCase() === 'RELIC' ? title : baseName;

export const gemWikiName = (name: string, support: boolean) => (support && !name.endsWith(' Support') ? `${name} Support` : name);

export const wikiUrl = (name: string, poe2 = false) => `https://www.${poe2 ? 'poe2wiki' : 'poewiki'}.net/wiki/${name.replaceAll(' ', '_')}`;

export const hoveredWikiName = (doc: Document = document) => {
  const hovered = doc.querySelectorAll(':hover');
  const el = hovered[hovered.length - 1] as HTMLElement | undefined;
  return el?.closest<HTMLElement>('[data-wiki]')?.dataset.wiki || undefined;
};

export const openWiki = (name: string, poe2 = false) => window.open(wikiUrl(name, poe2), '_blank', 'noopener');
