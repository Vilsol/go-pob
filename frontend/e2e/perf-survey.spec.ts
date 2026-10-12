import { test, expect, type Page, type CDPSession } from '@playwright/test';
import { writeFileSync, mkdirSync } from 'node:fs';
import { resolve } from 'node:path';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { PERF } from './support/perf';

test.skip(!PERF, 'perf-survey records samples; run it with E2E_PERF=1');
import { RESULTS_DIR } from './support/perf';
import { expectedLines } from './support/stats';

const BUILD = process.env.SURVEY_BUILD || '0001-necromancer-unknown';
const PROFILE = process.env.SURVEY_PROFILE === '1';
const QUIET_MS = 500;
const TRACES = (process.env.SURVEY_TRACE ?? '').split(',').filter((x) => x !== '');
const TRACE_CATEGORIES = ['disabled-by-default-v8.cpu_profiler', 'v8', 'devtools.timeline', 'disabled-by-default-devtools.timeline'];
const tracePath = (step: string) => resolve(RESULTS_DIR, `trace-${step.replace(/[^a-z0-9]+/gi, '_')}.json`);

type Step = {
  step: string;
  waitMs: number;
  workerIdleMs: number;
  longTasks: number;
  longTaskMs: number;
  maxLongTaskMs: number;
  profile?: { fn: string; selfMs: number }[];
};

const steps: Step[] = [];

const quiet = (page: Page, from: number) =>
  page.evaluate(
    ([t0, q]) =>
      new Promise<{ end: number; tasks: { start: number; dur: number }[] }>((done) => {
        const perf = (globalThis as unknown as { __pobPerf: { longtasks: { start: number; dur: number }[] } }).__pobPerf;
        const origin = performance.timeOrigin;
        let busy = 0;
        const tick = () => {
          const tasks = perf.longtasks.filter((l) => l.start + origin >= t0 - 5);
          const now = performance.now();
          if (document.documentElement.dataset.calcPending !== '0' && document.documentElement.dataset.calcPending !== undefined) busy = now;
          busy = Math.max(busy, ...tasks.map((l) => l.start + l.dur));
          if (now - busy >= q) {
            done({ end: busy + origin, tasks });
            return;
          }
          setTimeout(tick, 20);
        };
        requestAnimationFrame(() => {
          busy = performance.now();
          tick();
        });
      }),
    [from, QUIET_MS] as const
  );

const summarise = (profile: {
  nodes: { id: number; callFrame: { functionName: string; url: string; lineNumber: number } }[];
  samples: number[];
  timeDeltas: number[];
}) => {
  const byId = new Map(profile.nodes.map((n) => [n.id, n]));
  const self = new Map<string, number>();
  for (let i = 0; i < profile.samples.length; i++) {
    const n = byId.get(profile.samples[i])!;
    const f = n.callFrame;
    const key = `${f.functionName || '(anon)'} ${f.url.split('/').pop()}:${f.lineNumber + 1}`;
    self.set(key, (self.get(key) ?? 0) + (profile.timeDeltas[i] ?? 0) / 1000);
  }
  return [...self.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 25)
    .map(([fn, selfMs]) => ({ fn, selfMs: +selfMs.toFixed(1) }));
};

const measure = async (app: App, cdp: CDPSession | undefined, step: string, action: () => Promise<unknown>) => {
  if (cdp) await cdp.send('Profiler.start');
  const tracing = TRACES.some((t) => step.includes(t));
  if (tracing)
    await app.page
      .context()
      .browser()!
      .startTracing(app.page, { path: tracePath(step), categories: TRACE_CATEGORIES });
  const t0 = await app.now();
  await action();
  const q = await quiet(app.page, t0);
  const idleAt = await app.page.evaluate(async () => {
    const w = (globalThis as unknown as { cachedWorker: { syncWrap: { GetTreeVersion: () => Promise<string> } } }).cachedWorker.syncWrap;
    for (;;) {
      const s = performance.now();
      await w.GetTreeVersion();
      if (performance.now() - s < 30) return performance.timeOrigin + s;
      await new Promise((r) => setTimeout(r, 50));
    }
  });
  if (tracing) await app.page.context().browser()!.stopTracing();
  const durs = q.tasks.map((t) => t.dur);
  const s: Step = {
    step,
    waitMs: +(q.end - t0).toFixed(1),
    workerIdleMs: +(Math.max(idleAt, q.end) - t0).toFixed(1),
    longTasks: durs.length,
    longTaskMs: +durs.reduce((a, b) => a + b, 0).toFixed(1),
    maxLongTaskMs: +Math.max(0, ...durs).toFixed(1)
  };
  if (cdp) s.profile = summarise((await cdp.send('Profiler.stop')).profile as never);
  steps.push(s);
  console.log(
    `${step.padEnd(28)} wait ${String(s.waitMs).padStart(8)} ms  worker idle ${String(s.workerIdleMs).padStart(8)} ms  long tasks ${s.longTasks} / ${s.longTaskMs} ms (max ${s.maxLongTaskMs})`
  );
};

