import { describe, expect, it } from 'vitest';

const base = process.env.IMPORT_BASE ?? 'http://127.0.0.1:8788';
const live = describe.skipIf(!process.env.IMPORT_LIVE);

const ask = async (path: string) => {
  const res = await fetch(base + path);
  const body = await res.text();
  console.log(`${res.status} ${path} ${res.headers.get('Retry-After') ?? ''} ${JSON.stringify(body.slice(0, 80))}`);
  return { status: res.status, body };
};

live('import proxy, live', () => {
  it.each([
    ['POBBin', 'h-M3H3It2fDw'],
    ['pastebin', 'DsS6Qsxa'],
    ['Maxroll', 'm5ui4t0x']
  ])('/api/build %s %s answers a build code', async (site, id) => {
    const { status, body } = await ask(`/api/build?site=${site}&id=${id}`);
    expect(status).toBe(200);
    expect(body.trim()).toMatch(/^[\w-]+=*$/);
  });

  it("/api/poe reads Vilsol#3184's characters, profile name, items and passives", async () => {
    const account = encodeURIComponent('Vilsol#3184');
    const chars = await ask(`/api/poe/characters?realm=pc&account=${account}`);
    expect(chars.status).toBe(200);
    const list = JSON.parse(chars.body) as { name: string }[];
    expect(list.length).toBeGreaterThan(0);
    await ask(`/api/poe/profile?realm=pc&account=${account}`);
    const character = encodeURIComponent(list[0].name);
    const items = await ask(`/api/poe/items?realm=pc&account=${account}&character=${character}`);
    expect(items.status).toBe(200);
    const passives = await ask(`/api/poe/passives?realm=pc&account=${account}&character=${character}`);
    expect(passives.status).toBe(200);
  });
});
