import { expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { App } from './app';
import { DATA_HOST, cachedAsset } from './data_cache';
import { poe2Nav, poe2Page } from './poe2';
import { routeTreeAssets } from './tree_assets';
import { installTreeProbe } from './tree_probe';
import * as tree from './tree_model';

type Frame = { at: number; ms: number; renderer?: string; complete?: boolean };
type Phase = { name: string; at: number; ms?: number };
export type Revisit = {
  game: string;
  renderer: string;
  firstOpenMs: number;
  returnMs: number;
  returnFrames: number;
  contexts: number;
  glWarnings: string[];
  // the first open's phases, in ms from the tab click; the frames it drew and the longest gap between two
  phases?: Phase[];
  firstFrames?: number;
  longestGapMs?: number;
};

// the browser's WebGL warnings (Firefox reports a texture it had to clear or fill lazily), counted by their text
const glWarnings = (page: Page) => {
  const seen: string[] = [];
  page.on('console', (m) => {
    if (/WebGL/i.test(m.text())) seen.push(m.text().replace(/\d+/g, 'N').slice(0, 160));
  });
  page.on('pageerror', (e) => console.log(`pageerror: ${e.message}\n${e.stack ?? ''}`));
  return seen;
};

// counts the WebGL2 contexts the page asks for, each one a renderer built (or the hardware probe)
const countContexts = () => {
  const getContext = Object.getOwnPropertyDescriptor(HTMLCanvasElement.prototype, 'getContext')!.value as (...a: unknown[]) => RenderingContext | null;
  const g = globalThis as unknown as { __webgl2: number };
  g.__webgl2 = 0;
  HTMLCanvasElement.prototype.getContext = function (this: HTMLCanvasElement, id: string, ...rest: unknown[]) {
    if (id === 'webgl2') g.__webgl2++;
    return getContext.call(this, id, ...rest);
  } as typeof HTMLCanvasElement.prototype.getContext;
};
const contexts = (page: Page) => page.evaluate(() => (globalThis as unknown as { __webgl2: number }).__webgl2);

const frames = (page: Page, probe: '__treeProbe' | '__tree2Probe', from: number) =>
  page.evaluate(
    ([name, start]) => {
      const p = (globalThis as unknown as Record<string, { frames?: Frame[]; state?: { frames: Frame[] } }>)[name];
      return (p.state?.frames ?? p.frames!).slice(start);
    },
    [probe, from] as const
  );

// the time from the tab click to the first frame drawn with every sprite sheet it asked for, and the frames before it
const untilComplete = async (page: Page, probe: '__treeProbe' | '__tree2Probe', click: () => Promise<void>, start?: number) => {
  const from = start ?? (await frames(page, probe, 0)).length;
  const before = await contexts(page);
  let t0 = await page.evaluate(() => performance.now());
  await click();
  if (start !== undefined) t0 = (await frames(page, probe, start))[0]?.at ?? t0;
  let done: Frame | undefined;
  let seen: Frame[] = [];
  await expect
    .poll(
      async () => {
        seen = await frames(page, probe, from);
        done = seen.find((f) => f.complete);
        return !!done;
      },
      { timeout: 60_000, intervals: [20] }
    )
    .toBe(true)
    .catch((e: unknown) => {
      throw new Error(`no frame drew every sheet: ${seen.length} frames, the last ${JSON.stringify(seen.at(-1))}\n${String(e)}`);
    });
  const drawn = seen.slice(0, seen.indexOf(done!) + 1);
  const gap = Math.max(0, ...drawn.slice(1).map((f, i) => f.at - drawn[i].at));
  return {
    t0,
    ms: +(done!.at - t0).toFixed(1),
    frames: drawn.length,
    firstFrameMs: +(drawn[0].at - t0).toFixed(1),
    gapMs: +gap.toFixed(1),
    renderer: done!.renderer ?? '?',
    contexts: (await contexts(page)) - before
  };
};

export const revisitPoE1 = async (page: Page, renderer: 'webgl' | 'canvas'): Promise<Revisit> => {
  const warnings = glWarnings(page);
  await installTreeProbe(page);
  await page.addInitScript(countContexts);
  await page.addInitScript((r) => ((globalThis as unknown as { __treeProbe: { renderer: string } }).__treeProbe.renderer = r), renderer);
  // a new build lands on the tree: its first open is timed from the first frame the tree draws
  const app = await App.open(page, 'new-build');
  await tree.prepareTree();
  const first = await untilComplete(page, '__treeProbe', () => app.openTree(tree.treeVersion()), 0);
  await app.goTo('Skills');
  await expect(page.getByRole('img', { name: 'Passive skill tree' })).toBeHidden();
  const back = await untilComplete(page, '__treeProbe', () => app.goTo('Tree'));
  app.assertNoErrors();
  return {
    game: 'PoE1',
    renderer: back.renderer,
    firstOpenMs: first.ms,
    returnMs: back.ms,
    returnFrames: back.frames,
    contexts: back.contexts,
    glWarnings: warnings
  };
};

const POE2_BUILD = resolve(import.meta.dirname, '../../../wasm2/exposition/testdata/sorceress_spark.xml');

export const revisitPoE2 = async (page: Page, renderer: 'webgl' | 'canvas'): Promise<Revisit> => {
  const warnings = glWarnings(page);
  await page.addInitScript(countContexts);
  await page.addInitScript((kind) => {
    localStorage.setItem('options:poe2', 'true');
    const seen: unknown[] = [];
    const phases: Phase[] = [];
    (globalThis as unknown as { __tree2Probe: unknown }).__tree2Probe = {
      frames: seen,
      phases,
      renderer: kind,
      frame: (f: object) => seen.push({ ...f, at: performance.now() }),
      phase: (name: string, ms?: number) => phases.push({ name, at: performance.now(), ms })
    };
  }, renderer);
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  const code = deflateSync(readFileSync(POE2_BUILD)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
  await page.goto(`/calcs?import=${encodeURIComponent(code)}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  const first = await untilComplete(page, '__tree2Probe', () => poe2Nav(page, 'Tree').click());
  // the mipmaps follow the complete frame: the phases are read once every sheet asked for has them
  await expect
    .poll(() =>
      page.evaluate(() => {
        const names = (globalThis as unknown as { __tree2Probe: { phases: Phase[] } }).__tree2Probe.phases.map((p) => p.name);
        return names.filter((n) => n.endsWith('mipmapped')).length - names.filter((n) => n.endsWith('asked')).length;
      })
    )
    .toBe(0);
  const phases = (await page.evaluate(() => (globalThis as unknown as { __tree2Probe: { phases: Phase[] } }).__tree2Probe.phases.slice())).map((p) => ({
    ...p,
    at: +(p.at - first.t0).toFixed(1),
    ms: p.ms === undefined ? undefined : +p.ms.toFixed(1)
  }));
  phases.push({ name: 'first frame', at: first.firstFrameMs }, { name: 'complete', at: first.ms });
  phases.sort((a, b) => a.at - b.at);
  await poe2Nav(page, 'Skills').click();
  await expect(page.getByRole('img', { name: 'Passive skill tree' })).toBeHidden();
  const back = await untilComplete(page, '__tree2Probe', () => poe2Nav(page, 'Tree').click());
  return {
    game: 'PoE2',
    renderer: back.renderer,
    firstOpenMs: first.ms,
    returnMs: back.ms,
    returnFrames: back.frames,
    contexts: back.contexts,
    glWarnings: warnings,
    phases,
    firstFrames: first.frames,
    longestGapMs: first.gapMs
  };
};

// the return draws its finished art on its first frame, with the WebGL context the tree had before it left
export const expectKept = (r: Revisit) => {
  expect(r.contexts, `${r.game}: WebGL2 contexts made on the return`).toBe(0);
  expect(r.returnFrames, `${r.game}: frames drawn on the return before the art was complete`).toBe(1);
};

export const report = (page: Page, r: Revisit) => {
  const load = readFileSync('/proc/loadavg', 'utf8').split(' ').slice(0, 3).join(' ');
  const browser = page.context().browser()?.browserType().name() ?? '?';
  console.log(
    `tree revisit ${r.game} ${browser} ${r.renderer}: first open ${r.firstOpenMs} ms, Tree -> Skills -> Tree ${r.returnMs} ms ` +
      `(${r.returnFrames} frames, ${r.contexts} new WebGL2 contexts) | loadavg ${load}`
  );
  if (r.phases) {
    console.log(`  first open: ${r.firstFrames} frames, longest gap ${r.longestGapMs} ms`);
    for (const p of r.phases) console.log(`  ${String(p.at).padStart(8)} ms  ${p.name}${p.ms === undefined ? '' : ` (${p.ms} ms of GL calls)`}`);
  }
  const counts = new Map<string, number>();
  for (const w of r.glWarnings) counts.set(w, (counts.get(w) ?? 0) + 1);
  for (const [w, n] of counts) console.log(`  ${n}x ${w}`);
};
