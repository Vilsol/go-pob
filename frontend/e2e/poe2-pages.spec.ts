import { test, expect, type Page } from '@playwright/test';
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { PHONE, bar } from './support/phone';
import { poe2Nav, poe2Page, poe2Stat } from './support/poe2';

test.use({ serviceWorkers: 'block' });

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const code = (path: string) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

const open = async (page: Page, fixture = FIXTURE) => {
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
  await page.goto(`/?import=${encodeURIComponent(code(fixture))}`);
  const view = poe2Page(page);
  await expect(view).toBeVisible({ timeout: 180_000 });
  return { view: page, errors };
};

// PoB-PoE2's Calcs tab for the fixture: testdata/oracle/poe2_calcs_sorceress_spark.json (Life:0:0:0, Life:0:2:0;
// EnergyShield:0:8:0, a row a true output gates)
test('the PoE2 Calcs page shows PoB-PoE2 values', async ({ page }) => {
  const { view, errors } = await open(page);
  await poe2Nav(page, 'Calcs').click();
  const life = view.getByRole('region', { name: 'Calculations' }).getByRole('region', { name: 'Life', exact: true });
  await expect(life.locator('[data-key]').filter({ hasText: 'Base from Gear' })).toContainText('90');
  await expect(life.locator('[data-key]').filter({ hasText: 'Total Base' })).toContainText('650');
  const es = view.getByRole('region', { name: 'Calculations' }).getByRole('region', { name: 'Energy Shield', exact: true });
  await expect(es.locator('[data-key]').filter({ hasText: 'Recharge Delay' })).toContainText('4s');
  expect(errors).toEqual([]);
});

// a PoB colour code drawn as text
const COLOR_CODE = /\^(x[0-9A-Fa-f]{6}|\d)/;

const ITEMS_FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_items.xml');

// PoB-PoE2's Skills and Items tabs for the fixture: testdata/oracle/poe2_pages_sorceress_items.json
test('the PoE2 Skills and Items pages show PoB-PoE2 groups and items, and the main group recalculates', async ({ page }) => {
  const { view, errors } = await open(page, ITEMS_FIXTURE);
  await poe2Nav(page, 'Skills').click();
  // the Skills page PoE1 builds show
  const groups = page.locator('section[aria-label="Socket groups"] button[aria-current]');
  await expect(groups).toHaveCount(2);
  await expect(groups.nth(0)).toContainText('Spark');
  await expect(groups.nth(0)).toContainText('MAIN');
  await expect(groups.nth(1)).toContainText('Backup');
  await expect(page.getByRole('checkbox', { name: 'Rapid Casting I enabled' }).first()).not.toBeChecked();
  await groups.nth(1).click();
  await page.getByRole('button', { name: 'Set as main' }).click();
  await expect(groups.nth(1)).toContainText('MAIN');
  // the frame sidebar's picker counts groups from 0
  await expect(page.getByRole('complementary', { name: 'Main skill and stats' }).getByRole('combobox', { name: 'Main skill' })).toHaveValue('1');

  await poe2Nav(page, 'Items').click();
  // the Items page PoE1 builds show, on weapon set II
  await expect(view.getByRole('group', { name: 'Weapon set' }).getByRole('button', { name: 'II' })).toHaveAttribute('aria-pressed', 'true');
  const weapon = view.getByRole('button', { name: 'Weapon 1: Glyph Pillar' });
  await expect(weapon).toBeVisible();
  // a {range:0.25}(40-60)% line, as the tooltip shows it
  await weapon.hover();
  await expect(page.getByRole('tooltip')).toContainText('45% increased Spell Damage');
  const body = view.getByRole('button', { name: 'Body Armour: Storm Shell' });
  await body.hover();
  await expect(page.getByRole('tooltip')).toContainText('Bonded: +20 to maximum Life');
  const ring = view.getByRole('button', { name: 'Ring 1: Gloom Loop' });
  await ring.hover();
  await expect(page.getByRole('tooltip')).toContainText('Corrupted');
  // ItemsTab.lua:3721-3727 (PoB-PoE2): the title's rarity colour code is drawn as colour
  await expect(page.getByRole('tooltip')).toContainText('Gloom Loop');
  await expect(page.getByRole('tooltip')).not.toContainText(COLOR_CODE, { useInnerText: true });
  await expect(page.locator('body')).not.toContainText(COLOR_CODE, { useInnerText: true });
  expect(errors).toEqual([]);
});

