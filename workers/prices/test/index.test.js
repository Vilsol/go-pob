import { test, beforeEach } from "node:test";
import assert from "node:assert/strict";

import worker, { upstreamPath, USER_AGENT, TTL } from "../src/index.js";

const ORIGIN = "https://prices.example";
const LEAGUES = "/poe1/api/economy/leagues";
const GEMS = "/poe1/api/economy/stash/current/item/overview?league=Covens&type=SkillGem";

let store;
let calls;
let upstream;

class MemoryCache {
  async match(req) {
    const hit = store.get(req.url);
    return hit ? hit.clone() : undefined;
  }
  async put(req, res) {
    store.set(req.url, res.clone());
  }
}

beforeEach(() => {
  store = new Map();
  calls = [];
  upstream = () => new Response('{"lines":[]}', { headers: { ETag: 'W/"1"' } });
  globalThis.caches = { default: new MemoryCache() };
  globalThis.fetch = async (url, init) => {
    calls.push({ url, init });
    return upstream(url, init);
  };
});

async function get(path, init = {}) {
  const waits = [];
  const res = await worker.fetch(new Request(ORIGIN + path, init), {}, { waitUntil: (p) => waits.push(p) });
  await Promise.all(waits);
  return res;
}

test("allowlist", () => {
  const ok = [
    LEAGUES,
    "/poe1/api/economy/stash/current/currency/overview?league=Covens&type=Currency",
    "/poe1/api/economy/stash/current/item/overview?type=UniqueArmour&league=Hardcore+Covens",
    "/poe1/api/economy/stash/current/item/overview?league=Standard&type=ClusterJewel",
    "/poe1/api/economy/exchange/current/overview?league=Covens&type=Currency",
  ];
  for (const p of ok) {
    assert.notEqual(upstreamPath(new URL(ORIGIN + p)), null, p);
  }
  const bad = [
    "/",
    LEAGUES + "?x=1",
    "/poe1/api/economy/stash/current/currency/overview?league=Covens&type=Fragment",
    "/poe1/api/economy/stash/current/item/overview?league=Covens&type=Currency",
    "/poe1/api/economy/stash/current/item/overview?league=Covens&type=DivinationCard",
    "/poe1/api/economy/stash/current/item/overview?type=SkillGem",
    "/poe1/api/economy/stash/current/item/overview?league=&type=SkillGem",
    "/poe1/api/economy/stash/current/item/overview?league=" + "x".repeat(65) + "&type=SkillGem",
    "/poe1/api/economy/stash/current/item/overview?league=Covens&type=SkillGem&extra=1",
    "/poe1/api/economy/exchange/current/overview?league=Covens&type=Fragment",
    "/poe1/api/economy/exchange/current/overview?type=Currency",
    "/poe1/api/builds/profile?account=x",
    "/poe2/api/economy/leagues",
  ];
  for (const p of bad) {
    assert.equal(upstreamPath(new URL(ORIGIN + p)), null, p);
  }
});

test("query order is normalised", () => {
  const a = upstreamPath(new URL(ORIGIN + "/poe1/api/economy/stash/current/item/overview?type=SkillGem&league=Covens"));
  assert.equal(a, GEMS);
});

test("anything else is 404 and never fetched", async () => {
  const res = await get("/poe1/api/economy/stash/current/item/overview?league=Covens&type=Map");
  assert.equal(res.status, 404);
  assert.equal(calls.length, 0);
});

test("GET only", async () => {
  const res = await get(LEAGUES, { method: "POST", body: "x" });
  assert.equal(res.status, 405);
  assert.equal(calls.length, 0);
});

test("fetches once, then answers from the cache for six hours", async () => {
  const first = await get(GEMS);
  assert.equal(first.status, 200);
  assert.equal(await first.text(), '{"lines":[]}');
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, "https://poe.ninja" + GEMS);
  const second = await get("/poe1/api/economy/stash/current/item/overview?type=SkillGem&league=Covens");
  assert.equal(await second.text(), '{"lines":[]}');
  assert.equal(calls.length, 1);
  assert.equal(store.get(ORIGIN + GEMS).headers.get("Cache-Control"), `public, max-age=${TTL}`);
});

test("forwards nothing from the client", async () => {
  await get(LEAGUES, {
    headers: { Cookie: "a=b", "X-Forwarded-For": "1.2.3.4", "CF-Connecting-IP": "1.2.3.4", "User-Agent": "client", Authorization: "x" },
  });
  const sent = new Headers(calls[0].init.headers);
  assert.deepEqual([...sent.keys()].sort(), ["accept", "user-agent"]);
  assert.equal(sent.get("User-Agent"), USER_AGENT);
});

test("answers a matching If-None-Match with 304", async () => {
  await get(GEMS);
  const res = await get(GEMS, { headers: { "If-None-Match": 'W/"1"' } });
  assert.equal(res.status, 304);
  assert.equal(res.headers.get("ETag"), 'W/"1"');
});

test("serves the stale copy when poe.ninja fails", async () => {
  await get(GEMS);
  store.delete(ORIGIN + GEMS);
  upstream = () => new Response("down", { status: 503 });
  const res = await get(GEMS);
  assert.equal(res.status, 200);
  assert.equal(res.headers.get("X-Prices-Stale"), "1");
  assert.equal(await res.text(), '{"lines":[]}');
  assert.equal(new Headers(calls[1].init.headers).get("If-None-Match"), 'W/"1"');
});

test("serves the stale copy when the fetch throws", async () => {
  await get(GEMS);
  store.delete(ORIGIN + GEMS);
  upstream = () => {
    throw new Error("network");
  };
  const res = await get(GEMS);
  assert.equal(res.status, 200);
  assert.equal(res.headers.get("X-Prices-Stale"), "1");
});

test("refreshes the stale copy on 304", async () => {
  await get(GEMS);
  store.delete(ORIGIN + GEMS);
  upstream = () => new Response(null, { status: 304 });
  const res = await get(GEMS);
  assert.equal(res.status, 200);
  assert.equal(res.headers.get("X-Prices-Stale"), null);
  assert.equal(await res.text(), '{"lines":[]}');
  assert.ok(store.has(ORIGIN + GEMS));
});

test("502 when poe.ninja fails and nothing is cached", async () => {
  upstream = () => new Response("down", { status: 500 });
  const res = await get(GEMS);
  assert.equal(res.status, 502);
  assert.equal(store.size, 0);
});
