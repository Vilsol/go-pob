import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { pctl, record } from './support/perf';
import { expectedBlock, type DiffBlock, type WhatIfOutputs } from './support/stat_diff';
import { heatChannel, heatValues } from '../src/lib/skill_tree/heat';
import { ascendancyOffsets, nodeCentre } from '../src/lib/skill_tree/geometry';
import { expectedLines } from './support/stats';
import * as tree from './support/tree_model';
import { TREE_ASSET_PATTERN } from './support/tree_assets';
import { frameCount, framesSince, installTreeProbe, lastFrame, pointerAt, waitFrame } from './support/tree_probe';
import { canvasBox, clusterPlaced, emptySpot, hitTest, parkPointer, placedNodes, screenOf, touchDistance, treeSelect, wheelAt } from './support/tree_view';
import { showPowerRows, type PowerRow } from '../src/lib/design/tree';
import { displayStats } from '../src/lib/display/stats';
import { formatNumSep, luaFormat } from '../src/lib/display/format_stat';

const FRAME_MS = 1000 / 60;

const gpuName = (page: Page) =>
  page.evaluate(() => {
    const gl = document.createElement('canvas').getContext('webgl2');
    const ext = gl?.getExtension('WEBGL_debug_renderer_info');
    return gl ? String(gl.getParameter(ext ? ext.UNMASKED_RENDERER_WEBGL : gl.RENDERER)) : 'no webgl2';
  });

const setup = async (page: Page, file: string) => {
  await installTreeProbe(page);
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await tree.prepareTree();
  const active = go.fresh(build, (b) => [...(b.Build.PassiveNodes ?? [])]);
  return { app, build, active, gpu: await gpuName(page) };
};

const firstPaint = async (page: Page, app: App, file: string, active: number[], webgl: string) => {
  const t0 = await page.evaluate(() => performance.now());
  await app.goTo('Tree');
  const first = await waitFrame(page, 0, `(f) => f.className !== undefined && f.active === ${active.length}`, 60_000);
  expect(first.version, 'the tree drew another version than the build is calculated on').toBe(tree.treeVersion());
  const gpu = `${first.renderer ?? 'canvas-2d (before WebGL)'} | webgl: ${webgl}`;
  record({ kind: 'tree-latency', build: file, what: 'first-paint', ms: +(first.at - t0).toFixed(1), renderMs: +first.ms.toFixed(2), gpu });

  await page.waitForFunction(
    (pattern) => {
      const rs = performance.getEntriesByType('resource').filter((r) => new RegExp(pattern).test(r.name));
      return rs.length > 0 && rs.every((r) => (r as PerformanceResourceTiming).responseEnd > 0);
    },
    TREE_ASSET_PATTERN,
    { timeout: 60_000 }
  );
  const loaded = await page.evaluate(
    (pattern) =>
      Math.max(
        ...performance
          .getEntriesByType('resource')
          .filter((r) => new RegExp(pattern).test(r.name))
          .map((r) => (r as PerformanceResourceTiming).responseEnd)
      ),
    TREE_ASSET_PATTERN
  );
  const complete = await waitFrame(page, 0, `(f) => f.at > ${loaded}`, 5_000).catch(() => undefined);
  record({ kind: 'tree-latency', build: file, what: 'complete-paint', ms: complete ? +(complete.at - t0).toFixed(1) : null, gpu });
  return gpu;
};

const measure = async (page: Page, app: App, file: string, gesture: string, gpu: string, drive: () => Promise<void>) => {
  await app.flushPerf(`tree-${gesture}-before`);
  const from = await frameCount(page);
  await page.evaluate(() => {
    const f = { gaps: [] as number[], last: performance.now(), on: true };
    (globalThis as unknown as { __frames: typeof f }).__frames = f;
    const loop = (t: number) => {
      f.gaps.push(t - f.last);
      f.last = t;
      if (f.on) requestAnimationFrame(loop);
    };
    requestAnimationFrame(loop);
  });
  const t0 = Date.now();
  await drive();
  const durationMs = Date.now() - t0;
  const gaps = await page.evaluate(() => {
    const f = (globalThis as unknown as { __frames: { gaps: number[]; on: boolean } }).__frames;
    f.on = false;
    return f.gaps.slice(1);
  });
  const lt = await page.evaluate(() => {
    const perf = (globalThis as unknown as { __pobPerf: { longtasks: { dur: number }[] } }).__pobPerf;
    const d = perf.longtasks.map((l) => l.dur);
    perf.longtasks.length = 0;
    return d;
  });
  const draws = await framesSince(page, from);
  const dropped = gaps.reduce((n, g) => n + Math.max(0, Math.round(g / FRAME_MS) - 1), 0);
  record({
    kind: 'frames',
    build: file,
    gesture,
    frames: gaps.length,
    dropped,
    maxGapMs: +Math.max(0, ...gaps).toFixed(1),
    durationMs,
    gapMs: pctl(gaps),
    draws: draws.length,
    renderMs: pctl(draws.map((d) => d.ms)),
    longTasks: lt.length,
    longTaskMs: +lt.reduce((a, b) => a + b, 0).toFixed(1),
    gpu
  });
};

const panSweep = async (page: Page) => {
  const box = await canvasBox(page);
  const t = tree.rawTree();
  const spot = emptySpot(t, placedNodes(t), (await lastFrame(page))!.view);
  const cx = box.x + spot.x;
  const cy = box.y + spot.y;
  await page.mouse.move(cx, cy);
  await page.mouse.down();
  const dir = cx > box.x + box.width / 2 ? -1 : 1;
  for (let i = 1; i <= 60; i++) await page.mouse.move(cx + dir * i * 8, cy - i * 4);
  for (let i = 59; i >= 0; i--) await page.mouse.move(cx + dir * i * 8, cy - i * 4);
  await page.mouse.up();
};

const zoomTo = async (page: Page, scaling: number) => {
  const box = await canvasBox(page);
  const now = (await lastFrame(page))!.view.scaling;
  const steps = Math.round((scaling - now) / 1.2);
  if (steps) await wheelAt(page, { x: box.width / 2, y: box.height / 2 }, steps);
};

test('passive tree benchmark: pan, zoom, hover, allocate', async ({ page }) => {
  const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
  const opened = await setup(page, file);
  const { app, build, active } = opened;
  const gpu = await firstPaint(page, app, file, active, opened.gpu);

  for (const z of [3, 10, 25]) {
    await zoomTo(page, z);
    await measure(page, app, file, `pan@${z}`, gpu, () => panSweep(page));
  }

  await measure(page, app, file, 'zoom', gpu, async () => {
    const box = await canvasBox(page);
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    for (let i = 0; i < 40; i++) await page.mouse.wheel(0, i < 20 ? -120 : 120);
  });

  await zoomTo(page, 10);
  const t = tree.rawTree();
  const state = tree.treeState(build);
  const view = (await lastFrame(page))!.view;
  const placed = placedNodes(t);
  const targets = placed
    .map((p) => ({ p, s: screenOf(t, view, p.at), len: state.path(p.node.skill!)?.length ?? 0 }))
    .filter(({ p, s, len }) => !active.includes(p.node.skill!) && len > 0 && s.x > 450 && s.y > 50 && s.x < view.width - 50 && s.y < view.height - 50)
    .filter(({ p, s }) => hitTest(t, placed, view, s) === p)
    .sort((a, b) => a.s.x - b.s.x)
    .filter((_, i, all) => i % Math.max(1, Math.floor(all.length / 40)) === 0)
    .slice(0, 40);
  expect(targets.length, 'too few unallocated nodes in view to sweep').toBeGreaterThan(20);
  const box = await canvasBox(page);
  const rest = emptySpot(t, placed, view);
  await measure(page, app, file, 'hover', gpu, async () => {
    for (const { p, s } of targets) {
      await parkPointer(page, rest);
      const from = await frameCount(page);
      await page.mouse.move(box.x + s.x, box.y + s.y);
      const f = await waitFrame(page, from, `(f) => f.hovered === ${p.node.skill} && f.hoverPath > 0`);
      record({ kind: 'tree-latency', build: file, what: 'hover', ms: +(f.at - (await pointerAt(page))).toFixed(1), renderMs: +f.ms.toFixed(2), gpu });
    }
  });

  const far = placed
    .map((p) => ({ node: p.node, len: state.path(p.node.skill!)?.length ?? 0 }))
    .filter((c) => c.len > 0 && !!c.node.name && touchDistance(c.node) > 0 && !c.node.isMastery)
    .sort((a, b) => Math.abs(a.len - 40) - Math.abs(b.len - 40))[0];
  const button = await app.treeNode(tree.resultName({ id: far.node.skill!, name: far.node.name! }));
  const from = await frameCount(page);
  const t0 = await page.evaluate(() => performance.now());
  await app.act(
    'allocate-path',
    () => button.click(),
    async () => {
      state.allocate(far.node.skill!);
      return expectedLines(await go.outputs(build));
    }
  );
  const shown = await waitFrame(page, from, `(f) => f.active === ${active.length + far.len}`);
  record({ kind: 'tree-latency', build: file, what: `allocate-${far.len}`, ms: +(shown.at - t0).toFixed(1), renderMs: +shown.ms.toFixed(2), gpu });

  const cdp = await page.context().newCDPSession(page);
  await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 });
  await measure(page, app, file, 'pan@10-cpu4x', gpu, () => panSweep(page));
  await cdp.send('Emulation.setCPUThrottlingRate', { rate: 1 });
  await cdp.detach();

  await app.memory('tree');
  await app.flushPerf('tree');
  app.assertNoErrors();
});

