import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { choose, entries } from './support/dropdown';
import type * as Types from '../src/lib/types';

type Build = Awaited<ReturnType<typeof go.loadXml>>;

const board = (page: Page) => page.getByRole('region', { name: 'Equipped items' });
const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const tile = (page: Page, slot: string) => board(page).getByRole('button', { name: new RegExp(`^${escapeRe(slot)}: `) });

const setup = async (page: Page, file: string) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await expect(board(page)).toBeVisible();
  return { app, build };
};

const equippedName = async (build: Build, slot: string) => {
  const m = await go.model();
  const eq = ((await m.exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === slot);
  if (!eq) return undefined;
  const p = (await m.exposition.ParseItem(build, eq.Text, go.NO_TIP_STATS)).unwrap();
  return p.Title || p.Name;
};

// ItemsTab.lua:244-258
test('weapon set I / II: the swap weapons and the main socket group follow (0111)', async ({ page }) => {
  const file = '0111-berserker-earthshatter';
  const { app, build } = await setup(page, file);
  const m = await go.model();
  const weaponSet = page.getByRole('group', { name: 'Weapon set' });
  const main = () => go.fresh(build, (b) => b.Build.MainSocketGroup);
  const second = () => go.fresh(build, (b) => b.Items.ItemSets.find((s) => s.ID === String(b.Items.ActiveItemSet))?.UseSecondWeaponSet ?? false);
  expect(main()).toBe(1);
  expect(second()).toBe(false);
  await expect(weaponSet.getByRole('button', { name: 'I', exact: true })).toHaveAttribute('aria-pressed', 'true');
  const setOne = await equippedName(build, 'Weapon 1');

  await app.act(
    'weapon-set',
    () => weaponSet.getByRole('button', { name: 'II', exact: true }).click(),
    async () => {
      expect(await m.exposition.SetWeaponSet(build, true)).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  expect(second()).toBe(true);
  expect(main(), 'the model moved the main group to the first set-2 group').toBe(3);
  await expect(weaponSet.getByRole('button', { name: 'II', exact: true })).toHaveAttribute('aria-pressed', 'true');
  const swap = await equippedName(build, 'Weapon 1 Swap');
  expect(swap, 'vacuity: 0111 has no swap weapon').toBeDefined();
  expect(swap).not.toBe(setOne);
  await expect(tile(page, 'Weapon 1')).toHaveAccessibleName(`Weapon 1: ${swap}`);

  await app.act(
    'weapon-set',
    () => weaponSet.getByRole('button', { name: 'I', exact: true }).click(),
    async () => {
      expect(await m.exposition.SetWeaponSet(build, false)).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  expect(main()).toBe(1);
  await expect(tile(page, 'Weapon 1')).toHaveAccessibleName(`Weapon 1: ${setOne}`);

  await app.act(
    'weapon-set',
    () => weaponSet.getByRole('button', { name: 'II', exact: true }).click(),
    async () => {
      await m.exposition.SetWeaponSet(build, true);
      return expectedLines(await go.outputs(build));
    }
  );
  const exported = await go.loadXml(await go.decode(await app.exportCode()));
  const state = (b: Build) =>
    go.fresh(b, (x) => [x.Build.MainSocketGroup, x.Items.ItemSets.find((s) => s.ID === String(x.Items.ActiveItemSet))?.UseSecondWeaponSet]);
  expect(state(exported)).toEqual(state(build));
  app.assertNoErrors();
});

test('manage item sets: new, copy, rename, reorder, switch, delete (0339)', async ({ page }) => {
  const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
  const { app, build } = await setup(page, file);
  const m = await go.model();
  const sets = (b: Build) => go.fresh(b, (x) => [x.Items.ActiveItemSet, x.Items.ItemSets.map((s) => `${s.ID}:${s.Title ?? 'Default'}`)]);
  const lines = async () => expectedLines(await go.outputs(build));
  await page.getByRole('button', { name: 'Manage…' }).click();
  const dialog = page.getByRole('dialog', { name: 'Manage item sets' });
  const list = dialog.getByRole('listbox', { name: 'Item sets' });
  const option = (title: string) => list.getByRole('option', { name: new RegExp(`^${escapeRe(title)}\\s*(current)?$`) });
  const shown = async () => {
    const [active, rows] = sets(build) as [number, string[]];
    await expect(list.getByRole('option')).toHaveText(rows.map((r) => r.replace(/^\d+:/, '') + (r.startsWith(`${active}:`) ? 'current' : '')));
  };
  await shown();
  const name = async (title: string) => {
    await dialog.getByRole('textbox', { name: 'Name for this item set' }).fill(title);
    return dialog.getByRole('button', { name: 'Save' }).click();
  };

  await dialog.getByRole('button', { name: 'New' }).click();
  await app.act(
    'item-set-new',
    () => name('pvp'),
    async () => (expect(await m.exposition.NewItemSet(build, 'pvp')).toBe(3), lines())
  );
  await shown();

  await option('enmity').click();
  await dialog.getByRole('button', { name: 'Copy' }).click();
  await expect(dialog.getByRole('textbox', { name: 'Name for this item set' })).toHaveValue('enmity');
  await app.act(
    'item-set-copy',
    () => name('enmity 2'),
    async () => (expect(await m.exposition.CopyItemSet(build, 2, 'enmity 2')).toBe(4), lines())
  );
  await shown();

  await option('pvp').click();
  await dialog.getByRole('button', { name: 'Rename' }).click();
  await app.act(
    'item-set-rename',
    () => name('pvp boss'),
    async () => (await m.exposition.RenameItemSet(build, 3, 'pvp boss'), lines())
  );
  await shown();

  await list.getByRole('group', { name: 'enmity 2' }).focus();
  await app.act(
    'item-set-move',
    () => page.keyboard.press('Alt+ArrowUp'),
    async () => (await m.exposition.MoveItemSet(build, 4, 3), lines())
  );
  expect(sets(build)[1]).toEqual(['1:Default', '2:enmity', '4:enmity 2', '3:pvp boss']);
  await shown();

  await app.act(
    'item-set',
    () => option('enmity 2').dblclick(),
    async () => (await m.exposition.SetActiveItemSet(build, 4), lines())
  );
  await shown();
  await expect(page.getByRole('combobox', { name: 'Item set' })).toHaveValue('4');

  await dialog.getByRole('button', { name: 'Delete' }).click();
  const confirm = dialog.getByRole('group', { name: 'Confirm delete' });
  await expect(confirm).toContainText("delete 'enmity 2'");
  await app.act(
    'item-set-delete',
    () => confirm.getByRole('button', { name: 'Delete' }).click(),
    async () => (await m.exposition.DeleteItemSet(build, 4), lines())
  );
  expect(sets(build)[0], 'the model hands the set in use over to the one before it').toBe(2);
  await shown();
  await dialog.getByRole('button', { name: 'Done' }).click();
  await expect(page.getByRole('combobox', { name: 'Item set' })).toHaveValue('2');

  const exported = await go.loadXml(await go.decode(await app.exportCode()));
  expect(sets(exported), 'the exported build holds the model’s sets, in order').toEqual(sets(build));
  app.assertNoErrors();
});

// ItemListControl.lua:23-75, 331-369, ItemsTab.lua:1768-1815
test('delete, del unused, del all (0339)', async ({ page }) => {
  const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
  const { app, build } = await setup(page, file);
  const m = await go.model();
  const lines = async () => expectedLines(await go.outputs(build));
  const ids = () => go.fresh(build, (b) => (b.Items.Items ?? []).map((it) => it.ID));
  const nameOf = async (id: number) => {
    const text = go.fresh(build, (b) => (b.Items.Items ?? []).find((it) => it.ID === id)?.Text ?? '');
    const p = (await m.exposition.ParseItem(build, text, go.NO_TIP_STATS)).unwrap();
    return p.Title || p.Name;
  };
  const stash = page.getByRole('region', { name: 'Stash items' });
  const confirm = page.getByRole('dialog', { name: /^Delete/ });
  await page.getByRole('tab', { name: /^Stash/ }).click();

  const names = new Map<number, string>();
  for (const id of ids()) names.set(id, await nameOf(id));
  const unique = (id: number) => [...names.values()].filter((n) => n === names.get(id)).length === 1;
  const unusedId = 8;
  expect(unique(unusedId), 'vacuity: another item shares its name').toBe(true);
  expect(await m.exposition.DeleteItemPrompt(build, unusedId)).toBe('');
  await app.act(
    'item-delete',
    () => stash.getByRole('button', { name: `Delete ${names.get(unusedId)}`, exact: true }).click(),
    async () => (expect(await m.exposition.DeleteItem(build, unusedId)).toBe(true), lines())
  );
  expect(ids()).not.toContain(unusedId);
  await expect(stash.getByRole('button', { name: names.get(unusedId), exact: true })).toHaveCount(0);

  const prompt16 = await m.exposition.DeleteItemPrompt(build, 16);
  expect(prompt16).toContain("in set 'Default'");
  await stash.getByRole('button', { name: `Delete ${names.get(16)}`, exact: true }).click();
  await expect(confirm).toContainText(prompt16.split('\n')[0]);
  await app.act(
    'item-delete',
    () => confirm.getByRole('button', { name: 'Delete' }).click(),
    async () => (await m.exposition.DeleteItem(build, 16), lines())
  );
  expect(ids()).not.toContain(16);

  const prompt61 = await m.exposition.DeleteItemPrompt(build, 61);
  expect(prompt61).toContain('is currently equipped in Helmet.');
  await page.getByRole('tab', { name: 'Equipped' }).click();
  await board(page).getByRole('button', { name: 'Change Helmet' }).click();
  await page.getByRole('menu', { name: 'Helmet' }).getByRole('menuitem', { name: 'Delete item…' }).click();
  await expect(confirm).toContainText(prompt61.split('\n')[0]);
  await app.act(
    'item-delete',
    () => confirm.getByRole('button', { name: 'Delete' }).click(),
    async () => (await m.exposition.DeleteItem(build, 61), lines())
  );
  await expect(tile(page, 'Helmet')).toHaveAccessibleName('Helmet: Empty');

  await page.getByRole('tab', { name: /^Stash/ }).click();
  await app.act(
    'item-del-unused',
    () => stash.getByRole('button', { name: 'Del Unused' }).click(),
    async () => (expect(await m.exposition.DeleteUnusedItems(build)).toBe(true), lines())
  );
  expect(ids().length, 'Del Unused kept no item').toBeGreaterThan(0);
  await expect(stash.getByRole('cell', { name: 'Unused', exact: true })).toHaveCount(0);

  await stash.getByRole('button', { name: 'Del All…' }).click();
  await expect(confirm).toContainText('Are you sure you want to delete all items in this build?');
  await app.act(
    'item-del-all',
    () => confirm.getByRole('button', { name: 'Delete' }).click(),
    async () => (await m.exposition.DeleteAllItems(build), lines())
  );
  expect(ids()).toEqual([]);
  await page.getByRole('tab', { name: 'Equipped' }).click();
  await expect(tile(page, 'Body Armour')).toHaveAccessibleName('Body Armour: Empty');

  const exported = await go.loadXml(await go.decode(await app.exportCode()));
  const state = (b: Build) =>
    go.fresh(b, (x) => [(x.Items.Items ?? []).length, x.Items.ItemSets.map((s) => (s.Slots ?? []).filter((sl) => sl.ItemID > 0).length)]);
  expect(state(exported)).toEqual(state(build));
  app.assertNoErrors();
});

// ItemsTab.lua:1731-1765
// ItemListControl.lua:290-329
test('stash: sort, ctrl+click equip, double-click, copy (0339)', async ({ page }) => {
  const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
  const { app, build } = await setup(page, file);
  const m = await go.model();
  const lines = async () => expectedLines(await go.outputs(build));
  const items = () => go.fresh(build, (b) => (b.Items.Items ?? []).map((it) => ({ id: it.ID, text: it.Text })));
  const nameOf = async (text: string) => {
    const p = (await m.exposition.ParseItem(build, text, go.NO_TIP_STATS)).unwrap();
    return p.Title || p.Name;
  };
  const stash = page.getByRole('region', { name: 'Stash items' });
  await page.getByRole('tab', { name: /^Stash/ }).click();
  const rowNames = () =>
    stash
      .getByRole('table')
      .getByRole('button', { name: /^(?!Delete |Actions for )/ })
      .allTextContents();

  await app.act(
    'item-sort',
    () => stash.getByRole('button', { name: 'Sort', exact: true }).click(),
    async () => (await m.exposition.SortItems(build), lines())
  );
  const sorted = await Promise.all(items().map((it) => nameOf(it.text)));
  // ItemListControl.lua:119-121
  await expect
    .poll(async () => {
      const shown = await rowNames();
      let k = 0;
      for (const n of sorted) if (n === shown[k]) k++;
      return k === shown.length && shown.length === sorted.length;
    })
    .toBe(true);

  await app.act(
    'item-ctrl-click',
    () => stash.getByRole('button', { name: 'Empyrean Veil', exact: true }).click({ modifiers: ['Control'] }),
    async () => (expect(await m.exposition.ToggleEquipItem(build, 16, false)).toBe(true), lines())
  );
  await app.act(
    'item-ctrl-click',
    () => stash.getByRole('button', { name: 'Hypnotic Hold', exact: true }).click({ modifiers: ['Control', 'Shift'] }),
    async () => (expect(await m.exposition.ToggleEquipItem(build, 20, true)).toBe(true), lines())
  );
  const slotOf = (id: number) =>
    go.fresh(build, (b) => b.Items.ItemSets.find((s) => s.ID === String(b.Items.ActiveItemSet))?.Slots?.find((s) => s.ItemID === id)?.Name);
  expect(slotOf(16)).toBe('Helmet');
  expect(slotOf(20)).toBe('Ring 2');
  await page.getByRole('tab', { name: 'Equipped' }).click();
  await expect(tile(page, 'Helmet')).toHaveAccessibleName('Helmet: Empyrean Veil');
  await expect(tile(page, 'Ring 2')).toHaveAccessibleName('Ring 2: Hypnotic Hold');

  await page.context().grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.getByRole('tab', { name: /^Stash/ }).click();
  const text = items()
    .find((it) => it.id === 32)!
    .text.replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&');
  const name32 = await nameOf(text);
  await stash.getByRole('button', { name: name32, exact: true }).focus();
  await page.keyboard.press('Control+c');
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe(text.replace(/\r?\n/g, '\r\n'));
  await stash.getByRole('button', { name: name32, exact: true }).dblclick();
  await expect(page.getByRole('heading', { name: `Editing ${name32}` })).toBeVisible();
  await expect(page.getByRole('textbox', { name: /Item text/ })).toHaveValue(text);
  await expect(page.getByRole('dialog')).toHaveCount(0);
  app.assertNoErrors();
});

const ABYSS_BELT = `Item Class: Belts
Rarity: Rare
Abyss Clasp
Stygian Vise
--------
Item Level: 84
--------
Has 1 Abyssal Socket (implicit)
--------
+40 to maximum Life
+30% to Fire Resistance`;

// ItemsTab.lua:175-189, ItemSlotControl.lua:98-111
// ItemSlotControl.lua:48-56
// ItemsTab.lua:208-222, ItemSlotControl.lua:61-73
test('slots: abyssal sockets, menu entry tooltips, tree socket slots (0339)', async ({ page }) => {
  const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
  const { app, build } = await setup(page, file);
  const m = await go.model();
  const lines = async () => expectedLines(await go.outputs(build));
  const abyss = page.getByRole('region', { name: 'Abyssal sockets' });
  await expect(abyss).toHaveCount(0);

  await page.getByRole('textbox', { name: /Item text/ }).fill(ABYSS_BELT);
  await app.act(
    'item-equip',
    () => page.getByRole('region', { name: 'Equip impact' }).getByRole('button', { name: 'Equip in Belt' }).click(),
    async () => ((await m.exposition.EquipItem(build, ABYSS_BELT, 'Belt')).unwrap(), lines())
  );
  expect((await m.exposition.ItemsTabSlots(build)).Abyssal).toEqual(['Belt Abyssal Socket 1']);
  await expect(abyss.getByRole('button', { name: 'Belt Abyssal #1: Empty' })).toBeVisible();
  const choices = ((await m.exposition.SlotItemLists(build)) ?? []).find((l) => l.SlotName === 'Belt Abyssal Socket 1')?.ItemIDs ?? [];
  expect(choices, 'vacuity: 0339 has no abyss jewel').toContain(14);
  await abyss.getByRole('button', { name: 'Change Belt Abyssal #1' }).click();
  await app.act(
    'item-equip-id',
    () =>
      page
        .getByRole('menu', { name: 'Belt Abyssal #1' })
        .getByRole('menuitem', { name: /^Equip Ancient Arbiter/ })
        .click(),
    async () => (expect((await m.exposition.EquipItemID(build, 'Belt Abyssal Socket 1', 14)).unwrap()).toBe(true), lines())
  );
  await expect(abyss.getByRole('button', { name: 'Belt Abyssal #1: Ancient Arbiter' })).toBeVisible();

  await board(page).getByRole('button', { name: 'Change Belt' }).click();
  await app.act(
    'item-equip-id',
    () => page.getByRole('menu', { name: 'Belt' }).getByRole('menuitem', { name: "Equip Arn's Anguish" }).first().click(),
    async () => {
      const arn = ((await m.exposition.SlotItemLists(build)) ?? []).find((l) => l.SlotName === 'Belt')!.ItemIDs!.find((id) => id === 10 || id === 71)!;
      expect((await m.exposition.EquipItemID(build, 'Belt', arn)).unwrap()).toBe(true);
      return lines();
    }
  );
  await expect(abyss).toHaveCount(0);
  const slotOf = (id: number) =>
    go.fresh(build, (b) => b.Items.ItemSets.find((s) => s.ID === String(b.Items.ActiveItemSet))?.Slots?.find((s) => s.ItemID === id)?.Name);
  expect(slotOf(14), 'the model kept the jewel in a hidden abyssal slot').toBeUndefined();

  await board(page).getByRole('button', { name: 'Change Helmet' }).click();
  await page.getByRole('menu', { name: 'Helmet' }).getByRole('menuitem', { name: 'Equip Empyrean Veil' }).hover();
  const tip = page.getByRole('tooltip');
  await expect(tip).toContainText('Empyrean Veil');
  await expect(tip.locator('[data-header^="Equipping this item in Helmet"]')).toHaveCount(1);
  await page.keyboard.press('Escape');

  const sockets = (await m.exposition.ItemsTabSlots(build)).Sockets ?? [];
  const k = sockets.findIndex((s) => s.NodeID === 10532);
  expect(k, 'vacuity: 10532 is not a socket slot').toBeGreaterThanOrEqual(0);
  const select = page.getByRole('region', { name: 'Tree jewels' }).getByRole('combobox', { name: `Socket #${k + 1}` });
  const jewelName = async (id: number) => {
    const text = go.fresh(build, (b) => (b.Items.Items ?? []).find((it) => it.ID === id)?.Text ?? '');
    const p = (await m.exposition.ParseItem(build, text.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&'), go.NO_TIP_STATS)).unwrap();
    return p.Title || p.Name;
  };
  const name22 = await jewelName(22);
  expect(await entries(select)).toHaveLength(1 + (sockets[k].Choices ?? []).length);
  await app.act(
    'socket-jewel',
    () => choose(select, 'None'),
    async () => (expect((await m.exposition.SocketTreeJewel(build, 10532, 0)).unwrap()).toBe(true), lines())
  );
  await expect(select).toHaveText('None');

  await select.click();
  const list = page.locator(`[id="${await select.getAttribute('aria-controls')}"]`);
  await list.getByRole('option', { name: name22 }).first().hover();
  await expect(page.getByRole('tooltip').filter({ hasText: name22 })).toBeVisible();
  await expect(select).toBeFocused();
  await page.mouse.move(0, 0);
  await select.press('Home');
  await expect(page.getByRole('tooltip').filter({ hasText: name22 })).toHaveCount(0);
  const order = (await list.getByRole('option').allTextContents()).map((t) => t.trim());
  for (let n = 0; n < order.indexOf(name22); n++) await select.press('ArrowDown');
  await expect(page.getByRole('tooltip').filter({ hasText: name22 })).toBeVisible();
  await expect(select).toBeFocused();
  await app.act(
    'socket-jewel',
    () => select.press('Enter'),
    async () => (expect((await m.exposition.SocketTreeJewel(build, 10532, 22)).unwrap()).toBe(true), lines())
  );
  await expect(select).toHaveText(name22);
  await expect(page.getByRole('tooltip').filter({ hasText: name22 })).toHaveCount(0);
  app.assertNoErrors();
});

// ItemsTab.lua:4569-4593
// ItemsTab.lua:5081-5088, Main.lua:114
test('tooltips: socket jewel, affected-slot-only compare (0339)', async ({ page }) => {
  const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
  const { build } = await setup(page, file);
  const m = await go.model();
  const strip = (t: string) => t.replace(/\^(?:[x#][0-9A-Fa-f]{6}|\d)/g, '');

  const sockets = (await m.exposition.ItemsTabSlots(build)).Sockets ?? [];
  const k = sockets.findIndex((s) => s.NodeID === 10532);
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const [want, ok] = misc.SocketJewelTooltip(10532, await go.itemTooltipStats(build));
  misc.release();
  expect(ok).toBe(true);
  await page
    .getByRole('region', { name: 'Tree jewels' })
    .getByRole('group', { name: `Socket #${k + 1}` })
    .hover();
  const lines = (want.Lines ?? []).map((l) => (l.Separator ? '--' : strip(l.Text)));
  while (lines[0] === '--') lines.shift();
  while (lines.at(-1) === '--') lines.pop();
  await expect
    .poll(() =>
      page
        .getByRole('tooltip')
        .locator('[data-item-lines] > *')
        .evaluateAll((els) => els.map((el) => (el.tagName === 'HR' ? '--' : (el.textContent ?? ''))))
    )
    .toEqual(lines);
  await page.mouse.move(0, 0);

  const blocks = page.getByRole('tooltip').locator('[data-header]');
  const hoverEntry = async () => {
    await board(page).getByRole('button', { name: 'Change Ring 1' }).click();
    await page.getByRole('menu', { name: 'Ring 1' }).getByRole('menuitem', { name: 'Equip Hypnotic Hold' }).hover();
  };
  await hoverEntry();
  await expect(blocks).toHaveCount(1);
  await expect(blocks.first()).toHaveAttribute('data-header', /^Equipping this item in Ring 1/);
  await page.keyboard.press('Escape');

  await page.getByRole('button', { name: 'Settings' }).click();
  const sheet = page.getByRole('dialog', { name: 'Settings' });
  await sheet.getByRole('tab', { name: /^Items/ }).click();
  await sheet.getByRole('checkbox', { name: 'Show tooltips only for affected slots' }).uncheck();
  await page.keyboard.press('Escape');
  await hoverEntry();
  const all = (await m.exposition.GetMiscCalculator(build))!;
  const slotsFit = ((await all.CompareItem(go.fresh(build, (b) => b.Items.Items!.find((it) => it.ID === 20)!.Text))).unwrap() ?? []).length;
  all.release();
  expect(slotsFit, 'vacuity: the ring fits one slot').toBeGreaterThan(1);
  await expect(blocks).toHaveCount(slotsFit);
});

// ItemsTab.lua:378-715, 1522-1541, 2335-2382
test('item controls and craft (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await setup(page, file);
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const controls = page.getByRole('region', { name: 'Item controls' });
  const helmet = (await import('node:fs')).readFileSync(new URL('./fixtures/items/helmet.txt', import.meta.url), 'utf8');
  const edit = async (text: string, e: Partial<Types.calculator.ItemEdit>) =>
    (await m.exposition.EditItem(text, { Kind: '', Index: 0, Value: 0, Second: 0, Float: 0, Text: '', ...e })).unwrap();

  await box.fill(helmet);
  await expect(controls).toBeVisible();
  let want = await edit(helmet, { Kind: 'quality', Value: 12 });
  await controls.getByRole('spinbutton', { name: 'Quality' }).fill('12');
  await controls.getByRole('spinbutton', { name: 'Quality' }).press('Enter');
  await expect(box).toHaveValue(want);

  // ItemsTab.lua:501-503
  expect(await controls.getByRole('button', { name: 'Add a socket' }).count()).toBe(0);
  want = await edit(want, { Kind: 'sockets', Text: 'B-B-B-R' });
  await controls.getByRole('combobox', { name: 'Socket 4 colour' }).selectOption('R');
  await expect(box).toHaveValue(want);
  want = await edit(want, { Kind: 'sockets', Text: 'B-B-B R' });
  await controls.getByRole('checkbox', { name: 'Link sockets 3 and 4' }).uncheck();
  await expect(box).toHaveValue(want);

  const shaped = await edit(want, { Kind: 'influence', Text: 'Shaper' });
  await controls.getByRole('combobox', { name: 'Influence 1' }).selectOption('Shaper');
  await expect(box).toHaveValue(shaped);

  const disabled = await edit(shaped, { Kind: 'disable', Index: 0 });
  await controls
    .getByRole('list', { name: /modifiers$/ })
    .first()
    .getByRole('button')
    .first()
    .click();
  await expect(box).toHaveValue(disabled);

  const removed = await edit(disabled, { Kind: 'remove', Index: (await m.exposition.DisplayItem(disabled))[0].Mods!.findIndex((x) => x.Removable) });
  await controls.getByRole('button', { name: /^Remove / }).click();
  await expect(box).toHaveValue(removed);

  await app.act(
    'item-save',
    () => page.getByRole('button', { name: 'Save to build' }).click(),
    async () => {
      const misc = (await m.exposition.GetMiscCalculator(build))!;
      (await misc.AddItem(removed)).unwrap();
      misc.release();
      return expectedLines(await go.outputs(build));
    }
  );

  await page.getByRole('button', { name: 'Craft item…' }).click();
  const craft = page.getByRole('dialog', { name: 'Craft Item' });
  await craft.getByRole('textbox', { name: 'Name' }).fill('Test Crown');
  await choose(craft.getByRole('combobox', { name: 'Type' }), 'Helmet');
  const base = (await entries(craft.getByRole('combobox', { name: 'Base' })))[0];
  await expect(craft.getByRole('group', { name: 'Base preview' })).toContainText(base);
  await craft.getByRole('button', { name: 'Create' }).click();
  const [crafted, ok] = await m.exposition.CraftItemText(2, 'Test Crown', base);
  expect(ok).toBe(true);
  await expect(page.getByRole('article', { name: 'Item: Test Crown' })).toBeVisible();
  await expect(box).toBeHidden();
  await page.getByRole('button', { name: 'Edit as text' }).click();
  await expect(box).toHaveValue(crafted);
  app.assertNoErrors();
});

// ItemsTab.lua:2519-2605
test('an empty slot opens Craft Item offering only the types that fit it, on the board and in the list (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file).replace(/<Slot name="Gloves" itemId="\d+"/, '<Slot name="Gloves" itemId="0"');
  expect(xml, 'vacuity: the fixture still names a Gloves slot').toContain('<Slot name="Gloves" itemId="0"');
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await expect(tile(page, 'Gloves')).toHaveAccessibleName('Gloves: Empty');

  await tile(page, 'Gloves').click();
  let craft = page.getByRole('dialog', { name: 'Craft Item for Gloves' });
  await expect(craft).toBeVisible();
  const m = await go.model();
  const gloves = ((await m.exposition.CraftBaseGroups()) ?? []).filter((g) => g.Type.split(': ')[0] === 'Gloves').map((g) => g.Type);
  expect(gloves.length, 'vacuity: there are glove bases to craft').toBeGreaterThan(0);
  expect(await entries(craft.getByRole('combobox', { name: 'Type' }))).toEqual(gloves);
  await craft.getByRole('textbox', { name: 'Name' }).fill('Test Grip');
  const base = (await craft.getByRole('combobox', { name: 'Base' }).textContent())!.trim();
  const before = expectedLines(await go.outputs(build));
  await app.act(
    'craft-into-empty-slot',
    () => craft.getByRole('button', { name: 'Create' }).click(),
    async () => {
      const [text, ok] = await m.exposition.CraftItemText(2, 'Test Grip', base);
      expect(ok, `vacuity: Go crafts a ${base}`).toBe(true);
      (await m.exposition.EquipItem(build, text, 'Gloves')).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(tile(page, 'Gloves')).toHaveAccessibleName('Gloves: Test Grip');
  await expect(page.getByRole('button', { name: 'Equip in Gloves' }), 'it is equipped, not left in the paste panel').toHaveCount(0);
  const undo = page.getByRole('button', { name: 'Undo: Equipped Test Grip in Gloves (Ctrl+Z)' });
  await app.act(
    'undo',
    () => undo.click(),
    async () => before
  );
  await expect(tile(page, 'Gloves'), 'one undo step takes the crafted item back out').toHaveAccessibleName('Gloves: Empty');

  await page.getByRole('button', { name: 'Craft item…' }).click();
  craft = page.getByRole('dialog', { name: 'Craft Item' });
  await craft.getByRole('textbox', { name: 'Name' }).fill('Loose Grip');
  await choose(craft.getByRole('combobox', { name: 'Type' }), 'Gloves');
  await choose(craft.getByRole('combobox', { name: 'Base' }), base);
  await craft.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('article', { name: 'Item: Loose Grip' }), 'Craft item… still opens the item in the paste panel').toBeVisible();
  await expect(page.getByRole('button', { name: 'Equip in Gloves' })).toBeVisible();
  await expect(tile(page, 'Gloves')).toHaveAccessibleName('Gloves: Empty');

  await page.getByRole('group', { name: 'Layout' }).getByRole('button', { name: 'List' }).click();
  await page.getByRole('table', { name: 'Equipped items' }).getByRole('button', { name: 'Gloves: Empty' }).click();
  await expect(page.getByRole('dialog', { name: 'Craft Item for Gloves' })).toBeVisible();
  app.assertNoErrors();
});

// ItemsTab.lua:2519-2603, ItemDBControl.lua:361-367
test('Craft Item’s Unique rarity picks from the unique database, filtered to the slot, and creates it as Browse items opens it (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file).replace(/<Slot name="Gloves" itemId="\d+"/, '<Slot name="Gloves" itemId="0"');
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');

  await tile(page, 'Gloves').click();
  let craft = page.getByRole('dialog', { name: 'Craft Item for Gloves' });
  await craft.getByRole('combobox', { name: 'Rarity' }).selectOption('Unique');
  await expect(craft.getByRole('textbox', { name: 'Name' })).toHaveCount(0);
  const gloves = await entries(craft.getByRole('combobox', { name: 'Unique' }));
  expect(gloves).toContain("Shaper's Touch");
  expect(gloves).not.toContain("Kaom's Heart");
  const touch = craft.getByRole('combobox', { name: 'Unique' });
  await touch.click();
  await page.getByRole('searchbox', { name: 'Filter Unique' }).fill("shaper's touch");
  await page.getByRole('option', { name: "Shaper's Touch", exact: true }).click();
  const m = await go.model();
  await app.act(
    'craft-unique-into-empty-slot',
    () => craft.getByRole('button', { name: 'Create' }).click(),
    async () => {
      const [raw, ok] = await m.exposition.ItemDBText('UNIQUE', "Shaper's Touch, Crusader Gloves");
      expect(ok).toBe(true);
      (await m.exposition.EquipItem(build, await m.exposition.NormaliseDisplayItem(build, raw, true), 'Gloves')).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(tile(page, 'Gloves')).toHaveAccessibleName("Gloves: Shaper's Touch");
  await expect(page.getByRole('button', { name: "Undo: Equipped Shaper's Touch in Gloves (Ctrl+Z)" })).toBeEnabled();

  await page.getByRole('button', { name: 'Craft item…' }).click();
  craft = page.getByRole('dialog', { name: 'Craft Item' });
  await craft.getByRole('combobox', { name: 'Rarity' }).selectOption('Unique');
  const unique = craft.getByRole('combobox', { name: 'Unique' });
  await unique.click();
  await page.getByRole('searchbox', { name: 'Filter Unique' }).fill("kaom's heart");
  await page.getByRole('option', { name: "Kaom's Heart", exact: true }).click();
  await expect(craft.getByRole('group', { name: 'Base preview' })).toContainText("Kaom's Heart");
  await craft.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('article', { name: "Item: Kaom's Heart" })).toBeVisible();
  await page.getByRole('button', { name: 'Edit as text' }).click();
  await expect(page.getByRole('textbox', { name: /Item text/ })).toHaveValue(/^Rarity: UNIQUE\nKaom's Heart\nGlorious Plate\n/);
  app.assertNoErrors();
});

// ItemsTab.lua:2655, Item.lua:960-991, 1772-1780
test('an unnamed crafted item is named after its rarity and base, numbered when the build has that name, and keeps it through a save (0280)', async ({
  page
}) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file).replace(/<Slot name="Gloves" itemId="\d+"/, '<Slot name="Gloves" itemId="0"');
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const asText = page.getByRole('button', { name: 'Edit as text' });

  await tile(page, 'Gloves').click();
  let craft = page.getByRole('dialog', { name: 'Craft Item for Gloves' });
  const base = (await craft.getByRole('combobox', { name: 'Base' }).textContent())!.trim();
  await app.act(
    'craft-equip',
    () => craft.getByRole('button', { name: 'Create' }).click(),
    async () => {
      const [rare, ok] = await m.exposition.CraftItemText(2, `Rare ${base}`, base);
      expect(ok).toBe(true);
      (await m.exposition.EquipItem(build, rare, 'Gloves')).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(tile(page, 'Gloves')).toHaveAccessibleName(`Gloves: Rare ${base}`);
  await expect(page.getByRole('button', { name: `Undo: Equipped Rare ${base} in Gloves (Ctrl+Z)` })).toBeEnabled();
  const pageXml = await go.decode(await app.exportCode());
  await app.goTo('Items');
  expect(pageXml, 'the equipped item is titled after its rarity and base').toMatch(
    new RegExp(`Rarity: RARE\\s*\\n\\s*Rare ${escapeRe(base)}\\s*\\n\\s*${escapeRe(base)}\\s*\\n`)
  );

  await page.getByRole('button', { name: 'Craft item…' }).click();
  craft = page.getByRole('dialog', { name: 'Craft Item' });
  await choose(craft.getByRole('combobox', { name: 'Type' }), 'Gloves');
  await choose(craft.getByRole('combobox', { name: 'Base' }), base);
  await craft.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('article', { name: `Item: Rare ${base} 2` })).toBeVisible();

  await page.getByRole('button', { name: 'Craft item…' }).click();
  await craft.getByRole('combobox', { name: 'Rarity' }).selectOption('Magic');
  await choose(craft.getByRole('combobox', { name: 'Type' }), 'Gloves');
  await choose(craft.getByRole('combobox', { name: 'Base' }), base);
  await craft.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('article', { name: `Item: Magic ${base}` })).toBeVisible();
  await asText.click();
  const magic = await box.inputValue();
  const parsed = (await m.exposition.ParseItem(build, magic, go.NO_TIP_STATS)).unwrap();
  expect([parsed.Name, parsed.BaseName]).toEqual([`Magic ${base}`, base]);
  app.assertNoErrors();
});

test.describe('touch', () => {
  test.use({ hasTouch: true });
  // ItemSlotControl.lua:48-56
  test("a tree socket's jewel list previews the tapped entry and picks it on a second tap (0339)", async ({ page }) => {
    const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
    const { app, build } = await setup(page, file);
    const m = await go.model();
    const sockets = (await m.exposition.ItemsTabSlots(build)).Sockets ?? [];
    const k = sockets.findIndex((s) => s.NodeID === 10532);
    expect(k, 'vacuity: 10532 is not a socket slot').toBeGreaterThanOrEqual(0);
    const text = go.fresh(build, (b) => (b.Items.Items ?? []).find((it) => it.ID === 22)?.Text ?? '');
    const p = (await m.exposition.ParseItem(build, text.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&'), go.NO_TIP_STATS)).unwrap();
    const name22 = p.Title || p.Name;
    const select = page.getByRole('region', { name: 'Tree jewels' }).getByRole('combobox', { name: `Socket #${k + 1}` });
    await app.act(
      'socket-jewel',
      () => choose(select, 'None'),
      async () => (expect((await m.exposition.SocketTreeJewel(build, 10532, 0)).unwrap()).toBe(true), expectedLines(await go.outputs(build)))
    );

    await select.tap();
    const list = page.locator(`[id="${await select.getAttribute('aria-controls')}"]`);
    const entry = list.getByRole('option', { name: name22 }).first();
    await entry.tap();
    await expect(page.getByRole('tooltip').filter({ hasText: name22 })).toBeVisible();
    await expect(select).toHaveText('None');
    await app.act(
      'socket-jewel',
      () => entry.tap(),
      async () => (expect((await m.exposition.SocketTreeJewel(build, 10532, 22)).unwrap()).toBe(true), expectedLines(await go.outputs(build)))
    );
    await expect(select).toHaveText(name22);
    app.assertNoErrors();
  });
});
