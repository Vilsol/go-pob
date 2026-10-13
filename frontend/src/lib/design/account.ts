// ImportTab.lua:18-22, 368-404, 906-1094

/* ImportTab.lua:18-22 */
export const REALM_LIST = [
  { label: 'PC', id: 'PC', realmCode: 'pc' },
  { label: 'Xbox', id: 'XBOX', realmCode: 'xbox' },
  { label: 'Sony', id: 'SONY', realmCode: 'sony' }
];
export type Realm = (typeof REALM_LIST)[number];

/* ImportTab.lua:1095-1104 */
export type AccountCharacter = { name: string; realm: string; class: string; league: string; level: number };

export type ImportStage = 'GETACCOUNTNAME' | 'DOWNLOADCHARLIST' | 'SELECTCHAR' | 'IMPORTING';

export const NEGATIVE = '^xDD0022';
export const POSITIVE = '^x33FF77';

/* ImportTab.lua:368-370 */
export const canStart = (buf: string) => /\S[#-]\d{4}$/.test(buf);
/* ImportTab.lua:400-404 */
export const missingDiscriminator = (buf: string) => buf !== '' && !/[#-]\d{4}$/.test(buf);

/* ImportTab.lua:906-913 */
export const normaliseAccount = (realm: Realm, buf: string) => {
  const name =
    realm.realmCode === 'pc'
      ? buf.replace(/\s+/g, '')
      : buf
          .replace(/^[\s?]+/, '')
          .replace(/[\s?]+$/, '')
          .replace(/\s/g, '+');
  return name.replace(/^(.*)[#-]/s, '$1#');
};

/* ImportTab.lua:973-990 */
export const leaguesOf = (characters: AccountCharacter[]) => [...new Set(characters.map((c) => c.league))].sort();

/* ImportTab.lua:1026-1094 */
export const characterList = (realmCode: string, league: string | undefined, characters: AccountCharacter[]) =>
  characters
    .filter((c) => c.realm === realmCode && (league === undefined || c.league === league))
    .map((c) => ({
      char: c,
      label: c.name ?? '?',
      detail: `${c.class ?? '?'} lvl ${c.level ?? 0}${league === undefined ? ` in ${c.league ?? '?'}` : ''}`
    }))
    .sort((a, b) => (a.char.name.toLowerCase() < b.char.name.toLowerCase() ? -1 : a.char.name.toLowerCase() > b.char.name.toLowerCase() ? 1 : 0));
