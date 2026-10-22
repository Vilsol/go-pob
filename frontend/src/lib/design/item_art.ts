import { DESKTOP } from '$lib/pwa/desktop';

export const ART_CDN = 'https://web.poecdn.com/image/';

/* The desktop host keeps its own copy of the CDN's art on disk, the web app's service worker caches it */
export const artUrl = (path: string) => (DESKTOP ? `/desktop/art/${path}.png` : `${ART_CDN}${path}.png`);

export type ArtQuery = { Name: string; Base: string; Unique: boolean; Gem: boolean };
export type ArtResolver = (queries: ArtQuery[]) => Promise<string[]>;

let resolver: ArtResolver | undefined;
const known = new Map<string, string>();
let queued: { key: string; query: ArtQuery; done: (path: string) => void }[] = [];
const pending = new Map<string, Promise<string>>();

export function setArtResolver(r: ArtResolver | undefined) {
  resolver = r;
  known.clear();
  pending.clear();
}

const flush = async () => {
  const batch = queued;
  queued = [];
  let paths: string[] = [];
  try {
    paths = resolver ? await resolver(batch.map((b) => b.query)) : [];
  } catch {
    // unresolved queries are asked again next time
  }
  batch.forEach((b, i) => {
    pending.delete(b.key);
    if (i < paths.length) known.set(b.key, paths[i]);
    b.done(paths[i] ?? '');
  });
};

/* Lookups made in one task go to the worker as one batch */
export function artPath(query: ArtQuery): Promise<string> {
  const key = `${query.Gem ? 'g' : query.Unique ? 'u' : 'b'}|${query.Name}|${query.Base}`;
  const hit = known.get(key);
  if (hit !== undefined) return Promise.resolve(hit);
  const wait = pending.get(key);
  if (wait) return wait;
  const p = new Promise<string>((done) => {
    if (queued.length === 0) queueMicrotask(() => void flush());
    queued.push({ key, query, done });
  });
  pending.set(key, p);
  return p;
}
