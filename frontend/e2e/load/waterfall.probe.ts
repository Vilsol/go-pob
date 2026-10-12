import { chromium, test, type Page, type Worker, type CDPSession } from '@playwright/test';
import { appendFileSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

import base from '../playwright.config';
import { startHosts, type AppHost } from './hosts';

type Profile = { name: string; downMbps: number; rttMs: number; serverMs: number };

const PROFILES: Record<string, Profile> = {
  local: { name: 'local', downMbps: 0, rttMs: 0, serverMs: 0 },
  broadband: { name: 'broadband', downMbps: 50, rttMs: 20, serverMs: 45 },
  mobile: { name: 'mobile', downMbps: 9, rttMs: 150, serverMs: 45 }
};

const RUNS = Number(process.env.RUNS || 1);
const WANT = (process.env.PROFILES || 'local').split(',');
const OUT = resolve(import.meta.dirname, '../.results/load-waterfall.jsonl');
for (const name of WANT) {
  if (!PROFILES[name]) throw new Error(`REFUSING: unknown profile ${name}; known: ${Object.keys(PROFILES).join(', ')}`);
}
rmSync(OUT, { force: true });

const BUILD = resolve(import.meta.dirname, '../../build');
const APP_HOST = (process.env.APP_HOST || 'gh') as AppHost;
if (APP_HOST !== 'gh' && APP_HOST !== 'cf') throw new Error(`REFUSING: unknown APP_HOST ${String(APP_HOST)}; known: gh, cf`);

type Marks = {
  origin: number;
  marks: { name: string; t: number }[];
  data: { url: string; start: number; end: number; bytes: number }[];
};

function marks(target: Page | Worker): Promise<Marks> {
  return (target as Page).evaluate(() => ({
    origin: performance.timeOrigin,
    marks: performance.getEntriesByType('mark').map((m) => ({ name: m.name, t: m.startTime })),
    data: (performance.getEntriesByType('resource') as PerformanceResourceTiming[])
      .filter((r) => r.name.startsWith('https://data.pob.gg/'))
      .map((r) => ({ url: r.name.slice(20), start: r.startTime, end: r.responseEnd, bytes: r.transferSize }))
  }));
}

async function throttle(cdp: CDPSession, p: Profile) {
  await cdp.send('Network.enable');
  await cdp.send('Network.emulateNetworkConditions', {
    offline: false,
    latency: p.rttMs,
    downloadThroughput: p.downMbps ? (p.downMbps * 1e6) / 8 : -1,
    uploadThroughput: -1
  });
}

async function visit(page: Page, appURL: string, label: string, profile: Profile) {
  const t0 = Date.now();
  await page.goto(appURL);
  const newBuild = page.getByRole('button', { name: 'New build', exact: true });
  await newBuild.waitFor({ timeout: 300_000 });
  const ready = Date.now();
  await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 300_000 });

  const main = await marks(page);
  const worker = page.workers()[0];
  const wk: Marks = worker ? await marks(worker) : { origin: main.origin, marks: [], data: [] };
  const nav = await page.evaluate(() => {
    const n = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming;
    const res = performance.getEntriesByType('resource') as PerformanceResourceTiming[];
    const wasm = res.find((r) => r.name.includes('go-pob.wasm'));
    const js = res.filter((r) => r.initiatorType === 'script' || r.name.endsWith('.js'));
    return {
      domContentLoaded: n.domContentLoadedEventEnd,
      jsBytes: js.reduce((a, r) => a + r.transferSize, 0),
      jsEnd: Math.max(0, ...js.map((r) => r.responseEnd)),
      wasmStart: wasm?.startTime ?? NaN,
      wasmEnd: wasm?.responseEnd ?? NaN,
      wasmTransfer: wasm?.transferSize ?? NaN,
      wasmEncoded: wasm?.encodedBodySize ?? NaN
    };
  });

  const shift = wk.origin - main.origin;
  const m = (name: string) => main.marks.find((x) => x.name === name)?.t ?? NaN;
  const w = (name: string) => (wk.marks.find((x) => x.name === name)?.t ?? NaN) + shift;
  const stages = wk.marks.filter((x) => x.name.startsWith('pob:data:')).map((x) => ({ name: x.name.slice(9), t: x.t + shift }));

  const before = await page.evaluate(() => Number(document.documentElement.dataset.calcSeq ?? 0));
  const c0 = await page.evaluate(() => performance.now());
  await newBuild.click();
  await page.waitForFunction((b) => Number(document.documentElement.dataset.calcSeq ?? 0) > b && document.documentElement.dataset.calcPending === '0', before, {
    timeout: 300_000
  });
  const c1 = await page.evaluate(() => performance.now());
  const deferredAt = worker
    ? await (worker as unknown as Page).evaluate(async () => {
        const start = performance.now();
        while (performance.now() - start < 60_000) {
          const mark = performance.getEntriesByName('pob:data-deferred-loaded')[0];
          if (mark) return mark.startTime;
          await new Promise((r) => setTimeout(r, 20));
        }
        return NaN;
      })
    : NaN;

  const dataReqs = wk.data.map((r) => ({ ...r, start: r.start + shift, end: r.end + shift }));
  const row = {
    profile: profile.name,
    appHost: APP_HOST,
    visit: label,
    wallToReadyMs: ready - t0,
    domContentLoaded: nav.domContentLoaded,
    jsEnd: nav.jsEnd,
    jsBytes: nav.jsBytes,
    wasmFetchStart: nav.wasmStart,
    wasmFetchEnd: nav.wasmEnd,
    wasmTransferBytes: nav.wasmTransfer,
    wasmFetched: m('pob:wasm-fetched'),
    wasmDecoded: m('pob:wasm-decoded'),
    instantiateStart: w('pob:instantiate-start'),
    instantiated: w('pob:instantiated'),
    goMain: w('pob:go-main'),
    storageReady: w('pob:storage-ready'),
    booted: m('pob:wasm-booted'),
    dataLoaded: m('pob:data-loaded'),
    firstStatsMs: c1 - c0,
    firstStatsAt: c1,
    deferredLoaded: deferredAt + shift,
    dataRequests: dataReqs.length,
    dataBytes: dataReqs.reduce((a, r) => a + r.bytes, 0),
    dataFetchSpanMs: dataReqs.length ? Math.max(...dataReqs.map((r) => r.end)) - Math.min(...dataReqs.map((r) => r.start)) : 0,
    stages,
    dataReqs
  };
  mkdirSync(resolve(OUT, '..'), { recursive: true });
  appendFileSync(OUT, JSON.stringify(row) + '\n');

  const f = (x: number) => (Number.isFinite(x) ? x.toFixed(0).padStart(6) : '     -');
  console.log(
    `${profile.name.padEnd(9)} ${label.padEnd(4)} ` +
      `js ${f(row.jsEnd)} | wasm fetch ${f(row.wasmFetchStart)}->${f(row.wasmFetchEnd)} (${(row.wasmTransferBytes / 1e6).toFixed(2)} MB) ` +
      `gunzip->${f(row.wasmDecoded)} | compile ${f(row.instantiateStart)}->${f(row.instantiated)} | go init->${f(row.goMain)} | storage->${f(row.storageReady)} | booted ${f(row.booted)} ` +
      `| data->${f(row.dataLoaded)} (${row.dataRequests} req, ${(row.dataBytes / 1e6).toFixed(2)} MB, fetch span ${f(row.dataFetchSpanMs)}) | first stats +${f(row.firstStatsMs)} | full ${f(row.deferredLoaded)}`
  );
  return row;
}

for (const name of WANT) {
  const profile = PROFILES[name];
  for (let run = 0; run < RUNS; run++) {
    test(`load waterfall ${name} #${run}`, async () => {
      test.setTimeout(20 * 60_000);
      const hosts = await startHosts(BUILD, profile.serverMs, APP_HOST);
      const profileDir = mkdtempSync(join(tmpdir(), 'pob-load-profile-'));
      const context = await chromium.launchPersistentContext(profileDir, {
        ...base.use?.launchOptions,
        viewport: base.use?.viewport,
        args: [`--host-resolver-rules=MAP data.pob.gg:443 127.0.0.1:${hosts.dataPort}`, '--ignore-certificate-errors']
      });
      try {
        const page = context.pages()[0] ?? (await context.newPage());
        if (profile.downMbps) await throttle(await context.newCDPSession(page), profile);
        await visit(page, hosts.appURL, 'cold', profile);
        await visit(page, hosts.appURL, 'warm', profile);
      } finally {
        await context.close();
        await hosts.close();
        rmSync(profileDir, { recursive: true, force: true });
      }
    });
  }
}
