import { test, expect, type Page } from '@playwright/test';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { gunzipSync, gzipSync } from 'node:zlib';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import * as tree from './support/tree_model';
import { compareDrawLists, fromCanvasCalls, type DrawList, type Item } from './support/draw_list';
import { installDrawRecorder, frameCount, lastCalls, lastFrame, waitFrame, waitLastFrame } from './support/tree_probe';
import { isTreeAsset, treeAssetsFailed } from './support/tree_assets';
import { canvasBox, emptySpot, parkPointer, placedNodes, screenOf, touchDistance, wheelAt } from './support/tree_view';

const FIXTURES = resolve(import.meta.dirname, 'fixtures/tree-drawlists');
const RECORD = process.env.TREE_DRAWLIST === 'record';

const BUILDS = [
  { name: 'empty', file: undefined },
  { name: 'mid', file: '0280-elementalist-fireball' },
  { name: 'large', file: '13276-ladder-cuba_5717.cubamrc' }
];

const ZOOMS = [
  { name: 'initial', steps: 0 },
  { name: 'in', steps: -6 },
  { name: 'out', steps: 19 }
];

const fixturePath = (build: string, zoom: string) => resolve(FIXTURES, `${build}-${zoom}.json.gz`);
const readFixture = (p: string) => JSON.parse(gunzipSync(readFileSync(p)).toString('utf8')) as DrawList;

const trackAssetRequests = (page: Page) => {
  const pending = new Set<string>();
  page.on('request', (r) => isTreeAsset(r.url()) && pending.add(r.url()));
  const done = (r: { url(): string }) => pending.delete(r.url());
  page.on('requestfinished', done);
  page.on('requestfailed', done);
  return pending;
};

const settledDrawList = async (page: Page, pending: Set<string>, at: { x: number; y: number }): Promise<DrawList> => {
  const box = await canvasBox(page);
  const dpr = await page.evaluate(() => devicePixelRatio);
  const t = tree.rawTree();
  if ((await lastFrame(page))?.complete !== undefined) {
    const f = await waitLastFrame(page, '(f) => f.complete === true');
    const reported = await page.evaluate(() => (globalThis as unknown as { __treeProbe: { drawList?: () => { items: Item[] } } }).__treeProbe.drawList?.());
    if (reported) return { view: f.view, min: { x: t.min_x, y: t.min_y }, items: reported.items };
    return fromCanvasCalls(await lastCalls(page), f.view, { x: t.min_x, y: t.min_y }, dpr);
  }
  for (let attempt = 0; attempt < 100; attempt++) {
    if (pending.size === 0) {
      const f = (await lastFrame(page))!;
      const reported = await page.evaluate(() => (globalThis as unknown as { __treeProbe: { drawList?: () => unknown } }).__treeProbe.drawList?.());
      if (reported) {
        const r = reported as { complete: boolean; items: Item[] };
        if (r.complete) return { view: f.view, min: { x: t.min_x, y: t.min_y }, items: r.items };
      } else {
        const calls = await lastCalls(page);
        const loaded = await page.evaluate(() => (globalThis as unknown as { __treeImagesLoaded: () => boolean }).__treeImagesLoaded());
        if (loaded && calls.every((c) => c.op !== 'img' || c.ok)) return fromCanvasCalls(calls, f.view, { x: t.min_x, y: t.min_y }, dpr);
      }
    }
    const from = await frameCount(page);
    await page.mouse.move(box.x + at.x + (attempt % 2), box.y + at.y);
    await waitFrame(page, from, '() => true');
  }
  throw new Error(`REFUSING: tree images never finished loading (failed: ${treeAssetsFailed().join(', ') || 'none'})`);
};

