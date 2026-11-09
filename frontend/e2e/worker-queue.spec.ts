import { test, expect, type Page } from '@playwright/test';
import { DATA_HOST, cachedAsset } from './support/data_cache';
import { isTreeAsset, routeTreeAssets } from './support/tree_assets';
import { buildXml } from './support/builds';
import * as go from './support/go_model';

type QueueWorker = {
  ImportCode(code: string): Promise<unknown>;
  GetTimelessLists(jewelType: number): Promise<{ sockets: unknown[] }>;
  LoadCompareCode(code: string): Promise<unknown>;
  GetTree(version: string): Promise<string>;
  GetTreeVersion(): Promise<string>;
  CalculateAllocationPaths(version: string, active: number[], roots: number[]): Promise<Record<number, number> | undefined>;
};
type Worker = { cachedWorker: { syncWrap: QueueWorker } };

const freshPage = async (page: Page, errors: string[]) => {
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
};

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

// the compared build's calculation is the first to read the passive tree, and is not a calculation of the page's build
test('a call made while a compared build is calculated for the first time waits for its data reads instead of stopping Go', async ({ page }) => {
  const errors: string[] = [];
  await freshPage(page, errors);
  const code = await go.encode(buildXml('0280-elementalist-fireball'));
  const out = await page.evaluate(
    async ([c, ms]) => {
      const w = (globalThis as unknown as Worker).cachedWorker.syncWrap;
      const version = await w.GetTreeVersion();
      const compared = w.LoadCompareCode(c);
      const paths = w.CalculateAllocationPaths(version, [], [0]).then(() => 'answered');
      const timeout = new Promise<'timeout'>((done) => setTimeout(() => done('timeout'), ms));
      return { compared: await Promise.race([compared.then(() => 'compared'), timeout]), paths: await Promise.race([paths, timeout]) };
    },
    [code, 30_000] as const
  );
  expect(out.compared).toBe('compared');
  expect(out.paths, 'CalculateAllocationPaths answered').toBe('answered');
  expect(errors.filter((e) => /deadlock|fatal error|unreachable|already exited/i.test(e))).toEqual([]);
});

// a tree the program has not read yet loads through the cache while the Go call that asked for it waits
test('a call reaching a passive tree that another call is still loading waits for it instead of stopping Go', async ({ page }) => {
  const errors: string[] = [];
  await freshPage(page, errors);
  const out = await page.evaluate(async (ms) => {
    const w = (globalThis as unknown as Worker).cachedWorker.syncWrap;
    const version = '3_20';
    const tree = w.GetTree(version).then((t) => t.length);
    const paths = w.CalculateAllocationPaths(version, [], [0]).then(() => 'answered');
    const timeout = new Promise<'timeout'>((done) => setTimeout(() => done('timeout'), ms));
    return { tree: await Promise.race([tree, timeout]), paths: await Promise.race([paths, timeout]) };
  }, 30_000);
  expect(out.tree, 'GetTree answered').not.toBe('timeout');
  expect(out.paths, 'CalculateAllocationPaths answered').toBe('answered');
  expect(errors.filter((e) => /deadlock|fatal error|unreachable|already exited/i.test(e))).toEqual([]);
});

// nothing has read this tree yet: the call itself is the first to reach its load
test('a tree call that is the first to read its passive tree answers instead of handing back nothing', async ({ page }) => {
  const errors: string[] = [];
  await freshPage(page, errors);
  const out = await page.evaluate(async (ms) => {
    const w = (globalThis as unknown as Worker).cachedWorker.syncWrap;
    const paths = w.CalculateAllocationPaths('3_18', [], [0]).then((p) => (p ? 'answered' : 'empty'));
    const timeout = new Promise<'timeout'>((done) => setTimeout(() => done('timeout'), ms));
    return Promise.race([paths, timeout]);
  }, 30_000);
  expect(out).toBe('answered');
  expect(errors.filter((e) => /deadlock|fatal error|unreachable|already exited/i.test(e))).toEqual([]);
});
