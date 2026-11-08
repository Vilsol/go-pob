import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = deflateSync(readFileSync(FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
// longer than the history joins two edits of one kind into one step
const APART_MS = 1_200;

type Probe = { frames: { alloc: number; view: { zoomLevel: number } }[]; screenOf?: (id: number) => { x: number; y: number } | undefined };
const probe = (page: Page) => page.evaluate(() => (globalThis as unknown as { __tree2Probe: Probe }).__tree2Probe.frames.at(-1));

const screenOf = async (page: Page, id: number) => {
  const at = await page.evaluate((n) => (globalThis as unknown as { __tree2Probe: Probe }).__tree2Probe.screenOf?.(n), id);
  const box = await page.getByRole('img', { name: 'Passive skill tree' }).boundingBox();
  if (!at || !box) throw new Error(`node ${id} has no screen position`);
  return { x: box.x + at.x, y: box.y + at.y };
};

// node 44871 is +10 to maximum Energy Shield (PoB-PoE2's sidebar for the fixture: Energy Shield 10 with it, none without)
test('Ctrl+Z on the PoE2 tree page undoes exactly one step of the whole-build history, which the top bar names', async ({ page }) => {
  const errors: string[] = [];
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
  await page.goto(`/?import=${encodeURIComponent(code)}`);

  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  await expect(page.getByRole('button', { name: 'Nothing to undo' })).toBeDisabled();
  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
  await expect.poll(async () => (await probe(page))?.alloc).toBe(1);
  let at = await screenOf(page, 44871);
  await page.mouse.move(at.x, at.y);
  for (let i = 0; i < 14; i++) await page.mouse.wheel(0, -100);
  await expect.poll(async () => (await probe(page))?.view.zoomLevel).toBe(17);
  at = await screenOf(page, 44871);
  await page.mouse.click(at.x, at.y);
  await expect(poe2Stat(page, 'EnergyShield')).toHaveText('10');
  await page.waitForTimeout(APART_MS);
  await page.mouse.click(at.x, at.y);
  await expect(poe2Stat(page, 'EnergyShield')).toHaveCount(0);
  const deallocated = page.getByRole('button', { name: /^Undo: Deallocated .+ \(Ctrl\+Z\)$/ });
  await expect(deallocated).toBeEnabled();

  await page.keyboard.press('Control+z');
  await expect(poe2Stat(page, 'EnergyShield')).toHaveText('10');
  await expect(page.getByRole('button', { name: /^Undo: Allocated .+ \(Ctrl\+Z\)$/ })).toBeEnabled();
  await expect(page.getByRole('button', { name: /^Redo: Deallocated .+ \(Ctrl\+Y \/ Ctrl\+Shift\+Z\)$/ })).toBeEnabled();

  await page.keyboard.press('Control+z');
  await expect(poe2Stat(page, 'EnergyShield')).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Nothing to undo' })).toBeDisabled();

  await page.keyboard.press('Control+y');
  await expect(poe2Stat(page, 'EnergyShield')).toHaveText('10');

  // the history menu jumps to a step, and names each step's stat changes on hover
  await page.getByRole('button', { name: 'Undo history (right-click or long-press Undo)' }).click();
  const menu = page.getByRole('menu', { name: 'Undo history' });
  await expect(menu.getByRole('menuitem')).toHaveCount(2);
  const top = menu.getByRole('menuitem').first();
  await top.hover();
  await expect(page.getByRole('tooltip', { name: 'Stat change' })).toContainText('Energy Shield');
  await top.click();
  await expect(poe2Stat(page, 'EnergyShield')).toHaveCount(0);
  expect(errors).toEqual([]);
});
