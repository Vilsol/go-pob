import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page } from './support/poe2';

test.use({ serviceWorkers: 'block', viewport: { width: 1280, height: 900 } });

// radius_jewel.xml: a Warrior whose socket 26725 holds a Time-Lost Ruby, with allocated normal, notable and attribute
// nodes
const FIXTURE = resolve(import.meta.dirname, '../../calculator2/testdata/radius_jewel.xml');
const code = () => deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

type Sprite = { url: string; x: number; y: number; w: number; h: number };
type Frame = { renderer: string; sprites?: number; lines?: number; view: { zoomLevel: number }; compared?: number; alloc: number };
type Probe = {
  frames: Frame[];
  focus: (id: number) => void;
  screenOf: (id: number) => { x: number; y: number } | undefined;
  sprite: (name: string) => Sprite | undefined;
  icon: (id: number) => string | undefined;
  scale: () => number;
  sheetCounts: () => Record<string, number>;
};
type Win = { __tree2Probe: Probe };

const lastFrame = (page: Page) => page.evaluate(() => (globalThis as unknown as Win).__tree2Probe.frames.at(-1));

/* the mean absolute difference, per channel in 0-255, between the canvas around (cx, cy) and the sprite drawn there
   at half size half (screen pixels) and multiplied by tint, over the inner box a node's frame leaves uncovered; and
   the sprite's own mean deviation, which a flat vector disc in its place would differ from it by */
const compare = async (page: Page, at: { sprite: Sprite; cx: number; cy: number; half: number; tint: number }) => {
  const shot = (await page.getByRole('img', { name: 'Passive skill tree' }).screenshot()).toString('base64');
  return page.evaluate(
    async ({ b64, sprite, cx, cy, half, tint }) => {
      const load = async (src: string) => {
        const img = new Image();
        img.crossOrigin = 'anonymous';
        img.src = src;
        await img.decode();
        return img;
      };
      const [screen, sheet] = await Promise.all([load(`data:image/png;base64,${b64}`), load(sprite.url)]);
      const c = document.createElement('canvas');
      [c.width, c.height] = [screen.naturalWidth, screen.naturalHeight];
      const ctx = c.getContext('2d')!;
      ctx.drawImage(screen, 0, 0);
      const box = Math.max(4, Math.round(half * 0.5));
      const got = ctx.getImageData(Math.round(cx - box / 2), Math.round(cy - box / 2), box, box).data;
      const e = document.createElement('canvas');
      [e.width, e.height] = [Math.round(half * 2), Math.round(half * 2)];
      const ex = e.getContext('2d')!;
      ex.drawImage(sheet, sprite.x, sprite.y, sprite.w, sprite.h, 0, 0, e.width, e.height);
      const want = ex.getImageData(Math.round(half - box / 2), Math.round(half - box / 2), box, box).data;
      const n = want.length / 4;
      const mean = [0, 0, 0];
      for (let i = 0; i < want.length; i += 4) for (let k = 0; k < 3; k++) mean[k] += (want[i + k] * tint) / n;
      let diff = 0;
      let dev = 0;
      for (let i = 0; i < want.length; i += 4) {
        for (let k = 0; k < 3; k++) {
          diff += Math.abs(got[i + k] - want[i + k] * tint);
          dev += Math.abs(want[i + k] * tint - mean[k]);
        }
      }
      return { diff: diff / (n * 3), dev: dev / (n * 3) };
    },
    { b64: shot, ...at }
  );
};

