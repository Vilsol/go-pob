import { handleShare } from '../../src/lib/import_proxy/proxy';

export const onRequest = (ctx: PagesContext) => handleShare(ctx.request, { fetch });
