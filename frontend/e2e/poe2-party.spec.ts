import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';
import { displayStats2 } from '../src/lib/display/stats2';
import { formatNumSep, luaFormat } from '../src/lib/display/format_stat';

test.use({ serviceWorkers: 'block' });

const FIXTURES = resolve(import.meta.dirname, '../../wasm2/exposition/testdata');
const ORACLE = resolve(import.meta.dirname, '../../testdata/oracle');
const encode = (xml: string | Buffer) => deflateSync(xml).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
// the member's export as PoB-PoE2 writes it (oracle/poe2_party_import.lua), and the receiver's calculation with and
// without it (oracle/poe2_party.lua)
const member = (JSON.parse(readFileSync(resolve(ORACLE, 'poe2_party_import.json'), 'utf8')) as { inputs: Record<string, string> }).inputs.giver;
const receiver = (
  JSON.parse(readFileSync(resolve(ORACLE, 'poe2_party_giver.json'), 'utf8')) as {
    receivers: { build: string; without: { player: Record<string, number> }; with: { player: Record<string, number> } }[];
  }
).receivers.find((r) => r.build === 'warrior_dual_mace.xml')!;
const fmt = (stat: string, v: number) => {
  const s = displayStats2.find((d) => d.stat === stat)!;
  return formatNumSep(luaFormat(s.fmt ?? 'd', v * (s.pc || s.mod ? 100 : 1)));
};

const open = async (page: Page, name: string) => {
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
  await page.goto(`/?import=${encodeURIComponent(encode(readFileSync(resolve(FIXTURES, name))))}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  return errors;
};

test("a PoE2 party member's code moves the build's DPS to PoB-PoE2's, and switching the member off moves it back", async ({ page }) => {
  const [before, after] = [fmt('TotalDPS', receiver.without.player.TotalDPS), fmt('TotalDPS', receiver.with.player.TotalDPS)];
  expect(after, 'vacuity: the member moves the shown DPS').not.toBe(before);
  const errors = await open(page, 'warrior_dual_mace.xml');
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(before, { timeout: 60_000 });
  await poe2Nav(page, 'Party').click();
  const imp = page.getByRole('region', { name: 'Party import' });
  await imp.getByRole('textbox', { name: /Enter a build code/ }).fill(encode(member));
  await imp.getByRole('button', { name: 'Import' }).click();
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(after, { timeout: 60_000 });
  await expect(page.getByRole('region', { name: 'Curses' })).toContainText('Despair');
  await page.getByRole('checkbox', { name: 'Warrior active' }).click();
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(before);
  expect(errors).toEqual([]);
});
