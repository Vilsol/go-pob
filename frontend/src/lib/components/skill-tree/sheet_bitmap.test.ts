import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { resetJobs } from '$lib/activity/jobs';
import { loadBitmap } from './sheet_bitmap';

type Pending = { url: string; resolve: (r: Response) => void };
let pending: Pending[] = [];

beforeEach(() => {
  resetJobs();
  pending = [];
  vi.stubGlobal('fetch', (url: string) => new Promise<Response>((resolve) => pending.push({ url, resolve })));
  vi.stubGlobal('createImageBitmap', () => Promise.resolve({ width: 1, height: 1 } as ImageBitmap));
});
afterEach(() => {
  resetJobs();
  vi.unstubAllGlobals();
});

const answer = async (url: string, status = 200) => {
  const i = pending.findIndex((p) => p.url === url);
  pending.splice(i, 1)[0].resolve(new Response(new Blob([new Uint8Array(4)]), { status }));
  await vi.waitFor(() => undefined);
};
const logged = () => (globalThis.__pobJobs ?? []).map((j) => ({ state: j.state, error: j.error, phases: j.phases.map((p) => [p.name, p.done, p.total]) }));

describe('loadBitmap', () => {
  it('counts the images asked for while the job runs, and ends it when the last arrives', async () => {
    const a = loadBitmap('a');
    const b = loadBitmap('b');
    await answer('a');
    await a;
    const c = loadBitmap('c');
    await answer('b');
    await answer('c');
    await Promise.all([b, c]);
    expect(logged()).toEqual([{ state: 'ok', error: undefined, phases: [['Sprite sheets and art', 3, 3]] }]);

    // the next image starts a new job
    const d = loadBitmap('d');
    await answer('d');
    await d;
    expect(logged().map((j) => j.phases)).toEqual([[['Sprite sheets and art', 3, 3]], [['Sprite sheets and art', 1, 1]]]);
  });

  it('ends in error when an image did not load', async () => {
    const a = loadBitmap('a');
    const b = loadBitmap('b').catch((e: unknown) => e);
    await answer('b', 404);
    await answer('a');
    await a;
    expect(await b).toBeInstanceOf(Error);
    expect(logged()).toEqual([{ state: 'error', error: '1 of 2 tree images did not load', phases: [['Sprite sheets and art', 2, 2]] }]);
  });
});
