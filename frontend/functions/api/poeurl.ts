import { handlePoeUrl } from '../../src/lib/import_proxy/proxy';

export const onRequest = (ctx: PagesContext) => handlePoeUrl(ctx.request, { fetch });
