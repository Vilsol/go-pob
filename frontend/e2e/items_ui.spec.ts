import { type Page } from '@playwright/test';
import { test, expect } from './support/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { plainText } from '../src/lib/design/crafting';

const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2';
const board = (page: Page) => page.getByRole('region', { name: 'Equipped items' });
const unescapeXml = (t: string) => t.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&');

const setup = async (page: Page, file = BUILD) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  const t0 = await app.now();
  await app.goTo('Items');
  await app.followWorker(t0, () => expect(board(page)).toBeVisible());
  return { app, build };
};

test("an equipped item's Mod table adds a modifier to it, and Clone as new item adds a copy (0339)", async ({ page }) => {
  test.setTimeout(180_000);
  const { app, build } = await setup(page);
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const helmet = ((await m.exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === 'Helmet')!;
  const text = unescapeXml(helmet.Text);

  // ItemsTab.lua:988-993
  await board(page)
    .getByRole('button', { name: /^Helmet: / })
    .click();
  const editor = page.getByRole('dialog', { name: /^Item / });
  await editor.getByRole('button', { name: 'Mod table' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Modifier to Item' });
  await expect(dialog).toBeVisible();
  await expect(editor).toHaveCount(0);
  await expect(box).toHaveValue(text);
  await dialog.getByRole('combobox', { name: 'Source' }).selectOption({ label: 'Custom' });
  await dialog.getByRole('textbox', { name: 'Custom modifier' }).fill('+7 to Dexterity');
  await dialog.getByRole('button', { name: 'Add', exact: true }).click();
  const [added, ok] = await m.exposition.CustomModText(text, 'CUSTOM', '', '+7 to Dexterity');
  expect(ok).toBe(true);
  await expect(box).toHaveValue(added);
  await app.act(
    'item-replace',
    () => page.getByRole('button', { name: 'Save', exact: true }).click(),
    async () => {
      const misc = (await m.exposition.GetMiscCalculator(build))!;
      (await misc.ReplaceItem(helmet.ItemID, added)).unwrap();
      misc.release();
      return expectedLines(await go.outputs(build));
    }
  );

  await board(page)
    .getByRole('button', { name: /^Ring 1: / })
    .click();
  await expect(page.getByRole('dialog', { name: /^Item / }).getByRole('button', { name: 'Mod table' })).toHaveAttribute(
    'title',
    'PoB adds modifiers to magic and rare items only'
  );
  await expect(page.getByRole('dialog', { name: /^Item / }).getByRole('button', { name: 'Mod table' })).toBeDisabled();
  await page.keyboard.press('Escape');

  // ItemsTab.lua:332-334
  const ring = ((await m.exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === 'Ring 2')!;
  await board(page)
    .getByRole('button', { name: /^Ring 2: / })
    .click();
  await page
    .getByRole('dialog', { name: /^Item / })
    .getByRole('button', { name: 'Clone as new item' })
    .click();
  await expect(box).toHaveValue(unescapeXml(ring.Text));
  await expect(page.getByRole('heading', { name: 'Paste an item' })).toBeVisible();
  const count = () => go.fresh(build, (b) => b.Items.Items.length);
  const before = count();
  await app.act(
    'item-clone',
    () => page.getByRole('button', { name: 'Save to build' }).click(),
    async () => {
      const misc = (await m.exposition.GetMiscCalculator(build))!;
      (await misc.AddItem(unescapeXml(ring.Text))).unwrap();
      misc.release();
      return expectedLines(await go.outputs(build));
    }
  );
  expect(count()).toBe(before + 1);
  app.assertNoErrors();
});

test("the stash's row actions, keyboard and tooltip (0339)", async ({ page }) => {
  test.setTimeout(180_000);
  const { app, build } = await setup(page);
  const m = await go.model();
  const lines = async () => expectedLines(await go.outputs(build));
  const box = page.getByRole('textbox', { name: /Item text/ });
  const idOf = async (name: string) => {
    for (const it of go.fresh(build, (b) => (b.Items.Items ?? []).map((x) => ({ id: x.ID, text: x.Text })))) {
      const p = (await m.exposition.ParseItem(build, it.text, go.NO_TIP_STATS)).unwrap();
      if ((p.Title || p.Name) === name) return { id: it.id, text: unescapeXml(it.text) };
    }
    throw new Error(`no item ${name}`);
  };
  await page
    .getByRole('tablist', { name: 'Item views' })
    .getByRole('tab', { name: /^Stash/ })
    .click();
  const table = page.getByRole('table', { name: 'Stash' });
  const nameButton = (name: string) => table.getByRole('button', { name, exact: true });
  const actions = (name: string) => table.getByRole('button', { name: `Actions for ${name}` });
  const menu = (name: string) => page.getByRole('menu', { name });

  // ItemListControl.lua:260-269
  await nameButton('Dawnbreaker').hover();
  const tipId = await nameButton('Dawnbreaker').getAttribute('aria-describedby');
  const tip = page.locator(`[id="${tipId}"]`);
  await expect(tip).toBeVisible();
  const [tipBox, listBox] = [await tip.boundingBox(), await table.boundingBox()];
  expect(tipBox!.x).toBeGreaterThanOrEqual(listBox!.x + listBox!.width);

  // ItemListControl.lua:290-314
  const dawn = await idOf('Dawnbreaker');
  await actions('Dawnbreaker').click();
  await app.act(
    'stash-menu-equip',
    () =>
      menu('Dawnbreaker')
        .getByRole('menuitem', { name: /^Equip or unequip/ })
        .click(),
    async () => (expect(await m.exposition.ToggleEquipItem(build, dawn.id, false)).toBe(true), lines())
  );

  // ItemListControl.lua:331-369
  await nameButton('Dawnbreaker').focus();
  await page.keyboard.press('ArrowDown');
  const second = (await table.getByRole('row').nth(2).getByRole('button').first().textContent())!;
  await expect(table.getByRole('row').nth(2).getByRole('button').first()).toBeFocused();
  await page.keyboard.press('ArrowUp');
  await expect(nameButton('Dawnbreaker')).toBeFocused();
  expect(second).not.toBe('Dawnbreaker');
  const meadow = "Ahkeli's Meadow";
  const gone = await idOf(meadow);
  await nameButton(meadow).focus();
  await app.act(
    'stash-delete-key',
    () => page.keyboard.press('Delete'),
    async () => (expect(await m.exposition.DeleteItem(build, gone.id)).toBe(true), lines())
  );
  await expect(nameButton(meadow)).toHaveCount(0);

  const names = await table.getByRole('row').locator('button[data-pob-color]').allTextContents();
  const prism = names.find((n) => n !== 'Dawnbreaker' && names.filter((x) => x === n).length === 1)!;
  const p = await idOf(prism);
  await actions(prism).click();
  await menu(prism)
    .getByRole('menuitem', { name: /^Clone as new item/ })
    .click();
  await expect(box).toHaveValue(p.text);
  await expect(page.getByRole('heading', { name: 'Paste an item' })).toBeVisible();
  await page
    .getByRole('tablist', { name: 'Item views' })
    .getByRole('tab', { name: /^Stash/ })
    .click();
  await actions(prism).click();
  await menu(prism)
    .getByRole('menuitem', { name: /^Edit as text/ })
    .click();
  await expect(box).toHaveValue(p.text);
  await expect(page.getByRole('heading', { name: `Editing ${prism}` })).toBeVisible();

  // SharedItemListControl.lua:93-104
  await page
    .getByRole('tablist', { name: 'Item views' })
    .getByRole('tab', { name: /^Stash/ })
    .click();
  await actions(prism).click();
  await menu(prism)
    .getByRole('menuitem', { name: /^Share/ })
    .click();
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Shared' }).click();
  await expect(page.getByRole('table', { name: 'Shared' }).getByRole('button', { name: prism, exact: true })).toHaveCount(1);
  app.assertNoErrors();
});

test('drag a stash row onto another to reorder, and onto a slot to equip (0339)', async ({ page }) => {
  test.setTimeout(180_000);
  const { app, build } = await setup(page);
  const m = await go.model();
  const tabs = page.getByRole('tablist', { name: 'Item views' });
  await tabs.getByRole('tab', { name: /^Stash/ }).click();
  const table = page.getByRole('table', { name: 'Stash' });
  const names = () => table.getByRole('row').locator('button[data-pob-color]').allTextContents();
  const modelNames = async () => {
    const out: string[] = [];
    for (const it of go.fresh(build, (b) => (b.Items.Items ?? []).map((x) => x.Text))) {
      const p = (await m.exposition.ParseItem(build, it, go.NO_TIP_STATS)).unwrap();
      out.push(p.Title || p.Name);
    }
    return out;
  };
  const ids = () => go.fresh(build, (b) => (b.Items.Items ?? []).map((x) => x.ID));
  expect(await names()).toEqual(await modelNames());

  // ItemListControl.lua:286-288
  const [from, to] = [ids()[0], ids()[4]];
  const rows = table.getByRole('row');
  await rows.nth(1).locator('button[data-pob-color]').hover();
  await page.mouse.down();
  await rows.nth(3).hover();
  await rows.nth(5).hover({ position: { x: 20, y: 10 } });
  await page.mouse.up();
  go.fresh(build, (b) => b.MoveItem(from, to));
  const want = await modelNames();
  expect(want.indexOf('Dawnbreaker'), 'vacuity: the move moved it').toBe(3);
  await expect.poll(names).toEqual(want);

  // ItemSlotControl.lua:114-130
  const dawn = table.getByRole('button', { name: 'Dawnbreaker', exact: true });
  await app.act(
    'stash-drop-slot',
    async () => {
      await dawn.hover();
      await page.mouse.down();
      await tabs.getByRole('tab', { name: 'Equipped' }).hover();
      await tabs.getByRole('tab', { name: 'Equipped' }).hover({ position: { x: 5, y: 5 } });
      await board(page).getByRole('group', { name: 'Weapon 2 slot' }).hover();
      await page.mouse.up();
    },
    async () => {
      expect(await m.exposition.DropItemOnSlot(build, 'Weapon 2', from, '', '')).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(board(page).getByRole('button', { name: /^Weapon 2: / })).toHaveAccessibleName('Weapon 2: Dawnbreaker');
  app.assertNoErrors();
});

test('the board lines up the gloves, belt and boots bottoms, and the gloves and boots with the rings (0339)', async ({ page }) => {
  test.setTimeout(120_000);
  const { app } = await setup(page);
  const box = async (slot: string) =>
    (await board(page)
      .getByRole('group', { name: `${slot} slot` })
      .boundingBox())!;
  const [gloves, belt, boots, ring1, ring2] = await Promise.all(['Gloves', 'Belt', 'Boots', 'Ring 1', 'Ring 2'].map(box));
  const bottom = (b: { y: number; height: number }) => b.y + b.height;
  expect(Math.abs(bottom(gloves) - bottom(belt))).toBeLessThanOrEqual(1);
  expect(Math.abs(bottom(boots) - bottom(belt))).toBeLessThanOrEqual(1);
  expect(Math.abs(gloves.x + gloves.width - (ring1.x + ring1.width))).toBeLessThanOrEqual(1);
  expect(Math.abs(boots.x - ring2.x)).toBeLessThanOrEqual(1);
  const all = await board(page).getByRole('group').all();
  const boxes = await Promise.all(all.map(async (g) => ({ name: await g.getAttribute('aria-label'), b: (await g.boundingBox())! })));
  for (const [i, a] of boxes.entries())
    for (const c of boxes.slice(i + 1)) {
      const apart = a.b.x + a.b.width <= c.b.x || c.b.x + c.b.width <= a.b.x || a.b.y + a.b.height <= c.b.y || c.b.y + c.b.height <= a.b.y;
      expect(apart, `${a.name} overlaps ${c.name}`).toBe(true);
    }
  app.assertNoErrors();
});

test("a crafted item's affix dropdown filters as the user types, and Enter picks (0339)", async ({ page }) => {
  test.setTimeout(180_000);
  const { app } = await setup(page);
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const [crafted] = await m.exposition.CraftItemText(2, '', 'Hubris Circlet');
  await box.fill(crafted);
  const [controls] = await m.exposition.ItemAffixControls(crafted);
  const series = controls.Controls![0].Series!.map((x) => plainText(x.Label));
  const affix = page.getByRole('group', { name: 'Affixes' }).getByRole('combobox', { name: 'Affix 1' });

  const want = series.filter((l) => l.toLowerCase().includes('maximum') && l.toLowerCase().includes('life'));
  expect(want.length, 'vacuity: the filter keeps some entries and drops others').toBeGreaterThan(0);
  expect(want.length).toBeLessThan(series.length);
  await affix.focus();
  await page.keyboard.type('Life maxim');
  const list = page.locator(`[id="${await affix.getAttribute('aria-controls')}"]`);
  await expect(list.getByRole('option')).toHaveText(want);
  await expect(page.getByRole('searchbox', { name: 'Filter Affix 1' })).toHaveValue('Life maxim');
  await page.keyboard.press('Enter');
  const [withLife, ok] = await m.exposition.AffixEditText(crafted, 0, series.indexOf(want[0]), controls.Controls![0].SliderVal);
  expect(ok).toBe(true);
  await expect(box).toHaveValue(withLife);
  await expect(affix).toHaveText(want[0]);
  app.assertNoErrors();
});

test('Craft item… picks the type and base by typing, previews each hovered base, and the crafted item equips (0339)', async ({ page }) => {
  test.setTimeout(180_000);
  const { app, build } = await setup(page);
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });

  // ItemsTab.lua:2573-2587
  await page.getByRole('button', { name: 'Craft item…' }).click();
  const craft = page.getByRole('dialog', { name: 'Craft Item' });
  await craft.getByRole('textbox', { name: 'Name' }).fill('Test Crown');
  const group = ((await m.exposition.CraftBaseGroups()) ?? []).find((g) => g.Bases?.includes('Hubris Circlet'))!;
  const type = craft.getByRole('combobox', { name: 'Type' });
  await type.focus();
  await page.keyboard.type(group.Type);
  await page.keyboard.press('Enter');
  await expect(type).toHaveText(group.Type);
  const base = craft.getByRole('combobox', { name: 'Base' });
  await base.click();
  await page.keyboard.type('hubris');
  const list = page.locator(`[id="${await base.getAttribute('aria-controls')}"]`);
  await expect(list.getByRole('option')).toHaveText(['Hubris Circlet']);
  const preview = craft.getByRole('group', { name: 'Base preview' });
  await expect(preview).toContainText('Hubris Circlet');
  await expect(preview).toContainText('Equipping this item in Helmet will give you');
  await page.getByRole('searchbox', { name: 'Filter Base' }).fill('');
  const other = (await list.getByRole('option').first().textContent())!.trim();
  await list.getByRole('option').first().hover();
  await expect(preview).toContainText(other);
  await list.getByRole('option', { name: 'Hubris Circlet', exact: true }).click();
  await expect(base).toHaveText('Hubris Circlet');
  await expect(preview).toContainText('Hubris Circlet');

  await craft.getByRole('button', { name: 'Create' }).click();
  const [crafted, ok] = await m.exposition.CraftItemText(2, 'Test Crown', 'Hubris Circlet');
  expect(ok).toBe(true);
  await expect(page.getByRole('article', { name: 'Item: Test Crown' })).toBeVisible();
  await expect(box).toBeHidden();
  await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.getByRole('button', { name: 'Copy item' }).click();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(crafted.replace(/\r?\n/g, '\r\n'));
  const asText = page.getByRole('button', { name: 'Edit as text' });
  await expect(asText).toHaveAttribute('aria-pressed', 'false');
  await asText.click();
  await expect(asText).toHaveAttribute('aria-pressed', 'true');
  await expect(box).toHaveValue(crafted);
  await app.act(
    'craft-equip',
    () => page.getByRole('button', { name: 'Equip in Helmet' }).click(),
    async () => {
      (await m.exposition.EquipItem(build, crafted, 'Helmet')).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(board(page).getByRole('button', { name: /^Helmet: / })).toHaveAccessibleName('Helmet: Test Crown');
  app.assertNoErrors();
});
