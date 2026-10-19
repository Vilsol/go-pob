import { expect, type Page } from '@playwright/test';

import { DATA_HOST, cachedAsset } from './data_cache';
import { PERF, record } from './perf';
import { routeTreeAssets } from './tree_assets';
import { diffLines, type StatLine } from './stats';

type Timing = { seq: number; pending: number; startedAt: number; calcMs: number; dumpMs: number; sentAt: number; receivedAt: number; renderedAt: number };

const initPerf = () => {
  const perf = { longtasks: [] as { start: number; dur: number }[], events: [] as { name: string; dur: number; id: number }[] };
  (globalThis as unknown as { __pobPerf: typeof perf }).__pobPerf = perf;
  try {
    new PerformanceObserver((l) => {
      for (const e of l.getEntries()) perf.longtasks.push({ start: e.startTime, dur: e.duration });
    }).observe({ type: 'longtask', buffered: true });
    new PerformanceObserver((l) => {
      for (const e of l.getEntries() as unknown as { name: string; duration: number; interactionId: number }[]) {
        if (e.interactionId) perf.events.push({ name: e.name, dur: e.duration, id: e.interactionId });
      }
    }).observe({ type: 'event', buffered: true, durationThreshold: 16 } as PerformanceObserverInit);
  } catch {
    // empty
  }
};

const SETTLE_TIMEOUT = 180_000;
const ACTION_TIMEOUT = 30_000;
const FOLLOW_UP_TIMEOUT = 5_000;

export class App {
  readonly errors: string[] = [];
  private served404 = new Set<string>();

  private constructor(
    readonly page: Page,
    public build: string,
    readonly phone = false
  ) {}

  // phone: the page is below the phone breakpoint, so the stats live in the closed stats sheet behind the strip
  static async open(page: Page, build: string, { phone = false } = {}): Promise<App> {
    const app = new App(page, build, phone);

    await page.context().route(`${DATA_HOST}**`, async (route) => {
      const url = route.request().url();
      let asset;
      try {
        asset = cachedAsset(url);
      } catch (e) {
        app.errors.push(String(e));
        return route.abort();
      }
      if (asset.status === 404) {
        app.served404.add(url);
        return route.fulfill({ status: 404, headers: { 'access-control-allow-origin': '*' } });
      }
      return route.fulfill({ status: 200, body: asset.body, headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' } });
    });
    await routeTreeAssets(page);

    page.on('console', (m) => {
      if (process.env.E2E_DEBUG === '1') console.log(`[browser ${m.type()}] ${m.text().slice(0, 300)}`);
      if (m.type() !== 'error') return;
      if (m.text().startsWith('Failed to load resource') && app.served404.has(m.location().url)) return;
      const what = m.text().startsWith('Failed to load resource') ? ` (${m.location().url})` : '';
      app.errors.push(`console.error: ${m.text().slice(0, 500)}${what}`);
    });
    page.on('pageerror', (e) => app.errors.push(`pageerror: ${e.message.slice(0, 500)}`));

    await page.addInitScript(initPerf);
    await page.goto('/');

    const stalled = async (stage: string, e: unknown) => {
      const reached = await page
        .evaluate(() => performance.getEntriesByType('mark').map((m) => `${m.name}@${m.startTime.toFixed(0)}`))
        .catch(() => ['(page gone)']);
      throw new Error(`App.open stalled waiting for ${stage}; marks: ${reached.join(', ') || 'none'}\n${String(e)}`);
    };
    const newBuild = page.getByRole('button', { name: 'New build', exact: true });
    await newBuild.waitFor({ timeout: SETTLE_TIMEOUT }).catch((e: unknown) => stalled('the New build button', e));
    await page
      .waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: SETTLE_TIMEOUT })
      .catch((e: unknown) => stalled('pob:data-loaded', e));
    const marks = await page.evaluate(() => ({
      boot: performance.getEntriesByName('pob:wasm-booted')[0]?.startTime ?? NaN,
      data: performance.getEntriesByName('pob:data-loaded')[0]?.startTime ?? NaN
    }));

    const before = await app.seq();
    const t0 = await app.now();
    await newBuild.click();
    await app.waitSettled(before);
    if (phone) await expect(page.locator('[data-stats-strip] [data-strip-cell]').first()).toBeVisible();
    else await expect(app.statsRegion().getByRole('button').first()).toBeVisible();
    const last = (await app.timingsSince(before)).at(-1)!;

    record({ kind: 'cold-start', build, navToBootMs: marks.boot, navToDataMs: marks.data, newBuildToStatsMs: last.renderedAt - t0 });
    await app.flushPerf('cold-start');
    return app;
  }

  statsRegion() {
    return this.page.getByRole('region', { name: 'Build stats', includeHidden: this.phone });
  }

  savedBuild(name: string) {
    return this.page
      .getByRole('grid', { name: 'Builds' })
      .getByRole('row')
      .filter({ has: this.page.getByRole('link', { name, exact: true }) });
  }

