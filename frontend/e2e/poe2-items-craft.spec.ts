import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync, gunzipSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const ITEMS_DIR = resolve(import.meta.dirname, '../../calculator2/testdata');
const code = (path: string) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');
const PASTES = readFileSync(resolve(ITEMS_DIR, 'items_edit_pastes.txt'), 'utf8').replace(/\n$/, '').split('\n=====\n');
// calculator2/testdata/item_compare_pob.jsonl.gz: PoB-PoE2's item tooltips over items_edit.xml, with each calculation
type CompareRow = { base?: Record<string, number>; item?: string; lines?: string[]; calcs?: { slot: string; output: Record<string, number> }[] };
const COMPARE = gunzipSync(readFileSync(resolve(ITEMS_DIR, 'item_compare_pob.jsonl.gz')))
  .toString('utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as CompareRow);
// calculator2/testdata/item_affix_pob.jsonl.gz: PoB-PoE2's affix drop-downs, step by step over crafted items
type AffixRow = { case: string; step: string; raw: string; controls: { list: { label: string }[] }[] };
const AFFIX = gunzipSync(readFileSync(resolve(ITEMS_DIR, 'item_affix_pob.jsonl.gz')))
  .toString('utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as AffixRow);
// calculator2/testdata/item_range_pob.jsonl: PoB-PoE2's modifier range sliders over item_affix_pastes.txt and uniques
type RangeRow = { case: string; step: string; raw: string };
const RANGES = readFileSync(resolve(ITEMS_DIR, 'item_range_pob.jsonl'), 'utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as RangeRow);
const AFFIX_PASTES = readFileSync(resolve(ITEMS_DIR, 'item_affix_pastes.txt'), 'utf8').replace(/\n$/, '').split('\n=====\n');
const plain = (s: string) => s.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '');
const tooltipLine = (item: string, text: RegExp) => plain(COMPARE.find((r) => r.item === item)!.lines!.find((l) => text.test(plain(l)))!);

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
  await page.goto(`/?import=${encodeURIComponent(code(resolve(ITEMS_DIR, 'items_edit.xml')))}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  await poe2Nav(page, 'Items').click();
  return errors;
};

test("the PoE2 Items page shows PoB-PoE2's equip impact, browses and equips uniques, and crafts an item from a base", async ({ page }) => {
  test.setTimeout(300_000);
  const errors = await open(page);

  // ItemsTab.lua:4425-4538 (PoB-PoE2): the pasted helmet's Helmet block
  await page.getByRole('textbox', { name: 'Item text' }).fill(PASTES[0]);
  const impact = page.getByRole('region', { name: 'Equip impact' }).or(page.locator('section[aria-label="Equip impact"]'));
  await expect(impact.locator('[data-block-header]')).toHaveText(['Equipping this item in Helmet will give you:']);
  // "+42 Total Life (+6.2%)": the line shows its label beside its value
  const [, value, percent] = /^(\S+) Total Life( \(.+\))?$/.exec(tooltipLine('paste helmet', /Total Life/))!;
  await expect(impact.locator('[data-line]').filter({ hasText: /^Total Life/ })).toContainText(value + (percent ?? ''));
  await page.getByRole('button', { name: 'Equip in Helmet' }).click();
  await expect(page.getByRole('button', { name: 'Helmet: Grim Visage' })).toBeVisible();

  // ItemDBControl.lua (PoB-PoE2): the unique database, its tooltip's stat differences and Ctrl+click
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  await page.getByRole('searchbox', { name: 'Search uniques' }).fill('headhunter');
  const list = page.getByRole('table', { name: 'Unique list' });
  await expect(list.getByRole('row').locator('button[data-pob-color]')).toHaveText(['Headhunter']);
  const life = Number(await poe2Stat(page, 'Life').textContent());
  await list.getByRole('button', { name: 'Equip Headhunter, Heavy Belt' }).click();
  const hh = COMPARE.find((r) => r.item === 'Headhunter')!;
  const gain = hh.calcs!.at(-1)!.output.Life - COMPARE[0].base!.Life;
  await expect(poe2Stat(page, 'Life')).toHaveText(String(life + gain));
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Equipped' }).click();
  await expect(page.getByRole('button', { name: 'Belt: Headhunter' })).toBeVisible();

  // ItemsTab.lua:2762-2868 (PoB-PoE2): Craft Item makes a rare from a base and opens it in the editor
  await page.getByRole('button', { name: 'Craft item…' }).click();
  const craft = page.getByRole('dialog', { name: 'Craft Item' });
  await craft.getByRole('textbox', { name: 'Name' }).fill('Doom Grip');
  const pick = async (name: string, option: string) => {
    const box = craft.getByRole('combobox', { name });
    await box.click();
    await page
      .locator(`[id="${await box.getAttribute('aria-controls')}"]`)
      .getByRole('option', { name: option, exact: true })
      .click();
    await expect(box).toHaveText(option);
  };
  await pick('Type', 'Gloves: Armour/Energy Shield');
  await pick('Base', 'Adherent Cuffs');
  await craft.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('article', { name: 'Item: Doom Grip' })).toContainText('Rare Adherent Cuffs');
  await page.getByRole('button', { name: 'Edit as text' }).click();
  await expect(page.getByRole('textbox', { name: 'Item text' })).toHaveValue(/^Rarity: RARE\nDoom Grip\nAdherent Cuffs\n/);
  await expect(page.locator('section[aria-label="Equip impact"] [data-block-header]')).toHaveText([/Equipping this item in Gloves/]);
  expect(errors).toEqual([]);
});

test("the PoE2 item editor picks a crafted item's affix and its roll as PoB-PoE2 does", async ({ page }) => {
  test.setTimeout(300_000);
  const errors = await open(page);

  // ItemsTab.lua:2762-2868 (PoB-PoE2): a rare Ruby Ring opens in the editor with three prefix and three suffix slots
  await page.getByRole('button', { name: 'Craft item…' }).click();
  const craft = page.getByRole('dialog', { name: 'Craft Item' });
  const pick = async (name: string, option: string) => {
    const box = craft.getByRole('combobox', { name });
    await box.click();
    await page
      .locator(`[id="${await box.getAttribute('aria-controls')}"]`)
      .getByRole('option', { name: option, exact: true })
      .click();
    await expect(box).toHaveText(option);
  };
  await pick('Type', 'Ring');
  await pick('Base', 'Ruby Ring');
  await craft.getByRole('button', { name: 'Create' }).click();
  const prefixes = page.getByRole('group', { name: 'Prefixes' });
  await expect(prefixes.getByRole('group', { name: /^Prefix \d$/ })).toHaveCount(3);
  await expect(page.getByRole('group', { name: 'Suffixes' }).getByRole('group', { name: /^Suffix \d$/ })).toHaveCount(3);

  // ItemsTab.lua:932-947, 1067-1075 (PoB-PoE2): the first prefix's second entry, then its roll slider at 0.7
  const crafted = AFFIX.find((r) => r.case === '3 Ruby Ring' && r.step === 'craft')!;
  const slid = AFFIX.find((r) => r.case === '3 Ruby Ring' && r.step === 'slide 1 0.7')!;
  const affix = page.getByRole('combobox', { name: 'Affix 1' });
  await affix.click();
  await page
    .locator(`[id="${await affix.getAttribute('aria-controls')}"]`)
    .getByRole('option', { name: plain(crafted.controls[0].list[1].label), exact: true })
    .click();
  const roll = page.getByRole('slider', { name: 'Roll of affix 1' });
  await expect(roll).toBeVisible();
  await roll.fill('0.7');
  await page.getByRole('button', { name: 'Edit as text' }).click();
  await expect(page.getByRole('textbox', { name: 'Item text' })).toHaveValue(slid.raw);

  // Main.lua:171-189 (PoB-PoE2): the rare database lists the crafted templates
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  await page.getByRole('tablist', { name: 'Item database' }).getByRole('tab', { name: 'Templates' }).click();
  await page.getByRole('searchbox', { name: 'Search templates' }).fill('critical quiver');
  await expect(page.getByRole('table', { name: 'Template list' }).getByRole('row').locator('button[data-pob-color]')).toHaveText([
    '+2 Critical Quiver, Visceral Quiver'
  ]);
  expect(errors).toEqual([]);
});

test("the PoE2 item editor moves a pasted item's modifier range as PoB-PoE2 does", async ({ page }) => {
  test.setTimeout(300_000);
  const errors = await open(page);

  // ItemsTab.lua:1147-1166, 2564-2577 (PoB-PoE2): the pasted ring's range line at 0.37
  await page.getByRole('textbox', { name: 'Item text' }).fill(AFFIX_PASTES[0]);
  const roll = page.getByRole('slider', { name: 'Roll of range line 1' });
  await expect(roll).toBeVisible();
  await roll.fill('0.37');
  const want = RANGES.find((r) => r.case === 'paste 1' && r.step === 'range 1 0.37')!;
  await expect(page.getByRole('textbox', { name: 'Item text' })).toHaveValue(want.raw);
  expect(errors).toEqual([]);
});
