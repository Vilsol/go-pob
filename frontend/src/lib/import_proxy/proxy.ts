// BuildSiteTools.lua:88-113, ImportTab.lua:813-960

import { SITES } from '../design/share';

export const USER_AGENT = 'go-pob (+https://pob.gg)';

export const BUILD_CAP = 1 << 20;

const DAY = 24 * 60 * 60;
const IMMUTABLE = new Set(['Maxroll', 'PoBCodes', 'POBBin', 'PoeNinja']);
export const IMMUTABLE_TTL = 7 * DAY;
export const EDITABLE_TTL = 5 * 60;

export type Deps = {
  fetch: typeof fetch;
  cache?: Cache;
  waitUntil?: (p: Promise<unknown>) => void;
};

const text = (status: number, body: string, headers: Record<string, string> = {}) =>
  new Response(body, { status, headers: { 'Content-Type': 'text/plain; charset=utf-8', ...headers } });

export const readCapped = async (res: Response, cap: number): Promise<string | undefined> => {
  if (Number(res.headers.get('Content-Length') ?? 0) > cap) {
    await res.body?.cancel();
    return undefined;
  }
  if (!res.body) return '';
  const reader = res.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > cap) {
      await reader.cancel();
      return undefined;
    }
    chunks.push(value);
  }
  const all = new Uint8Array(size);
  let at = 0;
  for (const c of chunks) {
    all.set(c, at);
    at += c.byteLength;
  }
  return new TextDecoder().decode(all);
};

const cached = async (key: string, deps: Deps, make: () => Promise<Response>): Promise<Response> => {
  const hit = await deps.cache?.match(key);
  if (hit) return hit;
  const res = await make();
  if (deps.cache && /s-maxage=\d+/.test(res.headers.get('Cache-Control') ?? '')) {
    const put = deps.cache.put(key, res.clone());
    if (deps.waitUntil) deps.waitUntil(put);
    else await put;
  }
  return res;
};

const upstreamError = (who: string, res: Response) =>
  text(res.status, `${who} answered ${res.status}${res.headers.get('cf-mitigated') ? ` (${res.headers.get('cf-mitigated')})` : ''}`);

