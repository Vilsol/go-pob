import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = (path = FIXTURE) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

type Probe = {
  frames: { hovered?: number; alloc: number; renderer: string; overlays: number }[];
  screenOf?: (id: number) => { x: number; y: number } | undefined;
};

const screenOf = async (page: Page, id: number) => {
  const at = await page.evaluate((n) => (globalThis as unknown as { __tree2Probe: Probe }).__tree2Probe.screenOf?.(n), id);
  const box = await page.getByRole('img', { name: 'Passive skill tree' }).boundingBox();
  if (!at || !box) throw new Error(`node ${id} has no screen position`);
  return { x: box.x + at.x, y: box.y + at.y };
};

const lastFrame = (page: Page) => page.evaluate(() => (globalThis as unknown as { __tree2Probe: Probe }).__tree2Probe.frames.at(-1));

// PoB-PoE2's sidebar for the fixture with node 44871 (+10 to maximum Energy Shield) allocated, from
// oracle/poe2_sidebar.lua: Energy Shield 10, Effective Hit Pool 546 (542 without it)
const openPoe2 = async (page: Page, errors: string[], path?: string) => {
  page.on('pageerror', (e) => errors.push(e.message));
  await page.addInitScript(() => {
    localStorage.setItem('options:poe2', 'true');
    const frames: unknown[] = [];
    (globalThis as unknown as { __tree2Probe: unknown }).__tree2Probe = { frames, frame: (f: unknown) => frames.push(f) };
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
  await page.goto(`/?import=${encodeURIComponent(code(path))}`);
};

test('with the PoE2 flag on, clicking a tree node allocates it and recalculates the PoE2 sidebar', async ({ page }) => {
  const errors: string[] = [];
  await openPoe2(page, errors);

  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await expect(poe2Stat(page, 'TotalEHP')).toHaveText('542');
  await expect(poe2Stat(page, 'EnergyShield')).toHaveCount(0);

  await poe2Nav(page, 'Tree').click();
  const canvas = view.getByRole('img', { name: 'Passive skill tree' });
  await expect(canvas).toBeVisible({ timeout: 60_000 });
  await expect.poll(async () => (await lastFrame(page))?.alloc).toBe(1);

  // zoom in about the node; the point under the cursor stays put (PassiveTreeView.lua:1500-1514)
  let at = await screenOf(page, 44871);
  await page.mouse.move(at.x, at.y);
  for (let i = 0; i < 14; i++) await page.mouse.wheel(0, -100);
  await expect
    .poll(
      async () =>
        await page.evaluate(
          () => (globalThis as unknown as { __tree2Probe: { frames: { view: { zoomLevel: number } }[] } }).__tree2Probe.frames.at(-1)?.view.zoomLevel
        )
    )
    .toBe(17);
  at = await screenOf(page, 44871);
  await page.mouse.move(at.x + 1, at.y + 1);
  await page.mouse.move(at.x, at.y);
  const tip = page.getByRole('tooltip', { name: 'Node tooltip' });
  await expect(tip).toContainText('Energy Shield');
  await expect(tip).toContainText('+10 to maximum Energy Shield');
  await expect.poll(async () => (await lastFrame(page))?.hovered).toBe(44871);

  await page.mouse.down();
  await page.mouse.up();
  await expect(poe2Stat(page, 'EnergyShield')).toHaveText('10');
  await expect(poe2Stat(page, 'TotalEHP')).toHaveText('546');
  await expect(view.getByRole('status', { name: 'Points used' })).toContainText('Points 1 / 123');

  // a second click deallocates it
  await page.mouse.down();
  await page.mouse.up();
  await expect(poe2Stat(page, 'TotalEHP')).toHaveText('542');
  await expect(poe2Stat(page, 'EnergyShield')).toHaveCount(0);
  await expect(view.getByRole('status', { name: 'Points used' })).toContainText('Points 0 / 123');
  expect(errors).toEqual([]);
});

// radius_jewel.xml socket 26725 holds a Large jewel (jewelRadiusIndex 3 in PoB-PoE2); PassiveTreeView.lua:1340-1347
// rings an allocated socket's jewel radius, and hovering the socket rings every full-disc radius (1318-1338)
test('a socketed radius jewel draws its radius ring on the PoE2 tree', async ({ page }) => {
  const errors: string[] = [];
  await openPoe2(page, errors, resolve(import.meta.dirname, '../../calculator2/testdata/radius_jewel.xml'));
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
  await expect.poll(async () => (await lastFrame(page))?.overlays).toBe(1);
  const at = await screenOf(page, 26725);
  await page.mouse.move(at.x, at.y);
  await expect.poll(async () => (await lastFrame(page))?.hovered).toBe(26725);
  await expect.poll(async () => (await lastFrame(page))?.overlays).toBe(5);
  expect(errors).toEqual([]);
});

// TreeTab.lua:365-380, PassiveSpecListControl.lua:39-46 and PassiveTreeView.lua:556-564 (PoB-PoE2): the shared tree
// page's tree list and the Shift+right-click node note on a PoE2 build
test('the PoE2 tree page manages trees and keeps a node note', async ({ page }) => {
  const errors: string[] = [];
  await openPoe2(page, errors);
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
  await expect(view.getByRole('combobox', { name: 'Class' })).toHaveValue('Sorceress');

  await page.keyboard.press('Control+m');
  const manager = page.getByRole('dialog', { name: 'Manage passive trees' });
  await manager.getByRole('button', { name: 'New' }).click();
  await manager.getByRole('textbox', { name: 'Name for this passive tree' }).fill('Second');
  await manager.getByRole('button', { name: 'Save' }).click();
  await expect(manager.getByRole('listbox')).toContainText('Second');
  await manager.getByRole('button', { name: 'Done' }).click();

  let at = await screenOf(page, 44871);
  await page.mouse.move(at.x, at.y);
  for (let i = 0; i < 14; i++) await page.mouse.wheel(0, -100);
  at = await screenOf(page, 44871);
  await page.mouse.move(at.x + 1, at.y + 1);
  await page.mouse.move(at.x, at.y);
  await expect.poll(async () => (await lastFrame(page))?.hovered).toBe(44871);
  await page.keyboard.down('Shift');
  await page.mouse.click(at.x, at.y, { button: 'right' });
  await page.keyboard.up('Shift');
  const note = page.getByRole('dialog', { name: 'Note: Energy Shield' });
  await note.getByRole('textbox', { name: 'Node note' }).fill('take this first');
  await note.getByRole('button', { name: 'Save' }).click();
  await expect(note).toBeHidden();
  await page.keyboard.down('Shift');
  await page.mouse.click(at.x, at.y, { button: 'right' });
  await page.keyboard.up('Shift');
  await expect(note.getByRole('textbox', { name: 'Node note' })).toHaveValue('take this first');
  expect(errors).toEqual([]);
});
