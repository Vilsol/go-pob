import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_tree_jewels.xml');
const code = (path: string) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

const open = async (page: Page) => {
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await page.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  await page.goto(`/?import=${encodeURIComponent(code(FIXTURE))}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  return errors;
};

// testdata/tree_jewels_pob.jsonl: PoB-PoE2's main output after each of the same socket edits through its Items tab
const ROWS = readFileSync(resolve(import.meta.dirname, '../../wasm2/exposition/testdata/tree_jewels_pob.jsonl'), 'utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as { step: string; output: Record<string, number> });
// the sidebar's Hit DPS, as stats2's '.1f' writes it
const hitDPS = (step: string) => ROWS.find((r) => r.step === step)!.output.TotalDPS.toFixed(1);

// ItemSlotControl.lua:117-131 (PoB-PoE2): each tree socket's list holds the build's jewels, whose bases are Diamond,
// Sapphire, Ruby and Emerald rather than "… Jewel"; picking one equips it and the build recalculates
test('the PoE2 Items page shows and swaps the jewels in tree sockets', async ({ page }) => {
  const errors = await open(page);
  await poe2Nav(page, 'Items').click();
  const jewels = page.getByRole('region', { name: 'Tree jewels' });
  const names = ['Prism of Belief', 'Doom Ornament', 'Split Personality', 'Heart of the Well', 'Gale Stone', 'Rune Spark'];
  await expect(jewels.getByRole('combobox')).toHaveText(names);
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('load'));

  await page.locator('[data-socket-node="21984"]').hover();
  await expect(page.getByRole('tooltip')).toContainText('20% increased Critical Damage Bonus');

  const socket2 = jewels.getByRole('combobox', { name: 'Socket #2' });
  const list = page.getByRole('listbox', { name: 'Socket #2' });
  await socket2.click();
  await expect(list.getByRole('option')).toHaveText(['None', ...names]);
  await list.getByRole('option', { name: 'None' }).click();
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('empty socket 21984'));
  await socket2.click();
  await list.getByRole('option', { name: 'Rune Spark' }).click();
  await expect(socket2).toHaveText('Rune Spark');
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('Rune Spark in socket 21984'));
  expect(errors).toEqual([]);
});