  async openSaved(name: string) {
    await this.savedBuild(name).dblclick({ position: { x: 4, y: 4 } });
  }

  async now(): Promise<number> {
    return this.page.evaluate(() => performance.timeOrigin + performance.now());
  }

  async seq(): Promise<number> {
    return this.page.evaluate(() => Number(document.documentElement.dataset.calcSeq ?? 0));
  }

  async waitSettled(before: number, timeout = SETTLE_TIMEOUT) {
    await this.page.waitForFunction(
      (b) => {
        const d = document.documentElement.dataset;
        return Number(d.calcSeq ?? 0) > b && d.calcPending === '0';
      },
      before,
      { timeout }
    );
  }

  async followWorker<T>(since: number, wait: () => Promise<T>): Promise<T> {
    for (;;) {
      try {
        return await wait();
      } catch (e) {
        if (!(await this.idleJobEnded(since))) throw e;
        since = await this.now();
      }
    }
  }

  async idleJobDone() {
    await expect.poll(() => this.idleJobEnded(0), { timeout: SETTLE_TIMEOUT }).toBe(true);
  }

  private async idleJobEnded(since: number): Promise<boolean> {
    const worker = this.page.workers()[0];
    if (!worker) return false;
    const read = () =>
      worker.evaluate(() => {
        const at = (n: string) => performance.getEntriesByName(n).map((m) => performance.timeOrigin + m.startTime);
        return { start: at('pob:idle-job-start'), end: at('pob:idle-job-end') };
      });
    let marks = await read();
    if (marks.start.length === 0 || Math.max(...marks.start, ...marks.end) < since) return false;
    await expect.poll(async () => (marks = await read()).end.length, { timeout: SETTLE_TIMEOUT }).toBe(marks.start.length);
    return true;
  }

  async timingsSince(before: number): Promise<Timing[]> {
    return this.page.evaluate((b) => (globalThis.__pobCalcTimings ?? []).filter((t) => t.seq > b), before);
  }

  async readStats(): Promise<StatLine[]> {
    return this.statsRegion().evaluate((root) => {
      const split = (t: string) => {
        const m = /^(.*?):[ \u00a0](.*)$/s.exec(t);
        return m ? { label: m[1], value: m[2] } : { label: t, value: '<missing>' };
      };
      const out: { label: string; value: string }[] = [];
      for (const el of root.querySelectorAll('button, ul[aria-label="Full DPS by skill"] > li')) {
        if (el.tagName === 'LI') {
          const [head, ...details] = [...el.children];
          out.push(split(head?.textContent ?? ''));
          for (const d of details) out.push({ label: '', value: d.textContent ?? '' });
          continue;
        }
        if (el.hasAttribute('aria-pressed') || el.hasAttribute('aria-expanded')) continue;
        out.push(split(el.textContent ?? ''));
      }
      return out;
    });
  }

  async act(action: string, fn: () => Promise<unknown>, want: () => Promise<StatLine[]>) {
    const before = await this.seq();
    const t0 = await this.now();
    await fn();
    try {
      await this.followWorker(t0, () => this.waitSettled(before, ACTION_TIMEOUT));
    } catch {
      const seen = await this.page
        .evaluate(() => {
          const el = document.activeElement as HTMLInputElement | null;
          const d = document.documentElement.dataset;
          return `focused ${el?.tagName.toLowerCase()} "${el?.getAttribute('aria-label') ?? ''}" = ${JSON.stringify(el?.value)}, calcSeq ${d.calcSeq}, calcPending ${d.calcPending}`;
        })
        .catch((e: unknown) => String(e));
      throw new Error(
        `${action} on ${this.build}: no recalculation was rendered within ${ACTION_TIMEOUT / 1000}s — the write never reached Go, or reached it without a Tick (${seen}; before: calcSeq ${before})`
      );
    }

    const expected = await want();
    expect(expected.length, `${action}: the model produced no sidebar lines, so a match would be vacuous`).toBeGreaterThan(0);
    let diffs = diffLines(expected, await this.readStats());
    while (diffs.length > 0) {
      try {
        await this.waitSettled(await this.seq(), FOLLOW_UP_TIMEOUT);
      } catch {
        diffs = diffLines(expected, await this.readStats());
        break;
      }
      diffs = diffLines(expected, await this.readStats());
    }

    const ticks = await this.timingsSince(before);
    const first = ticks[0];
    const last = ticks.at(-1)!;
    const sum = (f: (t: Timing) => number) => ticks.reduce((a, t) => a + f(t), 0);
    record({
      kind: 'action',
      action,
      build: this.build,
      settleMs: last.renderedAt - t0,
      ticks: ticks.length,
      calcMs: sum((t) => t.calcMs),
      dumpMs: sum((t) => t.dumpMs),
      messageMs: sum((t) => t.receivedAt - t.sentAt),
      renderMs: sum((t) => t.renderedAt - t.receivedAt),
      dispatchMs: first.startedAt - t0
    });
    await this.flushPerf(action);

    expect(diffs, `${action} on ${this.build}: displayed stats differ from Go`).toEqual([]);
  }

