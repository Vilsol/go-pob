// Pages answers a path with no file with the app's index.html; a missing file must say so instead
const FILE = /\.(png|ico|txt|xml|json|webmanifest)$/i;

export const missingFile = (path: string, res: Response) => FILE.test(path) && !!res.headers.get('Content-Type')?.startsWith('text/html');

export const onRequest = async (ctx: PagesContext): Promise<Response> => {
  const res = await ctx.next();
  if (!missingFile(new URL(ctx.request.url).pathname, res)) return res;
  return new Response('Not Found\n', { status: 404, headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
};