// calculator2/testdata/skills_edit_pob.jsonl: PoB-PoE2's main output after each of the same edits through its Skills tab,
// for the same build (the fixture is sorceress_spark.xml)
const SKILLS_EDIT = readFileSync(resolve(import.meta.dirname, '../../calculator2/testdata/skills_edit_pob.jsonl'), 'utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as { step: string; output: Record<string, number> });
// the sidebar's Hit DPS, as stats2's '.1f' writes it
const hitDPS = (step: string) => SKILLS_EDIT.find((r) => r.step === step)!.output.TotalDPS.toFixed(1);

test('the PoE2 Skills page adds a group with a skill and a support, edits them, and the build saves and reopens with them', async ({ page }) => {
  const { errors } = await open(page);
  await poe2Nav(page, 'Skills').click();
  const groups = page.locator('section[aria-label="Socket groups"] button[aria-current]');
  await expect(groups).toHaveCount(1);
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('load'));

  await page.getByRole('button', { name: '+ New group' }).click();
  await expect(groups).toHaveCount(2);
  await expect(groups.nth(1)).toHaveAttribute('aria-current', 'true');
  await page.getByRole('button', { name: 'Set as main' }).click();
  await expect(groups.nth(1)).toContainText('MAIN');
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('main group 2'));

  // GemSelectControl.lua (PoB-PoE2): typing a name and Enter picks the first match
  const box = page.getByRole('combobox', { name: 'Add a gem' });
  for (const [name, step] of [
    ['Fireball', 'add Fireball'],
    ['Elemental Focus', 'add Elemental Focus']
  ]) {
    await box.click();
    await box.fill(name);
    await expect(page.getByRole('listbox', { name: 'Gem results' }).getByRole('option').first()).toContainText(name);
    await box.press('Enter');
    await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS(step));
  }
  const editor = page.getByRole('region', { name: 'Selected group' });
  for (const [label, value, step] of [
    ['Fireball level', '15', 'Fireball level 15'],
    ['Fireball quality', '20', 'Fireball quality 20']
  ]) {
    const spin = editor.getByRole('spinbutton', { name: label });
    await spin.fill(value);
    await spin.press('Enter');
    await expect(spin).toHaveValue(value);
    await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS(step));
  }
  const support = editor.getByRole('checkbox', { name: 'Elemental Focus enabled' }).first();
  await support.click();
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('disable Elemental Focus'));
  await support.click();
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('enable Elemental Focus'));

  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog', { name: 'Save build as' });
  await dialog.getByRole('textbox', { name: 'Build name' }).fill('Fire group');
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await page.getByRole('link', { name: 'Back to builds' }).click();
  await page.getByRole('link', { name: 'Fire group', exact: true }).click();
  // the build reopens on the page it was left on, with the edits
  await expect(groups).toHaveCount(2, { timeout: 60_000 });
  await expect(groups.nth(1)).toContainText('Fireball');
  await expect(groups.nth(1)).toContainText('MAIN');
  await expect(poe2Stat(page, 'TotalDPS')).toHaveText(hitDPS('enable Elemental Focus'));
  await groups.nth(1).click();
  await expect(editor.getByRole('spinbutton', { name: 'Fireball level' })).toHaveValue('15');
  await expect(editor.getByRole('spinbutton', { name: 'Fireball quality' })).toHaveValue('20');
  expect(errors).toEqual([]);
});