const open = async (page: Page, renderer: string, treeFetches: string[]) => {
  await page.addInitScript((kind) => {
    localStorage.setItem('options:poe2', 'true');
    const seen: unknown[] = [];
    (globalThis as unknown as { __tree2Probe: unknown }).__tree2Probe = { frames: seen, renderer: kind, frame: (f: unknown) => seen.push(f) };
  }, renderer);
  page.context().on('request', (r) => {
    if (/\/tree\/data\.json/.test(r.url())) treeFetches.push(r.url());
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
  await page.goto(`/calcs?import=${encodeURIComponent(code())}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  await poe2Nav(page, 'Tree').click();
  // every sheet has loaded once the sprite and line counts stop growing
  let last = -1;
  await expect
    .poll(
      async () => {
        const f = await lastFrame(page);
        const n = (f?.sprites ?? 0) + (f?.lines ?? 0);
        const settled = n > 1000 && n === last;
        last = n;
        return settled;
      },
      { timeout: 90_000, intervals: [1000] }
    )
    .toBe(true);
};

const HALF_GRAY = 128 / 255;

for (const renderer of ['webgl', 'canvas']) {
  test(`the PoE2 tree draws its node icons, jewels, connectors and backgrounds from the published sheets, ${renderer}`, async ({ page }) => {
    test.setTimeout(240_000);
    const errors: string[] = [];
    page.on('pageerror', (e) => errors.push(e.message));
    const treeFetches: string[] = [];
    await open(page, renderer, treeFetches);
    expect((await lastFrame(page))!.renderer).toBe(renderer);

    // the tree's art comes with the tree the worker already holds: the PoE2 tree data downloads once
    expect(treeFetches.filter((u) => u.includes('/data/0.5/'))).toEqual(['https://data.pob.gg/data/0.5/tree/data.json.br']);

    // every sheet a Warrior with a socketed jewel shows draws from it
    const counts = await page.evaluate(() => (globalThis as unknown as Win).__tree2Probe.sheetCounts());
    for (const sheet of ['skills-3.webp', 'frame-3.png', 'mastery-active-effect-3.webp', 'ascendancy-background-3.webp', 'jewel-3.png', 'line-3.png']) {
      expect(counts[`tree/assets/${sheet}`] ?? 0, sheet).toBeGreaterThan(0);
    }
    // the connectors are line art, not vector strokes
    expect((await lastFrame(page))!.lines).toBeGreaterThan(3000);

    const box = (await page.getByRole('img', { name: 'Passive skill tree' }).boundingBox())!;
    const at = async (id: number, base: number, name: string, tint: number) => {
      await page.evaluate((n) => (globalThis as unknown as Win).__tree2Probe.focus(n), id);
      await page.mouse.move(box.x + box.width - 2, box.y + box.height - 2);
      await page.waitForTimeout(500);
      const pos = (await page.evaluate((n) => (globalThis as unknown as Win).__tree2Probe.screenOf(n), id))!;
      const scale = await page.evaluate(() => (globalThis as unknown as Win).__tree2Probe.scale());
      const sprite = await page.evaluate((n) => (globalThis as unknown as Win).__tree2Probe.sprite(n), name);
      expect(sprite, name).toBeTruthy();
      return compare(page, { sprite: sprite!, cx: pos.x, cy: pos.y, half: base * scale, tint });
    };
    const icon = (id: number) => page.evaluate((n) => (globalThis as unknown as Win).__tree2Probe.icon(n)!, id);
    // PassiveTree.lua:821-879 sizes; PassiveTreeView.lua:1108-1110 draws an unallocated icon at half gray, 961-969
    // a socket's jewel over its frame
    const probes: [number, number, string, number][] = [
      [3936, 37, await icon(3936), 1],
      [5710, 54, await icon(5710), 1],
      [11230, 82, await icon(11230), HALF_GRAY],
      [26725, 76, 'Time-Lost Ruby', 1]
    ];
    for (const [k, [id, base, name, tint]] of probes.entries()) {
      const r = await at(id, base, name, tint);
      // the canvas reproduces the sprite: a flat disc in its place would miss it by about the sprite's own deviation,
      // and another node's sprite by more than this one
      const other = await at(id, base, probes[(k + 1) % probes.length][2], tint);
      const msg = `${id} ${name}: ${JSON.stringify({ r, other })}`;
      expect(r.diff, msg).toBeLessThan(Math.min(30, r.dev * 0.75));
      expect(r.diff, msg).toBeLessThan(other.diff * 0.75);
    }

    // an allocated attribute node draws its chosen attribute's icon (PassiveTree.lua:592-613), not the tree's
    const attr = [];
    for (const n of ['plusattribute', 'plusstrength', 'plusdexterity', 'plusintelligence'])
      attr.push(await at(13397, 37, `Art/2DArt/SkillIcons/passives/${n}.dds`, 1));
    const best = attr.indexOf(attr.reduce((a, b) => (b.diff < a.diff ? b : a)));
    expect(best, JSON.stringify(attr)).toBeGreaterThan(0);
    expect(errors).toEqual([]);
  });
}

// PassiveTreeView.lua:206-209 and TreeTab.lua:121-140, 444 (PoB-PoE2): PageUp and PageDown zoom the tree a step, three
// with Shift, and the Compare box hands the compared tree to the view
test('the PoE2 tree zooms with PageUp and PageDown and draws a compared tree', async ({ page }) => {
  test.setTimeout(240_000);
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await open(page, 'canvas', []);
  const zoom = (await lastFrame(page))!.view.zoomLevel;
  await page.mouse.move(2, 2);
  await page.keyboard.press('PageUp');
  await expect.poll(async () => (await lastFrame(page))!.view.zoomLevel).toBe(zoom + 1);
  await page.keyboard.press('Shift+PageDown');
  await expect.poll(async () => (await lastFrame(page))!.view.zoomLevel).toBe(zoom - 2);

  expect((await lastFrame(page))!.compared).toBeUndefined();
  await poe2Page(page).getByRole('checkbox', { name: 'Compare:' }).check();
  // the one tree compared with itself: its own allocated nodes
  await expect.poll(async () => (await lastFrame(page))!.compared).toBeGreaterThan(0);
  expect((await lastFrame(page))!.compared).toBe((await lastFrame(page))!.alloc);
  expect(errors).toEqual([]);
});

// the Canvas 2D renderer draws a tree that fits once into a layer and pans by moving it; the layer matches the tree drawn
// straight to the canvas
test('the PoE2 Canvas 2D tree draws the same through its layer as straight to the canvas', async ({ page }) => {
  await open(page, 'canvas', []);
  const shot = async () => (await page.getByRole('img', { name: 'Passive skill tree' }).screenshot()).toString('base64');
  const redraw = async () => {
    for (const key of ['PageUp', 'PageDown']) {
      const n = await page.evaluate(() => (globalThis as unknown as Win).__tree2Probe.frames.length);
      await page.keyboard.press(key);
      await expect.poll(async () => page.evaluate(() => (globalThis as unknown as Win).__tree2Probe.frames.length)).toBeGreaterThan(n);
    }
    await page.waitForTimeout(300);
  };
  await page.mouse.move(5, 5);
  await redraw();
  const layered = await shot();
  await page.evaluate(() => ((globalThis as unknown as { __tree2Probe: { layer?: boolean } }).__tree2Probe.layer = false));
  await redraw();
  const direct = await shot();
  const diff = await page.evaluate(
    async ({ a, b }) => {
      const pixels = async (b64: string) => {
        const img = new Image();
        img.src = `data:image/png;base64,${b64}`;
        await img.decode();
        const c = document.createElement('canvas');
        [c.width, c.height] = [img.naturalWidth, img.naturalHeight];
        const ctx = c.getContext('2d')!;
        ctx.drawImage(img, 0, 0);
        return ctx.getImageData(0, 0, c.width, c.height).data;
      };
      const [x, y] = await Promise.all([pixels(a), pixels(b)]);
      let sum = 0;
      let lit = 0;
      for (let i = 0; i < x.length; i += 4) {
        for (let k = 0; k < 3; k++) sum += Math.abs(x[i + k] - y[i + k]);
        if (y[i] + y[i + 1] + y[i + 2] > 60) lit++;
      }
      return { mean: sum / ((x.length / 4) * 3), lit: lit / (x.length / 4) };
    },
    { a: layered, b: direct }
  );
  // the tree covers the canvas, and the two differ by little more than the layer's rounding to whole pixels
  expect(diff.lit).toBeGreaterThan(0.05);
  expect(diff.mean).toBeLessThan(3);
});
