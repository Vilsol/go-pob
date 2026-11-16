# prices.pob.gg

A Cloudflare Worker that go-pob's MCP server (`go-pob mcp`, the `price_build` tool) reads item and gem prices
through. It passes a short list of [poe.ninja](https://poe.ninja) PoE1 economy requests through unchanged and caches
each answer for six hours.

## Why

poe.ninja's API page asks desktop apps and other clients to proxy their requests through their own backend, so that
responses are cached and the request volume stays under control. go-pob runs on users' machines, so it never calls
poe.ninja itself: every request goes to `https://prices.pob.gg/<poe.ninja path>`.

## What it serves

Only these GETs; anything else is a 404 and is never sent upstream:

| path | query |
|---|---|
| `/poe1/api/economy/leagues` | none |
| `/poe1/api/economy/stash/current/currency/overview` | `league` (1-64 chars), `type=Currency` |
| `/poe1/api/economy/exchange/current/overview` | `league` (1-64 chars), `type=Currency` (the Currency Exchange's rates) |
| `/poe1/api/economy/stash/current/item/overview` | `league` (1-64 chars), `type` one of `UniqueWeapon`, `UniqueArmour`, `UniqueAccessory`, `UniqueFlask`, `UniqueJewel`, `SkillGem`, `BaseType`, `ClusterJewel` |

* Answers are cached in the Cloudflare cache for six hours, keyed by the normalised URL. A second copy is kept
  for a week and served (with `X-Prices-Stale: 1`) when poe.ninja fails; with no copy at all the answer is a 502.
* Upstream requests carry only `User-Agent: go-pob-prices (+https://github.com/Vilsol/go-pob)`, `Accept` and the
  cached copy's `If-None-Match`. Nothing from the client (cookies, headers, addresses) is forwarded.
* Responses carry the upstream `ETag`; a matching `If-None-Match` from the client gets a 304.

This is market data. It is not part of go-pob-data and is never stored in this repository.

## Test

```shell
mise run test-prices-worker   # or: cd workers/prices && node --test test/
```

The tests mock `fetch` and the cache and make no network requests; no install is needed.

## Deploy

1. Uncomment the `routes` line in `wrangler.toml` (the `pob.gg` zone must be on the Cloudflare account).
2. From `workers/prices/`: `npx wrangler@4.143.1 login`, then `npx wrangler@4.143.1 deploy`.

With `custom_domain = true` Cloudflare creates the `prices.pob.gg` DNS record and certificate itself; if a
`prices` record already exists in the zone, delete it first.

To try it locally: `npx wrangler@4.143.1 dev`, then `go-pob mcp --prices-url http://localhost:8787`.