  async flushPerf(phase: string) {
    if (!PERF) return;
    const p = await this.page.evaluate(() => {
      const perf = (globalThis as unknown as { __pobPerf?: { longtasks: { dur: number }[]; events: { name: string; dur: number; id: number }[] } }).__pobPerf;
      if (!perf) return { longtasks: [], events: [] };
      const r = { longtasks: perf.longtasks.slice(), events: perf.events.slice() };
      perf.longtasks.length = 0;
      perf.events.length = 0;
      return r;
    });
    const durs = p.longtasks.map((l) => l.dur);
    record({ kind: 'longtask', build: this.build, phase, count: durs.length, totalMs: durs.reduce((a, b) => a + b, 0), maxMs: Math.max(0, ...durs) });
    const byInteraction = new Map<number, { name: string; dur: number }>();
    for (const e of p.events) {
      const prev = byInteraction.get(e.id);
      if (!prev || e.dur > prev.dur) byInteraction.set(e.id, e);
    }
    for (const e of byInteraction.values()) record({ kind: 'interaction', build: this.build, phase, name: e.name, durationMs: e.dur });
  }

  async memory(when: string) {
    const cdp = await this.page.context().newCDPSession(this.page);
    await cdp.send('HeapProfiler.collectGarbage');
    const main = (await cdp.send('Runtime.getHeapUsage')).usedSize;
    await cdp.detach();
    const worker = this.page.workers()[0];
    const w = worker
      ? await worker.evaluate(() => ({
          heap: (performance as unknown as { memory?: { usedJSHeapSize: number } }).memory?.usedJSHeapSize ?? null,
          wasm: (globalThis as unknown as { __pobWasmMemory?: WebAssembly.Memory }).__pobWasmMemory?.buffer.byteLength ?? null,
          live: (globalThis as unknown as { __pobLiveHeapMB?: () => number }).__pobLiveHeapMB?.() ?? null
        }))
      : { heap: null, wasm: null, live: null };
    const mb = (n: number | null) => (n === null ? null : +(n / 2 ** 20).toFixed(1));
    const goLiveHeapMB = w.live === null ? null : +w.live.toFixed(1);
    record({ kind: 'memory', build: this.build, when, mainHeapMB: mb(main)!, workerHeapMB: mb(w.heap), wasmMemMB: mb(w.wasm), goLiveHeapMB });
    return goLiveHeapMB;
  }

  async goTo(link: string) {
    const target = this.page.getByRole('link', { name: link });
    const href = await target.getAttribute('href');
    await target.click();
    if (href) await this.page.waitForURL((u) => u.pathname === new URL(href, u).pathname);
  }

  async openBuildControls() {
    await this.goTo('Tree');
  }

  async importCode(code: string, want: () => Promise<StatLine[]>) {
    await this.goTo('Import / Export');
    const region = this.page.getByRole('region', { name: 'Build import' });
    const box = region.getByRole('textbox', { name: /^To import a build/ });
    const t0 = await this.now();
    await box.fill(code);
    await this.followWorker(t0, () => expect(box).toHaveAccessibleDescription('Code is valid'));
    await expect(region.getByRole('radio', { name: /Import to this build/ })).toBeChecked();
    // ImportTab.lua:648-653
    await this.act(
      'import',
      async () => {
        await region.getByRole('button', { name: 'Import', exact: true }).click();
        await this.page.getByRole('dialog', { name: 'Build Import' }).getByRole('button', { name: 'Import', exact: true }).click();
      },
      want
    );
  }

  async openTree(version?: string) {
    await this.goTo('Tree');
    await expect(this.page.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
    await expect(this.page.getByRole('searchbox', { name: 'Search the tree' })).toBeVisible();
    if (version) {
      await expect(this.page.getByLabel('Tree version'), 'the tree page draws another version than the build is calculated on').toContainText(
        `Tree ${version.replace('_', '.')}`
      );
    }
  }

  /* PassiveSpec.lua:1614-1635 */
  async treeNode(name: string) {
    const id = / (#\d+)$/.exec(name)?.[1];
    if (!id) throw new Error(`treeNode: "${name}" is not a search result's name ("<node> #<id>")`);
    await this.page.getByRole('searchbox', { name: 'Search the tree' }).fill(id);
    return this.page.getByRole('region', { name: 'Search results' }).getByRole('button', { name: new RegExp(` ${id}$`) });
  }

  async exportCode(): Promise<string> {
    await this.goTo('Notes');
    await this.goTo('Import / Export');
    const box = this.page.getByRole('region', { name: 'Build sharing' }).getByRole('textbox', { name: 'Build code' });
    let code = '';
    await expect.poll(async () => (code = await box.inputValue())).not.toBe('');
    return code;
  }

  assertNoErrors() {
    expect(this.errors, `console errors / unhandled exceptions on ${this.build}`).toEqual([]);
  }
}
