import { execFileSync } from 'node:child_process';
import { test, expect, type Page } from '@playwright/test';
import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { record } from './support/perf';
import { expectedLines } from './support/stats';

type Rows = { ID: number; Name: string; Power: number; PathPower: number; PathDist: number; Offence: number; Defence: number }[];
type PoolWorker = {
  BuildPowerReportLive(stat: string, depth: number, workers: number): Promise<{ rows: Rows; ms: number } | undefined>;
  CancelPowerReport(): Promise<void>;
  PowerPoolWarm(workers: number): Promise<void>;
  PowerPoolStats(): Promise<{
    workers: number;
    warm?: number;
    running: number;
    bootMs: (number | undefined)[];
    memoryMB: (number | undefined)[];
    localShards: number;
  }>;
};
type Pool = { cachedWorker: { syncWrap: PoolWorker } };

const open = async (page: Page, file: string) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  return app;
};

const poolStats = (page: Page) => page.evaluate(() => (globalThis as unknown as Pool).cachedWorker.syncWrap.PowerPoolStats());

const report = (page: Page, stat: string, depth: number, workers: number) =>
  page.evaluate(
    async ([s, d, k]) => {
      const t0 = performance.now();
      const r = await (globalThis as unknown as Pool).cachedWorker.syncWrap.BuildPowerReportLive(s, d, k);
      const ms = performance.now() - t0;
      const bits = (v: number) => {
        const view = new DataView(new ArrayBuffer(8));
        view.setFloat64(0, v);
        return view.getBigUint64(0).toString(16);
      };
      return {
        ms,
        rows: r?.rows.map((x) => [x.ID, x.Name, bits(x.Power), bits(x.PathPower), x.PathDist, bits(x.Offence), bits(x.Defence)].join(' '))
      };
    },
    [stat, depth, workers] as const
  );

test('parallel workers: the pooled report is the one-worker report row for row; Cancel stops every worker', async ({ page }) => {
  test.setTimeout(15 * 60_000);
  const app = await open(page, '0280-elementalist-fireball');
  const single = await report(page, 'Life', 2, 1);
  expect(single.rows?.length, 'the one-worker report').toBeGreaterThan(100);
  expect((await poolStats(page)).workers, 'Off starts no worker').toBe(0);
  const cold = await report(page, 'Life', 2, 3);
  expect(cold.rows).toEqual(single.rows);
  expect((await poolStats(page)).workers, 'the cold report boots the workers').toBe(2);
  await expect.poll(async () => (await poolStats(page)).warm, { timeout: 60_000 }).toBe(2);
  const pooled = await report(page, 'Life', 2, 3);
  expect(pooled.rows).toEqual(single.rows);
  const stats = await poolStats(page);
  expect(stats.workers, 'the pool keeps its workers for the next report').toBe(2);
  expect(stats.running).toBe(0);
  expect(stats.localShards, 'shards the pool failed to run').toBe(0);
  console.log(`pool boot ms: ${stats.bootMs.map((v) => Math.round(v ?? -1)).join(', ')}`);

  const run = page.evaluate(() => (globalThis as unknown as Pool).cachedWorker.syncWrap.BuildPowerReportLive('FullDPS', 0, 3).then((r) => r === undefined));
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 5 * 60_000 }).toBe(2);
  await page.evaluate(() => (globalThis as unknown as Pool).cachedWorker.syncWrap.CancelPowerReport());
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 10_000 }).toBe(0);
  expect(await run, 'the cancelled report returned rows').toBe(true);

  await report(page, 'Life', 1, 1);
  expect((await poolStats(page)).workers).toBe(0);
  app.assertNoErrors();
});

test('parallel workers: Off by default; the panel runs the report on the workers the option names and closing it stops them', async ({ page }) => {
  test.setTimeout(15 * 60_000);
  const app = await open(page, '0280-elementalist-fireball');
  await page.getByRole('button', { name: 'Settings' }).click();
  const sheet = page.getByRole('dialog', { name: 'Settings' });
  await sheet.getByRole('tab', { name: /^Power report/ }).click();
  const option = sheet.getByRole('combobox', { name: 'Parallel workers' });
  await expect(option).toHaveValue('Off');
  await option.selectOption('2');
  expect(await page.evaluate(() => localStorage.getItem('options:powerWorkers'))).toBe('2');
  await sheet.getByRole('button', { name: 'Save' }).click();
  await expect(sheet).toBeHidden();
  expect(await page.evaluate(() => localStorage.getItem('options:powerWorkers')), 'Save keeps the option').toBe('2');

  await app.goTo('Tree');
  await page.getByRole('button', { name: /^Power/ }).click();
  const panel = page.getByRole('region', { name: 'Power report' });
  await panel.getByRole('combobox', { name: 'Power stat' }).selectOption('FullDPS');
  await panel.getByRole('button', { name: 'Start' }).click();
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 5 * 60_000 }).toBe(1);
  await panel.getByRole('button', { name: 'Stop' }).click();
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 10_000 }).toBe(0);
  await expect(panel.getByRole('progressbar')).toHaveCount(0);
  await expect(panel.getByRole('button', { name: 'Stop' })).toHaveCount(0);
  await panel.getByRole('button', { name: 'Start' }).click();
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 5 * 60_000 }).toBe(1);
  await panel.getByRole('button', { name: 'Close power report' }).click();
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 10_000 }).toBe(0);
  app.assertNoErrors();
});

