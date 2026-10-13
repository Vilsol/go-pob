import type { IncomingMessage, ServerResponse } from 'node:http';
import type { Plugin } from 'vite';
import { handleBuild, handlePoe } from './proxy';

const route = (req: IncomingMessage) => {
  const request = new Request(new URL(req.url ?? '/', 'http://localhost'), { method: req.method });
  const { pathname } = new URL(request.url);
  const deps = { fetch };
  if (pathname === '/api/build') return handleBuild(request, deps);
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
