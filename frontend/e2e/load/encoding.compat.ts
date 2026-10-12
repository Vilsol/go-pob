import { test, type Browser } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import { appendFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { createSecureServer } from 'node:http2';
import type { AddressInfo } from 'node:net';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { brotliCompressSync, deflateRawSync, deflateSync, gzipSync } from 'node:zlib';

const OUT = resolve(import.meta.dirname, '../.results/encoding-compat.jsonl');

const WASM = Buffer.from([
  0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x05, 0x01, 0x60, 0x00, 0x01, 0x7f, 0x03, 0x02, 0x01, 0x00, 0x07, 0x05, 0x01, 0x01, 0x66, 0x00, 0x00,
  0x0a, 0x06, 0x01, 0x04, 0x00, 0x41, 0x2a, 0x0b
]);
const PLAIN = Buffer.from(Array.from({ length: 4000 }, (_, i) => `row ${i} ${(i * 2654435761) % 1000}\n`).join(''));
const ENCODED: Record<string, Buffer> = {
  gzip: gzipSync(PLAIN),
  deflate: deflateSync(PLAIN),
  'deflate-raw': deflateRawSync(PLAIN),
  brotli: brotliCompressSync(PLAIN)
};

type Seen = { acceptEncoding: string[] };

function handler(seen: Seen) {
  return (req: IncomingMessage | import('node:http2').Http2ServerRequest, res: ServerResponse | import('node:http2').Http2ServerResponse) => {
    const url = new URL(req.url ?? '/', 'http://x');
    const h = { 'access-control-allow-origin': '*', 'cache-control': 'no-store' };
    if (url.pathname === '/') {
      (res as ServerResponse).writeHead(200, { ...h, 'content-type': 'text/html' }).end('<!doctype html><title>compat</title>');
      return;
    }
    if (url.pathname === '/ae') {
      seen.acceptEncoding.push(String(req.headers['accept-encoding'] ?? ''));
      (res as ServerResponse).writeHead(200, { ...h, 'content-type': 'text/plain' }).end('ok');
      return;
    }
    if (url.pathname.startsWith('/wasm/')) {
      const enc = url.pathname.slice(6);
      const body = enc === 'br' ? brotliCompressSync(WASM) : enc === 'gzip' ? gzipSync(WASM) : WASM;
      const headers: Record<string, string> = { ...h, 'content-type': 'application/wasm', 'content-length': String(body.length) };
      if (enc !== 'identity') headers['content-encoding'] = enc;
      (res as ServerResponse).writeHead(200, headers).end(body);
      return;
    }
    if (url.pathname.startsWith('/raw/')) {
      const body = ENCODED[url.pathname.slice(5)];
      (res as ServerResponse).writeHead(200, { ...h, 'content-type': 'application/octet-stream', 'content-length': String(body.length) }).end(body);
      return;
    }
    (res as ServerResponse).writeHead(404, h).end();
  };
}

async function probe(browser: Browser, origin: string, seen: Seen) {
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  try {
    const page = await context.newPage();
    await page.goto(origin + '/');
    seen.acceptEncoding.length = 0;
    return await page.evaluate(
      async ({ formats, plainLength }) => {
        await fetch('/ae');
        const wasm: Record<string, unknown> = {};
        for (const enc of ['identity', 'gzip', 'br']) {
          const r: Record<string, unknown> = {};
          try {
            const bytes = new Uint8Array(await (await fetch(`/wasm/${enc}`)).arrayBuffer());
            r.fetchDecoded = bytes[0] === 0x00 && bytes[1] === 0x61 && bytes[2] === 0x73 && bytes[3] === 0x6d;
          } catch (e) {
            r.fetchDecoded = String(e);
          }
          try {
            const { instance } = await WebAssembly.instantiateStreaming(fetch(`/wasm/${enc}`), {});
            r.streaming = (instance.exports.f as () => number)() === 42;
          } catch (e) {
            r.streaming = String(e);
          }
          wasm[enc] = r;
        }
        const stream: Record<string, unknown> = {};
        for (const f of formats) {
          try {
            const ds = new DecompressionStream(f as CompressionFormat);
            const out = await new Response((await fetch(`/raw/${f}`)).body!.pipeThrough(ds)).arrayBuffer();
            stream[f] = out.byteLength === plainLength;
          } catch (e) {
            stream[f] = String(e).slice(0, 80);
          }
        }
        return {
          userAgent: navigator.userAgent,
          instantiateStreaming: typeof WebAssembly.instantiateStreaming === 'function',
          compileCache: typeof (WebAssembly as unknown as { Module: unknown }).Module === 'function',
          wasm,
          stream
        };
      },
      { formats: Object.keys(ENCODED), plainLength: PLAIN.length }
    );
  } finally {
    await context.close();
  }
}

for (const scheme of ['http', 'https']) {
  test(`encoding compat ${scheme}`, async ({ browser, browserName }) => {
    const seen: Seen = { acceptEncoding: [] };
    let origin: string;
    let close: () => Promise<void>;
    if (scheme === 'http') {
      const s = createServer(handler(seen));
      await new Promise<void>((r) => s.listen(0, '127.0.0.1', r));
      origin = `http://localhost:${(s.address() as AddressInfo).port}`;
      close = () => new Promise((r) => s.close(() => r()));
    } else {
      const certDir = mkdtempSync(join(tmpdir(), 'pob-compat-cert-'));
      execFileSync('go', ['run', resolve(import.meta.dirname, 'cert.go'), certDir], { stdio: 'inherit' });
      const s = createSecureServer(
        { key: readFileSync(join(certDir, 'key.pem')), cert: readFileSync(join(certDir, 'cert.pem')), allowHTTP1: true },
        handler(seen)
      );
      await new Promise<void>((r) => s.listen(0, '127.0.0.1', r));
      origin = `https://localhost:${(s.address() as AddressInfo).port}`;
      close = () =>
        new Promise((r) =>
          s.close(() => {
            rmSync(certDir, { recursive: true, force: true });
            r();
          })
        );
    }
    try {
      const result = await probe(browser, origin, seen);
      const row = { engine: browserName, version: browser.version(), scheme, acceptEncoding: seen.acceptEncoding[0] ?? '', ...result };
      mkdirSync(resolve(OUT, '..'), { recursive: true });
      appendFileSync(OUT, JSON.stringify(row) + '\n');
      console.log(JSON.stringify(row));
    } finally {
      await close();
    }
  });
}
