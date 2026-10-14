import type { ImportMode } from '$lib/design/share';

export const UNAVAILABLE = {
  characters:
    'Not implemented: fetching a character needs the pathofexile.com account API, which the app does not reach. Paste the character JSON in the import box above to import its items and skills.',
  modes: {} as Partial<Record<ImportMode, string>>
};
export const REALMS = ['PC', 'Xbox', 'PS4', 'Garena', 'Tencent'];

const SITE_KEY = 'options:lastExportWebsite';
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
