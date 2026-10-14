// PassiveTreeView.lua:1658-1706

import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { record } from './support/perf';
import { expectedBlock, type DiffBlock, type WhatIfOutputs } from './support/stat_diff';
import { expectedLines } from './support/stats';
import * as tree from './support/tree_model';
import { installTreeProbe, lastFrame, pointerAt, waitFrame } from './support/tree_probe';
import { canvasBox, emptySpot, hitTest, parkPointer, placedNodes, screenOf, wheelAt, type Placed } from './support/tree_view';

type Build = Awaited<ReturnType<typeof go.loadXml>>;
type Shown = DiffBlock & { readyAt: number; calcMs: number };

const setup = async (page: Page, file: string) => {
  await installTreeProbe(page);
  // PassiveTreeView.lua:271-273
  const xml = go.withoutView(buildXml(file));
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await tree.prepareTree();
  const active = go.fresh(build, (b) => [...(b.Build.PassiveNodes ?? [])]);
  await app.goTo('Tree');
  await waitFrame(page, 0, `(f) => f.className !== undefined && f.active === ${active.length}`, 60_000);
  const box = await canvasBox(page);
  await wheelAt(page, { x: box.width / 2, y: box.height / 2 }, -3);
  return { app, build, active };
};

const diffRegion = (page: Page) => page.getByRole('tooltip').getByRole('group', { name: 'Stat differences' });

const hover = async (page: Page, target: Placed, rest: { x: number; y: number }) => {
  const t = tree.rawTree();
  await parkPointer(page, rest);
  const view = (await lastFrame(page))!.view;
  const s = screenOf(t, view, target.at);
  const box = await canvasBox(page);
  await page.mouse.move(box.x + s.x, box.y + s.y);
  const pointer = await pointerAt(page);
  const region = diffRegion(page);
  await expect(region).toHaveAttribute('data-node', String(target.node.skill));
  await expect(region).toHaveAttribute('aria-busy', 'false', { timeout: 60_000 });
  const blocks = await region.evaluate((el) => {
    const out: { header: string; readyAt: number; calcMs: number; sections: { actor?: string; lines: string[] }[] }[] = [];
    const text = (e: Element | undefined) => (e?.textContent ?? '').replace(/\s+/g, ' ').trim();
    for (const sec of el.querySelectorAll<HTMLElement>('[data-header]')) {
      let b = out.at(-1);
      if (!b || b.header !== sec.dataset.header) {
        b = { header: sec.dataset.header!, readyAt: Number(sec.dataset.readyAt), calcMs: Number(sec.dataset.calcMs), sections: [] };
        out.push(b);
      }
      const lines = [...sec.querySelectorAll('[data-line]')].map((l) => {
        const m = /^(\S+)(?: (.*))?$/.exec(text(l.children[1]))!;
        return `${m[1]} ${text(l.children[0])}${m[2] ? ` ${m[2]}` : ''}`;
      });
      b.sections.push({ actor: sec.dataset.actor || undefined, lines });
    }
    return out;
  });
  return { pointer, blocks: blocks as Shown[], empty: await region.evaluate((el) => el.lastElementChild?.textContent ?? '') };
};

/* PassiveTreeView.lua:1662-1698 */
const expected = async (build: Build, id: number): Promise<DiffBlock[]> => {
  const m = await go.model();
  const state = tree.treeState(build, await tree.clusterOf(build));
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  try {
    const base = misc.Base() as WhatIfOutputs;
    const calc = async (add: number[], remove: number[]) => (await misc.CalcNodes(add, remove)) as WhatIfOutputs;
    const alloc = state.active.includes(id);
    const path = alloc ? state.depends(id) : (state.path(id) ?? []);
    const blocks = [
      alloc
        ? expectedBlock('Unallocating this node will give you:', base, await calc([], [id]))
        : expectedBlock('Allocating this node will give you:', base, await calc([id], []))
    ];
    if (path.length > 1) {
      blocks.push(
        alloc
          ? expectedBlock('Unallocating this node and all nodes depending on it will give you:', base, await calc([], path), path.length)
          : expectedBlock('Allocating this node and all nodes leading to it will give you:', base, await calc(path, []), path.length)
      );
    }
    return blocks.filter((b): b is DiffBlock => !!b);
  } finally {
    misc.release();
  }
};

const strip = (b: Shown[]): DiffBlock[] => b.map(({ header, sections }) => ({ header, sections }));

