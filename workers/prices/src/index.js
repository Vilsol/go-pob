// A read-only pass-through to poe.ninja's PoE1 economy API for go-pob's price lookups: the paths go-pob uses, cached
// for six hours, nothing else.

export const UPSTREAM = "https://poe.ninja";
export const USER_AGENT = "go-pob-prices (+https://github.com/Vilsol/go-pob)";
export const TTL = 21600;
// how long a copy is kept to answer when poe.ninja fails after the fresh one expired
export const STALE_TTL = 7 * 86400;
export const MAX_LEAGUE = 64;

const API = "/poe1/api/economy";
const OVERVIEWS = {
  [API + "/stash/current/currency/overview"]: new Set(["Currency"]),
  [API + "/exchange/current/overview"]: new Set(["Currency"]),
  [API + "/stash/current/item/overview"]: new Set([
    "UniqueWeapon",
    "UniqueArmour",
    "UniqueAccessory",
    "UniqueFlask",
    "UniqueJewel",
    "SkillGem",
    "BaseType",
    "ClusterJewel",
  ]),
};

// upstreamPath is the normalised path and query url asks for, or null when it is not one go-pob uses.
export function upstreamPath(url) {
  const params = [...url.searchParams.keys()];
  if (url.pathname === API + "/leagues") {
    return params.length === 0 ? url.pathname : null;
  }
  const types = OVERVIEWS[url.pathname];
  if (!types || params.length !== 2) {
    return null;
  }
  const league = url.searchParams.get("league");
  const type = url.searchParams.get("type");
  if (!league || league.length > MAX_LEAGUE || !types.has(type)) {
    return null;
  }
  return `${url.pathname}?${new URLSearchParams({ league, type })}`;
}

function text(status, body) {
  return new Response(body + "\n", { status, headers: { "Content-Type": "text/plain; charset=utf-8" } });
}

function reply(request, cached, extra = {}) {
  const etag = cached.headers.get("ETag");
  const headers = { "Content-Type": "application/json", "Cache-Control": `public, max-age=${TTL}`, ...extra };
  if (etag) {
    headers.ETag = etag;
    if (request.headers.get("If-None-Match") === etag) {
      return new Response(null, { status: 304, headers });
    }
  }
  return new Response(cached.body, { status: 200, headers });
}

function stored(body, etag, maxAge) {
  const headers = { "Content-Type": "application/json", "Cache-Control": `public, max-age=${maxAge}` };
  if (etag) {
    headers.ETag = etag;
  }
  return new Response(body, { headers });
}

export default {
  async fetch(request, env, ctx) {
    if (request.method !== "GET") {
      return new Response(null, { status: 405, headers: { Allow: "GET" } });
    }
    const url = new URL(request.url);
    const path = upstreamPath(url);
    if (path === null) {
      return text(404, "not found");
    }
    const cache = caches.default;
    const key = new Request(url.origin + path);
    const staleKey = new Request(url.origin + "/stale" + path);
    const fresh = await cache.match(key);
    if (fresh) {
      return reply(request, fresh);
    }
    const stale = await cache.match(staleKey);
    const headers = { "User-Agent": USER_AGENT, Accept: "application/json" };
    const staleTag = stale?.headers.get("ETag");
    if (staleTag) {
      headers["If-None-Match"] = staleTag;
    }
    let upstream;
    try {
      upstream = await fetch(UPSTREAM + path, { headers, redirect: "follow" });
    } catch {
      upstream = null;
    }
    let body;
    let etag;
    if (upstream?.status === 200) {
      body = await upstream.arrayBuffer();
      etag = upstream.headers.get("ETag");
    } else if (upstream?.status === 304 && stale) {
      body = await stale.arrayBuffer();
      etag = staleTag;
    } else if (stale) {
      return reply(request, stale, { "X-Prices-Stale": "1" });
    } else {
      return text(502, `poe.ninja answered ${upstream ? upstream.status : "nothing"}`);
    }
    ctx.waitUntil(Promise.all([cache.put(key, stored(body, etag, TTL)), cache.put(staleKey, stored(body, etag, STALE_TTL))]));
    return reply(request, stored(body, etag, TTL));
  },
};
