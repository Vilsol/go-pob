// PassiveTreeView.lua:858-882 and 1528-1618 (PoB-PoE2)

import { luaPattern } from '$lib/design/gem_search';

export type Searchable2 = { type: string; name: string; stats: string[] | null };

/* PassiveTreeView.lua:862-876 */
export const prepSearch = (searchStr: string): string[] => {
  let search = searchStr.toLowerCase();
  const searchWords: string[] = [];
  for (const m of search.matchAll(/"([^"]*)"/g)) {
    searchWords.push(m[1]);
    search = search.replaceAll(`"${m[1]}"`, '');
  }
  for (const m of search.matchAll(/\S+/g)) searchWords.push(m[0]);
  return searchWords;
};

/* Common.lua:1009-1050 */
const orPatterns = (pattern: string): Set<string> => {
  const subGroups: { open: number; close: number; patterns: Set<string> }[] = [];
  for (let i = 0; i < pattern.length; i++) {
    if (pattern[i] !== '(') continue;
    let depth = 0;
    let j = i;
    for (; j < pattern.length; j++) {
      if (pattern[j] === '(') depth++;
      else if (pattern[j] === ')' && --depth === 0) break;
    }
    if (j >= pattern.length) break;
    subGroups.push({ open: i, close: j, patterns: orPatterns(pattern.slice(i + 1, j)) });
    i = j;
  }
  let generated = [pattern.slice(0, subGroups[0]?.open ?? pattern.length)];
  subGroups.forEach((g, i) => {
    const next = pattern.slice(g.close + 1, subGroups[i + 1]?.open ?? pattern.length);
    generated = generated.flatMap((p) => [...g.patterns].map((s) => p + s + next));
  });
  return new Set(generated.flatMap((p) => p.split('|').filter((s) => s !== '')));
};

// undefined is the PCall error: an invalid pattern
const matchOrPattern = (haystack: string, pattern: string): boolean | undefined => {
  for (const p of orPatterns(pattern)) {
    const re = luaPattern(p);
    if (!re) return undefined;
    if (re.test(haystack)) return true;
  }
  return false;
};

/* PassiveTreeView.lua:1528-1618, less the oil recipes, the mod names and the unlock constraints the view's tree does not carry */
export const doesNodeMatchSearchParams = (searchParams: string[], node: Searchable2): boolean => {
  if (node.type === 'ClassStart' || node.type === 'OnlyImage') return false;
  let need = [...searchParams];
  const search = (haystack: string): boolean => {
    const left: string[] = [];
    for (const n of need) {
      const m = matchOrPattern(haystack, n);
      if (m === undefined) return false;
      if (!m) left.push(n);
    }
    need = left;
    return true;
  };
  if (!search(node.name.toLowerCase())) return false;
  if (need.length === 0) return true;
  for (const line of node.stats ?? []) {
    if (!search(line.toLowerCase())) return false;
    if (need.length === 0) return true;
  }
  if (!search(node.type.toLowerCase())) return false;
  return need.length === 0;
};

/* PassiveTreeView.lua:877-881 */
export const searchResults = (searchStr: string, nodes: Iterable<{ id: number } & Searchable2>): Set<number> => {
  const params = prepSearch(searchStr);
  const out = new Set<number>();
  if (!params.length) return out;
  for (const n of nodes) if (doesNodeMatchSearchParams(params, n)) out.add(n.id);
  return out;
};