const candidates = async (page: Page, build: Build) => {
  const t = tree.rawTree();
  const state = tree.treeState(build, await tree.clusterOf(build));
  const view = (await lastFrame(page))!.view;
  const placed = placedNodes(t);
  const inView = placed
    .map((p) => ({ p, s: screenOf(t, view, p.at) }))
    .filter(({ p, s }) => !!p.node.name && !p.node.isMastery && s.x > 450 && s.y > 60 && s.x < view.width - 60 && s.y < view.height - 60)
    .filter(({ p, s }) => hitTest(t, placed, view, s) === p)
    .map(({ p }) => p);
  const unallocated = inView.filter((p) => !state.active.includes(p.node.skill!) && (state.path(p.node.skill!)?.length ?? 0) > 1);
  const allocated = inView.filter((p) => state.active.includes(p.node.skill!) && p.node.isNotable && state.depends(p.node.skill!).length > 1);
  return { unallocated, allocated, rest: emptySpot(t, placed, view) };
};

for (const file of ['0280-elementalist-fireball', '0008-necromancer-vaal-summon-skeletons', '0339-reliquarian-shield-crush-of-the-chieftain-2']) {
  test(`node tooltip stat differences match the model: ${file}`, async ({ page }) => {
    const { app, build } = await setup(page, file);
    const c = await candidates(page, build);
    expect(c.unallocated.length, 'no unallocated node with a path in view').toBeGreaterThan(0);

    const targets = [c.unallocated[Math.floor(c.unallocated.length / 2)], ...c.allocated.slice(0, 1)];
    let lines = 0;
    for (const target of targets) {
      const shown = await hover(page, target, c.rest);
      const want = await expected(build, target.node.skill!);
      expect(strip(shown.blocks), `stat differences for ${target.node.name} #${target.node.skill}`).toEqual(want);
      lines += want.flatMap((b) => b.sections.flatMap((s) => s.lines)).length;
      for (const b of shown.blocks) {
        record({
          kind: 'tree-latency',
          build: file,
          what: `stat-diff-${b.header.startsWith('Allocating this node and') || b.header.startsWith('Unallocating this node and') ? 'path' : 'node'}`,
          ms: +(b.readyAt - shown.pointer).toFixed(1)
        });
      }
    }
    expect(lines, 'every hovered node showed no stat difference, so nothing was graded').toBeGreaterThan(0);
    if (file.startsWith('0008')) {
      // Build.lua:1926-1933
      const shown = await hover(page, targets[0], c.rest);
      expect(
        shown.blocks.length > 0 && shown.blocks.every((b) => b.sections.every((s) => s.actor === 'Minion' || s.actor === 'Player')),
        'a minion main skill shows lines outside a Minion: / Player: section'
      ).toBe(true);
    }
    app.assertNoErrors();
  });
}

test('stat differences: hover latency on a heavy build, and a sweep cancels stale what-ifs', async ({ page }) => {
  const file = '1591-necromancer-summon-holy-relic-of-conviction-15';
  const { app, build } = await setup(page, file);
  const c = await candidates(page, build);
  const sample = c.unallocated.filter((_, i) => i % Math.max(1, Math.floor(c.unallocated.length / 10)) === 0).slice(0, 10);
  expect(sample.length, 'too few unallocated nodes in view').toBeGreaterThan(4);

  for (const target of sample) {
    const shown = await hover(page, target, c.rest);
    for (const [i, b] of shown.blocks.entries()) {
      const what = i === 0 ? 'node' : 'path';
      record({ kind: 'tree-latency', build: file, what: `stat-diff-${what}`, ms: +(b.readyAt - shown.pointer).toFixed(1) });
      record({ kind: 'tree-latency', build: file, what: `stat-diff-${what}-go-calc`, ms: +b.calcMs.toFixed(1) });
    }
  }

  const t = tree.rawTree();
  const view = (await lastFrame(page))!.view;
  const box = await canvasBox(page);
  await parkPointer(page, c.rest);
  for (const target of sample) {
    const s = screenOf(t, view, target.at);
    await page.mouse.move(box.x + s.x, box.y + s.y);
  }
  const last = sample.at(-1)!;
  const pointer = await pointerAt(page);
  const region = diffRegion(page);
  await expect(region).toHaveAttribute('data-node', String(last.node.skill));
  await expect(region).toHaveAttribute('aria-busy', 'false', { timeout: 60_000 });
  const settledAt = await region.evaluate((el) => Math.max(...[...el.querySelectorAll<HTMLElement>('[data-ready-at]')].map((b) => Number(b.dataset.readyAt))));
  record({ kind: 'tree-latency', build: file, what: `stat-diff-after-sweep-of-${sample.length}`, ms: +(settledAt - pointer).toFixed(1) });
  const shown = await hover(page, last, c.rest);
  expect(strip(shown.blocks)).toEqual(await expected(build, last.node.skill!));

  await app.flushPerf('whatif');
  app.assertNoErrors();
});
