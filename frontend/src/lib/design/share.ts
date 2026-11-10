// Modules/BuildSiteTools.lua:19-42, ImportTab.lua:574-640

export type Site = {
  label: string;
  id: string;
  match: RegExp;
  regex: RegExp;
  download: string;
  link?: string;
  post?: { url: string; fields: string; codeOut: string };
  game?: 'poe2';
};

export const SITES: Site[] = [
  {
    label: 'Maxroll',
    id: 'Maxroll',
    match: /^https:\/\/maxroll\.gg\/poe\/pob\/.*/,
    regex: /maxroll\.gg\/poe\/pob\/(.+)\s*$/,
    download: 'maxroll.gg/poe/api/pob/$1',
    link: 'https://maxroll.gg/poe/pob/',
    post: { url: 'https://maxroll.gg/poe/api/pob', fields: 'pobCode=', codeOut: 'https://maxroll.gg/poe/pob/' }
  },
  {
    label: 'pob.codes',
    id: 'PoBCodes',
    match: /^https:\/\/pob\.codes\/b\/.+/,
    regex: /pob\.codes\/b\/(.+)\s*$/,
    download: 'api.pob.codes/$1/raw',
    link: 'https://pob.codes/b/',
    post: { url: 'https://api.pob.codes/pob/plain', fields: '', codeOut: 'https://pob.codes/b/' }
  },
  {
    label: 'pobb.in',
    id: 'POBBin',
    match: /^https:\/\/pobb\.in\/.+/,
    regex: /pobb\.in\/(.+)\s*$/,
    download: 'pobb.in/pob/$1',
    link: 'https://pobb.in/',
    post: { url: 'https://pobb.in/pob/', fields: '', codeOut: 'https://pobb.in/' }
  },
  {
    label: 'PoeNinja',
    id: 'PoeNinja',
    match: /^https:\/\/poe\.ninja\/?p?o?e?1?\/pob\/\w+/,
    regex: /poe\.ninja\/?p?o?e?1?\/pob\/(\w+)\s*$/,
    download: 'poe.ninja/poe1/pob/raw/$1',
    link: 'https://poe.ninja/poe1/pob/',
    post: { url: 'https://poe.ninja/poe1/pob/api/upload', fields: 'code=', codeOut: '' }
  },
  { label: 'Pastebin.com', id: 'pastebin', match: /^https:\/\/pastebin\.com\/\w+/, regex: /pastebin\.com\/(\w+)\s*$/, download: 'pastebin.com/raw/$1' },
  {
    label: 'PastebinP.com',
    id: 'pastebinProxy',
    match: /^https:\/\/pastebinp\.com\/\w+/,
    regex: /pastebinp\.com\/(\w+)\s*$/,
    download: 'pastebinp.com/raw/$1'
  },
  {
    label: 'poedb.tw',
    id: 'PoEDB',
    match: /^https:\/\/poedb\.tw\/.+/,
    regex: /poedb\.tw\/pob\/(.+)\s*$/,
    download: 'poedb.tw/pob/$1/raw',
    link: 'https://poedb.tw/pob/',
    post: { url: 'https://poedb.tw/pob/api/gen', fields: '', codeOut: '' }
  },
  // BuildSiteTools.lua:19-35 (PoB-PoE2); pobb.in and the pastebins are the same rows as PoE1's
  {
    label: 'Maxroll (PoE2)',
    id: 'Maxroll2',
    match: /^https:\/\/maxroll\.gg\/poe2\/pob\/.*/,
    regex: /maxroll\.gg\/poe2\/pob\/(.+)\s*$/,
    download: 'maxroll.gg/poe2/api/pob/$1',
    link: 'https://maxroll.gg/poe2/pob/',
    post: { url: 'https://maxroll.gg/poe2/api/pob', fields: 'pobCode=', codeOut: 'https://maxroll.gg/poe2/pob/' },
    game: 'poe2'
  },
  {
    label: 'poe.ninja (PoE2)',
    id: 'PoeNinja2',
    match: /^https:\/\/poe2?\.ninja\/?p?o?e?2?\/pob\/.+/,
    regex: /poe2?\.ninja\/?p?o?e?2?\/pob\/(.+)\s*$/,
    download: 'poe.ninja/poe2/pob/raw/$1',
    link: 'https://poe.ninja/poe2/pob/',
    post: { url: 'https://poe.ninja/poe2/pob/api/upload', fields: 'code=', codeOut: '' },
    game: 'poe2'
  },
  {
    label: 'poe2db.tw',
    id: 'PoE2DB',
    match: /^https:\/\/poe2db\.tw\/pob\/.+/,
    regex: /poe2db\.tw\/pob\/(.+)\s*$/,
    download: 'poe2db.tw/pob/$1/raw',
    link: 'https://poe2db.tw/pob/',
    post: { url: 'https://poe2db.tw/pob/api/gen', fields: '', codeOut: '' },
    game: 'poe2'
  }
];
/* ImportTab.lua:522-532 */
export const EXPORT_SITES = SITES.filter((s) => s.post && !s.game);
/* ImportTab.lua:154-164 (PoB-PoE2): its own list's order, BuildSiteTools.lua:19-35 */
export const EXPORT_SITES2 = ['Maxroll2', 'POBBin', 'PoeNinja2', 'PoE2DB'].map((id) => SITES.find((s) => s.id === id)!);

