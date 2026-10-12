import { expect, type Page } from '@playwright/test';

import { ascendancyOffsets, drawnByPoB, nodeCentre, toScreen, type Point } from '../../src/lib/skill_tree/geometry';
import type { Tree, Node } from '../../src/lib/skill_tree/types';
import type { TreeView } from '../../src/lib/components/skill-tree/probe';
import { frameCount, lastFrame, waitFrame, waitLastFrame } from './tree_probe';

export const treeCanvas = (page: Page) => page.getByRole('img', { name: 'Passive skill tree' });

export type Placed = { node: Node; at: Point };

export const placedNodes = (tree: Tree): Placed[] => {
  const offsets = ascendancyOffsets(tree);
  const out: Placed[] = [];
  for (const group of Object.values(tree.groups)) {
    for (const id of group.nodes ?? []) {
      const node = tree.nodes[id];
      if (!drawnByPoB(tree, node)) continue;
      const at = nodeCentre(tree, offsets, node);
      if (at) out.push({ node, at });
    }
  }
  return out;
};

export const clusterPlaced = (nodes: { ID: number; Type: string; Name: string; X: number; Y: number }[]): Placed[] =>
  nodes.map((c) => ({
    node: {
      skill: c.ID,
      name: c.Name,
      isNotable: c.Type === 'Notable' || undefined,
      isKeystone: c.Type === 'Keystone' || undefined,
      isJewelSocket: c.Type === 'Socket' || undefined,
      isMastery: c.Type === 'Mastery' || undefined
    },
    at: { x: c.X, y: c.Y }
  }));

export const touchDistance = (n: Node) => {
  if (n.classStartIndex !== undefined || n.isAscendancyStart) return 0;
  if (n.isKeystone) return 110;
  if (n.isNotable || n.isJewelSocket) return 70;
  if (n.isMastery) return 85;
  return 50;
};

export const screenOf = (tree: Tree, view: TreeView, p: Point) => toScreen(tree, p, view.offsetX, view.offsetY, view.scaling);

export const hitTest = (tree: Tree, placed: Placed[], view: TreeView, at: Point) =>
  placed.find((p) => {
    const s = screenOf(tree, view, p.at);
    return Math.hypot(s.x - at.x, s.y - at.y) < touchDistance(p.node) / view.scaling;
  });

export const emptySpot = (tree: Tree, placed: Placed[], view: TreeView): Point => {
  let best: Point | undefined;
  let bestD = -1;
  for (let x = 20; x < view.width - 20; x += 20) {
    for (let y = 20; y < view.height - 20; y += 20) {
      if ((x < 420 && y < 420) || (x > view.width - 70 && y < 140)) continue;
      let d = Infinity;
      for (const p of placed) {
        const s = screenOf(tree, view, p.at);
        d = Math.min(d, Math.hypot(s.x - x, s.y - y) - (touchDistance(p.node) + 20) / view.scaling);
      }
      if (d > bestD) [best, bestD] = [{ x, y }, d];
    }
  }
  if (!best || bestD <= 0) throw new Error('REFUSING: no canvas point is clear of every node');
  return best;
};

export const canvasBox = async (page: Page) => {
  const box = await treeCanvas(page).boundingBox();
  if (!box) throw new Error('the tree canvas has no box');
  return box;
};

export const wheelAt = async (page: Page, at: Point, steps: number) => {
  const box = await canvasBox(page);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  for (let i = 0; i < Math.abs(steps); i++) {
    const from = await frameCount(page);
    const before = (await lastFrame(page))!.view.scaling;
    await page.mouse.wheel(0, Math.sign(steps) * 120);
    const f = await waitFrame(page, from, `(f) => f.view.scaling !== ${before} || ${before} <= 3 || ${before} >= 30`);
    expect(f).toBeTruthy();
  }
  return (await lastFrame(page))!;
};

export const parkPointer = async (page: Page, at: Point) => {
  const box = await canvasBox(page);
  await page.mouse.move(box.x + at.x, box.y + at.y);
  return waitLastFrame(page, '(f) => f.hovered === undefined && f.hoverPath === 0');
};

export const treeSelect = (page: Page) => {
  const box = page.getByRole('combobox', { name: 'Tree set' });
  const list = box.locator('xpath=..').getByRole('listbox');
  const open = async () => {
    if ((await box.getAttribute('aria-expanded')) !== 'true') await box.click();
  };
  return {
    box,
    entries: async () => {
      await open();
      const names = (await list.getByRole('option').allTextContents()).map((s) => s.trim());
      await box.press('Escape');
      return names;
    },
    choose: async (entry: string | number) => {
      await open();
      await (typeof entry === 'number' ? list.getByRole('option').nth(entry) : list.getByRole('option', { name: entry, exact: true })).click();
    }
  };
};
