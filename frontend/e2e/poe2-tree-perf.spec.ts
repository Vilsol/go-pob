import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

type Frame = { at: number; ms: number; renderer: string };
const frames = (page: import('@playwright/test').Page) =>
  page.evaluate(() => (globalThis as unknown as { __tree2Probe: { frames: Frame[] } }).__tree2Probe.frames.slice());
const pct = (xs: number[], p: number) => [...xs].sort((a, b) => a - b)[Math.min(xs.length - 1, Math.floor(xs.length * p))];

// reports the PoE2 tree's first paint and its frame rate while panning; it asserts only that frames arrive
for (const renderer of ['webgl', 'canvas']) {
  test(`PoE2 passive tree benchmark: first paint and pan, ${renderer}`, { tag: '@slow' }, async ({ page }) => {
    await page.addInitScript((renderer) => {
      localStorage.setItem('options:poe2', 'true');
      const frames: unknown[] = [];
      (globalThis as unknown as { __tree2Probe: unknown }).__tree2Probe = {
        frames,
        renderer,
        frame: (f: object) => frames.push({ ...f, at: performance.now() })
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
    await page.goto(`/?import=${encodeURIComponent(code())}`);
    const view = page.getByRole('region', { name: 'Path of Exile 2 build' });
    await expect(view).toBeVisible({ timeout: 180_000 });

    const clicked = await page.evaluate(() => performance.now());
    await view.getByRole('tab', { name: 'Tree' }).click();
    await expect.poll(async () => (await frames(page)).length, { timeout: 60_000 }).toBeGreaterThan(0);
    const first = (await frames(page))[0];

    const box = (await page.getByRole('img', { name: 'Passive skill tree' }).boundingBox())!;
    const cx = box.x + box.width / 2;
    const cy = box.y + box.height / 2;
    for (const zoom of [0, 10]) {
      await page.mouse.move(cx, cy);
      for (let i = 0; i < zoom; i++) await page.mouse.wheel(0, -100);
      await page.waitForTimeout(200);
      const before = (await frames(page)).length;
      // pointer events from inside the page, one per animation frame, so the input rate is not the bound
      const [t0, t1] = await page.evaluate(
        ({ cx, cy }) =>
          new Promise<[number, number]>((done) => {
            const canvas = document.querySelector('canvas')!;
            const ev = (type: string, x: number, y: number) => new PointerEvent(type, { clientX: x, clientY: y, button: 0, bubbles: true, pointerId: 1 });
            canvas.dispatchEvent(ev('pointerdown', cx, cy));
            const t0 = performance.now();
            let i = 0;
            const tick = () => {
              if (++i > 180) {
                window.dispatchEvent(ev('pointerup', cx, cy));
                requestAnimationFrame(() => done([t0, performance.now()]));
                return;
              }
              canvas.dispatchEvent(ev('pointermove', cx + 200 * Math.sin(i / 10), cy + 150 * Math.cos(i / 10)));
              requestAnimationFrame(tick);
            };
            requestAnimationFrame(tick);
          }),
        { cx, cy }
      );
      const pan = (await frames(page)).slice(before);
      expect(pan.length).toBeGreaterThan(10);
      const gaps = pan.slice(1).map((f, i) => f.at - pan[i].at);
      console.log(
        `PoE2 tree pan@zoom${zoom} (${first.renderer}): ${pan.length} frames in ${(t1 - t0).toFixed(0)} ms (${((pan.length * 1000) / (t1 - t0)).toFixed(1)} fps), ` +
          `frame gap p50 ${pct(gaps, 0.5).toFixed(1)} ms p95 ${pct(gaps, 0.95).toFixed(1)} ms, draw p50 ${pct(
            pan.map((f) => f.ms),
            0.5
          ).toFixed(2)} ms p95 ${pct(
            pan.map((f) => f.ms),
            0.95
          ).toFixed(2)} ms`
      );
    }
    // the frame rate with nothing to draw, the bound the browser itself sets
    const idle = await page.evaluate(
      () =>
        new Promise<number>((done) => {
          const t0 = performance.now();
          let n = 0;
          const tick = () => (++n < 60 ? requestAnimationFrame(tick) : done((n * 1000) / (performance.now() - t0)));
          requestAnimationFrame(tick);
        })
    );
    console.log(`PoE2 tree idle animation frames: ${idle.toFixed(1)} fps`);
    console.log(`PoE2 tree first paint (${first.renderer}): ${(first.at - clicked).toFixed(0)} ms after the tab click, its draw ${first.ms.toFixed(2)} ms`);
  });
}
