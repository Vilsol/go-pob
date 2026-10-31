import type { Sort } from '$lib/design/builds';

export const BUILD_SORT_KEY = 'options:buildSortMode';
const FALLBACK: Sort = { key: 'edited', dir: 'desc' };

/* Main.lua:539-541, 747 */
export const loadBuildSort = (): Sort => {
  try {
    const v = JSON.parse(globalThis.localStorage?.getItem(BUILD_SORT_KEY) ?? 'null') as Sort | null;
    return v && typeof v.key === 'string' && (v.dir === 'asc' || v.dir === 'desc') ? v : FALLBACK;
  } catch {
    return FALLBACK;
  }
};

export const saveBuildSort = (sort: Sort) => {
  try {
    globalThis.localStorage?.setItem(BUILD_SORT_KEY, JSON.stringify(sort));
  } catch {
    // empty
  }
};
