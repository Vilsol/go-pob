import type { ImportMode } from '$lib/design/share';

export const UNAVAILABLE = {
  share: 'Not implemented: sharing uploads the code to a third-party site. Copy the code instead.',
  characters:
    'Not implemented: fetching a character needs the pathofexile.com account API, which the app does not reach. Paste the character JSON in the import box above to import its items and skills.',
  modes: {} as Partial<Record<ImportMode, string>>
};
export const REALMS = ['PC', 'Xbox', 'PS4', 'Garena', 'Tencent'];