test('passive tree benchmark: large allocation with cluster jewels', async ({ page }) => {
  const file = '13276-ladder-cuba_5717.cubamrc';
  const opened = await setup(page, file);
  const gpu = await firstPaint(page, opened.app, file, opened.active, opened.gpu);
  const { app } = opened;
  await measure(page, app, file, 'pan@10', gpu, () => panSweep(page));
  await zoomTo(page, 25);
  await measure(page, app, file, 'pan@25', gpu, () => panSweep(page));
  await app.flushPerf('tree');
  app.assertNoErrors();
});

const openOnTree = async (page: Page, file: string) => {
  const opened = await setup(page, file);
  await opened.app.goTo('Tree');
  await waitFrame(page, 0, `(f) => f.className !== undefined && f.active === ${opened.active.length}`, 60_000);
  return opened;
};

const nodeOnScreen = async (page: Page, id: number, build?: Awaited<ReturnType<typeof go.loadXml>>) => {
  const t = tree.rawTree();
  const cluster = build ? ((await (await go.model()).exposition.GetTreeClusterNodes(build)) ?? []) : [];
  const placed = [...placedNodes(t), ...clusterPlaced(cluster)];
  const target = placed.find((p) => p.node.skill === id);
  if (!target) throw new Error(`node ${id} is not drawn`);
  for (let i = 0; i < 12; i++) {
    const view = (await lastFrame(page))!.view;
    const s = screenOf(t, view, target.at);
    const inside = s.x > 80 && s.y > 80 && s.x < view.width - 80 && s.y < view.height - 80;
    if (inside && hitTest(t, placed, view, s) === target) return s;
    if (!inside) {
      const from = emptySpot(t, placed, view);
      const box = await canvasBox(page);
      const n = await frameCount(page);
      await page.mouse.move(box.x + from.x, box.y + from.y);
      await page.mouse.down();
      const dx = view.width / 2 - s.x;
      const dy = view.height / 2 - s.y;
      for (let k = 1; k <= 10; k++) await page.mouse.move(box.x + from.x + (dx * k) / 10, box.y + from.y + (dy * k) / 10);
      await page.mouse.up();
      await waitFrame(page, n, '(f) => true');
    } else {
      await wheelAt(page, s, -2);
    }
  }
  throw new Error(`node ${id} could not be brought under the pointer`);
};

test('tree sets: the select and the manage list switch passive trees as Go does', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const trees = treeSelect(page);
  const select = trees.box;
  expect(await trees.entries()).toEqual(['Until Lab', 'Lab 2', 'White maps', 'Manage trees…']);
  await expect(select).toHaveText('Lab 2');

  await app.act(
    'switch-tree',
    () => trees.choose('White maps'),
    async () => {
      await m.exposition.SetActiveTreeSpec(build, 3);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(page.getByRole('combobox', { name: 'Class' })).toHaveValue('Witch');

  await trees.choose('Manage trees…');
  const dialog = page.getByRole('dialog', { name: 'Manage passive trees' });
  await dialog.getByRole('button', { name: 'New' }).click();
  await dialog.getByRole('textbox', { name: 'Name for this passive tree' }).fill('Fresh start');
  await dialog.getByRole('button', { name: 'Save' }).click();
  const fresh = dialog.getByRole('option', { name: /^Fresh start/ });
  await expect(fresh).toContainText('Elementalist, 0 points');
  await app.act(
    'switch-to-new-tree',
    () => fresh.dblclick(),
    async () => {
      await m.exposition.NewTreeSpec(build, 'Fresh start');
      await m.exposition.SetActiveTreeSpec(build, 4);
      return expectedLines(await go.outputs(build));
    }
  );
  await dialog.getByRole('button', { name: 'Done' }).click();
  await expect(select).toHaveText('Fresh start');
  app.assertNoErrors();
});

// PassiveSpec.lua:1614-1635
test('tattoos: a tattooed node’s tooltip is the tattoo’s name and stats, not the tree’s node', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const { app, build } = await openOnTree(page, '0020-assassin-storm-burst');
  const m = await go.model();
  const tips = m.exposition.TreeTattooNodes(build) ?? [];
  const tip = tips.find((t) => t.NodeID === 15549)!;
  expect(tip, 'the build tattoos node 15549').toBeDefined();
  const own = tree.rawTree().nodes[15549];
  expect(own.name, 'control: the tattoo renames the node').not.toBe(tip.Name);
  const at = await nodeOnScreen(page, 15549);
  const box = await canvasBox(page);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  const nodeTip = page.getByRole('tooltip').first();
  await expect(nodeTip.locator('span.text-15').first()).toHaveText(tip.Name);
  expect(await nodeTip.locator('.text-magic > span').allTextContents()).toEqual(tip.Stats);
  await page.mouse.move(0, 0);
  app.assertNoErrors();
});

// PassiveSpec.lua:1614-1635
test('timeless jewels: a conquered node’s tooltip is the conquest’s name and lines, not the tree’s node', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const { app, build } = await openOnTree(page, '0006-elementalist-storm-burst');
  const m = await go.model();
  const conquered = m.exposition.TreeConqueredNodes(await m.exposition.GetMiscCalculator(build)) ?? [];
  const t = tree.rawTree();
  const renamed = conquered.find((c) => t.nodes[c.NodeID]?.isNotable && c.Name !== t.nodes[c.NodeID].name)!;
  expect(renamed, 'control: Glorious Vanity renames a notable').toBeDefined();
  const added = conquered.find((c) => !c.Replaced && (c.Stats?.length ?? 0) > (t.nodes[c.NodeID]?.stats?.length ?? 0))!;
  for (const c of [renamed, ...(added ? [added] : [])]) {
    const at = await nodeOnScreen(page, c.NodeID);
    const box = await canvasBox(page);
    await page.mouse.move(box.x + at.x, box.y + at.y);
    const nodeTip = page.getByRole('tooltip').first();
    await expect(nodeTip.locator('span.text-15').first()).toHaveText(c.Name);
    expect(await nodeTip.locator('.text-magic > span').allTextContents()).toEqual(c.Stats);
    await page.mouse.move(0, 0);
  }
  app.assertNoErrors();
});

