import { describe, expect, it, vi } from 'vitest';
import { BUILD_CAP, EDITABLE_TTL, IMMUTABLE_TTL, POE_TTL, USER_AGENT, handleBuild, handlePoe, type Deps } from './proxy';

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

const deps = (answer: () => Response | Promise<Response>) => {
  const fetchMock = vi.fn((url: string, init?: RequestInit) => Promise.resolve(answer()).then((r) => (void url, void init, r)));
  const { m, cache } = memoryCache();
  return { fetchMock, m, d: { fetch: fetchMock as unknown as typeof fetch, cache } satisfies Deps };
};

const get = (q: string) => new Request(`https://pob.gg/api/build?${q}`);

describe('/api/build', () => {
  it('downloads the pin downloadURL with an honest User-Agent, and serves the second ask from the cache', async () => {
    const { fetchMock, d } = deps(() => new Response('eNrtXVtz'));
    const res = await handleBuild(get('site=POBBin&id=h-M3H3It2fDw'), d);
    expect(res.status).toBe(200);
    expect(await res.text()).toBe('eNrtXVtz');
    expect(fetchMock).toHaveBeenCalledWith('https://pobb.in/pob/h-M3H3It2fDw', { headers: { 'User-Agent': USER_AGENT } });
    expect(res.headers.get('Cache-Control')).toBe(`public, max-age=${IMMUTABLE_TTL}, s-maxage=${IMMUTABLE_TTL}`);
    const again = await handleBuild(get('site=POBBin&id=h-M3H3It2fDw'), d);
    expect(await again.text()).toBe('eNrtXVtz');
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['Maxroll', 'm5ui4t0x', 'https://maxroll.gg/poe/api/pob/m5ui4t0x', IMMUTABLE_TTL],
    ['PoBCodes', 'abc', 'https://api.pob.codes/abc/raw', IMMUTABLE_TTL],
    ['PoeNinja', 'Abc1', 'https://poe.ninja/poe1/pob/raw/Abc1', IMMUTABLE_TTL],
    ['pastebin', 'DsS6Qsxa', 'https://pastebin.com/raw/DsS6Qsxa', EDITABLE_TTL],
    ['pastebinProxy', 'DsS6Qsxa', 'https://pastebinp.com/raw/DsS6Qsxa', EDITABLE_TTL],
    ['rentry', 'abc', 'https://rentry.co/paste/abc/raw', EDITABLE_TTL],
    ['PoEDB', 'abc', 'https://poedb.tw/pob/abc/raw', EDITABLE_TTL]
  ])('%s %s → %s, cached %ss', async (site, id, url, ttl) => {
    const { fetchMock, d } = deps(() => new Response('code'));
    const res = await handleBuild(get(`site=${site}&id=${id}`), d);
    expect(fetchMock.mock.calls[0][0]).toBe(url);
    expect(res.headers.get('Cache-Control')).toContain(`s-maxage=${ttl}`);
  });

  it("rebuilds poe.ninja's character link from its four fields, cached for minutes", async () => {
    const { fetchMock, d } = deps(() => new Response('code'));
    const res = await handleBuild(get('site=PoeNinja&id=overview%2Fcode&account=Vilsol-3184&name=Some_Char&overview=mercenaries&type=exp'), d);
    expect(fetchMock.mock.calls[0][0]).toBe('https://poe.ninja/poe1/pob/raw/overview/code?account=Vilsol-3184&name=Some_Char&overview=mercenaries&type=exp');
    expect(res.headers.get('Cache-Control')).toContain(`s-maxage=${EDITABLE_TTL}`);
  });

  it.each([
    ['a missing poe.ninja field', 'site=PoeNinja&id=overview%2Fcode&account=a&name=b&overview=c', 'malformed PoeNinja character'],
    ['a query in a poe.ninja field', 'site=PoeNinja&id=overview%2Fcode&account=a%26x%3Dy&name=b&overview=c&type=d', 'malformed PoeNinja character'],
    ['the character path on another site', 'site=Maxroll&id=overview%2Fcode&account=a&name=b&overview=c&type=d', 'malformed Maxroll id'],
    ['an unknown site', 'site=evil&id=abc', 'unknown site'],
    ['a missing site', 'id=abc', 'unknown site'],
    ['a missing id', 'site=POBBin', 'malformed pobb.in id'],
    ['a path in a .+ id', 'site=Maxroll&id=../../admin', 'malformed Maxroll id'],
    ['a query in a .+ id', 'site=POBBin&id=a?b=c', 'malformed pobb.in id'],
    ['a host in a .+ id', 'site=PoBCodes&id=evil.com/x', 'malformed pob.codes id'],
    ['a dash in a %w+ id', 'site=pastebin&id=ab-cd', 'malformed Pastebin.com id'],
    ['an over-long id', `site=POBBin&id=${'a'.repeat(129)}`, 'malformed pobb.in id']
  ])('refuses %s without a request', async (_, q, msg) => {
    const { fetchMock, d } = deps(() => new Response('code'));
    const res = await handleBuild(get(q), d);
    expect(res.status).toBe(400);
    expect(await res.text()).toBe(msg);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('refuses anything but GET', async () => {
    const { fetchMock, d } = deps(() => new Response('code'));
    expect((await handleBuild(new Request('https://pob.gg/api/build?site=POBBin&id=a', { method: 'POST' }), d)).status).toBe(405);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("passes the site's status through and caches nothing", async () => {
    const { fetchMock, m, d } = deps(() => new Response('<html>nope</html>', { status: 404 }));
    const res = await handleBuild(get('site=pastebin&id=gone'), d);
    expect(res.status).toBe(404);
    expect(await res.text()).toBe('Pastebin.com answered 404');
    expect(m.size).toBe(0);
    await handleBuild(get('site=pastebin&id=gone'), d);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('names a Cloudflare challenge', async () => {
    const { d } = deps(() => new Response('<html>', { status: 403, headers: { 'cf-mitigated': 'challenge' } }));
    const res = await handleBuild(get('site=Maxroll&id=abc'), d);
    expect(res.status).toBe(403);
    expect(await res.text()).toBe('Maxroll answered 403 (challenge)');
  });

  it('answers 502 when the site is unreachable', async () => {
    const { d } = deps(() => Promise.reject(new Error('dns')));
    expect((await handleBuild(get('site=Maxroll&id=abc'), d)).status).toBe(502);
  });

  it('stops reading past the cap, by Content-Length or by counting', async () => {
    const big = 'x'.repeat(BUILD_CAP + 1);
    const declared = deps(() => new Response('x', { headers: { 'Content-Length': String(BUILD_CAP + 1) } }));
    expect((await handleBuild(get('site=Maxroll&id=abc'), declared.d)).status).toBe(502);
    const streamed = deps(
      () =>
        new Response(
          new ReadableStream({
            start(c) {
              c.enqueue(new TextEncoder().encode(big.slice(0, BUILD_CAP)));
              c.enqueue(new TextEncoder().encode('xy'));
              c.close();
            }
          })
        )
    );
    const res = await handleBuild(get('site=Maxroll&id=abc'), streamed.d);
    expect(res.status).toBe(502);
    expect(await res.text()).toBe(`Maxroll answered more than ${BUILD_CAP} bytes`);
    expect(streamed.m.size).toBe(0);
    const exact = deps(() => new Response(big.slice(0, BUILD_CAP)));
    expect((await handleBuild(get('site=Maxroll&id=abc'), exact.d)).status).toBe(200);
  });

  it('works without a cache (the dev server)', async () => {
    const fetchMock = vi.fn(() => Promise.resolve(new Response('code')));
    const res = await handleBuild(get('site=POBBin&id=abc'), { fetch: fetchMock as unknown as typeof fetch });
    expect(await res.text()).toBe('code');
  });
});

const poe = (endpoint: string, q: string, d: Deps) => handlePoe(endpoint, new Request(`https://pob.gg/api/poe/${endpoint}?${q}`), d);

describe('/api/poe', () => {
  it.each([
    ['characters', 'realm=pc&account=Vilsol%233184', 'https://www.pathofexile.com/character-window/get-characters?accountName=Vilsol%233184&realm=pc'],
    ['profile', 'realm=pc&account=Vilsol%233184', 'https://www.pathofexile.com/account/view-profile/Vilsol%233184'],
    [
      'items',
      'realm=xbox&account=Some%2BName%231234&character=Foo_Bar',
      'https://www.pathofexile.com/character-window/get-items?accountName=Some+Name%231234&character=Foo_Bar&realm=xbox'
    ],
    [
      'passives',
      'realm=sony&account=N%C3%A9e%231234&character=X',
      'https://www.pathofexile.com/character-window/get-passive-skills?accountName=N%C3%A9e%231234&character=X&realm=sony'
    ]
  ])("%s asks for ImportTab.lua's URL", async (endpoint, q, url) => {
    const { fetchMock, d } = deps(() => new Response(endpoint === 'profile' ? '<a href="/account/view-profile/Vilsol-3184/characters">' : '[]'));
    const res = await poe(endpoint, q, d);
    expect(res.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledWith(url, { headers: { 'User-Agent': USER_AGENT } });
    expect(res.headers.get('Cache-Control')).toBe(`public, max-age=${POE_TTL}, s-maxage=${POE_TTL}`);
  });

  it("answers the profile page's account name, not the page", async () => {
    const { d } = deps(() => new Response('<html>…<a href="/account/view-profile/N%C3%A9e-1234/characters">x</a>…'));
    expect(await (await poe('profile', 'realm=pc&account=nee%231234', d)).json()).toEqual({ accountName: 'Née-1234' });
    const none = deps(() => new Response('<html>private</html>'));
    expect((await poe('profile', 'realm=pc&account=nee%231234', none.d)).status).toBe(502);
  });

  it.each([
    ['an unknown endpoint', 'stash', 'realm=pc&account=a%231', 404, 'unknown endpoint'],
    ['an inherited name', 'constructor', 'realm=pc&account=a%231', 404, 'unknown endpoint'],
    ['an unknown realm', 'characters', 'realm=ps5&account=a%231', 400, 'unknown realm'],
    ['a missing realm', 'characters', 'account=a%231', 400, 'unknown realm'],
    ['a missing account', 'characters', 'realm=pc', 400, 'malformed account name'],
    ['a query in the account', 'characters', 'realm=pc&account=a%26realm%3Dx', 400, 'malformed account name'],
    ['a path in the account', 'profile', 'realm=pc&account=..%2F..%2Fx', 400, 'malformed account name'],
    ['a missing character', 'items', 'realm=pc&account=a%231', 400, 'malformed character name'],
    ['a query in the character', 'passives', 'realm=pc&account=a%231&character=x%26y', 400, 'malformed character name']
  ])('refuses %s without a request', async (_, endpoint, q, status, msg) => {
    const { fetchMock, d } = deps(() => new Response('[]'));
    const res = await poe(endpoint, q, d);
    expect(res.status).toBe(status);
    expect(await res.text()).toBe(msg);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('serves a second ask within the minute from the cache', async () => {
    const { fetchMock, d } = deps(() => new Response('[{"name":"X"}]'));
    await poe('characters', 'realm=pc&account=a%231', d);
    expect(await (await poe('characters', 'realm=pc&account=a%231', d)).text()).toBe('[{"name":"X"}]');
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("passes GGG's 429 through with Retry-After and its rate-limit headers, never retries, and holds off until Retry-After", async () => {
    const { fetchMock, d } = deps(
      () => new Response('', { status: 429, headers: { 'Retry-After': '30', 'X-Rate-Limit-Ip': '45:60:60', 'X-Rate-Limit-Ip-State': '46:60:30' } })
    );
    const res = await poe('characters', 'realm=pc&account=a%231', d);
    expect(res.status).toBe(429);
    expect(res.headers.get('Retry-After')).toBe('30');
    expect(res.headers.get('X-Rate-Limit-Ip-State')).toBe('46:60:30');
    expect(res.headers.get('Cache-Control')).toBe('s-maxage=30');
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect((await poe('characters', 'realm=pc&account=a%231', d)).status).toBe(429);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it.each([
    [401, 'pathofexile.com answered 401'],
    [403, 'pathofexile.com answered 403'],
    [404, 'pathofexile.com answered 404']
  ])('passes %s through (ImportTab.lua:918-925 tells them apart) and caches nothing', async (status, msg) => {
    const { m, d } = deps(() => new Response('{"error":{}}', { status }));
    const res = await poe('characters', 'realm=pc&account=a%231', d);
    expect(res.status).toBe(status);
    expect(await res.text()).toBe(msg);
    expect(m.size).toBe(0);
  });

  it('stops reading past the cap', async () => {
    const { d } = deps(() => new Response('x', { headers: { 'Content-Length': String(64 << 20) } }));
    expect((await poe('items', 'realm=pc&account=a%231&character=b', d)).status).toBe(502);
  });
});