// calculator2/testdata/items_edit_pob.jsonl: PoB-PoE2's main output after each of the same edits through its Items tab,
// for the same build
const ITEMS_DIR = resolve(import.meta.dirname, '../../calculator2/testdata');
const ITEMS_EDIT = readFileSync(resolve(ITEMS_DIR, 'items_edit_pob.jsonl'), 'utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as { step: string; output: Record<string, number> });
const PASTES = readFileSync(resolve(ITEMS_DIR, 'items_edit_pastes.txt'), 'utf8').replace(/\n$/, '').split('\n=====\n');
const itemsStat = (step: string, stat: string) => String(ITEMS_EDIT.find((r) => r.step === step)!.output[stat]);

test('the PoE2 Items page pastes and equips items, swaps weapon sets, sockets runes and switches item sets, and the build saves and reopens with them', async ({
  page
}) => {
  const { view, errors } = await open(page, resolve(ITEMS_DIR, 'items_edit.xml'));
  await poe2Nav(page, 'Items').click();
  const expectStats = async (step: string) => {
    await expect(poe2Stat(page, 'Life')).toHaveText(itemsStat(step, 'Life'));
    await expect(poe2Stat(page, 'Mana')).toHaveText(itemsStat(step, 'Mana'));
  };
  await expectStats('load');
  for (const [text, step] of [
    [PASTES[0], 'paste helmet'],
    [PASTES[1], 'paste wand']
  ]) {
    await view.getByRole('textbox', { name: 'Item text' }).fill(text);
    await view.getByRole('button', { name: 'Save to build' }).click();
    await expectStats(step);
  }
  await expect(view.getByRole('button', { name: 'Helmet: Grim Visage' })).toBeVisible();
  await expect(view.getByRole('button', { name: 'Weapon 1: Doom Song' })).toBeVisible();

  const weaponSet = view.getByRole('group', { name: 'Weapon set' });
  await weaponSet.getByRole('button', { name: 'II' }).click();
  await expectStats('weapon set II');
  await expect(view.getByRole('button', { name: 'Weapon 1: Empty' })).toBeVisible();
  await weaponSet.getByRole('button', { name: 'I', exact: true }).click();
  await expectStats('weapon set I');

  // ItemListControl.lua:321-328, ItemsTab.lua:613-624, 792-800 (PoB-PoE2): two runes in the body armour
  await view.getByRole('button', { name: 'Body Armour: Storm Shell' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Edit as text' }).click();
  const sockets = view.getByRole('spinbutton', { name: 'Sockets' });
  await sockets.fill('2');
  await sockets.press('Enter');
  await view.getByRole('combobox', { name: 'Rune #1' }).selectOption('Desert Rune');
  await expect(view.getByRole('combobox', { name: 'Rune #1' })).toHaveValue('Desert Rune');
  await view.getByRole('combobox', { name: 'Rune #2' }).selectOption('Glacial Rune');
  await expect(view.getByRole('combobox', { name: 'Rune #2' })).toHaveValue('Glacial Rune');
  await view.getByRole('button', { name: 'Save', exact: true }).click();
  await expectStats('body armour runes');
  await view.getByRole('button', { name: 'Body Armour: Storm Shell' }).hover();
  await expect(page.getByRole('tooltip')).toContainText('+14% to Cold Resistance');

  // ItemSetService.lua:17-23 (PoB-PoE2): a new set is empty and active; the first set brings the items back
  await view.getByRole('button', { name: 'Manage…' }).first().click();
  const manage = page.getByRole('dialog', { name: 'Manage item sets' });
  await manage.getByRole('button', { name: 'New' }).click();
  await manage.getByRole('textbox', { name: 'Name for this item set' }).fill('Second');
  await manage.getByRole('button', { name: 'Save' }).click();
  await expectStats('new item set');
  await page.keyboard.press('Escape');
  await expect(manage).toBeHidden();
  await view.getByRole('combobox', { name: 'Item set' }).selectOption({ label: 'Default' });
  await expectStats('body armour runes');

  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog', { name: 'Save build as' });
  await dialog.getByRole('textbox', { name: 'Build name' }).fill('Items journey');
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await page.getByRole('link', { name: 'Back to builds' }).click();
  await page.getByRole('link', { name: 'Items journey', exact: true }).click();
  // the build reopens on the page it was left on, with the edits
  await expect(view.getByRole('button', { name: 'Helmet: Grim Visage' })).toBeVisible({ timeout: 60_000 });
  await expectStats('body armour runes');
  await view.getByRole('button', { name: 'Body Armour: Storm Shell' }).hover();
  await expect(page.getByRole('tooltip')).toContainText('+14% to Cold Resistance');
  await expect(view.getByRole('combobox', { name: 'Item set' }).locator('option')).toHaveText(['Default', 'Default', 'Second']);
  expect(errors).toEqual([]);
});

test.describe('390px wide', () => {
  test.use(PHONE);
  test('the PoE2 Skills, Items and Calcs pages fit a phone', async ({ page }) => {
    const { view, errors } = await open(page, ITEMS_FIXTURE);
    for (const [tab, pane] of [
      ['Skills', page.locator('section[aria-label="Socket groups"]')],
      ['Items', view.locator('[data-poe2-page][aria-label="Items"]')],
      ['Calcs', view.getByRole('region', { name: 'Calculations' })]
    ] as const) {
      await bar(page).getByRole('link', { name: tab, exact: true }).click();
      await expect(pane).toBeVisible();
      // a pane still loading (aria-busy) or drawn in a fallback face is not the layout being checked
      await expect(pane).not.toHaveAttribute('aria-busy', 'true');
      await expect(pane.locator('[aria-busy=true]')).toHaveCount(0);
      await page.evaluate(async () => {
        await document.fonts.ready;
      });
      // the pane sits inside the phone, takes its width below the stats, and its content does not run past it (a Calcs
      // damage table scrolls inside itself; a tab strip wider than the phone scrolls sideways, as layout-phone.spec allows)
      const fit = await pane.evaluate((el) => {
        const scrolls = (e: Element) => {
          for (let p = e.parentElement; p && p !== el; p = p.parentElement) if (getComputedStyle(p).overflowX !== 'visible') return true;
          return false;
        };
        const right = el.getBoundingClientRect().right;
        const kids = [...el.querySelectorAll('*')].filter((e) => !e.closest('[role=tablist]') && !scrolls(e) && e.getBoundingClientRect().width > 0);
        return {
          right: Math.round(right),
          width: el.clientWidth,
          overflow: Math.max(0, ...kids.map((e) => e.getBoundingClientRect().right - right))
        };
      });
      const [inner, scroll] = await page.evaluate(() => [window.innerWidth, (document.scrollingElement ?? document.documentElement).scrollWidth]);
      expect([tab, inner, scroll, fit.right <= PHONE.viewport.width, fit.width >= 300, fit.overflow <= 1]).toEqual([
        tab,
        PHONE.viewport.width,
        PHONE.viewport.width,
        true,
        true,
        true
      ]);
    }
    expect(errors).toEqual([]);
  });
});

// NotesTab.lua:75-88 (PoB-PoE2): the Notes page edits the build's <Notes>, which Save writes and a reopen reads back
test('the PoE2 Notes page edits the notes the build saves with', async ({ page }) => {
  const { view, errors } = await open(page);
  await poe2Nav(page, 'Notes').click();
  const box = view.getByRole('textbox', { name: 'Notes' });
  await expect(box).toBeVisible();
  await box.click();
  await page.keyboard.type('Spark plan');
  await expect(page.getByRole('button', { name: 'Save (unsaved changes)' })).toBeVisible();

  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog', { name: 'Save build as' });
  await dialog.getByRole('textbox', { name: 'Build name' }).fill('Spark notes');
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await page.getByRole('link', { name: 'Back to builds' }).click();
  await page.getByRole('link', { name: 'Spark notes', exact: true }).click();
  // the build reopens on the page it was left on
  await expect(view.getByRole('textbox', { name: 'Notes' })).toContainText('Spark plan', { timeout: 60_000 });
  expect(errors).toEqual([]);
});

// ItemTools.lua:475-485 (PoB-PoE2): F1 over an item opens its poe2wiki.net page, not poewiki.net's
test('F1 over a PoE2 item opens its poe2wiki.net page', async ({ page }) => {
  const { view, errors } = await open(page, ITEMS_FIXTURE);
  await page.evaluate(() => {
    const w = window as unknown as { opened: string[] };
    w.opened = [];
    window.open = (url?: string | URL) => (w.opened.push(String(url)), null);
  });
  await poe2Nav(page, 'Items').click();
  // the fixture's stash lists its items; the Equipped board marks none for the wiki
  await view.getByRole('tab', { name: 'Stash6' }).click();
  const item = view.locator('[data-wiki]').first();
  const name = await item.getAttribute('data-wiki');
  expect(name, 'the Items page names no item for the wiki').toBeTruthy();
  await item.hover();
  await page.keyboard.press('F1');
  await expect
    .poll(() => page.evaluate(() => (window as unknown as { opened: string[] }).opened))
    .toEqual([`https://www.poe2wiki.net/wiki/${name!.replaceAll(' ', '_')}`]);
  expect(errors).toEqual([]);
});

// TreeTab.lua:588-611 (PoB-PoE2): a build saved on an older tree opens on 0_5, the only PoE2 tree go-pob-data publishes,
// and says how many saved passives that tree no longer allocates (here a node id no tree has)
test('a PoE2 build saved on an older tree says it was converted and how many passives it lost', async ({ page }) => {
  const xml = readFileSync(ITEMS_FIXTURE, 'utf8').replace(
    /(<Spec [^>]*?)nodes="([^"]*)"([^>]*?)treeVersion="0_5"/,
    '$1nodes="$2,999999999"$3treeVersion="0_3"'
  );
  expect(xml, 'the fixture spec was not rewritten').toContain('treeVersion="0_3"');
  const file = join(mkdtempSync(join(tmpdir(), 'poe2-old-tree-')), 'build.xml');
  writeFileSync(file, xml);
  const { errors } = await open(page, file);
  await expect(page.getByRole('status').filter({ hasText: 'converted to 0_5' })).toContainText(/saved on 0_3 .* 1 of its \d+ passives are no longer allocated/);
  expect(errors).toEqual([]);
});