test('radius jewels: a node in a jewel’s radius names the jewel and its radius lines', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const file = '0006-elementalist-storm-burst';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const rings = (await m.exposition.GetTreeJewelRings(build, [])) ?? [];
  expect(rings.map((r) => r.Art)).toEqual(['Glorious Vanity']);
  const radius = ((await m.exposition.GetTreeSocketRadii(build, rings[0].NodeID)) ?? []).find((x) => x.Outer === rings[0].Outer)!.Nodes ?? [];
  const t = tree.rawTree();
  const inside = radius.find((n) => t.nodes[n]?.isNotable)!;
  const outside = placedNodes(t).find((p) => p.node.isNotable && !p.node.ascendancyName && !radius.includes(p.node.skill!))!.node.skill!;
  expect(buildXml(file)).toContain('Passives in radius are Conquered by the Vaal');

  const conquered = m.exposition.TreeConqueredNodes(await m.exposition.GetMiscCalculator(build)) ?? [];
  const tipOf = async (id: number) => {
    const at = await nodeOnScreen(page, id);
    const box = await canvasBox(page);
    await page.mouse.move(box.x + at.x, box.y + at.y);
    const nodeTip = page.getByRole('tooltip').first();
    await expect(nodeTip.locator('span.text-15').first()).toHaveText(conquered.find((c) => c.NodeID === id)?.Name ?? t.nodes[id].name!);
    return nodeTip;
  };
  const tip = (await tipOf(inside)).getByLabel('Jewels affecting this node');
  await expect(tip).toContainText('In radius of Glorious Vanity');
  const lines = await tip.locator('[data-line]').allTextContents();
  expect(lines).toContain('Passives in radius are Conquered by the Vaal');
  for (const l of lines) expect(l).toMatch(/radius|ring/i);
  await page.mouse.move(0, 0);
  await expect((await tipOf(outside)).getByLabel('Jewels affecting this node')).toHaveCount(0);
  await page.mouse.move(0, 0);
  app.assertNoErrors();
});

