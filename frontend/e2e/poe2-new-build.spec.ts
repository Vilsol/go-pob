// Build.lua:34-55, TreeTab.lua:47-49, PassiveSpec.lua:33-43 (PoB-PoE2): a new build is a Ranger on the latest tree

import { test, expect, type Page } from '@playwright/test';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

type Probe = { frames: { hovered?: number; alloc: number }[]; screenOf?: (id: number) => { x: number; y: number } | undefined };

const screenOf = async (page: Page, id: number) => {
  const at = await page.evaluate((n) => (globalThis as unknown as { __tree2Probe: Probe }).__tree2Probe.screenOf?.(n), id);
  const box = await page.getByRole('img', { name: 'Passive skill tree' }).boundingBox();
  if (!at || !box) throw new Error(`node ${id} has no screen position`);
  return { x: box.x + at.x, y: box.y + at.y };
};
const lastFrame = (page: Page) => page.evaluate(() => (globalThis as unknown as { __tree2Probe: Probe }).__tree2Probe.frames.at(-1));

// PoB-PoE2's values: a new build has 65 Life; as a Witch with node 44871 (+10 to maximum Energy Shield) it has 10
// Energy Shield and its level follows the one point spent to 2
test('a new PoE2 build opens in the PoE2 engine, takes a class and a node, saves and reopens as PoE2', async ({ page }) => {
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
  await page.goto('/');

  await page.getByRole('button', { name: 'New build for another game' }).click({ timeout: 180_000 });
  await page.getByRole('menu', { name: 'New build for' }).getByRole('menuitem', { name: 'Path of Exile 2' }).click();
  const view = poe2Page(page);
  await expect(poe2Stat(page, 'Life')).toHaveText('65', { timeout: 180_000 });

  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('img', { name: 'Passive skill tree' })).toBeVisible({ timeout: 60_000 });
  const cls = view.getByRole('combobox', { name: 'Class' });
  await expect(cls).toHaveValue('Ranger');
  await cls.selectOption('Witch');
  await expect(cls).toHaveValue('Witch');
  await expect.poll(async () => (await lastFrame(page))?.alloc).toBe(1);

  let at = await screenOf(page, 44871);
  await page.mouse.move(at.x, at.y);
  for (let i = 0; i < 14; i++) await page.mouse.wheel(0, -100);
  at = await screenOf(page, 44871);
  await page.mouse.move(at.x + 1, at.y + 1);
  await page.mouse.move(at.x, at.y);
  await expect.poll(async () => (await lastFrame(page))?.hovered).toBe(44871);
  await page.mouse.down();
  await page.mouse.up();
  await expect(poe2Stat(page, 'EnergyShield')).toHaveText('10');

  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog', { name: 'Save build as' });
  await dialog.getByRole('textbox', { name: 'Build name' }).fill('Fresh witch');
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await page.getByRole('link', { name: 'Back to builds' }).click();

  const link = page.getByRole('link', { name: 'Fresh witch', exact: true });
  const row = page.getByRole('row').filter({ has: link });
  await expect(row.locator('[data-game="poe2"]')).toHaveText('PoE2');
  await expect(row.locator('[data-col="cls"]')).toHaveText('Witch');
  await expect(row.locator('[data-col="level"]')).toHaveText('2');

  await link.click();
  await expect(poe2Stat(page, 'EnergyShield')).toHaveText('10', { timeout: 60_000 });
  await poe2Nav(page, 'Tree').click();
  await expect(view.getByRole('combobox', { name: 'Class' })).toHaveValue('Witch');
  expect(errors).toEqual([]);
});
