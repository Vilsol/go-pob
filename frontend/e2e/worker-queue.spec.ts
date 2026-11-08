import { test, expect } from '@playwright/test';
import { DATA_HOST, cachedAsset } from './support/data_cache';
import { isTreeAsset, routeTreeAssets } from './support/tree_assets';
import { buildXml } from './support/builds';
import * as go from './support/go_model';

type QueueWorker = {
  ImportCode(code: string): Promise<unknown>;
  GetTimelessLists(jewelType: number): Promise<{ sockets: unknown[] }>;
};
type Worker = { cachedWorker: { syncWrap: QueueWorker } };

// a fresh worker: the first calculation is the first to read the passive tree, which waits on the data cache part way
test('a call made while the first calculation reads the passive tree waits for it instead of stopping Go', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  page.on('console', (m) => {
    if (m.type() === 'error') errors.push(m.text());
  });
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const url = route.request().url();
    if (isTreeAsset(url)) return route.fallback();
    const asset = cachedAsset(url);
    if (asset.status === 404) return route.fulfill({ status: 404, headers: { 'access-control-allow-origin': '*' } });
    return route.fulfill({ status: 200, body: asset.body, headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' } });
  });
  await routeTreeAssets(page);
  await page.goto('/');
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });

  const code = await go.encode(buildXml('0280-elementalist-fireball'));
  const out = await page.evaluate(
    async ([c, ms]) => {
      const w = (globalThis as unknown as Worker).cachedWorker.syncWrap;
      const opened = w.ImportCode(c);
      const lists = w.GetTimelessLists(2).then((l) => l.sockets.length);
      const timeout = new Promise<'timeout'>((done) => setTimeout(() => done('timeout'), ms));
      return { opened: await Promise.race([opened.then(() => 'opened'), timeout]), sockets: await Promise.race([lists, timeout]) };
    },
    [code, 30_000] as const
  );
  expect(out.opened).toBe('opened');
  expect(out.sockets, 'GetTimelessLists answered').not.toBe('timeout');
  expect(out.sockets, 'it read the opened build').toBeGreaterThan(0);
  expect(errors.filter((e) => /deadlock|fatal error|unreachable/i.test(e))).toEqual([]);
});
