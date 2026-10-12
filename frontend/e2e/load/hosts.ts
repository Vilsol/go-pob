import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdtempSync, readFileSync, statSync } from 'node:fs';
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { createSecureServer, type Http2ServerRequest, type Http2ServerResponse } from 'node:http2';
import { tmpdir } from 'node:os';
import { extname, join, normalize, resolve } from 'node:path';
import { brotliCompressSync, gzipSync } from 'node:zlib';
import type { AddressInfo } from 'node:net';

import { cachedAsset } from '../support/data_cache';

const TYPES: Record<string, string> = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.woff2': 'font/woff2',
  '.wasm': 'application/wasm',
  '.gz': 'application/gzip'
};
const GZIPPED = new Set(['.html', '.js', '.css', '.json', '.svg']);

export type Hosts = { appURL: string; dataPort: number; close: () => Promise<void> };

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export type AppHost = 'gh' | 'cf';

export async function startHosts(buildDir: string, dataServerMs: number, appHost: AppHost = 'gh'): Promise<Hosts> {
  if (!existsSync(join(buildDir, 'index.html'))) {
    throw new Error(`REFUSING: ${buildDir} has no index.html; run vite build first`);
  }

  const gz = new Map<string, Buffer>();
  const app = createServer((req: IncomingMessage, res: ServerResponse) => {
    const path = decodeURIComponent((req.url ?? '/').split('?')[0]);
    let file = normalize(join(buildDir, path));
    if (!file.startsWith(buildDir) || !existsSync(file) || statSync(file).isDirectory()) {
      file = join(buildDir, 'index.html');
    }
    const st = statSync(file);
    const etag = `"${st.size.toString(16)}-${st.mtimeMs.toString(16)}"`;
    const ext = extname(file);
    const headers: Record<string, string> = {
      'content-type': TYPES[ext] ?? 'application/octet-stream',
      'cache-control': 'max-age=600',
      etag,
      'access-control-allow-origin': '*',
      vary: 'Accept-Encoding'
    };
    if (req.headers['if-none-match'] === etag) {
      res.writeHead(304, headers).end();
      return;
    }
    let body = readFileSync(file);
    const accept = String(req.headers['accept-encoding']);
    if (appHost === 'cf' && file.endsWith('.wasm.br')) {
      headers['content-type'] = 'application/wasm';
      headers['content-encoding'] = 'br';
    } else if (appHost === 'cf' && GZIPPED.has(ext) && /\bbr\b/.test(accept)) {
      let z = gz.get(file);
      if (!z) gz.set(file, (z = brotliCompressSync(body)));
      body = z;
      headers['content-encoding'] = 'br';
    } else if (GZIPPED.has(ext) && /\bgzip\b/.test(accept)) {
      let z = gz.get(file);
      if (!z) gz.set(file, (z = gzipSync(body)));
      body = z;
      headers['content-encoding'] = 'gzip';
    }
    headers['content-length'] = String(body.length);
    res.writeHead(200, headers).end(body);
  });

  const certDir = mkdtempSync(join(tmpdir(), 'pob-load-cert-'));
  execFileSync('go', ['run', resolve(import.meta.dirname, 'cert.go'), certDir], { stdio: 'inherit' });
  const data = createSecureServer(
    { key: readFileSync(join(certDir, 'key.pem')), cert: readFileSync(join(certDir, 'cert.pem')), allowHTTP1: true },
    (req: Http2ServerRequest, res: Http2ServerResponse) => {
      void (async () => {
        await sleep(dataServerMs);
        let asset;
        try {
          asset = cachedAsset(`https://data.pob.gg${req.url}`);
        } catch (e) {
          console.error(String(e));
          res.writeHead(500).end();
          return;
        }
        const headers = { 'access-control-allow-origin': '*', 'timing-allow-origin': '*', 'cache-control': 'public, max-age=0, must-revalidate' };
        if (asset.status === 404) {
          res.writeHead(404, headers).end();
          return;
        }
        const etag = `"${createHash('sha1').update(asset.body).digest('hex')}"`;
        if (req.headers['if-none-match'] === etag) {
          res.writeHead(304, { ...headers, etag }).end();
          return;
        }
        res.writeHead(200, { ...headers, etag, 'content-type': 'application/octet-stream', 'content-length': String(asset.body.length) }).end(asset.body);
      })();
    }
  );

  await new Promise<void>((r) => app.listen(0, '127.0.0.1', r));
  await new Promise<void>((r) => data.listen(0, '127.0.0.1', r));
  return {
    appURL: `http://localhost:${(app.address() as AddressInfo).port}`,
    dataPort: (data.address() as AddressInfo).port,
    close: async () => {
      await new Promise((r) => app.close(r));
      await new Promise((r) => data.close(r));
    }
  };
}
