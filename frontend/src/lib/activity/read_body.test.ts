import { describe, expect, it, vi } from 'vitest';
import { gzipSync } from 'node:zlib';
import { fetchWasm, readBody, type WasmReport } from './read_body';

const stream = (...parts: number[][]) =>
  new ReadableStream<Uint8Array>({
    start(c) {
      for (const p of parts) c.enqueue(new Uint8Array(p));
      c.close();
    }
  });

describe('readBody', () => {
  it('returns every byte in order and reports the count after each chunk, with the length as the total', async () => {
    const seen: [number, number | undefined][] = [];
    const res = new Response(stream([1, 2], [3], [4, 5, 6]), { headers: { 'content-length': '6' } });
    const buf = await readBody(res, (d, t) => seen.push([d, t]));
    expect([...new Uint8Array(buf)]).toEqual([1, 2, 3, 4, 5, 6]);
    expect(seen).toEqual([
      [2, 6],
      [3, 6],
      [6, 6]
    ]);
  });

  it('gives no total when the length counts encoded bytes, or there is none', async () => {
    const seen: (number | undefined)[] = [];
    await readBody(new Response(stream([1]), { headers: { 'content-length': '9', 'content-encoding': 'br' } }), (_, t) => seen.push(t));
    await readBody(new Response(stream([1])), (_, t) => seen.push(t));
    expect(seen).toEqual([undefined, undefined]);
  });

  it('returns every byte when the length is short of the body or beyond it', async () => {
    const short = await readBody(new Response(stream([1, 2], [3], [4, 5]), { headers: { 'content-length': '3' } }), () => {});
    expect([...new Uint8Array(short)]).toEqual([1, 2, 3, 4, 5]);
    const long = await readBody(new Response(stream([1, 2], [3]), { headers: { 'content-length': '8' } }), () => {});
    expect(long.byteLength).toBe(3);
    expect([...new Uint8Array(long)]).toEqual([1, 2, 3]);
  });
});

const WASM = [0x00, 0x61, 0x73, 0x6d, 1, 0, 0, 0];

const recorder = () => {
  const seen: string[] = [];
  const report: WasmReport = {
    phase: (name) => seen.push(`phase ${name}`),
    bytes: (done, total) => seen.push(`bytes ${done}/${total}`),
    fetched: (url) => seen.push(`fetched ${url}`)
  };
  return { seen, report };
};

describe('fetchWasm', () => {
  it('keeps a plain module as it arrives, its download counted in bytes', async () => {
    vi.stubGlobal('fetch', () => Promise.resolve(new Response(new Uint8Array(WASM), { headers: { 'content-length': '8' } })));
    const { seen, report } = recorder();
    const buf = await fetchWasm('https://x/e.wasm', 'https://x/e.wasm.gz', 'the engine', report);
    expect([...new Uint8Array(buf)]).toEqual(WASM);
    expect(seen).toEqual(['phase Downloading the engine', 'bytes 8/8', 'fetched https://x/e.wasm']);
    vi.unstubAllGlobals();
  });

  it('decompresses a .gz served as stored, counting its compressed bytes', async () => {
    const gz = gzipSync(new Uint8Array(WASM));
    vi.stubGlobal('fetch', () => Promise.resolve(new Response(gz, { headers: { 'content-length': String(gz.byteLength) } })));
    const { seen, report } = recorder();
    const buf = await fetchWasm('https://x/e.wasm.gz', 'https://x/e.wasm.gz', 'the engine', report);
    expect([...new Uint8Array(buf)]).toEqual(WASM);
    expect(seen).toEqual([
      'phase Downloading the engine (compressed)',
      `bytes ${gz.byteLength}/${gz.byteLength}`,
      'fetched https://x/e.wasm.gz',
      'phase Decompressing the engine'
    ]);
    vi.unstubAllGlobals();
  });

  it('fetches the fallback when a file arrives still encoded, and refuses a fallback that is not a module either', async () => {
    const asked: string[] = [];
    const gz = gzipSync(new Uint8Array(WASM));
    vi.stubGlobal('fetch', (url: string) => {
      asked.push(url);
      return Promise.resolve(new Response(url.endsWith('.br') ? new Uint8Array([0x0b, 1, 2]) : gz));
    });
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    const { seen, report } = recorder();
    const buf = await fetchWasm('https://x/e.wasm.br', 'https://x/e.wasm.gz', 'the engine', report);
    expect([...new Uint8Array(buf)]).toEqual(WASM);
    expect(asked).toEqual(['https://x/e.wasm.br', 'https://x/e.wasm.gz']);
    expect(seen.filter((s) => s.startsWith('fetched'))).toEqual(['fetched https://x/e.wasm.gz']);
    await expect(fetchWasm('https://x/e.wasm.br', 'https://x/e.wasm.br', 'the engine', report)).rejects.toThrow('Failed to load the engine: not a wasm module');
    vi.unstubAllGlobals();
  });

  it('names the engine and the file when the request fails or the host refuses it', async () => {
    const { report } = recorder();
    vi.stubGlobal('fetch', () => Promise.reject(new TypeError('Failed to fetch')));
    await expect(fetchWasm('https://x/e.wasm', 'https://x/e.wasm', 'the engine', report)).rejects.toThrow(
      'Failed to load the engine: Failed to fetch https://x/e.wasm'
    );
    vi.stubGlobal('fetch', () => Promise.resolve(new Response('', { status: 404 })));
    await expect(fetchWasm('https://x/e.wasm', 'https://x/e.wasm', 'the engine', report)).rejects.toThrow('Failed to load the engine: 404 https://x/e.wasm');
    vi.unstubAllGlobals();
  });
});
