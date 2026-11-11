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
  frames: { hovered?: number; alloc: number; renderer: string; overlays: number; heat?: number }[];
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
  const tip = view.getByRole('tooltip');
  await expect(tip).toContainText('Energy Shield');
  await expect(tip).toContainText('+10 to maximum Energy Shield');
  await expect.poll(async () => (await lastFrame(page))?.hovered).toBe(44871);
  // PassiveTreeView.lua:1966-2025 (PoB-PoE2): the stat differences and the pathing line; Ctrl+D hides the differences
  const diffs = tip.getByRole('group', { name: 'Stat differences' });
  await expect(diffs).toContainText('Allocating this node will give you:');
  await expect(diffs).toContainText('Energy Shield');
  await expect(tip.getByLabel('Node tips')).toContainText('1 points to node');
  await page.keyboard.press('Control+d');
  await expect(diffs).toContainText('Tip: Press Ctrl+D to enable the display of stat differences.');
  await page.keyboard.press('Control+d');
  await expect(diffs).toContainText('Allocating this node will give you:');
  // PassiveTreeView.lua:276-350, 2014-2016: with Shift held the node starts a trace
  await page.keyboard.down('Shift');
  await expect(tip.getByLabel('Node tips')).toContainText('1 nodes in trace path');
  await page.keyboard.up('Shift');
  await expect(tip.getByLabel('Node tips')).toContainText('1 points to node');

  await page.mouse.down();
  await page.mouse.up();
  await expect(poe2Stat(page, 'EnergyShield')).toHaveText('10');
  await expect(poe2Stat(page, 'TotalEHP')).toHaveText('546');
  await expect(view.getByRole('status', { name: 'Points used' })).toContainText('Points 1 / 123');
  // PassiveTreeView.lua:2026-2057 (PoB-PoE2): unallocating it costs data.goldRespecPrices[40], 736 Gold
  await page.mouse.move(at.x + 1, at.y + 1);
  await page.mouse.move(at.x, at.y);
  await expect(tip.getByLabel('Node tips')).toContainText('736 Gold required to unallocate this node');

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
  // PassiveTreeView.lua:565-572, 1704-1727 (PoB-PoE2): the socket's tooltip is its jewel's, and a right-click opens the
  // Items page on its slot
  const tip = view.getByRole('tooltip');
  await expect(tip.getByLabel('Socketed jewel')).toContainText('Time-Lost Ruby');
  await expect(tip.getByRole('group', { name: 'Stat differences' })).toHaveCount(0);
  await expect(tip.getByLabel('Node tips')).toContainText('Tip: Right click this socket');
  await page.mouse.click(at.x, at.y, { button: 'right' });
  await expect(page).toHaveURL(/\/items\?socket=26725/);
  await expect(page.locator('[data-socket-node="26725"] [role="combobox"]')).toBeFocused();
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

// PassiveTreeView.lua:200-211, 251-255 and ItemTools.lua:480-485 (PoB-PoE2): Ctrl+C copies the hovered node's name and
// stats, F1 opens its poe2wiki.net page, and a Ctrl+click zooms two steps in (out with the right button) without
// allocating; node 44871 is TreeData/0_5/tree.lua's "Energy Shield", "+10 to maximum Energy Shield"
test('the PoE2 tree copies a node, opens its wiki page and zooms with Ctrl+click', async ({ page }) => {
  const errors: string[] = [];
  await openPoe2(page, errors);
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
  await page.evaluate(() => {
    const seen = ((globalThis as unknown as { __seen: string[] }).__seen = [] as string[]);
    Object.defineProperty(navigator, 'clipboard', { value: { writeText: (t: string) => (seen.push(`copy:${t}`), Promise.resolve()) }, configurable: true });
    window.open = (url?: string | URL) => (seen.push(`open:${String(url)}`), null);
  });
  const seen = () => page.evaluate(() => (globalThis as unknown as { __seen: string[] }).__seen);
  const zoomLevel = () =>
    page.evaluate(() => (globalThis as unknown as { __tree2Probe: { frames: { view: { zoomLevel: number } }[] } }).__tree2Probe.frames.at(-1)?.view.zoomLevel);

  let at = await screenOf(page, 44871);
  await page.mouse.move(at.x, at.y);
  for (let i = 0; i < 10; i++) await page.mouse.wheel(0, -100);
  await expect.poll(zoomLevel).toBe(13);
  at = await screenOf(page, 44871);
  await page.mouse.move(at.x + 1, at.y + 1);
  await page.mouse.move(at.x, at.y);
  await expect.poll(async () => (await lastFrame(page))?.hovered).toBe(44871);
  await page.keyboard.press('Control+c');
  await page.keyboard.press('F1');
  await expect.poll(seen).toEqual(['copy:# Energy Shield\n+10 to maximum Energy Shield\n', 'open:https://www.poe2wiki.net/wiki/Energy_Shield']);

  const alloc = (await lastFrame(page))?.alloc;
  await page.keyboard.down('Control');
  await page.mouse.click(at.x, at.y);
  await expect.poll(zoomLevel).toBe(15);
  await page.mouse.click(at.x, at.y, { button: 'right' });
  await expect.poll(zoomLevel).toBe(13);
  await page.keyboard.up('Control');
  expect((await lastFrame(page))?.alloc).toBe(alloc);
  await expect(poe2Stat(page, 'EnergyShield')).toHaveCount(0);
  expect(errors).toEqual([]);
});