test('parallel workers: the tree marks the node every running worker calculates', async ({ page }) => {
  test.setTimeout(15 * 60_000);
  const app = await open(page, '0280-elementalist-fireball');
  await page.getByRole('button', { name: 'Settings' }).click();
  const sheet = page.getByRole('dialog', { name: 'Settings' });
  await sheet.getByRole('tab', { name: /^Power report/ }).click();
  await sheet.getByRole('combobox', { name: 'Parallel workers' }).selectOption('3');
  await sheet.getByRole('button', { name: 'Save' }).click();
  await expect(sheet).toBeHidden();

  await app.goTo('Tree');
  await page.getByRole('button', { name: /^Power/ }).click();
  const panel = page.getByRole('region', { name: 'Power report' });
  await panel.getByRole('combobox', { name: 'Power stat' }).selectOption('FullDPS');
  await panel.getByRole('button', { name: 'Start' }).click();
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 5 * 60_000 }).toBe(2);
  const marked = new Set<string>();
  await expect
    .poll(
      async () => {
        const ids = await page.locator('[data-power-current]').evaluateAll((els) => els.map((e) => e.getAttribute('data-power-current') ?? ''));
        if (ids.length > 1) ids.forEach((id) => marked.add(id));
        return ids.length;
      },
      { timeout: 60_000 }
    )
    .toBeGreaterThan(1);
  await panel.getByRole('button', { name: 'Stop' }).click();
  await expect.poll(async () => (await poolStats(page)).running, { timeout: 10_000 }).toBe(0);
  await expect(page.locator('[data-power-current]')).toHaveCount(0);
  expect(marked.size).toBeGreaterThan(1);
  app.assertNoErrors();
});

const rendererMB = () => {
  const out = execFileSync('ps', ['-eo', 'rss=,args='], { encoding: 'utf8' });
  return (
    out
      .split('\n')
      .filter((l) => /chrom/i.test(l) && l.includes('--type=renderer'))
      .reduce((s, l) => s + Number(l.trim().split(/\s+/)[0]), 0) / 1024
  );
};

test('parallel workers: 0280 Full DPS wall-clock at 1, 2 and 4 workers', async ({ page }) => {
  test.skip(process.env.E2E_POWER_BENCH !== '1', 'E2E_POWER_BENCH=1');
  test.setTimeout(60 * 60_000);
  const file = '0280-elementalist-fireball';
  await open(page, file);
  let want: string[] | undefined;
  for (const workers of [1, 2, 4]) {
    const before = rendererMB();
    const load = execFileSync('cat', ['/proc/loadavg'], { encoding: 'utf8' }).trim();
    let peak = before;
    const sampler = setInterval(() => (peak = Math.max(peak, rendererMB())), 1000);
    const r = await report(page, 'FullDPS', 0, workers);
    clearInterval(sampler);
    const stats = await poolStats(page);
    want ??= r.rows;
    expect(r.rows, `${workers} workers`).toEqual(want);
    const row = {
      kind: 'power-pool',
      build: file,
      workers,
      ms: Math.round(r.ms),
      load,
      bootMs: stats.bootMs,
      poolWasmMB: stats.memoryMB.map((v) => Math.round(v ?? -1)),
      rendererMB: Math.round(before),
      peakMB: Math.round(peak)
    };
    console.log(JSON.stringify(row));
    record({ kind: 'tree-latency', build: file, what: `power-report-FullDPS-${workers}w`, ms: Math.round(r.ms) });
    if (workers > 1) {
      await report(page, 'Life', 1, 1);
      const t0 = Date.now();
      await report(page, 'Life', 1, workers);
      const warm = await poolStats(page);
      console.log(JSON.stringify({ kind: 'power-pool-warm', workers, bootMs: warm.bootMs, lifeReportMs: Date.now() - t0 }));
      await report(page, 'Life', 1, 1);
    }
  }
});

test('parallel workers: break-even by stat and depth, cold and warm', async ({ page }) => {
  test.skip(process.env.E2E_POWER_BENCH !== '1', 'E2E_POWER_BENCH=1');
  test.setTimeout(60 * 60_000);
  const file = process.env.E2E_POWER_BUILD ?? '0280-elementalist-fireball';
  await open(page, file);
  const cases = (process.env.E2E_POWER_CASES ?? 'Life:2 Life:1 Life:0 CombinedDPS:2 FullDPS:2').split(' ');
  for (const c of cases) {
    const [stat, depth] = [c.split(':')[0], Number(c.split(':')[1])];
    const load = () => execFileSync('cat', ['/proc/loadavg'], { encoding: 'utf8' }).trim().split(' ')[0];
    const one = await report(page, stat, depth, 1);
    const cold = await report(page, stat, depth, 4);
    await expect.poll(async () => (await poolStats(page)).warm, { timeout: 60_000 }).toBe(3);
    const warm = await report(page, stat, depth, 4);
    expect(cold.rows).toEqual(one.rows);
    expect(warm.rows).toEqual(one.rows);
    console.log(
      JSON.stringify({
        kind: 'power-pool-break-even',
        build: file,
        case: c,
        rows: one.rows?.length,
        off: Math.round(one.ms),
        cold: Math.round(cold.ms),
        warm: Math.round(warm.ms),
        load: load()
      })
    );
  }
});
