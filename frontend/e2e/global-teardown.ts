import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { RESULTS_DIR, SAMPLES, type Sample } from './support/perf';

const BASELINE = resolve(import.meta.dirname, 'perf-baseline.json');

const q = (xs: number[], p: number) => {
  if (xs.length === 0) return null;
  const s = [...xs].sort((a, b) => a - b);
  return +s[Math.min(s.length - 1, Math.ceil((p / 100) * s.length) - 1)].toFixed(1);
};
const dist = (xs: number[]) => ({ n: xs.length, p50: q(xs, 50), p95: q(xs, 95), max: q(xs, 100) });
const mean = (xs: number[]) => (xs.length ? +(xs.reduce((a, b) => a + b, 0) / xs.length).toFixed(1) : null);
const groupBy = <T>(xs: T[], key: (x: T) => string) => {
  const m = new Map<string, T[]>();
  for (const x of xs) m.set(key(x), [...(m.get(key(x)) ?? []), x]);
  return m;
};
const of = <K extends Sample['kind']>(all: Sample[], kind: K) => all.filter((s): s is Extract<Sample, { kind: K }> => s.kind === kind);

export default function globalTeardown() {
  if (!existsSync(SAMPLES)) {
    console.log('\nperf: no samples were recorded');
    return;
  }
  const all = readFileSync(SAMPLES, 'utf8')
    .split('\n')
    .filter(Boolean)
    .map((l) => JSON.parse(l) as Sample);

  const cold = of(all, 'cold-start');
  const actions = of(all, 'action');
  const out = {
    note: 'Measured by `task e2e` (Chromium, one worker, production build via vite preview, game data served from the local Go bundle cache). Not a gate.',
    coldStart: {
      navToWasmBootedMs: dist(cold.map((c) => c.navToBootMs)),
      navToDataLoadedMs: dist(cold.map((c) => c.navToDataMs)),
      newBuildToFirstStatsMs: dist(cold.map((c) => c.newBuildToStatsMs))
    },
    actionToSettledMs: Object.fromEntries(
      [...groupBy(actions, (a) => a.action)].map(([k, xs]) => [
        k,
        {
          ...dist(xs.map((x) => x.settleMs)),
          meanTicks: mean(xs.map((x) => x.ticks)),
          where: {
            dispatchToWorkerMs: mean(xs.map((x) => x.dispatchMs)),
            goCalcMs: mean(xs.map((x) => x.calcMs)),
            workerSerializeMs: mean(xs.map((x) => x.dumpMs)),
            messageMs: mean(xs.map((x) => x.messageMs)),
            uiUpdateMs: mean(xs.map((x) => x.renderMs))
          }
        }
      ])
    ),
    actionToSettledByBuildMs: Object.fromEntries(
      [...groupBy(actions, (a) => a.build)].map(([b, xs]) => [
        b,
        Object.fromEntries([...groupBy(xs, (x) => x.action)].map(([k, ys]) => [k, dist(ys.map((y) => y.settleMs))]))
      ])
    ),
    longTasks: Object.fromEntries(
      [...groupBy(of(all, 'longtask'), (l) => l.phase)].map(([k, xs]) => [
        k,
        {
          samples: xs.length,
          meanCount: mean(xs.map((x) => x.count)),
          meanTotalMs: mean(xs.map((x) => x.totalMs)),
          maxMs: q(
            xs.map((x) => x.maxMs),
            100
          )
        }
      ])
    ),
    interactionToNextPaintMs: Object.fromEntries([...groupBy(of(all, 'interaction'), (i) => i.phase)].map(([k, xs]) => [k, dist(xs.map((x) => x.durationMs))])),
    treeFrames: of(all, 'frames'),
    treeLatency: of(all, 'tree-latency'),
    memory: of(all, 'memory')
  };

  writeFileSync(resolve(RESULTS_DIR, 'perf.json'), JSON.stringify(out, null, 2) + '\n');
  if (process.env.E2E_BASELINE === '1') {
    writeFileSync(BASELINE, JSON.stringify(out, null, 2) + '\n');
  }

  console.log('\n=== e2e performance (ms) ===');
  const row = (name: string, d: { n: number; p50: number | null; p95: number | null; max: number | null }) =>
    console.log(
      `  ${name.padEnd(34)} n=${String(d.n).padStart(4)}  p50=${String(d.p50).padStart(8)}  p95=${String(d.p95).padStart(8)}  max=${String(d.max).padStart(8)}`
    );
  row('cold: nav -> wasm booted', out.coldStart.navToWasmBootedMs);
  row('cold: nav -> data loaded', out.coldStart.navToDataLoadedMs);
  row('cold: New Build -> first stats', out.coldStart.newBuildToFirstStatsMs);
  for (const [k, d] of Object.entries(out.actionToSettledMs)) {
    row(`action: ${k}`, d);
    console.log(
      `  ${''.padEnd(34)} ticks=${d.meanTicks} dispatch=${d.where.dispatchToWorkerMs} go=${d.where.goCalcMs} serialize=${d.where.workerSerializeMs} msg=${d.where.messageMs} ui=${d.where.uiUpdateMs}`
    );
  }
  for (const [k, d] of Object.entries(out.interactionToNextPaintMs)) row(`INP-ish: ${k}`, d);
  for (const [k, d] of Object.entries(out.longTasks))
    console.log(`  long tasks: ${k.padEnd(22)} mean count=${d.meanCount} mean total=${d.meanTotalMs} max=${d.maxMs}`);
  for (const f of out.treeFrames)
    console.log(
      `  tree ${f.gesture}: ${f.frames} frames, ${f.dropped} dropped, max gap ${f.maxGapMs} ms over ${f.durationMs} ms` +
        (f.renderMs
          ? ` | gap p50/95/99 ${f.gapMs!.p50}/${f.gapMs!.p95}/${f.gapMs!.p99} | ${f.draws} draws, render p50/95/99 ${f.renderMs.p50}/${f.renderMs.p95}/${f.renderMs.p99} | long tasks ${f.longTasks} (${f.longTaskMs} ms)`
          : '')
    );
  for (const [k, xs] of groupBy(out.treeLatency, (l) => `${l.build.slice(0, 4)} ${l.what}`)) {
    const ms = xs.map((x) => x.ms);
    if (ms.some((m) => m === null)) console.log(`  tree latency ${k.padEnd(26)} never (${ms.filter((m) => m === null).length} of ${ms.length})`);
    else row(`tree latency ${k}`, dist(ms as number[]));
  }
  for (const m of out.memory)
    console.log(`  memory ${m.build} ${m.when}: main heap ${m.mainHeapMB} MB, worker heap ${m.workerHeapMB} MB, wasm ${m.wasmMemMB} MB`);
  console.log(`  written: ${resolve(RESULTS_DIR, 'perf.json')}${process.env.E2E_BASELINE === '1' ? ` and ${BASELINE}` : ''}`);
}