/* BuildSiteTools.lua:19-42 */
const idPattern = (regex: RegExp) => new RegExp(`^(?:${/\((.+?)\)\\s\*\$$/.exec(regex.source)?.[1] ?? '(?!)'})$`);
const SAFE_ID = /^[\w-]{1,128}$/;
/* Main.lua:69-70, BuildSiteTools.lua:93-94 */
const NINJA_CHARACTER = 'overview/code';
const NINJA_FIELDS = ['account', 'name', 'overview', 'type'];
const NINJA_FIELD = /^[^\s&?#/\\=]{1,64}$/u;

/* BuildSiteTools.lua:100 */
export const handleBuild = async (request: Request, deps: Deps): Promise<Response> => {
  if (request.method !== 'GET') return text(405, 'GET only', { Allow: 'GET' });
  const { origin, searchParams: params } = new URL(request.url);
  const site = SITES.find((s) => s.id === params.get('site'));
  if (!site) return text(400, 'unknown site');
  let id = params.get('id') ?? '';
  let ttl = IMMUTABLE.has(site.id) ? IMMUTABLE_TTL : EDITABLE_TTL;
  if (site.id === 'PoeNinja' && id === NINJA_CHARACTER) {
    const fields = NINJA_FIELDS.map((f) => params.get(f) ?? '');
    if (!fields.every((v) => NINJA_FIELD.test(v))) return text(400, `malformed ${site.label} character`);
    id = `${NINJA_CHARACTER}?${NINJA_FIELDS.map((f, i) => `${f}=${encodeURIComponent(fields[i])}`).join('&')}`;
    ttl = EDITABLE_TTL;
  } else if (!SAFE_ID.test(id) || !idPattern(site.regex).test(id)) return text(400, `malformed ${site.label} id`);
  const url = `https://${site.download.replace('$1', id)}`;
  return cached(`${origin}/api/build?site=${site.id}&id=${id}`, deps, async () => {
    let res: Response;
    try {
      res = await (0, deps.fetch)(url, { headers: { 'User-Agent': USER_AGENT } });
    } catch {
      return text(502, `${site.label} is unreachable`);
    }
    if (!res.ok) {
      await res.body?.cancel();
      return upstreamError(site.label, res);
    }
    const body = await readCapped(res, BUILD_CAP);
    if (body === undefined) return text(502, `${site.label} answered more than ${BUILD_CAP} bytes`);
    return text(200, body, { 'Cache-Control': `public, max-age=${ttl}, s-maxage=${ttl}` });
  });
};

/* ImportTab.lua:18-22 */
export const REALMS = ['pc', 'xbox', 'sony'];
const POE_HOST = 'https://www.pathofexile.com/';
export const POE_CAP = 4 << 20;
export const POE_TTL = 60;

/* ImportTab.lua:916, 946, 859, 821 */
const ENDPOINTS: Record<string, { path: (a: string, c: string, r: string) => string; character: boolean }> = {
  characters: { path: (a, _, r) => `character-window/get-characters?accountName=${a}&realm=${r}`, character: false },
  profile: { path: (a) => `account/view-profile/${a}`, character: false },
  items: { path: (a, c, r) => `character-window/get-items?accountName=${a}&character=${c}&realm=${r}`, character: true },
  passives: { path: (a, c, r) => `character-window/get-passive-skills?accountName=${a}&character=${c}&realm=${r}`, character: true }
};

// ImportTab.lua:905-913
const ACCOUNT = /^[^\s#&?/\\%=]{1,64}(?:#\d{1,8})?$/u;
const CHARACTER = /^[^\s#&?/\\%=]{1,64}$/u;

/* ImportTab.lua:911 */
const queryValue = (v: string) => encodeURIComponent(v).replace(/%2B/g, '+');

/* ImportTab.lua:954-956 */
export const profileName = (html: string) => {
  const name = /\/view-profile\/([^/]+)\/characters/.exec(html)?.[1];
  try {
    return name && decodeURIComponent(name);
  } catch {
    return name;
  }
};

const rateLimited = (res: Response) => {
  const headers: Record<string, string> = { 'Retry-After': res.headers.get('Retry-After') ?? String(POE_TTL) };
  res.headers.forEach((v, k) => {
    if (k.toLowerCase().startsWith('x-rate-limit')) headers[k] = v;
  });
  const wait = Number(headers['Retry-After']) || POE_TTL;
  return text(429, 'rate limited by pathofexile.com', { ...headers, 'Cache-Control': `s-maxage=${wait}` });
};

export const handlePoe = async (endpoint: string, request: Request, deps: Deps): Promise<Response> => {
  if (request.method !== 'GET') return text(405, 'GET only', { Allow: 'GET' });
  const ep = Object.hasOwn(ENDPOINTS, endpoint) ? ENDPOINTS[endpoint] : undefined;
  if (!ep) return text(404, 'unknown endpoint');
  const { origin, searchParams: params } = new URL(request.url);
  const realm = params.get('realm') ?? '';
  if (!REALMS.includes(realm)) return text(400, 'unknown realm');
  const account = params.get('account') ?? '';
  if (!ACCOUNT.test(account)) return text(400, 'malformed account name');
  const character = ep.character ? (params.get('character') ?? '') : '';
  if (ep.character && !CHARACTER.test(character)) return text(400, 'malformed character name');
  const path = ep.path(queryValue(account), queryValue(character), realm);
  const key = `${origin}/api/poe/${endpoint}?realm=${realm}&account=${queryValue(account)}&character=${queryValue(character)}`;
  return cached(key, deps, async () => {
    let res: Response;
    try {
      res = await (0, deps.fetch)(POE_HOST + path, { headers: { 'User-Agent': USER_AGENT } });
    } catch {
      return text(502, 'pathofexile.com is unreachable');
    }
    if (res.status === 429) {
      await res.body?.cancel();
      return rateLimited(res);
    }
    if (!res.ok) {
      await res.body?.cancel();
      return upstreamError('pathofexile.com', res);
    }
    const body = await readCapped(res, POE_CAP);
    if (body === undefined) return text(502, `pathofexile.com answered more than ${POE_CAP} bytes`);
    const cache = { 'Cache-Control': `public, max-age=${POE_TTL}, s-maxage=${POE_TTL}` };
    if (endpoint !== 'profile') return new Response(body, { headers: { 'Content-Type': 'application/json', ...cache } });
    const name = profileName(body);
    if (!name) return text(502, 'the profile page names no account');
    return new Response(JSON.stringify({ accountName: name }), { headers: { 'Content-Type': 'application/json', ...cache } });
  });
};