test(`perf survey on ${BUILD}`, async ({ page }) => {
  test.skip(process.env.SURVEY !== '1', 'a measurement, run on request (SURVEY=1)');
  test.setTimeout(30 * 60_000);
  const xml = buildXml(BUILD);
  const build = await go.loadXml(xml);
  const want = async () => expectedLines(await go.outputs(build));

  if (TRACES.includes('cold'))
    await page
      .context()
      .browser()!
      .startTracing(page, { path: tracePath('cold'), categories: TRACE_CATEGORIES });
  const app = await App.open(page, BUILD);
  if (TRACES.includes('cold')) await page.context().browser()!.stopTracing();
  const marks = await page.evaluate(() => performance.getEntriesByType('mark').map((m) => [m.name, +m.startTime.toFixed(0)]));
  const workerMarks = await page
    .workers()[0]
    ?.evaluate(() => performance.getEntriesByType('mark').map((m) => [m.name, +(m.startTime + performance.timeOrigin).toFixed(0)]));
  console.log('main marks', JSON.stringify(marks));
  console.log('worker marks (epoch)', JSON.stringify(workerMarks), 'main origin', await page.evaluate(() => performance.timeOrigin));

  const cdp = PROFILE ? await page.context().newCDPSession(page) : undefined;
  if (cdp) {
    await cdp.send('Profiler.enable');
    await cdp.send('Profiler.setSamplingInterval', { interval: 200 });
  }

  const code = await go.encode(xml);
  await measure(app, cdp, 'import', () => app.importCode(code, want));
  await app.memory('after-import');

  for (const round of [1, 2]) {
    for (const link of ['Tree', 'Skills', 'Items', 'Calcs', 'Config', 'Notes', 'Import / Export']) {
      await measure(app, cdp, `nav ${link} #${round}`, () => app.goTo(link));
    }
  }

  await app.goTo('Tree');
  const level = page.getByRole('spinbutton', { name: 'Character level' });
  const original = build.Build.Level;
  for (const n of [original - 1, original]) {
    await measure(app, cdp, `set-level ${n}`, () =>
      app.act(
        'set-level',
        () => level.fill(String(n)),
        async () => {
          // Build.lua:218-224
          build.SetCharacterLevelAutoMode(false);
          build.SetLevel(n);
          return want();
        }
      )
    );
  }

  for (let r = 0; r < 3; r++) {
    const before = await app.seq();
    await level.fill(String(original - 1));
    await app.waitSettled(before);
    const second = await app.seq();
    const t0 = await app.now();
    await level.fill(String(original));
    await app.waitSettled(second);
    const rendered = (await app.timingsSince(second)).at(-1)!.renderedAt;
    const s: Step = { step: `back-to-back level #${r}`, waitMs: +(rendered - t0).toFixed(1), workerIdleMs: NaN, longTasks: 0, longTaskMs: 0, maxLongTaskMs: 0 };
    steps.push(s);
    console.log(`${s.step.padEnd(28)} second edit rendered after ${s.waitMs} ms`);
    await page.waitForTimeout(1500);
  }

  const select = page.getByRole('combobox', { name: 'Main skill' });
  const opts = await select.evaluate((el) => {
    const s = el as HTMLSelectElement;
    return { original: s.value, other: [...s.options].map((o) => o.value).find((v) => v !== s.value) };
  });
  if (opts.other) {
    for (const v of [opts.other, opts.original]) {
      await measure(app, cdp, `main-skill ${v}`, async () => {
        const before = await app.seq();
        await select.selectOption(v);
        await app.waitSettled(before);
      });
    }
  }

  const restoreCalc = async (before: number) => {
    const ticks = await app.timingsSince(before);
    console.log(`${''.padEnd(28)} ticks ${ticks.length}, calc ${ticks.map((t) => t.calcMs.toFixed(1)).join(' + ')} ms`);
  };
  for (const step of ['Calcs', 'Config', 'Items', 'Skills']) {
    await app.goTo(step);
    let before = await app.seq();
    await measure(app, cdp, `set-level on ${step}`, async () => {
      before = await app.seq();
      await page.keyboard.press('Control+z');
      await app.waitSettled(before).catch(() => undefined);
    });
    await restoreCalc(before);
    await measure(app, cdp, `redo on ${step}`, async () => {
      before = await app.seq();
      await page.keyboard.press('Control+Shift+z');
      await app.waitSettled(before).catch(() => undefined);
    });
    await restoreCalc(before);
  }
  await app.memory('end');

  mkdirSync(RESULTS_DIR, { recursive: true });
  const out = resolve(RESULTS_DIR, `survey-${BUILD}${PROFILE ? '-profile' : ''}.json`);
  writeFileSync(out, JSON.stringify({ build: BUILD, marks, workerMarks, steps }, null, 1));
  console.log('wrote', out);
  expect(steps.length).toBeGreaterThan(0);
});
