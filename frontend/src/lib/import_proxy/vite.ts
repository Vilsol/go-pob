import type { IncomingMessage, ServerResponse } from 'node:http';
import type { Plugin } from 'vite';
import { Readable } from 'node:stream';
import { handleBuild, handlePoe, handlePoeUrl, handleShare } from './proxy';

const route = (req: IncomingMessage) => {
  const url = new URL(req.url ?? '/', 'http://localhost');
  const deps = { fetch };
  const { pathname } = url;
  if (pathname === '/api/share') {
    const body = req.method === 'POST' ? (Readable.toWeb(req) as ReadableStream<Uint8Array>) : undefined;
    return handleShare(new Request(url, { method: req.method, body, duplex: 'half' } as RequestInit), deps);
  }
  const request = new Request(url, { method: req.method });
  if (pathname === '/api/build') return handleBuild(request, deps);
  if (pathname === '/api/poeurl') return handlePoeUrl(request, deps);
  const poe = /^\/api\/poe\/([^/]+)$/.exec(pathname);
  return poe ? handlePoe(poe[1], request, deps) : undefined;
};

const middleware = (req: IncomingMessage, res: ServerResponse, next: () => void) => {
  const answer = route(req);
  if (!answer) {
    next();
    return;
  }
  void answer.then(async (r) => {
    res.statusCode = r.status;
    r.headers.forEach((v, k) => res.setHeader(k, v));
    res.end(Buffer.from(await r.arrayBuffer()));
  });
};

export const importProxy = (): Plugin => ({
  name: 'go-pob-import-proxy',
  configureServer: (server) => void server.middlewares.use(middleware),
  configurePreviewServer: (server) => void server.middlewares.use(middleware)
});
