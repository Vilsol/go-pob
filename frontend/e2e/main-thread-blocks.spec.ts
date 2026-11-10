// Main-thread blocks over a walk through the app: long tasks per phase, and a CPU profile and a trace per phase written
// to .results/main-thread/<phase>.{cpuprofile,trace.json}. A profiling run, not a check:
//   E2E_MAIN_THREAD=1 E2E_SLOW=1 E2E_PERF=1 npx playwright test -c e2e/playwright.config.ts e2e/main-thread-blocks.spec.ts
// (E2E_GPU=1 for the WebGL renderers). Headless Chromium composites in software even then: each WebGL frame is read back
// inside the main thread's Commit, which first waits for every queued GPU command, so a long Commit after a texture
// upload is the harness's cost, not one a GPU-composited page pays.
import { test, expect, type CDPSession, type Page } from '@playwright/test';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { DATA_HOST, cachedAsset } from './support/data_cache';
import * as go from './support/go_model';
import { RESULTS_DIR } from './support/perf';
import { expectedLines } from './support/stats';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page } from './support/poe2';

const OUT = resolve(RESULTS_DIR, 'main-thread');
test.skip(process.env.E2E_MAIN_THREAD !== '1', 'a profiling run: E2E_MAIN_THREAD=1');
const SETTLE_MS = 3000;

type Perf = { longtasks: { start: number; dur: number }[] };

const longtasks = (page: Page) =>
  page.evaluate(() => {
    const perf = (globalThis as unknown as { __pobPerf?: Perf }).__pobPerf;
    if (!perf) return [];
    const r = perf.longtasks.slice();
    perf.longtasks.length = 0;
    return r;
  });

class Phases {
  private constructor(
    private page: Page,
    private cdp: CDPSession,
    private prefix: string
  ) {}

  static async start(page: Page, prefix: string) {
    mkdirSync(OUT, { recursive: true });
    const cdp = await page.context().newCDPSession(page);
    await cdp.send('Profiler.enable');
    await cdp.send('Profiler.setSamplingInterval', { interval: 250 });
    await cdp.send('Profiler.start');
    await Phases.trace(page);
    return new Phases(page, cdp, prefix);
  }

  private static trace = (page: Page) =>
    page
      .context()
      .browser()!
      .startTracing(page, { categories: ['devtools.timeline', 'disabled-by-default-devtools.timeline', 'v8', 'blink', 'toplevel'] });

  async finish() {
    await this.page.context().browser()!.stopTracing();
    await this.cdp.send('Profiler.stop');
  }

  async end(phase: string) {
    await this.page.waitForTimeout(SETTLE_MS);
    const lts = await longtasks(this.page);
    const { profile } = await this.cdp.send('Profiler.stop');
    const name = `${this.prefix}-${phase}`;
    writeFileSync(resolve(OUT, `${name}.cpuprofile`), JSON.stringify(profile));
    writeFileSync(resolve(OUT, `${name}.trace.json`), await this.page.context().browser()!.stopTracing());
    const durs = lts.map((l) => Math.round(l.dur));
    const line = `${name}\tcount=${durs.length}\ttotal=${durs.reduce((a, b) => a + b, 0)}\tmax=${Math.max(0, ...durs)}\t[${durs.filter((d) => d > 50).join(',')}]`;
    console.log(line);
    writeFileSync(resolve(OUT, 'longtasks.tsv'), line + '\n', { flag: 'a' });
    await this.cdp.send('Profiler.start');
    await Phases.trace(this.page);
  }
}

test('PoE1: boot, import the largest build, every page, the item DB, a recalculation', { tag: '@slow' }, async ({ page }) => {
  test.setTimeout(600_000);
  const file = '0001-necromancer-unknown';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const p = await Phases.start(page, 'poe1');
  const app = await App.open(page, file);
  await p.end('boot');
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.openTree();
  await p.end('import-to-tree');
  for (const link of ['Items', 'Calcs', 'Skills', 'Config', 'Tree']) {
    await app.goTo(link);
    await p.end(`page-${link}`);
  }
  await app.goTo('Items');
  await expect(page.getByRole('region', { name: 'Equipped items' })).toBeVisible();
  await p.end('page-Items-again');
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  await expect(page.getByRole('table', { name: 'Unique list' })).toBeVisible({ timeout: 60_000 });
  await p.end('item-db');
  await app.goTo('Tree');
  await p.end('page-Tree-again');
  const level = page.getByRole('spinbutton', { name: 'Character level' });
  const before = await app.seq();
  await level.fill('50');
  await app.waitSettled(before);
  await p.end('recalc-level');
  await app.goTo('Compare');
  await p.end('page-Compare');
  await p.finish();
});

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_items.xml');
const code = (path: string) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

test('PoE2: open a build, every page, the tree', { tag: '@slow' }, async ({ page }) => {
  test.setTimeout(600_000);
  await page.addInitScript(() => {
    localStorage.setItem('options:poe2', 'true');
    const perf: Perf = { longtasks: [] };
    (globalThis as unknown as { __pobPerf: Perf }).__pobPerf = perf;
    new PerformanceObserver((l) => {
      for (const e of l.getEntries()) perf.longtasks.push({ start: e.startTime, dur: e.duration });
    }).observe({ type: 'longtask', buffered: true });
  });
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  const p = await Phases.start(page, 'poe2');
  await page.goto(`/calcs?import=${encodeURIComponent(code(FIXTURE))}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  await p.end('boot-open');
  // the second Tree visit is named apart, so its files do not overwrite the first visit's
  for (const [link, phase] of [
    ['Tree', 'Tree'],
    ['Items', 'Items'],
    ['Skills', 'Skills'],
    ['Config', 'Config'],
    ['Calcs', 'Calcs'],
    ['Tree', 'Tree-again']
  ]) {
    await poe2Nav(page, link).click();
    await p.end(`page-${phase}`);
  }
  await p.finish();
});
