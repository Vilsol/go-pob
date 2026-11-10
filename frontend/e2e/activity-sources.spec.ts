import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

type Phase = { name: string; done?: number; total?: number; unit?: string };
type Logged = { kind: string; state: string; phases: Phase[] };

const finished = (page: Page, kind: string) =>
  page.evaluate(
    (k) =>
      ((globalThis as unknown as { __pobJobs?: Logged[] }).__pobJobs ?? [])
        .filter((j) => j.kind === k)
        .map((j): Logged => ({ kind: j.kind, state: j.state, phases: j.phases.map(({ name, done, total, unit }) => ({ name, done, total, unit })) })),
    kind
  );

const routeData = async (page: Page) => {
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
};

test('the game data job counts the files it downloads and the tables it loads', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await routeData(page);
  await page.goto('/');
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });
  const [data] = await finished(page, 'data');
  expect(data.state).toBe('ok');
  const [download, ...rest] = data.phases;
  expect(download.name).toBe('Downloading');
  expect(download.total).toBeGreaterThan(1);
  expect(download.done).toBe(download.total);
  expect(download.unit).toBe('count');
  const tables = rest.slice(0, -1);
  expect(tables.length).toBeGreaterThan(1);
  expect(tables.map((p) => [p.done, p.total])).toEqual(tables.map((_, i) => [i, tables.length]));
  expect(rest.at(-1)).toEqual({ name: 'Indexing', done: undefined, total: undefined, unit: undefined });
  expect(errors).toEqual([]);
});

test('a PoE2 link boots the PoE2 engine as a job with its download in bytes, and its game data loads as a job', async ({ page }) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await page.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
  await routeData(page);
  const xml = readFileSync(resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml'));
  const code = deflateSync(xml).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
  await page.route('**/api/build?*', (route) => route.fulfill({ status: 200, contentType: 'text/plain; charset=utf-8', body: code }));
  await page.goto(`/?import=${encodeURIComponent('https://poe.ninja/poe2/pob/Abc1')}`);
  await expect(poe2Stat(page, 'Life')).toHaveText('683', { timeout: 180_000 });

  const [boot] = await finished(page, 'poe2-boot');
  expect(boot.state).toBe('ok');
  const names = boot.phases.map((p) => p.name);
  expect(names[0]).toBe('Loading the engine code');
  expect(names[1]).toMatch(/^Downloading the engine( \(compressed\))?$/);
  expect(names.at(-1)).toBe('Booting the engine');
  const download = boot.phases[1];
  expect(download.unit).toBe('bytes');
  expect(download.done).toBeGreaterThan(1_000_000);
  if (download.total !== undefined) expect(download.done).toBe(download.total);

  await expect
    .poll(async () => (await finished(page, 'poe2-data')).map((j) => [j.state, j.phases.map((p) => [p.name, p.done, p.total])]), { timeout: 60_000 })
    .toEqual([
      [
        'ok',
        [
          ['Passive tree', undefined, undefined],
          ['Item tables', 0, 5],
          ['Mod parser', 1, 5],
          ['Gems', 2, 5],
          ['Skill data', 3, 5],
          ['Cost divisors', 4, 5]
        ]
      ]
    ]);
  expect(errors).toEqual([]);
});
