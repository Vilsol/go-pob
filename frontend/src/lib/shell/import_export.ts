import type { ImportMode } from '$lib/design/share';

export const UNAVAILABLE = {
  characters:
    'Not implemented: fetching a character needs the pathofexile.com account API, which the app does not reach. Paste the character JSON in the import box above to import its items and skills.',
  modes: {} as Partial<Record<ImportMode, string>>
};
export const REALMS = ['PC', 'Xbox', 'PS4', 'Garena', 'Tencent'];

export const SITE_KEY = 'options:lastExportWebsite';
/* ImportTab.lua:535, 538 */
export const loadExportSite = () => {
  try {
    return globalThis.localStorage?.getItem(SITE_KEY) ?? undefined;
  } catch {
    return undefined;
  }
};
export const saveExportSite = (site: string) => {
  try {
    globalThis.localStorage?.setItem(SITE_KEY, site);
  } catch {
    // empty
  }
};

/* ImportTab.lua:369-371 (PoB-PoE2) */
export const BUILD_PLANNER_NAME = 'Unnamed Build';
export const BUILD_PLANNER_AUTHOR = 'Author';

// BuildExportPoE2.lua:316-327 (PoB-PoE2) writes the file into the planner's folder; the browser saves it where the
// viewer picks
export const saveBuildFile = (name: string, json: string) => {
  const a = document.createElement('a');
  a.href = URL.createObjectURL(new Blob([json], { type: 'application/json' }));
  a.download = name;
  a.click();
  URL.revokeObjectURL(a.href);
};
