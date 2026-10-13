import { handleBuild } from '../../src/lib/import_proxy/proxy';

export const onRequestGet = (ctx: PagesContext) => handleBuild(ctx.request, { fetch, cache: caches.default, waitUntil: (p) => ctx.waitUntil(p) });
