// Modules/BuildSiteTools.lua:19-44, ImportTab.lua:574-640

export type Site = {
  label: string;
  id: string;
  match: RegExp;
  regex: RegExp;
  download: string;
  link?: string;
};

export const SITES: Site[] = [
  {
    label: 'Maxroll',
    id: 'Maxroll',
    match: /^https:\/\/maxroll\.gg\/poe\/pob\/.*/,
    regex: /maxroll\.gg\/poe\/pob\/(.+)\s*$/,
    download: 'maxroll.gg/poe/api/pob/$1',
    link: 'https://maxroll.gg/poe/pob/'
  },
  {
    label: 'pob.codes',
    id: 'PoBCodes',
    match: /^https:\/\/pob\.codes\/b\/.+/,
    regex: /pob\.codes\/b\/(.+)\s*$/,
    download: 'api.pob.codes/$1/raw',
    link: 'https://pob.codes/b/'
  },
  { label: 'pobb.in', id: 'POBBin', match: /^https:\/\/pobb\.in\/.+/, regex: /pobb\.in\/(.+)\s*$/, download: 'pobb.in/pob/$1', link: 'https://pobb.in/' },
  {
    label: 'PoeNinja',
    id: 'PoeNinja',
    match: /^https:\/\/poe\.ninja\/?p?o?e?1?\/pob\/\w+/,
    regex: /poe\.ninja\/?p?o?e?1?\/pob\/(\w+)\s*$/,
    download: 'poe.ninja/poe1/pob/raw/$1',
    link: 'https://poe.ninja/poe1/pob/'
  },
  { label: 'Pastebin.com', id: 'pastebin', match: /^https:\/\/pastebin\.com\/\w+/, regex: /pastebin\.com\/(\w+)\s*$/, download: 'pastebin.com/raw/$1' },
  {
    label: 'PastebinP.com',
    id: 'pastebinProxy',
    match: /^https:\/\/pastebinp\.com\/\w+/,
    regex: /pastebinp\.com\/(\w+)\s*$/,
    download: 'pastebinp.com/raw/$1'
  },
  { label: 'Rentry.co', id: 'rentry', match: /^https:\/\/rentry\.co\/\w+/, regex: /rentry\.co\/(\w+)\s*$/, download: 'rentry.co/paste/$1/raw' },
  {
    label: 'poedb.tw',
    id: 'PoEDB',
    match: /^https:\/\/poedb\.tw\/.+/,
    regex: /poedb\.tw\/pob\/(.+)\s*$/,
    download: 'poedb.tw/pob/$1/raw',
    link: 'https://poedb.tw/pob/'
  }
];
export const EXPORT_SITES = SITES.filter((s) => s.link);

export type ImportMode = 'new' | 'replace' | 'compare';

export type Verdict = { valid: boolean; text: string; site?: string; json?: boolean };

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
  return site ? { valid: true, text: `URL is valid (${site.label})`, site: site.label } : undefined;
};

/* BuildSiteTools.lua:100 */
export const downloadUrl = (raw: string): { site: string; url: string } | undefined => {
  const t = importLink(raw);
  const site = SITES.find((s) => s.match.test(t));
  return site && { site: site.label, url: t.replace(site.regex, site.download) };
};

/* ImportTab.lua:609-628 */
export const jsonVerdict = (raw: string): Verdict | undefined => {
  const t = importLink(raw);
  if (!/^\{[\s\S]*\}$/.test(t)) return undefined;
  let data: { character?: { equipment?: unknown; passives?: unknown } } | null;
  try {
    data = JSON.parse(t) as typeof data;
  } catch {
    return { valid: false, text: 'Invalid JSON format (decode error)' };
  }
  const character = data?.character;
  if (!character) return { valid: false, text: 'Invalid JSON format (character missing)' };
  if (!character.equipment || !character.passives) return { valid: false, text: 'Invalid JSON format (equipment or passives missing)' };
  return { valid: true, text: 'JSON is valid', json: true };
};

export async function codeVerdict(raw: string): Promise<Verdict> {
  const bad = { valid: false, text: 'Invalid input' };
  try {
    const bin = atob(raw.trim().replace(/-/g, '+').replace(/_/g, '/'));
    const bytes = Uint8Array.from(bin, (c) => c.charCodeAt(0));
    const out = new Blob([bytes]).stream().pipeThrough(new DecompressionStream('deflate'));
    const text = await new Response(out).text();
    return text ? { valid: true, text: 'Code is valid' } : bad;
  } catch {
    return bad;
  }
}