test('mastery effects: choosing an effect, and allocating a mastery by its choice, as Go does', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const t = tree.rawTree();
  const panel = page.getByRole('dialog');

  const held = 63268;
  await (await app.treeNode(tree.resultName({ id: held, name: t.nodes[held].name! }))).click();
  await expect(panel.getByRole('heading', { name: t.nodes[held].name! })).toBeVisible();
  const options = (await m.exposition.GetMasteryOptions(build, held)) ?? [];
  const next = options.find((o) => !o.Selected)!;
  await expect(panel.getByRole('button', { pressed: true })).toContainText(options.find((o) => o.Selected)!.Label);
  await app.act(
    'mastery-effect',
    () => panel.getByRole('group', { name: 'Mastery effects' }).getByRole('button').filter({ hasText: next.Label }).first().click(),
    async () => {
      (await m.exposition.SelectMasteryEffect(build, held, next.Effect, [])).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );

  const state = tree.treeState(build);
  const free = Object.values(t.nodes).find((n) => n.isMastery && n.masteryEffects?.length && !state.active.includes(n.skill!) && state.path(n.skill!));
  expect(free, 'no reachable unallocated mastery').toBeTruthy();
  const id = free!.skill!;
  await page.getByRole('searchbox', { name: 'Search the tree' }).fill(`#${id}`);
  const result = page.getByRole('region', { name: 'Search results' }).getByRole('button', { name: tree.resultName({ id, name: free!.name! }) });
  const before = await app.seq();
  await result.click();
  await expect(panel.getByText('Choose one effect to allocate it')).toBeVisible();
  await expect(result).toHaveAttribute('aria-pressed', 'false');
  expect(await app.seq(), 'opening a mastery recalculated').toBe(before);
  const first = ((await m.exposition.GetMasteryOptions(build, id)) ?? [])[0];
  const path = state.path(id)!;
  await app.act(
    'mastery-allocate',
    () => panel.getByRole('group', { name: 'Mastery effects' }).getByRole('button').first().click(),
    async () => {
      (await m.exposition.SelectMasteryEffect(build, id, first.Effect, path)).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(result).toHaveAttribute('aria-pressed', 'true');
  app.assertNoErrors();
});

test('power report: the panel lists Go’s report for the chosen stat, built off the interaction path', async ({ page }) => {
  test.setTimeout(20 * 60_000);
  const file = '0280-elementalist-fireball';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();

  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const builder = m.exposition.NewPowerBuilder(misc, 'Life', 0)!;
  while (builder.Step(1000) < 1);
  const rows = (builder.Report() ?? []) as PowerRow[];
  builder.release();
  misc.release();
  const want = showPowerRows(
    rows,
    displayStats.find((d) => d.stat === 'Life'),
    'unallocated',
    true,
    (f, v) => formatNumSep(luaFormat(f, v))
  );
  expect(want.length, 'the model reports no node with power').toBeGreaterThan(50);

  await page.getByRole('button', { name: /^Power/ }).click();
  const panel = page.getByRole('region', { name: 'Power report' });
  await expect(panel.getByRole('combobox', { name: 'Power stat' })).toHaveValue('Life');
  await expect(panel.getByRole('progressbar')).toHaveCount(0);
  await panel.getByRole('button', { name: 'Start' }).click();
  const table = panel.getByRole('table', { name: 'Node power' });
  await expect(table).toBeVisible({ timeout: 10 * 60_000 });
  const shown = await table
    .locator('[data-power-node]')
    .evaluateAll((els) => els.map((e) => [...e.querySelectorAll('[role="cell"]')].map((c) => c.textContent?.trim() ?? '')));
  const key = (r: string[]) => r.join(' | ');
  expect(shown.map(key).sort()).toEqual(want.map((r) => key([r.Type, r.Name, r.power, r.points, r.perPoint])).sort());
  const ms = Number(await panel.locator('[data-power-ms]').getAttribute('data-power-ms'));
  record({ kind: 'tree-latency', build: file, what: 'power-report-Life', ms });

  // PassiveTreeView.lua:1312-1321
  const first = want[0];
  const n = await frameCount(page);
  await table.locator(`[data-power-node="${first.ID}"]`).first().click();
  await waitFrame(page, n, '(f) => true');
  const t = tree.rawTree();
  const view = (await lastFrame(page))!.view;
  const at = screenOf(t, view, placedNodes(t).find((p) => p.node.skill === first.ID)!.at);
  expect(view.scaling).toBe(3);
  expect(Math.abs(at.x - view.width / 2)).toBeLessThan(2);
  expect(Math.abs(at.y - view.height / 2)).toBeLessThan(2);

  await table.locator(`[data-power-node="${first.ID}"]`).first().hover();
  const ring = page.locator(`[data-power-hover="${first.ID}"]`);
  const box = await canvasBox(page);
  const ringBox = (await ring.boundingBox())!;
  expect(Math.abs(ringBox.x + ringBox.width / 2 - (box.x + at.x))).toBeLessThan(2);
  expect(Math.abs(ringBox.y + ringBox.height / 2 - (box.y + at.y))).toBeLessThan(2);
  expect(await ring.evaluate((el) => parseFloat(el.style.width))).toBe((0.75 * 450) / 3);
  await page.mouse.move(box.x + 5, box.y + box.height - 5);
  await expect(page.locator('[data-power-hover]')).toHaveCount(0);

  const cols = await table
    .locator('[role="row"]')
    .evaluateAll((els) => els.slice(0, 40).map((r) => [...r.querySelectorAll('[role="cell"], [role="columnheader"]')].map((c) => c.getBoundingClientRect())));
  for (const row of cols) {
    expect(row.map((c) => Math.round(c.left))).toEqual(cols[0].map((c) => Math.round(c.left)));
    for (let k = 1; k < row.length; k++) expect(row[k].left - row[k - 1].right, `the gap before column ${k}`).toBeGreaterThanOrEqual(8);
  }

  // PowerReportListControl.lua:58-96
  await table.getByRole('button', { name: 'Points' }).click();
  const points = await table.locator('[data-power-node] [role="cell"]:nth-child(4)').allTextContents();
  const dists = points.map((p) => (p === 'Anoint' ? 1000 : Number(p)));
  expect(dists, 'sorted by points').toEqual([...dists].sort((a, b) => a - b));
  await panel.getByRole('combobox', { name: 'Power report filter' }).selectOption('clusters');
  await expect(table.getByRole('button', { name: 'Life' })).toHaveAttribute('aria-pressed', 'true');
  await expect(table.locator('[data-power-node] [role="cell"]:nth-child(4)', { hasText: 'Cluster' }).first()).toBeVisible();
  app.assertNoErrors();
});

test('power report: its time on the largest build (0001), Life', async ({ page }) => {
  test.setTimeout(8 * 60_000);
  const file = '0001-necromancer-unknown';
  const { app } = await openOnTree(page, file);
  await page.getByRole('button', { name: /^Power/ }).click();
  const panel = page.getByRole('region', { name: 'Power report' });
  await expect(panel.getByRole('combobox', { name: 'Power stat' })).toHaveValue('Life');
  await panel.getByRole('button', { name: 'Start' }).click();
  await expect(panel.getByRole('progressbar')).toBeVisible();
  await expect.poll(() => page.locator('[data-power-flash]').count(), { timeout: 60_000 }).toBeGreaterThan(1);
  const flash = await page.locator('[data-power-flash]').first().elementHandle();
  await expect.poll(() => flash.evaluate((e) => e.isConnected), { timeout: 5_000 }).toBe(false);
  await expect.poll(async () => Number(await page.locator('[data-heat]').getAttribute('data-heat')), { timeout: 60_000 }).toBeGreaterThan(0);
  await expect(page.locator('[data-power-glow]')).toHaveCount(0);
  await page.getByRole('searchbox', { name: 'Search the tree' }).fill('life');
  await expect(page.getByRole('region', { name: 'Search results' }).locator('[data-result]').first()).toBeVisible();
  await expect(panel.getByRole('table', { name: 'Node power' })).toBeVisible({ timeout: 6 * 60_000 });
  const ms = Number(await panel.locator('[data-power-ms]').getAttribute('data-power-ms'));
  expect(ms).toBeGreaterThan(0);
  record({ kind: 'tree-latency', build: file, what: 'power-report-Life', ms });
  console.log(`power report on ${file}, Life: ${(ms / 1000).toFixed(1)} s`);
  app.assertNoErrors();
});

test('tattoos: "Reset Node" on a runegrafted mastery, "Add" on an Intelligence node, as Go does', async ({ page }) => {
  const file = '0007-elementalist-storm-burst-of-repulsion';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const node = 39836;
  const at = await nodeOnScreen(page, node);
  const box = await canvasBox(page);
  // PassiveTreeView.lua:1646-1656
  await page.mouse.move(box.x + at.x, box.y + at.y);
  await expect(page.getByLabel('Tattoo tip')).toHaveText('Tip: Right click to edit the runegraft for this node');
  await page.mouse.click(box.x + at.x, box.y + at.y, { button: 'right' });
  await page.getByRole('menu').getByRole('menuitem', { name: 'Tattoo / runegraft…' }).click();
  let dialog = page.getByRole('dialog', { name: /^Replace modifier of / });
  await expect(dialog).toContainText('Runegraft of Treachery');
  await expect(dialog.getByLabel('Tattoo count')).toHaveText(`Tattoo count: ${go.fresh(build, (b) => b.TattooCount())} / 50`);
  await app.act(
    'reset-tattoo',
    () => dialog.getByRole('button', { name: 'Reset Node' }).click(),
    async () => {
      expect(build.RemoveTattoo(node)).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  // PassiveTreeView.lua:527-529
  await page.mouse.click(box.x + at.x, box.y + at.y, { button: 'right' });
  await expect(page.getByRole('menu').getByRole('menuitem', { name: 'Tattoo / runegraft…' })).toBeDisabled();
  await page.keyboard.press('Escape');
  // TreeTab.lua:846-878
  const attr = 39841;
  const attrAt = await nodeOnScreen(page, attr);
  await page.mouse.click(box.x + attrAt.x, box.y + attrAt.y, { button: 'right' });
  await page.getByRole('menu').getByRole('menuitem', { name: 'Tattoo / runegraft…' }).click();
  dialog = page.getByRole('dialog', { name: 'Replace modifier of Intelligence' });
  const select = dialog.getByRole('combobox', { name: 'Modifier' });
  const choices = (legacy: boolean) => (m.exposition.GetTattooChoices(build, attr, legacy) ?? []).map((c) => c.DN);
  await expect(select.getByRole('option')).toHaveText(choices(false));
  await dialog.getByRole('checkbox', { name: 'Show Legacy Tattoos' }).check();
  await expect(select.getByRole('option')).toHaveText(choices(true));
  const dn = choices(false).at(-1) ?? '';
  expect(dn).not.toBe('');
  await select.selectOption(dn);
  await app.act(
    'add-tattoo',
    () => dialog.getByRole('button', { name: 'Add' }).click(),
    async () => {
      expect(m.exposition.AddTattoo(build, attr, dn).ok).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  expect(go.fresh(build, (b) => b.TattooOn(attr))).toBe(dn);
  app.assertNoErrors();
});

// PassiveSpec.lua:562-754, Build.lua:255-278, PassiveTreeView.lua:390-497
test('class and ascendancy changes: an ascendancy click and Connect Path as Go does', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const t = tree.rawTree();

  const occultist = Object.values(t.nodes).find((n) => n.ascendancyName === 'Occultist' && n.isNotable && !n.isMultipleChoiceOption)!;
  const id = occultist.skill!;
  await app.act(
    'ascendancy-click',
    async () => (await app.treeNode(tree.resultName({ id, name: occultist.name! }))).click(),
    async () => {
      const r = await m.exposition.ClickTreeAscendancyNode(build, id, 0);
      expect(r.Handled && r.Changed).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(page.getByRole('combobox', { name: 'Ascendancy' })).toHaveValue('Occultist');
  expect(
    go.fresh(build, (b) => b.Build.PassiveNodes ?? []),
    'the clicked notable'
  ).toContain(id);

  const select = page.getByRole('combobox', { name: 'Class' });
  await select.selectOption('Shadow');
  const confirm = page.getByRole('dialog', { name: 'Class Change' });
  await expect(confirm).toContainText('Changing class to Shadow will reset your passive tree.');
  await expect(select).toHaveValue('Witch');
  await app.act(
    'connect-path',
    () => confirm.getByRole('button', { name: 'Connect Path' }).click(),
    async () => {
      const r = await m.exposition.ChangeTreeClass(build, 'Shadow', 2);
      expect(r.Changed).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(select).toHaveValue('Shadow');
  await expect(page.getByRole('combobox', { name: 'Ascendancy' })).toHaveValue('None');
  app.assertNoErrors();
});

// PassiveMasteryControl.lua:37-43
// PassiveTreeView.lua:1671-1705
test('mastery effects: hovering an effect shows its stat differences as Go calculates them', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const t = tree.rawTree();
  const panel = page.getByRole('dialog');
  const tip = page.locator('[aria-label="Mastery effect tooltip"]');
  const read = async () => {
    const region = tip.getByRole('group', { name: 'Stat differences' });
    await expect(region).toHaveAttribute('aria-busy', 'false', { timeout: 60_000 });
    return region.evaluate((el) => {
      const out: DiffBlock[] = [];
      const text = (e: Element | undefined) => (e?.textContent ?? '').replace(/\s+/g, ' ').trim();
      for (const sec of el.querySelectorAll<HTMLElement>('[data-header]')) {
        let b = out.at(-1);
        if (!b || b.header !== sec.dataset.header) out.push((b = { header: sec.dataset.header!, sections: [] }));
        const lines = [...sec.querySelectorAll('[data-line]')].map((l) => {
          const x = /^(\S+)(?: (.*))?$/.exec(text(l.children[1]))!;
          return `${x[1]} ${text(l.children[0])}${x[2] ? ` ${x[2]}` : ''}`;
        });
        b.sections.push({ actor: sec.dataset.actor || undefined, lines });
      }
      return out;
    });
  };
  const expected = async (id: number, effect: number, path: number[], alloc: boolean) => {
    const misc = (await m.exposition.GetMiscCalculator(build))!;
    try {
      const base = misc.Base() as WhatIfOutputs;
      const calc = async (add: number[]) => (await m.exposition.CalcMasteryEffect(misc, id, effect, add)) as WhatIfOutputs;
      const blocks = [expectedBlock(alloc ? 'Reallocating this node will give you:' : 'Allocating this node will give you:', base, await calc([id]))];
      if (!alloc && path.length > 1)
        blocks.push(expectedBlock('Allocating this node and all nodes leading to it will give you:', base, await calc(path), path.length));
      return blocks.filter((b): b is DiffBlock => !!b);
    } finally {
      misc.release();
    }
  };

  const held = 63268;
  await (await app.treeNode(tree.resultName({ id: held, name: t.nodes[held].name! }))).click();
  const options = (await m.exposition.GetMasteryOptions(build, held)) ?? [];
  const other = options.findIndex((o) => !o.Selected);
  const buttons = panel.getByRole('group', { name: 'Mastery effects' }).getByRole('button');
  await buttons.nth(other).hover();
  await expect(tip).toContainText(options[other].Label);
  const realloc = await read();
  expect(realloc).toEqual(await expected(held, options[other].Effect, [held], true));
  expect(realloc.length, 'no what-if shown for the reallocation').toBe(1);
  await page.mouse.move(0, 0);
  await expect(tip).toHaveCount(0);
  await page.keyboard.press('Escape');
  await expect(panel).toHaveCount(0);

  const state = tree.treeState(build);
  const reachable = Object.values(t.nodes).filter(
    (n) => n.isMastery && n.masteryEffects?.length && !state.active.includes(n.skill!) && (state.path(n.skill!)?.length ?? 0) > 1
  );
  let free = reachable[0];
  for (const n of reachable) {
    if (((await m.exposition.GetMasteryOptions(build, n.skill!)) ?? []).length < n.masteryEffects!.length) {
      free = n;
      break;
    }
  }
  const id = free.skill!;
  // PassiveTreeView.lua:1578-1599
  const offered = ((await m.exposition.GetMasteryOptions(build, id)) ?? []).map((o) => o.Label);
  const all = free.masteryEffects!.map((e) => e.stats.join(' / '));
  const at = await nodeOnScreen(page, id);
  const box = await canvasBox(page);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  const nodeTip = page.getByRole('tooltip').filter({ hasText: free.name! });
  await expect(nodeTip).toContainText('Available Mastery node options are:');
  const listed = await nodeTip.locator('.text-magic > span').allTextContents();
  expect(listed).toEqual(offered.flatMap((l) => l.split('\n')));
  expect(all.length, 'no effect of this mastery is held elsewhere: no control').toBeGreaterThan(offered.length);
  await page.mouse.move(0, 0);

  await (await app.treeNode(tree.resultName({ id, name: free.name! }))).click();
  await expect(panel.getByText('Choose one effect to allocate it')).toBeVisible();
  const first = ((await m.exposition.GetMasteryOptions(build, id)) ?? [])[0];
  await buttons.first().hover();
  const shown = await read();
  expect(shown).toEqual(await expected(id, first.Effect, state.path(id)!, false));
  expect(shown.length, 'no what-if shown for the unallocated mastery').toBeGreaterThan(0);
  await page.mouse.move(0, 0);

  app.assertNoErrors();
});

// PassiveTreeView.lua:960-976
test('dependents: hovering a cluster jewel socket reddens its subgraph nodes', async ({ page }) => {
  await page.addInitScript(() =>
    setTimeout(() => {
      const probe = (globalThis as unknown as { __treeProbe?: { renderer?: string } }).__treeProbe;
      if (probe) probe.renderer = 'webgl';
    })
  );
  const { app, build } = await openOnTree(page, '0719-gladiator-herald-of-agony-2');
  const m = await go.model();
  const cluster = (await m.exposition.GetTreeClusterNodes(build)) ?? [];
  const state = tree.treeState(build, cluster);
  const sockets = [...new Set(cluster.flatMap((c) => c.Linked ?? []).filter((id) => id < 1 << 16 && state.active.includes(id)))];
  const socket = sockets.sort((a, b) => state.depends(b).length - state.depends(a).length)[0];
  const deps = state.depends(socket);
  const inner = cluster.filter((c) => c.ID !== socket && deps.includes(c.ID) && c.Type !== 'Mastery');
  expect(inner.length, 'the socket carries no allocated subgraph node').toBeGreaterThan(0);
  const red = async (c: { X: number; Y: number }) => {
    const list = await page.evaluate(() => (globalThis as unknown as { __treeProbe: { drawList?: () => { items: unknown[] } } }).__treeProbe.drawList?.());
    return ((list?.items ?? []) as { k: string; q: number[]; t?: number[] }[]).some(
      (i) => i.k === 'sprite' && Math.abs(i.q[0] + i.q[2] / 2 - c.X) < 1 && Math.abs(i.q[1] + i.q[5] / 2 - c.Y) < 1 && i.t?.[0] === 1 && !i.t[1] && !i.t[2]
    );
  };
  const at = await nodeOnScreen(page, socket, build);
  expect(await red(inner[0])).toBe(false);
  const box = await canvasBox(page);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  for (const c of inner) await expect.poll(() => red(c), `subgraph node ${c.ID}`).toBe(true);
  // PassiveTreeView.lua:1731-1733
  await expect(page.getByRole('tooltip').first()).toContainText(`removes ${deps.length} nodes`);
  app.assertNoErrors();
});

// CalcsTab.lua:684
test('power report: a mastery shows the effect the report picks', async ({ page }) => {
  test.setTimeout(20 * 60_000);
  const { app, build } = await openOnTree(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const builder = m.exposition.NewPowerBuilder(misc, 'Life', 0)!;
  while (builder.Step(1000) < 1);
  const rows = (builder.Report() ?? []) as PowerRow[];
  builder.release();
  misc.release();
  const pick = rows.filter((r) => r.Picked && r.PathDist < 1000).sort((a, b) => b.Power - a.Power)[0];
  expect(pick, 'the model picks no mastery effect').toBeDefined();
  const own = rows.filter((r) => r.ID === pick.ID && !r.Allocated);
  expect(pick.Power, "the pick is the node's best row").toBe(Math.max(...own.map((r) => r.Power)));
  expect(own.filter((r) => r.Power === pick.Power)[0], 'the first of equal rows').toBe(pick);
  const label = (pick.Stats ?? []).join(' / ');

  await page.getByRole('button', { name: /^Power/ }).click();
  const panel = page.getByRole('region', { name: 'Power report' });
  await panel.getByRole('button', { name: 'Start' }).click();
  await expect(panel.getByRole('table', { name: 'Node power' })).toBeVisible({ timeout: 10 * 60_000 });

  const t = tree.rawTree();
  const at = await nodeOnScreen(page, pick.ID);
  const box = await canvasBox(page);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  const tip = page.getByRole('tooltip').filter({ hasText: t.nodes[pick.ID].name! });
  await expect(tip.locator('[data-power-pick]')).toHaveText(`Power report picks: ${label}`);

  await page.mouse.click(box.x + at.x, box.y + at.y);
  const effects = page.getByRole('dialog').getByRole('group', { name: 'Mastery effects' });
  await expect(effects.locator('[data-power-pick]')).toHaveCount(1);
  await expect(effects.locator('[data-power-pick]')).toContainText(label);
  await page.keyboard.press('Escape');

  await panel.getByRole('button', { name: 'Close power report' }).click();
  await page.mouse.move(box.x + 5, box.y + box.height - 5);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  await expect(tip).toBeVisible();
  await expect(tip.locator('[data-power-pick]')).toHaveCount(0);
  app.assertNoErrors();
});

test('mastery effects: the modal fits the viewport, the tooltip is not cut off and stays while it is scrolled', async ({ page }) => {
  const { app, build } = await openOnTree(page, '0280-elementalist-fireball');
  const t = tree.rawTree();
  const state = tree.treeState(build);
  const held = Object.values(t.nodes)
    .filter((n) => n.isMastery && n.masteryEffects?.length && !state.active.includes(n.skill!) && (state.path(n.skill!)?.length ?? 0) > 1)
    .sort((a, b) => b.masteryEffects!.length - a.masteryEffects!.length)[0].skill!;
  const panel = page.getByRole('dialog');
  const tip = page.locator('[aria-label="Mastery effect tooltip"]');
  const inside = (b: { x: number; y: number; width: number; height: number }, w: number, h: number, what: string) => {
    expect(b.x, `${what} left`).toBeGreaterThanOrEqual(0);
    expect(b.y, `${what} top`).toBeGreaterThanOrEqual(0);
    expect(b.x + b.width, `${what} right`).toBeLessThanOrEqual(w + 0.5);
    expect(b.y + b.height, `${what} bottom`).toBeLessThanOrEqual(h + 0.5);
  };
  for (const [w, h] of [
    [1600, 1000],
    [800, 360]
  ]) {
    await page.setViewportSize({ width: w, height: h });
    await (await app.treeNode(tree.resultName({ id: held, name: t.nodes[held].name! }))).click();
    await expect(panel).toBeVisible();
    inside((await panel.boundingBox())!, w, h, `the modal at ${w}x${h}`);
    expect(await panel.evaluate((d) => d.scrollHeight - d.clientHeight), `the modal's own overflow at ${w}x${h}`).toBeLessThanOrEqual(1);
    const buttons = panel.getByRole('group', { name: 'Mastery effects' }).getByRole('button');
    const last = buttons.last();
    await last.scrollIntoViewIfNeeded();
    inside((await last.boundingBox())!, w, h, `the last effect at ${w}x${h}`);
    const busy = tip.getByRole('group', { name: 'Stat differences' });
    let tallest = 0;
    let most = 0;
    for (let k = 0; k < (await buttons.count()); k++) {
      await buttons.nth(k).hover();
      await expect(busy).toHaveAttribute('aria-busy', 'false', { timeout: 60_000 });
      const height = await tip.evaluate((el) => el.scrollHeight);
      if (height > most) [tallest, most] = [k, height];
    }
    await buttons.nth(tallest).hover();
    await expect(busy).toHaveAttribute('aria-busy', 'false', { timeout: 60_000 });
    const clip = await tip.evaluate((el) => {
      const col = el.parentElement!;
      const box = el.querySelector('[role="tooltip"]')!.getBoundingClientRect();
      const c = col.getBoundingClientRect();
      return { right: box.right - c.right, left: c.left - box.left };
    });
    expect(clip.right, `the tooltip past its column at ${w}x${h}`).toBeLessThanOrEqual(0.5);
    expect(clip.left, `the tooltip before its column at ${w}x${h}`).toBeLessThanOrEqual(0.5);
    const col = await tip.evaluate((el) => {
      const r = el.parentElement!.getBoundingClientRect();
      return { x: r.x, y: r.y, width: r.width, height: r.height };
    });
    inside(col, w, h, `the tooltip's column at ${w}x${h}`);
    await page.mouse.move(col.x + col.width - 4, col.y + col.height / 2, { steps: 8 });
    await expect(tip).toBeVisible();
    const scroller = tip.locator('xpath=..');
    const overflow = await scroller.evaluate((el) => el.scrollHeight - el.clientHeight);
    if (h < 600) expect(overflow, 'the small viewport scrolls the tooltip').toBeGreaterThan(0);
    await page.mouse.wheel(0, 400);
    if (overflow > 0) await expect.poll(() => scroller.evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
    await expect(tip).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(panel).toHaveCount(0);
  }
  app.assertNoErrors();
});

// TreeTab.lua:51-104
test('tree sets: the open list moves nothing, the tooltip shows beside the select', async ({ page }) => {
  const { app } = await openOnTree(page, '0280-elementalist-fireball');
  const select = page.getByRole('combobox', { name: 'Tree set' });
  const top = () => page.getByRole('banner').evaluate((el) => el.getBoundingClientRect().top);
  const before = await top();
  await select.click();
  const list = select.locator('xpath=..').getByRole('listbox');
  await expect(list).toBeVisible();
  const view = page.viewportSize()!;
  const lb = (await list.boundingBox())!;
  expect(lb.y >= 0 && lb.y + lb.height <= view.height, `the list runs off screen: ${JSON.stringify(lb)}`).toBe(true);
  expect(await top(), 'the app moved').toBe(before);
  await select.locator('xpath=..').getByRole('option').first().hover();
  const tip = page.locator('[aria-label="Tree tooltip"]');
  await expect(tip).toBeVisible();
  const tb = (await tip.boundingBox())!;
  const sb = (await select.boundingBox())!;
  expect(
    tb.x + tb.width <= sb.x + 1 && Math.abs(tb.y + tb.height - (sb.y + sb.height)) <= 16,
    `tooltip ${JSON.stringify(tb)} select ${JSON.stringify(sb)}`
  ).toBe(true);
  expect(tb.y >= 0 && tb.x >= 0, 'the tooltip runs off screen').toBe(true);
  await select.press('Escape');
  expect(await top(), 'the app moved').toBe(before);
  app.assertNoErrors();
});

test('tree sets: a tree’s tooltip shows its details and what switching to it gives, as Go calculates', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { build } = await openOnTree(page, file);
  const m = await go.model();
  const specs = (await m.exposition.GetTreeSpecs(build)).Specs!;
  const points = (await m.exposition.GetTreeSpecsPoints(build))!;
  const details = (i: number) => [`Class: ${specs[i].ClassName}`, `Ascendancy: ${specs[i].AscendClassName}`, `Points used: ${points[i].Used}`];

  const select = page.getByRole('combobox', { name: 'Tree set' });
  await select.hover();
  const onSelect = page.locator('[aria-label="Tree tooltip"]');
  await expect(onSelect.locator('[aria-label="Details"] > span')).toHaveText(details(1));
  await expect(onSelect).toContainText(`Game Version: ${specs[1].VersionLabel}`);
  await expect(onSelect.getByRole('group', { name: 'Stat differences' })).toHaveCount(0);
  await page.mouse.move(0, 0);

  // TreeTab.lua:51-104
  await select.click();
  await select.locator('xpath=..').getByRole('option', { name: 'Until Lab', exact: true }).hover();
  await expect(onSelect.locator('[aria-label="Details"] > span')).toHaveText(details(0));
  await select.press('Escape');
  await page.mouse.move(0, 0);

  await treeSelect(page).choose('Manage trees…');
  const dialog = page.getByRole('dialog', { name: 'Manage passive trees' });
  await dialog.getByRole('option', { name: /^Until Lab/ }).hover();
  const tip = dialog.locator('[aria-label="Tree tooltip"]');
  await expect(tip.locator('[aria-label="Details"] > span')).toHaveText(details(0));
  const region = tip.getByRole('group', { name: 'Stat differences' });
  await expect(region).toHaveAttribute('aria-busy', 'false', { timeout: 60_000 });
  const shown = await region.evaluate((el) => {
    const out: DiffBlock[] = [];
    const text = (e: Element | undefined) => (e?.textContent ?? '').replace(/\s+/g, ' ').trim();
    for (const sec of el.querySelectorAll<HTMLElement>('[data-header]')) {
      let b = out.at(-1);
      if (!b || b.header !== sec.dataset.header) out.push((b = { header: sec.dataset.header!, sections: [] }));
      const lines = [...sec.querySelectorAll('[data-line]')].map((l) => {
        const x = /^(\S+)(?: (.*))?$/.exec(text(l.children[1]))!;
        return `${x[1]} ${text(l.children[0])}${x[2] ? ` ${x[2]}` : ''}`;
      });
      b.sections.push({ actor: sec.dataset.actor || undefined, lines });
    }
    return out;
  });
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  let want: DiffBlock[];
  try {
    const base = misc.Base() as WhatIfOutputs;
    want = [expectedBlock('Switching to this tree will give you:', base, (await m.exposition.CalcTreeSpec(misc, 1)) as WhatIfOutputs)].filter(
      (b): b is DiffBlock => !!b
    );
  } finally {
    misc.release();
  }
  expect(shown).toEqual(want);
  expect(shown.length, 'switching to a smaller tree changes nothing: no control').toBe(1);
});

// TreeTab.lua:629-839
test('tree links: Export Tree gives the tree’s link; Import Tree adds a tree from one, as Go does', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  await treeSelect(page).choose('Manage trees…');
  const dialog = page.getByRole('dialog', { name: 'Manage passive trees' });

  await dialog.getByRole('button', { name: 'Export Tree' }).click();
  const link = await m.exposition.ExportTreeLink(build);
  expect(link).toMatch(/^https:\/\/www\.pathofexile\.com\/passive-skill-tree\/3\.29\.0\/AAAABg/);
  await expect(dialog.getByRole('textbox', { name: 'Passive tree link' })).toHaveValue(link);
  await dialog.getByRole('group', { name: 'Export Tree' }).getByRole('button', { name: 'Done' }).click();

  await dialog.getByRole('button', { name: 'Import Tree' }).click();
  const form = dialog.getByRole('form', { name: 'Import Tree' });
  await form.getByRole('textbox', { name: 'Enter name for this passive tree' }).fill('From a link');
  await form.getByRole('textbox', { name: 'Enter passive tree link' }).fill('https://www.pathofexile.com/passive-skill-tree/AAAA');
  await form.getByRole('button', { name: 'Import' }).click();
  await expect(form.getByRole('alert')).toHaveText('Invalid tree link (unrecognised format)');
  await expect(dialog.getByRole('listbox', { name: 'Passive trees' }).getByRole('option')).toHaveCount(3);

  await form.getByRole('textbox', { name: 'Enter passive tree link' }).fill(link);
  await app.act(
    'import-tree',
    () => form.getByRole('button', { name: 'Import' }).click(),
    async () => {
      expect(await m.exposition.ImportTreeLink(build, 'From a link', link)).toBe('');
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(dialog.getByRole('option', { name: /^From a link/ })).toContainText('current');
  expect(go.fresh(build, (b) => b.Tree.ActiveSpec)).toBe(4);
  app.assertNoErrors();
});

// TreeTab.lua:759-789, 818-830
test('tree links: Shrink with PoEURL, and a PoEURL link imported as the link it redirects to', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const link = await m.exposition.ExportTreeLink(build);
  const asked: string[] = [];
  await page.route('**/api/poeurl?*', (route) => {
    const q = new URL(route.request().url()).searchParams;
    asked.push(q.get('id') ?? `shrink ${q.get('shrink')}`);
    if (q.get('id') === 'bad') return route.fulfill({ status: 502, contentType: 'text/plain; charset=utf-8', body: 'Failed to resolve PoEURL link' });
    return route.fulfill({ status: 200, contentType: 'text/plain; charset=utf-8', body: q.get('id') ? link : 'http://poeurl.com/dABz' });
  });
  await treeSelect(page).choose('Manage trees…');
  const dialog = page.getByRole('dialog', { name: 'Manage passive trees' });

  await dialog.getByRole('button', { name: 'Export Tree' }).click();
  const exported = dialog.getByRole('group', { name: 'Export Tree' });
  await expect(exported.getByRole('textbox', { name: 'Passive tree link' })).toHaveValue(link);
  await exported.getByRole('button', { name: 'Shrink with PoEURL' }).click();
  await expect(exported.getByRole('textbox', { name: 'Passive tree link' })).toHaveValue('http://poeurl.com/dABz');
  await expect(exported.getByRole('button', { name: 'Done' }).first()).toBeDisabled();
  await exported.getByRole('button', { name: 'Done' }).last().click();

  await dialog.getByRole('button', { name: 'Import Tree' }).click();
  const form = dialog.getByRole('form', { name: 'Import Tree' });
  await form.getByRole('textbox', { name: 'Enter name for this passive tree' }).fill('From PoEURL');
  await form.getByRole('textbox', { name: 'Enter passive tree link' }).fill('http://poeurl.com/bad');
  await form.getByRole('button', { name: 'Import' }).click();
  await expect(form.getByRole('alert')).toHaveText('Failed to resolve PoEURL link');
  await form.getByRole('textbox', { name: 'Enter passive tree link' }).fill('http://poeurl.com/dABz');
  await app.act(
    'import-tree-poeurl',
    () => form.getByRole('button', { name: 'Import' }).click(),
    async () => {
      expect(await m.exposition.ImportTreeLink(build, 'From PoEURL', link)).toBe('');
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(dialog.getByRole('option', { name: /^From PoEURL/ })).toContainText('current');
  expect(asked).toEqual([`shrink ${link}`, 'bad', 'dABz']);
  expect(app.errors.filter((e) => !e.includes('status of 502') || !e.includes('/api/poeurl?id=bad'))).toEqual([]);
});

// TreeTab.lua:322-340, 588-612
test('tree versions: an older tree converts to the latest as Go does', async ({ page }) => {
  const file = '0004-trickster-lightning-tendrils-of-eccentricity';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  await app.act(
    'switch-to-old-tree',
    () => treeSelect(page).choose(0),
    async () => {
      await m.exposition.SetActiveTreeSpec(build, 1);
      return expectedLines(await go.outputs(build));
    }
  );
  const banner = page.getByRole('status', { name: 'Convert older tree' });
  await expect(banner).toBeVisible();
  await expect(page.getByRole('combobox', { name: 'Version' })).toHaveValue('3_28');
  const before = (await m.exposition.GetTreeSpecs(build)).Specs!.length;
  await app.act(
    'convert-tree',
    () => banner.getByRole('button', { name: 'Convert to 3.29' }).click(),
    async () => {
      expect(await m.exposition.ConvertTreeSpec(build, '3_29', false)).toBe('');
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(page.getByRole('dialog', { name: 'Tree Converted' })).toContainText('The tree has been converted to 3.29.');
  await page.getByRole('dialog', { name: 'Tree Converted' }).getByRole('button', { name: 'OK' }).click();
  expect(await treeSelect(page).entries()).toHaveLength(before + 2);
  await expect(treeSelect(page).box).toHaveText((await m.exposition.GetTreeSpecs(build)).Specs![1].Title);
  await expect(banner).toHaveCount(0);
  app.assertNoErrors();
});

// TreeTab.lua:194-205, PassiveTreeView.lua:906-919
test('node power: the heat map colours nodes by Go’s power for the stat', async ({ page }) => {
  test.setTimeout(20 * 60_000);
  await page.addInitScript(() =>
    setTimeout(() => {
      const probe = (globalThis as unknown as { __treeProbe?: { renderer?: string } }).__treeProbe;
      if (probe) probe.renderer = 'webgl';
    })
  );
  const file = '0280-elementalist-fireball';
  const { build } = await openOnTree(page, file);
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const builder = m.exposition.NewPowerBuilder(misc, 'Life', 0)!;
  while (builder.Step(1000) < 1);
  const t = tree.rawTree();
  const ascendancy = new Set(
    Object.values(t.nodes)
      .filter((n) => n.ascendancyName)
      .map((n) => n.skill!)
  );
  // TreeTab.lua:1116
  const ascRows = (m.exposition.PowerAscendancy(builder) ?? []) as PowerRow[];
  const { values, max } = heatValues([...((builder.Report() ?? []) as PowerRow[]), ...ascRows], ascendancy);
  builder.release();
  misc.release();

  await page.getByRole('button', { name: /^Power/ }).click();
  const panel = page.getByRole('region', { name: 'Power report' });
  await panel.getByRole('checkbox', { name: 'Show node power' }).check();
  await expect(panel.getByRole('combobox', { name: 'Power stat' })).toHaveValue('Life');
  await panel.getByRole('button', { name: 'Start' }).click();
  const heat = page.locator('[data-heat]');
  await expect(heat).toHaveAttribute('data-heat', String(values.size), { timeout: 10 * 60_000 });
  await expect(panel.getByRole('table', { name: 'Node power' })).toBeVisible();

  const offsets = ascendancyOffsets(t);
  const best = [...values.entries()].sort((a, b) => b[1] - a[1])[0][0];
  const none = [...values.entries()].find(([id, v]) => v <= 0 && !t.nodes[id].ascendancyName)![0];
  // PassiveTreeView.lua:907-919
  const asc = ascRows.find((r) => !r.Allocated && r.Power > 0 && heatChannel(r.Power, max) < 1)!.ID;
  for (const id of [best, none, asc]) {
    const at = nodeCentre(t, offsets, t.nodes[id])!;
    const want = Math.round(255 * heatChannel(values.get(id)!, max));
    await expect
      .poll(async () => {
        const list = await page.evaluate(() => (globalThis as unknown as { __treeProbe: { drawList?: () => { items: unknown[] } } }).__treeProbe.drawList?.());
        const sprites = ((list?.items ?? []) as { k: string; q: number[]; t?: number[] }[]).filter(
          (i) => i.k === 'sprite' && Math.abs(i.q[0] + i.q[2] / 2 - at.x) < 1 && Math.abs(i.q[1] + i.q[5] / 2 - at.y) < 1
        );
        return sprites.length >= 2 && sprites.every((i) => Math.abs(Math.round(255 * (i.t?.[0] ?? 1)) - want) <= 1 && !i.t?.[1] && !i.t?.[2]);
      }, `node ${id}`)
      .toBe(true);
  }

  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
  await page.keyboard.press('p');
  await expect(panel.getByRole('checkbox', { name: 'Show node power' })).not.toBeChecked();
  await expect(page.locator('[data-heat]')).toHaveCount(0);
});

// PassiveTreeView.lua:1157-1248, 977-1003
test('jewel radii: rings over socketed radius jewels, and a hovered socket’s radii and nodes, as Go does', async ({ page }) => {
  const file = '0005-occultist-vaal-double-strike';
  const { build } = await openOnTree(page, file);
  const m = await go.model();
  const rings = (await m.exposition.GetTreeJewelRings(build, [])) ?? [];
  expect(rings.length, 'the fixture has radius jewels').toBeGreaterThan(1);
  await expect(page.locator('[data-jewel-ring]')).toHaveCount(rings.length);
  for (const r of rings) await expect(page.locator(`[data-jewel-ring="${r.NodeID}"]`).first()).toBeAttached();

  const socket = rings.find((r) => tree.rawTree().nodes[r.NodeID]?.isJewelSocket)!.NodeID;
  const radii = (await m.exposition.GetTreeSocketRadii(build, socket)) ?? [];
  const inRadius = new Set(radii.flatMap((r) => r.Nodes ?? []));
  const at = await nodeOnScreen(page, socket);
  const box = await canvasBox(page);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  await expect(page.locator('[data-socket-radius]')).toHaveCount(radii.length);
  await expect(page.locator('[data-in-radius]')).toHaveCount(inRadius.size);
  await page.mouse.move(box.x + 5, box.y + 5);
  await expect(page.locator('[data-socket-radius]')).toHaveCount(0);
});

// PassiveSpec.lua:2007, 2253
test('cluster jewels: a click on a subgraph node allocates its path through the subgraph, as Go does', async ({ page }) => {
  const file = '0021-assassin-blade-flurry';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const cluster = (await m.exposition.GetTreeClusterNodes(build)) ?? [];
  const state = tree.treeState(build, cluster);
  const target = cluster.find((c) => c.Type !== 'Mastery' && !state.active.includes(c.ID) && (state.path(c.ID)?.length ?? 0) > 1);
  expect(target, 'no unallocated subgraph node two or more points away').toBeDefined();
  const path = state.path(target!.ID)!;
  const at = await nodeOnScreen(page, target!.ID, build);
  const box = await canvasBox(page);
  await app.act(
    'allocate-cluster',
    () => page.mouse.click(box.x + at.x, box.y + at.y),
    async () => {
      state.allocate(target!.ID);
      return expectedLines(await go.outputs(build));
    }
  );
  expect(path.length).toBeGreaterThan(1);
  app.assertNoErrors();
});

// PassiveTreeView.lua:1478-1489
test('node tooltip: a socketed jewel shows its stats', async ({ page }) => {
  const file = '0009-trickster-cobra-lash';
  const { build, active } = await openOnTree(page, file);
  const m = await go.model();
  const misc = await m.exposition.GetMiscCalculator(build);
  const box = await canvasBox(page);
  const tip = page.getByRole('tooltip').getByLabel('Socketed jewel');
  const held = go.fresh(build, (b) => (b.Tree.Specs?.[b.Tree.ActiveSpec - 1]?.Sockets ?? []).filter((x) => x.ItemID).map((x) => x.NodeID));
  const drawn = new Set(placedNodes(tree.rawTree()).map((p) => p.node.skill));
  const sockets = [held.find((id) => active.includes(id)), held.find((id) => !active.includes(id) && drawn.has(id))];
  expect(
    sockets.every((id) => id !== undefined && misc!.SocketJewelTooltip(id, go.NO_TIP_STATS)[1]),
    `sockets ${sockets.join()}`
  ).toBe(true);
  for (const socket of sockets) {
    const [want] = misc!.SocketJewelTooltip(socket!, go.NO_TIP_STATS);
    const line = (want.Lines ?? [])
      .map((l) => l.Text.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, ''))
      .find((l) => l && !/^(Requires|Item Level|Limited|Radius|Quality)/.test(l));
    expect(line, `socket ${socket}: the model's jewel has no mod line`).toBeDefined();
    const at = await nodeOnScreen(page, socket!, build);
    await page.mouse.move(box.x + at.x, box.y + at.y);
    await expect(tip, `socket ${socket} (${active.includes(socket!) ? 'allocated' : 'unallocated'})`).toContainText(line!);
    await page.mouse.move(box.x + 5, box.y + 5);
  }
  misc!.release();
});

// PassiveTreeView.lua:83-111
// TreeTab.lua:180, Build.lua:1303
test('tree search: marks the build unsaved, is saved with it and restored when it opens', async ({ page }) => {
  const file = '0005-occultist-vaal-double-strike';
  const { app, build } = await openOnTree(page, file);
  const want = async () => expectedLines(await go.outputs(build));
  const name = `e2e-tree-search-${file}`;
  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(name);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await page.reload();
  await app.savedBuild(name).waitFor({ timeout: 180_000 });
  await app.act('open-saved', () => app.openSaved(name), want);
  await app.goTo('Tree');

  const search = page.getByRole('searchbox', { name: 'Search the tree' });
  await expect(page.getByRole('button', { name: 'Save (no unsaved changes)' })).toBeVisible();
  // PassiveTreeView.lua:96-98, 108
  await page.keyboard.press('Control+d');
  await search.fill('life');
  await expect(page.getByRole('button', { name: 'Save (unsaved changes)' }), 'the search left the build saved').toBeVisible();
  const result = page.getByRole('region', { name: 'Search results' }).getByRole('button').first();
  const offTip = page.getByRole('tooltip').getByText('Tip: Press Ctrl+D to enable the display of stat differences.');
  await result.hover();
  await expect(offTip).toBeVisible();
  await page.getByRole('button', { name: 'Save (unsaved changes)' }).click();
  await expect(page.getByRole('button', { name: 'Save (no unsaved changes)' })).toBeVisible();

  await page.reload();
  await app.savedBuild(name).waitFor({ timeout: 180_000 });
  await app.act('open-saved', () => app.openSaved(name), want);
  await app.goTo('Tree');
  await expect(search).toHaveValue('life');
  await result.hover();
  await expect(offTip).toBeVisible();
  app.assertNoErrors();
});

test('render time: bottom right of the tree, only in dev mode', async ({ page }) => {
  const { app } = await openOnTree(page, '0005-occultist-vaal-double-strike');
  const readout = page.getByRole('status', { name: 'Render time' });
  await expect(readout).toHaveCount(0);
  await page.getByRole('button', { name: 'Settings' }).click();
  const sheet = page.getByRole('dialog', { name: 'Settings' });
  await sheet.getByRole('tab', { name: /^General/ }).click();
  await sheet.getByRole('checkbox', { name: 'Dev mode' }).check();
  await page.keyboard.press('Escape');
  await expect(readout).toHaveText(/^\d+\.\dms$/);
  const r = (await readout.boundingBox())!;
  const c = await canvasBox(page);
  expect(r.x + r.width).toBeGreaterThan(c.x + c.width - 40);
  expect(r.y + r.height).toBeGreaterThan(c.y + c.height - 40);
  app.assertNoErrors();
});

// TreeTab.lua:129-154
test('reset tree / tattoos: the popup removes every tattoo, then resets the tree, as Go does', async ({ page }) => {
  const file = '0020-assassin-storm-burst';
  const { app, build } = await openOnTree(page, file);
  const m = await go.model();
  const open = async () => {
    await page.getByRole('button', { name: 'Reset Tree/Tattoos' }).click();
    const dialog = page.getByRole('dialog', { name: 'Reset Tree/Tattoos' });
    await expect(dialog).toContainText('Warning: resetting your passive tree or removing all tattoos cannot be undone.');
    return dialog;
  };
  expect(go.fresh(build, (b) => b.TattooCount())).toBeGreaterThan(0);
  let dialog = await open();
  await app.act(
    'remove-all-tattoos',
    () => dialog.getByRole('button', { name: 'Remove All Tattoos' }).click(),
    async () => {
      await m.exposition.RemoveAllTreeTattoos(build);
      return expectedLines(await go.outputs(build));
    }
  );
  dialog = await open();
  await app.act(
    'reset-tree',
    () => dialog.getByRole('button', { name: 'Reset Tree' }).click(),
    async () => {
      await m.exposition.ResetTreeNodes(build);
      return expectedLines(await go.outputs(build));
    }
  );
  await waitFrame(page, 0, `(f) => f.active === ${go.fresh(build, (b) => (b.Build.PassiveNodes ?? []).length)}`, 30_000);
  app.assertNoErrors();
});
