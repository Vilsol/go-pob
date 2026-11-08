import type { Sprite } from './types';

// The PoE2 tree's art from its data.json, as the worker hands it over with the tree: the names PassiveTreeView.lua
// asks GetAssetByName for, and the sprite sheets (PoE1's sprites format) that hold them.

type Overlay2 = { alloc: string; path: string; unalloc: string };
type Background2 = { image: string; x: number; y: number; width: number; height: number };

export type Tree2ArtData = {
  nodes: Record<
    string,
    { icon?: string; activeEffectImage?: string; nodeOverlay?: Overlay2; connectionArt?: string; options?: { name: string; icon?: string }[] }
  >;
  nodeOverlay?: Record<string, Overlay2>;
  connectionArt?: { default: string; ascendancy: string };
  classes?: {
    name: string;
    background?: Background2 & { active?: { width: number; height: number }; bg?: { width: number; height: number } };
    ascendancies?: { id?: string; name: string; background?: Background2; replace?: string; replaceBy?: string }[];
  }[];
  sprites?: Record<string, Record<string, Sprite>>;
};

// a sprite's place on its sheet, and its size in the source art (PassiveTree.lua:155-160 sizes an asset by it)
export type ArtSprite = { sheet: number; x: number; y: number; w: number; h: number; width: number; height: number };

export type Tree2Atlas = { sheets: string[]; byName: Map<string, ArtSprite> };

/* PassiveTree.lua:812-819: one name space over every sheet; PassiveTreeView.lua:1108-1110 darkens an unallocated icon
   itself, so a pre-darkened Inactive sprite is never the one drawn. A group's sheet holds its art scaled by the
   group's zoom, so the source size is the sheet size over it. */
export const buildAtlas2 = (sprites: Tree2ArtData['sprites']): Tree2Atlas => {
  const sheets: string[] = [];
  const byName = new Map<string, ArtSprite>();
  for (const [type, zooms] of Object.entries(sprites ?? {})) {
    if (/inactive$/i.test(type)) continue;
    const keys = Object.keys(zooms);
    if (!keys.length) continue;
    const key = keys.reduce((a, b) => (Number(b) > Number(a) ? b : a));
    const sprite = zooms[key];
    const zoom = Number(key) > 0 ? Number(key) : 1;
    let sheet = sheets.indexOf(sprite.filename);
    if (sheet < 0) sheet = sheets.push(sprite.filename) - 1;
    for (const [name, c] of Object.entries(sprite.coords)) {
      if (!byName.has(name)) byName.set(name, { sheet, x: c.x, y: c.y, w: c.w, h: c.h, width: c.w / zoom, height: c.h / zoom });
    }
  }
  return { sheets, byName };
};

// a sheet's path under the data version's root, as PoE1's scene names it
export const sheetPath2 = (filename: string) => 'tree/assets/' + filename.split('/').pop()!.split('?')[0];
