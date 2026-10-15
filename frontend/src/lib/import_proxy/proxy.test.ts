import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { deflateSync } from 'node:zlib';
import {
  BUILD_CAP,
  EDITABLE_TTL,
  IMMUTABLE_TTL,
  LINK_CAP,
  POE_CAP,
  POE_TTL,
  USER_AGENT,
  handleBuild,
  handlePoe,
  handlePoeUrl,
  handleShare,
  type Deps
} from './proxy';

type Spec = string | { prefix?: string; repeat?: string; count?: number; b64?: string; code?: Spec; suffix?: string };
type Step = {
  method?: string;
  path: Spec;
  body?: Spec;
  status: number;
  text?: Spec;
  json?: unknown;
  headers?: Record<string, string | null>;
  fetches?: number;
  cached?: number;
};
type Case = {
  name: string;
  upstream?: { status?: number; headers?: Record<string, string>; body?: Spec; unreachable?: boolean };
  cache?: boolean;
  steps: Step[];
  fetch?: { url: string; method?: string; headers: Record<string, string>; body?: Spec; redirect?: string }[];
};
const fixture = JSON.parse(readFileSync(new URL('../../../../testdata/import_proxy/cases.json', import.meta.url), 'utf8')) as {
  constants: Record<string, string | number>;
  cases: Case[];
};

const utf8 = new TextEncoder();
const bytes = (s: Spec): Uint8Array => {
  if (typeof s === 'string') return utf8.encode(s);
  let mid: Uint8Array = new Uint8Array();
  if (s.repeat !== undefined) mid = utf8.encode(s.repeat.repeat(s.count ?? 0));
  else if (s.b64 !== undefined) mid = Uint8Array.from(Buffer.from(s.b64, 'base64'));
  else if (s.code !== undefined) mid = utf8.encode(deflateSync(bytes(s.code)).toString('base64url'));
  return Uint8Array.from([...utf8.encode(s.prefix ?? ''), ...mid, ...utf8.encode(s.suffix ?? '')]);
};
const str = (s: Spec) => new TextDecoder().decode(bytes(s));

const memoryCache = () => {
  const m = new Map<string, Response>();
  return {
    m,
    cache: {
      match: (k: string) => Promise.resolve(m.get(k)?.clone()),
      put: (k: string, r: Response) => {
        m.set(k, r);
        return Promise.resolve();
      }
    } as unknown as Cache
  };
};

const route = (request: Request, d: Deps) => {
  const { pathname } = new URL(request.url);
  if (pathname === '/api/build') return handleBuild(request, d);
  if (pathname === '/api/share') return handleShare(request, d);
  if (pathname === '/api/poeurl') return handlePoeUrl(request, d);
  const poe = /^\/api\/poe\/([^/]+)$/.exec(pathname);
  if (!poe) throw new Error(`no route for ${pathname}`);
  return handlePoe(poe[1], request, d);
};

it('the constants are the contract', () => {
  expect({ USER_AGENT, BUILD_CAP, LINK_CAP, POE_CAP, IMMUTABLE_TTL, EDITABLE_TTL, POE_TTL }).toEqual(fixture.constants);
});

describe.each(fixture.cases.map((c) => [c.name, c] as const))('%s', (_, c) => {
  it('answers as the contract says', async () => {
    const up = c.upstream ?? {};
    const fetchMock = vi.fn((...call: [url: string, init?: RequestInit]) => {
      void call;
      if (up.unreachable) return Promise.reject(new TypeError('fetch failed'));
      const status = up.status ?? 200;
      const body = up.body === undefined ? null : bytes(up.body);
      return Promise.resolve(new Response(body, { status, headers: up.headers }));
    });
    const { m, cache } = memoryCache();
    const d: Deps = { fetch: fetchMock as unknown as typeof fetch, ...(c.cache === false ? {} : { cache }) };
    for (const step of c.steps) {
      const request = new Request(`https://pob.gg${str(step.path)}`, {
        method: step.method ?? 'GET',
        body: step.body === undefined ? undefined : bytes(step.body)
      });
      const res = await route(request, d);
      expect(res.status).toBe(step.status);
      for (const [k, v] of Object.entries(step.headers ?? {})) expect(res.headers.get(k), k).toBe(v);
      if (step.text !== undefined) expect(await res.text()).toBe(str(step.text));
      else if (step.json !== undefined) expect(await res.json()).toEqual(step.json);
      if (step.fetches !== undefined) expect(fetchMock).toHaveBeenCalledTimes(step.fetches);
      if (step.cached !== undefined) expect(m.size).toBe(step.cached);
    }
    if (c.fetch) {
      expect(fetchMock.mock.calls.map(([url, init]) => ({ url, ...init }))).toEqual(
        c.fetch.map(({ body, ...f }) => ({ ...f, ...(body === undefined ? {} : { body: str(body) }) }))
      );
    }
  });
});