for (const b of BUILDS)
  for (const renderer of ['webgl', 'canvas'] as const) {
    test(`tree draw list matches the reference: ${b.name} build, ${renderer}`, async ({ page }) => {
      await installDrawRecorder(page);
      await page.addInitScript((r) => ((globalThis as unknown as { __treeProbe: { renderer: string } }).__treeProbe.renderer = r), renderer);
      const pending = trackAssetRequests(page);
      const app = await App.open(page, b.file ?? 'new-build');
      let active: number[] = [];
      let build: Awaited<ReturnType<typeof go.loadXml>> | undefined;
      if (b.file) {
        // PassiveTreeView.lua:83-111
        const xml = buildXml(b.file).replace(/<TreeView\b[^>]*\/>/, '');
        build = await go.loadXml(xml);
        await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build!)));
        active = go.fresh(build, (x) => [...(x.Build.PassiveNodes ?? [])]);
      }
      await tree.prepareTree();
      await app.openTree(tree.treeVersion());
      const first = await waitFrame(page, 0, `(f) => f.className !== undefined && f.active === ${active.length}`, 60_000);
      expect(first.renderer, 'the tree drew with another renderer than the one asked for').toBe(renderer);
      expect(first.version, 'the tree drew another version than the build is calculated on').toBe(tree.treeVersion());

      const t = tree.rawTree();
      const placed = placedNodes(t);
      const check = async (state: string, at: { x: number; y: number }) => {
        const got = await settledDrawList(page, pending, at);
        expect(got.items.length, 'the frame drew nothing').toBeGreaterThan(0);

        const path = fixturePath(b.name, state);
        if (RECORD) {
          mkdirSync(FIXTURES, { recursive: true });
          writeFileSync(path, gzipSync(JSON.stringify(got), { level: 9 }));
          return;
        }
        expect(existsSync(path), `no reference at ${path}; record one with TREE_DRAWLIST=record`).toBe(true);
        const want = readFixture(path);
        if (renderer === 'webgl') {
          const problems = await page.evaluate(() => (globalThis as unknown as { __treeProbe: { selfCheck: () => string[] } }).__treeProbe.selfCheck());
          expect(problems, `${b.name}/${state}: the GPU was not given what the draw list says`).toEqual([]);
        }
        const c = compareDrawLists(want, got);
        console.log(`draw list ${b.name}/${state}: ${c.compared} items compared, ${c.overlappingPairs} overlapping pairs`);
        expect(c.compared, 'the reference has nothing in view, so a match would be vacuous').toBeGreaterThan(100);
        expect({ missing: c.missing.slice(0, 20), extra: c.extra.slice(0, 20), misordered: c.misordered.slice(0, 20) }, `${b.name}/${state}`).toEqual({
          missing: [],
          extra: [],
          misordered: []
        });
      };

      if (build) {
        const f = (await lastFrame(page))!;
        const state = tree.treeState(build);
        const target = placed.find((p) => {
          const s = screenOf(t, f.view, p.at);
          const len = active.includes(p.node.skill!) || touchDistance(p.node) === 0 ? 0 : (state.path(p.node.skill!)?.length ?? 0);
          return len >= 4 && len <= 12 && s.x > 450 && s.y > 450 && s.x < f.view.width - 50 && s.y < f.view.height - 50;
        });
        expect(target, 'no unallocated node a few hops out is in view').toBeDefined();
        const at = screenOf(t, f.view, target!.at);
        const len = state.path(target!.node.skill!)!.length + 1;
        const box = await canvasBox(page);
        const from = await frameCount(page);
        await page.mouse.move(box.x + at.x, box.y + at.y);
        await waitFrame(page, from, `(f) => f.hovered === ${target!.node.skill} && f.hoverPath === ${len}`);
        await check('hover', at);
      }

      for (const z of ZOOMS) {
        const box = await canvasBox(page);
        let f = (await lastFrame(page))!;
        if (z.steps) f = await wheelAt(page, { x: box.width / 2, y: box.height / 2 }, z.steps);
        const spot = emptySpot(t, placed, f.view);
        await parkPointer(page, spot);
        await check(z.name, spot);
      }
      app.assertNoErrors();
    });
  }

test('the draw-list comparison catches planted defects', () => {
  const path = fixturePath('large', 'initial');
  expect(existsSync(path), `no reference at ${path}`).toBe(true);
  const want = readFixture(path);
  expect(compareDrawLists(want, structuredClone(want))).toMatchObject({ missing: [], extra: [], misordered: [] });

  const firstIdx = (pred: (i: Item) => boolean) => {
    const inView = (x: Item) => compareDrawLists({ ...want, items: [x] }, { ...want, items: [x] }).compared === 1;
    const i = want.items.findIndex((x) => pred(x) && inView(x));
    expect(i, 'the reference lacks the item the defect is planted in').toBeGreaterThanOrEqual(0);
    return i;
  };

  const colour = structuredClone(want);
  const line = firstIdx((i) => i.k === 'line' && i.style === '#524518');
  (colour.items[line] as { style: string }).style = '#c89c01';
  expect(compareDrawLists(want, colour).missing.length, 'a connection in the wrong colour went unnoticed').toBe(1);

  const moved = structuredClone(want);
  const sprite = firstIdx((i) => i.k === 'sprite');
  (moved.items[sprite] as { q: number[] }).q[0] += 1;
  expect(compareDrawLists(want, moved).missing.length, 'a sprite moved by one tree unit went unnoticed').toBe(1);

  const swapped = structuredClone(want);
  const frame = firstIdx((i) => i.k === 'sprite' && i.src.includes('frame'));
  const q = (want.items[frame] as { q: number[] }).q;
  const icon = want.items.findIndex(
    (i, n) =>
      n < frame &&
      i.k === 'sprite' &&
      !i.src.includes('frame') &&
      Math.abs(i.q[0] + i.q[2] / 2 - (q[0] + q[2] / 2)) < 0.05 &&
      Math.abs(i.q[1] + i.q[5] / 2 - (q[1] + q[5] / 2)) < 0.05
  );
  expect(icon, 'no icon under the first frame').toBeGreaterThanOrEqual(0);
  [swapped.items[icon], swapped.items[frame]] = [swapped.items[frame], swapped.items[icon]];
  expect(compareDrawLists(want, swapped).misordered.length, 'a frame drawn under its icon went unnoticed').toBeGreaterThan(0);
});
