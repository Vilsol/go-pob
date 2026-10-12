// GemSelectControl.lua:105-152

export type SearchGem = {
  name: string;
  searchName: string;
  tags: string[];
  support: boolean;
  levelReq: number;
  legacy?: boolean;
  hideFromGemList?: boolean;
  legacyAwakened?: boolean;
};
export type PopulateOptions = { showSupport: string; matchLevel: boolean; characterLevel: number; showLegacy: boolean };

const exceptional = (g: SearchGem) => g.tags.includes('exceptional');

export function populateGems<G extends SearchGem>(gems: G[], o: PopulateOptions): G[] {
  const showAll = o.showSupport === 'ALL';
  const showExceptional = o.showSupport === 'EXCEPTIONAL';
  const showNormal = o.showSupport === 'NORMAL';
  return gems.filter((g) => {
    if (g.hideFromGemList) return false;
    if (o.matchLevel && o.characterLevel < g.levelReq) return false;
    const exceptionalLike = !!g.legacyAwakened || exceptional(g);
    let listed: boolean;
    if ((showExceptional || showAll) && exceptionalLike) listed = o.showLegacy || !g.legacy;
    else listed = showNormal || showAll;
    if (!listed) return false;
    if (g.legacy && !o.showLegacy) return false;
    return !g.support || showAll || (showNormal && !exceptionalLike) || (showExceptional && exceptionalLike);
  });
}

const CLASSES: Record<string, string> = {
  a: 'A-Za-z',
  l: 'a-z',
  u: 'A-Z',
  d: '0-9',
  s: '\\s',
  w: 'A-Za-z0-9',
  x: '0-9A-Fa-f',
  p: '!-/:-@\\[-`{-~',
  c: '\\x00-\\x1f\\x7f'
};
const escapeRe = (c: string) => c.replace(/[\\^$.*+?()[\]{}|/-]/g, '\\$&');

export function luaPattern(p: string): RegExp | null {
  let out = '';
  let i = 0;
  if (p[0] === '^') {
    out += '^';
    i = 1;
  }
  const classOf = (c: string, inSet: boolean) => {
    const k = CLASSES[c.toLowerCase()];
    if (!k) return escapeRe(c);
    if (c === c.toLowerCase()) return inSet ? k : `[${k}]`;
    return inSet ? null : `[^${k}]`;
  };
  while (i < p.length) {
    if (p[i] === '$' && i === p.length - 1) {
      out += '$';
      break;
    }
    let item: string | null;
    const c = p[i];
    if (c === '%') {
      if (i + 1 >= p.length || p[i + 1] === 'b' || p[i + 1] === 'f') return null;
      item = classOf(p[i + 1], false);
      i += 2;
    } else if (c === '.') {
      item = '[\\s\\S]';
      i += 1;
    } else if (c === '[') {
      let j = i + 1;
      let set = '[';
      if (p[j] === '^') {
        set += '^';
        j++;
      }
      let first = true;
      while (j < p.length && (p[j] !== ']' || first)) {
        first = false;
        if (p[j] === '%') {
          if (j + 1 >= p.length) return null;
          const k = classOf(p[j + 1], true);
          if (k === null) return null;
          set += k;
          j += 2;
        } else if (p[j + 1] === '-' && p[j + 2] !== undefined && p[j + 2] !== ']') {
          set += `${escapeRe(p[j])}-${escapeRe(p[j + 2])}`;
          j += 3;
        } else {
          set += escapeRe(p[j]);
          j += 1;
        }
      }
      if (j >= p.length) return null;
      item = set + ']';
      i = j + 1;
    } else {
      item = escapeRe(c);
      i += 1;
    }
    if (item === null) return null;
    const q = p[i];
    if (q === '*' || q === '+' || q === '?') {
      out += item + q;
      i += 1;
    } else if (q === '-') {
      out += item + '*?';
      i += 1;
    } else out += item;
  }
  try {
    return new RegExp(out);
  } catch {
    return null;
  }
}

const TAG_ALIASES: Record<string, string> = { active: 'grants_active_skill', int: 'intelligence', str: 'strength', dex: 'dexterity', aoe: 'area' };
const TERM_ALIASES: Record<string, string> = { active: 'grants_active_skill', int: 'intelligence', str: 'strength', dex: 'dexterity' };

const splitColon = (s: string) => [...(s.startsWith(':') ? [''] : []), ...s.split(':').filter((f) => f !== '')];

export function searchGems<G extends SearchGem>(gems: G[], buf: string, dps?: (g: G) => number | undefined, canSupport?: (g: G) => boolean): G[] {
  // GemSelectControl.lua:364-384
  const sort = (list: G[]) =>
    list.sort((a, b) => {
      const ca = !!canSupport?.(a);
      const cb = !!canSupport?.(b);
      if (ca !== cb) return ca ? -1 : 1;
      if (dps) {
        const da = dps(a) ?? -Infinity;
        const db = dps(b) ?? -Infinity;
        if (da !== db) return db - da;
      }
      return a.searchName < b.searchName ? -1 : a.searchName > b.searchName ? 1 : 0;
    });
  if (!buf.length) return sort([...gems]);

  const tagsList = splitColon(buf);
  let searchTerm = tagsList.shift() ?? '';
  const lowerSearch = searchTerm.toLowerCase();
  const patterns = [`^ ${lowerSearch}$`, `^${lowerSearch.replace(/[A-Za-z]/g, ' $&%l+')}$`, `^ ${lowerSearch}`, lowerSearch].map(luaPattern);
  const added = new Set<G>();
  const out: G[] = [];
  for (const pattern of patterns) {
    const matchList: G[] = [];
    for (const g of gems) {
      if (!added.has(g) && pattern?.test(` ${g.searchName.toLowerCase()}`)) {
        let addThisGem = true;
        for (const tag of tagsList) {
          let tagName = tag.replace(/\s+/g, '').toLowerCase();
          const negate = tagName.startsWith('-');
          if (negate) tagName = tagName.slice(1);
          tagName = TAG_ALIASES[tagName] ?? tagName;
          if (negate ? g.tags.includes(tagName) : !g.tags.includes(tagName)) addThisGem = false;
        }
        if (addThisGem) {
          matchList.push(g);
          added.add(g);
        }
      } else {
        searchTerm = TERM_ALIASES[searchTerm] ?? searchTerm;
        if (!added.has(g) && g.tags.includes(searchTerm.toLowerCase())) {
          matchList.push(g);
          added.add(g);
        }
      }
    }
    out.push(...sort(matchList));
  }
  return out;
}
