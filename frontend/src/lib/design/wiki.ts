// ItemTools.lua:380-409

export const WIKI_KEY = 'F1';

export const itemWikiName = (rarity: string, title: string, baseName: string) =>
  rarity.toUpperCase() === 'UNIQUE' || rarity.toUpperCase() === 'RELIC' ? title : baseName;

export const gemWikiName = (name: string, support: boolean) => (support && !name.endsWith(' Support') ? `${name} Support` : name);

export const wikiUrl = (name: string) => `https://www.poewiki.net/wiki/${name.replaceAll(' ', '_')}`;

export const hoveredWikiName = (doc: Document = document) => {
  const hovered = doc.querySelectorAll(':hover');
  const el = hovered[hovered.length - 1] as HTMLElement | undefined;
  return el?.closest<HTMLElement>('[data-wiki]')?.dataset.wiki || undefined;
};

export const openWiki = (name: string) => window.open(wikiUrl(name), '_blank', 'noopener');