export const DROPPED_SITES: { label: string; match: RegExp; text: string }[] = [
  {
    label: 'Rentry.co',
    match: /^https:\/\/rentry\.co\/\w+/,
    text: "Rentry.co links cannot be imported: rentry only serves a paste with its owner's access code. Paste the build code instead."
  }
];

export type ImportMode = 'new' | 'replace' | 'compare';

export type Verdict = { valid: boolean; text: string; site?: string; json?: boolean; game?: 'poe2' };

/* ImportTab.lua:590-594 */
export const importLink = (raw: string): string => {
  const t = raw.replace(/^[\s?]+/, '').replace(/[\s?]+$/, '');
  if (!/youtube\.com\/redirect\?/.test(t) && !/google\.com\/url\?/.test(t)) return t;
  const nested = t.replace(/^[\s\S]*[?&]q=([^&]+)[\s\S]*$/, '$1');
  const bytes = nested.replace(/\+/g, ' ').replace(/%([0-9a-fA-F]{2})/g, (_, h: string) => String.fromCharCode(parseInt(h, 16)));
  return new TextDecoder().decode(Uint8Array.from(bytes, (c) => c.charCodeAt(0)));
};

export const urlVerdict = (raw: string): Verdict | undefined => {
  const t = importLink(raw);
  const site = SITES.find((s) => s.match.test(t));
  if (site) return { valid: true, text: `URL is valid (${site.label})`, site: site.label };
  const dropped = DROPPED_SITES.find((s) => s.match.test(t));
  return dropped && { valid: false, text: dropped.text };
};

/* BuildSiteTools.lua:100 */
export const downloadUrl = (raw: string): { site: string; url: string } | undefined => {
  const t = importLink(raw);
  const site = SITES.find((s) => s.match.test(t));
  return site && { site: site.label, url: t.replace(site.regex, site.download) };
};

/* ImportTab.lua:609-628 */
export const NOT_A_CODE = 'this is not a Path of Building build code: it could not be decoded';
export const unreadableBuild = (why: string) => `the build code holds no readable build (${why})`;
// Build.lua:2231-2233
export const ROOT_MISSING = "'PathOfBuilding' root element missing";

// the worker's decode and parse errors, in the words the import messages use
export const importFailure = (err: unknown) => {
  const text = err instanceof Error ? err.message : String(err);
  if (text.startsWith('failed to create a zlib reader:') || text.startsWith('failed to read from zlib reader:')) return NOT_A_CODE;
  // PartyTab.lua:196-198 words it "Error loading"
  const xml = /^(?:failed to parse build as xml|error loading the build code): ([\s\S]*)$/.exec(text)?.[1];
  if (text === ROOT_MISSING) return unreadableBuild(text);
  return xml === undefined ? text : unreadableBuild(xml);
};

export const POE2_CHARACTER_DISABLED = 'This is a Path of Exile 2 character. Path of Exile 2 builds are not supported yet.';
export const POE2_CHARACTER = 'This is a Path of Exile 2 character, and Path of Exile 2 characters cannot be imported yet. Import its build code instead.';

export const jsonVerdict = (raw: string): Verdict | undefined => {
  const t = importLink(raw);
  if (!/^\{[\s\S]*\}$/.test(t)) return undefined;
  let data: { character?: { equipment?: unknown; passives?: { specialisations?: unknown } } } | null;
  try {
    data = JSON.parse(t) as typeof data;
  } catch {
    return { valid: false, text: 'Invalid JSON format (decode error)' };
  }
  const character = data?.character;
  if (!character) return { valid: false, text: 'Invalid JSON format (character missing)' };
  if (!character.equipment || !character.passives) return { valid: false, text: 'Invalid JSON format (equipment or passives missing)' };
  // ImportTab.lua:946-953 (PoB-PoE2)
  if (character.passives.specialisations) return { valid: true, text: 'JSON is valid', json: true, game: 'poe2' };
  return { valid: true, text: 'JSON is valid', json: true };
};

export async function codeVerdict(raw: string): Promise<Verdict> {
  const bad = { valid: false, text: 'Invalid input' };
  try {
    const bin = atob(raw.trim().replace(/-/g, '+').replace(/_/g, '/'));
    // a plain loop: Uint8Array.from with a mapper takes about 25 times as long on a large build's code
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
    const out = new Blob([bytes]).stream().pipeThrough(new DecompressionStream('deflate'));
    const text = await new Response(out).text();
    return text ? { valid: true, text: 'Code is valid' } : bad;
  } catch {
    return bad;
  }
}