// TreeTab.lua:695-849 (PoB-PoE2): Export Tree gives PoB-PoE2's link for the active tree, Import Tree refuses what it
// refuses with its message and adds a tree from a link; the links and messages are PoB-PoE2's, from
// wasm2/exposition/testdata/tree_specs_oracle.lua
test('the PoE2 tree page exports and imports tree links as PoB-PoE2 does', async ({ page }) => {
  const rows = readFileSync(resolve(import.meta.dirname, '../../wasm2/exposition/testdata/tree_specs_links_pob.jsonl'), 'utf8')
    .trim()
    .split('\n')
    .map((l) => JSON.parse(l) as { step: string; link?: string; msg?: string; labels: string[] });
  const exported = rows.find((r) => r.step === 'export')!;
  const bad = rows.find((r) => r.step === 'import Bad link')!;
  const imported = rows.find((r) => r.step === 'import Radius link')!;
  const errors: string[] = [];
  await openPoe2(page, errors, resolve(import.meta.dirname, '../../wasm2/exposition/testdata/tree_specs.xml'));
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });

  await page.keyboard.press('Control+m');
  const dialog = page.getByRole('dialog', { name: 'Manage passive trees' });
  await dialog.getByRole('button', { name: 'Export Tree' }).click();
  await expect(dialog.getByRole('textbox', { name: 'Passive tree link' })).toHaveValue(exported.link!);
  await dialog.getByRole('group', { name: 'Export Tree' }).getByRole('button', { name: 'Done' }).click();

  await dialog.getByRole('button', { name: 'Import Tree' }).click();
  const form = dialog.getByRole('form', { name: 'Import Tree' });
  await form.getByRole('textbox', { name: 'Enter name for this passive tree' }).fill('Radius link');
  await form.getByRole('textbox', { name: 'Enter passive tree link' }).fill(bad.link!);
  await form.getByRole('button', { name: 'Import' }).click();
  await expect(form.getByRole('alert')).toHaveText(bad.msg!);
  await expect(dialog.getByRole('listbox', { name: 'Passive trees' }).getByRole('option')).toHaveCount(2);

  await form.getByRole('textbox', { name: 'Enter passive tree link' }).fill(imported.link!);
  await form.getByRole('button', { name: 'Import' }).click();
  await expect(form).toBeHidden();
  await expect(dialog.getByRole('listbox', { name: 'Passive trees' }).getByRole('option')).toHaveCount(imported.labels.length);
  await expect(dialog.getByRole('option', { name: /^Radius link/ })).toContainText('current');
  expect(errors).toEqual([]);
});

// TreeTab.lua:197-283, 932-1048 and PassiveTreeView.lua:1003-1030 (PoB-PoE2): the power report lists the nodes' power for
// the chosen stat and depth, and the tree draws each node's power
test('the PoE2 power report lists node power and the tree draws the heat map', async ({ page }) => {
  const errors: string[] = [];
  await openPoe2(page, errors);
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
  await view.getByRole('button', { name: /^Power/ }).click();
  const report = view.getByRole('region', { name: 'Power report' });
  await report.getByRole('combobox', { name: 'Power report depth' }).selectOption('5');
  // Offence/Defence, the first stat, has no report of its own (PowerReportListControl.lua:47-51)
  await expect(report.getByRole('combobox', { name: 'Power stat' })).toHaveValue('');
  await report.getByRole('combobox', { name: 'Power stat' }).selectOption({ label: 'Energy Shield' });
  await report.getByRole('button', { name: 'Start' }).click();
  const rows = report.getByRole('table', { name: 'Node power' }).getByRole('row');
  await expect.poll(async () => rows.count(), { timeout: 120_000 }).toBeGreaterThan(1);
  await expect(rows.nth(1).getByRole('cell').nth(2)).toHaveText(/^\d/);
  await expect.poll(async () => (await lastFrame(page))?.heat ?? 0).toBeGreaterThan(0);
  // PowerReportListControl.lua (PoB-PoE2) has no Show Masteries box
  await expect(report.getByRole('checkbox', { name: 'Show Masteries' })).toHaveCount(0);
  // a picked row centres its node; hovering the row rings it on the tree, as on the PoE1 tree
  const first = report.locator('[data-power-node]').first();
  const id = await first.getAttribute('data-power-node');
  await first.click();
  await first.hover();
  const ring = page.locator(`[data-power-hover="${id}"]`);
  await expect(ring).toBeVisible();
  const box = (await view.getByRole('img', { name: 'Passive skill tree' }).boundingBox())!;
  await expect
    .poll(async () => {
      const r = (await ring.boundingBox())!;
      return Math.max(Math.abs(r.x + r.width / 2 - (box.x + box.width / 2)), Math.abs(r.y + r.height / 2 - (box.y + box.height / 2)));
    })
    .toBeLessThan(2);
  await page.mouse.move(box.x + 5, box.y + box.height - 5);
  await expect(page.locator('[data-power-hover]')).toHaveCount(0);
  // switching the heat map off stops drawing it
  await page.keyboard.press('p');
  await expect.poll(async () => (await lastFrame(page))?.heat).toBeUndefined();
  expect(errors).toEqual([]);
});
