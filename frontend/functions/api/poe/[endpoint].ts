import { handlePoe } from '../../../src/lib/import_proxy/proxy';

export const onRequestGet = (ctx: PagesContext<'endpoint'>) =>
  handlePoe(ctx.params.endpoint, ctx.request, { fetch, cache: caches.default, waitUntil: (p) => ctx.waitUntil(p) });
