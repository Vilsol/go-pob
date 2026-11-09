import { describe, expect, it } from 'vitest';
import { readBody } from './read_body';

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
});
