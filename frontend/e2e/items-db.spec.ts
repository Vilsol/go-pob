// ItemsTab.lua:273-296

import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { choose } from './support/dropdown';
import { plainText } from '../src/lib/design/crafting';
import * as tree from './support/tree_model';
import { frameCount, installTreeProbe, lastFrame, waitFrame } from './support/tree_probe';
import { canvasBox, clusterPlaced, emptySpot, hitTest, placedNodes, screenOf, wheelAt } from './support/tree_view';

const board = (page: Page) => page.getByRole('region', { name: 'Equipped items' });

const setup = async (page: Page, file: string) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  const t0 = await app.now();
  await app.goTo('Items');
  await app.followWorker(t0, () => expect(board(page)).toBeVisible());
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  return { app, build, t0 };
};

const filter = (f: Partial<{ Slot: number; Type: number; League: number; Requirement: number; Obtainable: number; Search: string; SearchMode: number }>) => ({
  Slot: 1,
  Type: 1,
  League: 1,
  Requirement: 1,
  Obtainable: 1,
  Search: '',
  SearchMode: 1,
  ...f
});

test('the unique and rare DBs list what the model lists, filtered and sorted (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build, t0 } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const stats = await go.itemTooltipStats(build);
  const level = go.fresh(build, (b) => b.Build.Level);
  const rows = (kind: string, f: ReturnType<typeof filter>, stat = '') =>
    (misc.ItemDBRows(kind, f, level, stats.Str, stats.Dex, stats.Int, stat) ?? []).map((r) => (kind === 'RARE' ? r.Name : r.Title || r.Name));
  const list = page.getByRole('table', { name: 'Unique list' });
  const names = (l = list) => l.getByRole('row').getByRole('cell').locator('button[data-pob-color]');

  // ItemDBControl.lua:112-126
  const all = rows('UNIQUE', filter({}));
  expect(all.length, 'vacuity: the unique DB is empty').toBeGreaterThan(1000);
  await app.followWorker(t0, () => expect(names()).toHaveCount(all.length));
  await expect(names().first()).toHaveText(all[0]);

  const filters = page.getByRole('complementary', { name: 'Filters' });
  await filters.getByRole('combobox', { name: 'Slot' }).selectOption({ label: 'Helmet' });
  await filters.getByRole('searchbox', { name: 'Search uniques' }).fill('life');
  const helmets = rows('UNIQUE', filter({ Slot: 4, Search: 'life' }));
  expect(helmets.length).toBeGreaterThan(5);
  await expect(names()).toHaveText(helmets);

  await filters.getByRole('combobox', { name: 'Sort' }).selectOption({ label: 'Sort by Life' });
  const byLife = rows('UNIQUE', filter({ Slot: 4, Search: 'life' }), 'Life');
  expect(byLife).not.toEqual(helmets);
  await expect(names()).toHaveText(byLife);

  // ItemsTab.lua:273-296
  // Item.lua:1572
  await page.getByRole('tablist', { name: 'Item database' }).getByRole('tab', { name: 'Templates' }).click();
  const rares = page.getByRole('table', { name: 'Template list' });
  const templates = rows('RARE', filter({}));
  expect(templates[0], 'a template name carries its base').toMatch(/, /);
  await expect(names(rares)).toHaveText(templates);
  await expect(filters.getByRole('combobox', { name: 'League' })).toHaveCount(0);

  // ItemsTab.lua:2519-2603
  const base = templates[0].split(', ').at(-1)!;
  await rares
    .getByRole('button', { name: `Craft from ${base}` })
    .first()
    .click();
  const craft = page.getByRole('dialog', { name: 'Craft Item' });
  await expect(craft.getByRole('combobox', { name: 'Base' })).toHaveText(base);
  await craft.getByRole('button', { name: 'Cancel' }).click();
  app.assertNoErrors();
});

test('Equip, Add, open and the tooltip of a DB item (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const filters = page.getByRole('complementary', { name: 'Filters' });
  await filters.getByRole('searchbox', { name: 'Search uniques' }).fill("kaom's heart");
  const list = page.getByRole('table', { name: 'Unique list' });
  const name = "Kaom's Heart, Glorious Plate";

  // ItemDBControl.lua:320-328
  await list.getByRole('button', { name: "Kaom's Heart", exact: true }).hover();
  const tip = page.getByRole('tooltip');
  await expect(tip).toContainText('Variant: Current (3 variants)');
  await expect(tip).toContainText('Equipping this item in Body Armour will give you');

  await app.act(
    'db-equip',
    () => list.getByRole('button', { name: `Equip ${name}` }).click(),
    async () => {
      expect((await m.exposition.AddDBItem(build, 'UNIQUE', name, true, false))[1]).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  const body = ((await m.exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === 'Body Armour');
  expect(body?.Name).toBe(name);
  await expect(page.getByRole('region', { name: 'Uniques' }).getByRole('status')).toContainText(`Added ${name}`);

  // ItemListControl.lua:275-284
  const items = () => go.fresh(build, (b) => b.Items.Items.length);
  const n = items();
  expect((await m.exposition.AddDBItem(build, 'UNIQUE', name, false, false))[1]).toBe(true);
  expect(items()).toBe(n + 1);
  await list.getByRole('button', { name: `Add ${name}` }).click();
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Stash' }).click();
  await expect(page.getByRole('table', { name: 'Stash' }).getByRole('button', { name: "Kaom's Heart", exact: true })).toHaveCount(1);

  // ItemsTab.lua:4748-4794
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  await filters.getByRole('searchbox', { name: 'Search uniques' }).fill('hiltless');
  await list.getByRole('button', { name: 'Hiltless', exact: true }).hover();
  await expect(page.getByRole('tooltip')).toContainText('Tip: Hold Shift to display a tooltip for the granted skill.');
  await expect(page.getByRole('group', { name: 'Granted skill' })).toHaveCount(0);
  await page.keyboard.down('Shift');
  await expect(page.getByRole('group', { name: 'Granted skill' })).toContainText('Lifetap');
  await page.keyboard.up('Shift');
  await expect(page.getByRole('group', { name: 'Granted skill' })).toHaveCount(0);
  await filters.getByRole('searchbox', { name: 'Search uniques' }).fill("kaom's heart");

  await list.getByRole('button', { name: "Kaom's Heart", exact: true }).dblclick();
  const [text] = await m.exposition.ItemDBText('UNIQUE', name);
  await expect(page.getByRole('textbox', { name: /Item text/ })).toHaveValue(text);
  await expect(board(page).getByRole('button', { name: /^Body Armour: / })).toHaveAccessibleName("Body Armour: Kaom's Heart");
  app.assertNoErrors();
});

test('item set tooltips on the set select and the Manage list (0339)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  const m = await go.model();
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Equipped' }).click();
  const tips = (await m.exposition.ItemSetTooltips(build)) ?? [];
  const sets = go.fresh(build, (b) => b.Items.ItemSets.map((s) => ({ id: s.ID, title: s.Title || 'Default' })));
  const active = String(go.fresh(build, (b) => b.Items.ActiveItemSet));
  const lines = (id: string) => (tips.find((t) => t.ID === id)?.Lines ?? []).map(plainText);
  expect(lines(active).length, 'vacuity: the active set is empty').toBeGreaterThan(3);

  // ItemsTab.lua:105-112
  await page.getByRole('combobox', { name: 'Item set' }).hover();
  const activeTitle = sets.find((s) => s.id === active)!.title;
  const tip = page.getByRole('tooltip', { name: `Item set ${activeTitle}` });
  await expect(tip.locator(':scope > span')).toHaveText(lines(active));

  // ItemSetListControl.lua:80-83
  await page.getByRole('button', { name: 'Manage…' }).click();
  const other = sets.find((s) => s.id !== active)!;
  await page.getByRole('listbox', { name: 'Item sets' }).getByRole('option', { name: other.title }).hover();
  await expect(page.getByRole('tooltip', { name: `Item set ${other.title}` }).locator(':scope > span')).toHaveText(lines(other.id));
  app.assertNoErrors();
});

test('Foulborn toggle and variant groups in the item controls (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const search = page.getByRole('complementary', { name: 'Filters' }).getByRole('searchbox', { name: 'Search uniques' });
  const list = page.getByRole('table', { name: 'Unique list' });
  const controls = page.getByRole('region', { name: 'Item controls' });
  const open = async (title: string) => {
    await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
    await search.fill(title.toLowerCase());
    await list.getByRole('button', { name: title, exact: true }).dblclick();
    await expect(box).not.toHaveValue('');
    return box.inputValue();
  };

  // ItemsTab.lua:1034-1043
  const witchbane = await open('Witchbane');
  const [info] = await m.exposition.DisplayItem(witchbane);
  const k = (info.Mods ?? []).findIndex((x) => x.Mutable);
  expect(k, 'vacuity: Witchbane has no Foulborn line').toBeGreaterThanOrEqual(0);
  await controls.getByRole('checkbox', { name: `Foulborn: ${plainText(info.Mods![k].Line)}` }).check();
  const mutated = (await m.exposition.EditItem(witchbane, { Kind: 'mutate', Index: k, Value: 0, Second: 0, Float: 0, Text: '' })).unwrap();
  await expect(box).toHaveValue(mutated);
  await expect(controls.getByRole('checkbox', { name: /^Foulborn: / }).first()).toBeChecked();

  // ItemsTab.lua:1871-1885
  const zana = await open("Zana's Ingenuity");
  await controls.getByRole('combobox', { name: 'Variant', exact: true }).selectOption({ label: 'Mana' });
  const withMana = (await m.exposition.EditItem(zana, { Kind: 'variantGroup', Index: 1, Value: 3, Second: 0, Float: 0, Text: '' })).unwrap();
  await expect(box).toHaveValue(withMana);
  app.assertNoErrors();
});

test('drag a DB item onto a slot and onto the stash (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const name = "Kaom's Heart, Glorious Plate";
  await page.getByRole('complementary', { name: 'Filters' }).getByRole('searchbox', { name: 'Search uniques' }).fill("kaom's heart");
  const source = page.getByRole('table', { name: 'Unique list' }).getByRole('button', { name: "Kaom's Heart", exact: true });
  const drag = async (to: () => ReturnType<Page['getByRole']>, tab: string) => {
    await source.hover();
    await page.mouse.down();
    await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: tab }).hover();
    await page
      .getByRole('tablist', { name: 'Item views' })
      .getByRole('tab', { name: tab })
      .hover({ position: { x: 5, y: 5 } });
    await to().hover();
    await page.mouse.up();
  };

  // ItemSlotControl.lua:114-130
  await app.act(
    'db-drop-slot',
    () => drag(() => board(page).getByRole('group', { name: 'Body Armour slot' }), 'Equipped'),
    async () => {
      expect(await m.exposition.DropItemOnSlot(build, 'Body Armour', 0, 'UNIQUE', name)).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(board(page).getByRole('button', { name: /^Body Armour: / })).toHaveAccessibleName("Body Armour: Kaom's Heart");

  // ItemListControl.lua:275-284
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  await drag(() => page.getByRole('region', { name: 'Stash items' }), 'Stash');
  await expect(page.getByRole('table', { name: 'Stash' }).getByRole('button', { name: "Kaom's Heart", exact: true })).toHaveCount(1);
  app.assertNoErrors();
});

test('the stash loadout filter lists what the model lists (0339)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  const m = await go.model();
  await page
    .getByRole('tablist', { name: 'Item views' })
    .getByRole('tab', { name: /^Stash/ })
    .click();
  const loadout = page.getByRole('combobox', { name: 'Loadout filter' });
  const filters = (await m.exposition.ItemListLoadoutFilters(build)) ?? [];
  await expect(loadout.locator('option')).toHaveText(filters);
  const rows = page
    .getByRole('table', { name: 'Stash' })
    .getByRole('row')
    .filter({ has: page.locator('button[data-pob-color]') });
  for (const f of ['Unused Items', 'Current Loadout', filters[filters.length - 1]]) {
    await loadout.selectOption(f);
    const ids = (await m.exposition.ItemListFiltered(build, f)) ?? [];
    await expect(rows).toHaveCount(ids.length);
  }
  app.assertNoErrors();
});

test('the passive tree select and the Tree page socket jump (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Equipped' }).click();
  const select = page.getByRole('combobox', { name: 'Passive tree' });
  const specs = (await m.exposition.GetTreeSpecs(build)).Specs ?? [];
  await expect(select.locator('option')).toHaveText(specs.map((s) => s.Label));
  await expect(select).toHaveValue(String(go.fresh(build, (b) => b.Tree.ActiveSpec)));

  // ItemsTab.lua:193-198
  const other = go.fresh(build, (b) => (b.Tree.ActiveSpec === 1 ? 2 : 1));
  await app.act(
    'items-tree-select',
    () => select.selectOption(String(other)),
    async () => {
      await m.exposition.SetActiveTreeSpec(build, other);
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(select).toHaveValue(String(other));

  // ItemsTab.lua:203-205
  await page.getByRole('button', { name: 'Manage passive trees' }).click();
  await expect(page).toHaveURL(/\/tree\/?$/);
  const manager = page.getByRole('dialog', { name: 'Manage passive trees' });
  await expect(manager).toBeVisible();
  await expect(manager.getByRole('option')).toHaveCount(specs.length);
  app.assertNoErrors();
});

test('shared item sets: share, import, rename, delete (0339)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Equipped' }).click();
  const setSelect = page.getByRole('combobox', { name: 'Item set' });
  const before = await setSelect.locator('option').count();
  const active = String(go.fresh(build, (b) => b.Items.ActiveItemSet));
  const title = go.fresh(build, (b) => b.Items.ItemSets.find((s) => s.ID === active)?.Title ?? 'Default');

  await page.getByRole('button', { name: 'Manage…' }).click();
  const dialog = page.getByRole('dialog', { name: 'Manage item sets' });
  const region = dialog.getByRole('region', { name: 'Shared item sets' });
  const shared = region.getByRole('listbox', { name: 'Shared sets' });
  // SharedItemSetListControl.lua:76-93
  await region.getByRole('button', { name: 'Share set' }).click();
  await expect(shared.getByRole('option')).toHaveText([title]);
  await shared.getByRole('option', { name: title }).hover();
  await expect(page.getByRole('tooltip', { name: `Item set ${title}` })).toContainText('Body Armour:');

  // ItemSetListControl.lua:93-106
  await shared.getByRole('option', { name: title }).click();
  await app.act(
    'import-shared-set',
    () => region.getByRole('button', { name: 'Import' }).click(),
    async () => expectedLines(await go.outputs(build))
  );
  await expect(dialog.getByRole('listbox', { name: 'Item sets' }).getByRole('option')).toHaveCount(before + 1);

  await region.getByRole('button', { name: 'Retitle' }).click();
  await region.getByRole('textbox', { name: 'Name for this shared set' }).fill('Shared copy');
  await region.getByRole('button', { name: 'Save' }).click();
  await expect(shared.getByRole('option')).toHaveText(['Shared copy']);
  await shared.getByRole('option', { name: 'Shared copy' }).click();
  await region.getByRole('button', { name: 'Remove' }).click();
  await region.getByRole('group', { name: 'Confirm shared delete' }).getByRole('button', { name: 'Delete' }).click();
  await expect(shared.getByRole('option')).toHaveCount(0);
  app.assertNoErrors();
});

const nodeOnScreen = async (page: Page, id: number, build: Awaited<ReturnType<typeof go.loadXml>>) => {
  const t = tree.rawTree();
  const cluster = (await (await go.model()).exposition.GetTreeClusterNodes(build)) ?? [];
  const placed = [...placedNodes(t), ...clusterPlaced(cluster)];
  const target = placed.find((p) => p.node.skill === id);
  if (!target) throw new Error(`node ${id} is not drawn`);
  for (let i = 0; i < 12; i++) {
    const view = (await lastFrame(page))!.view;
    const s = screenOf(t, view, target.at);
    const inside = s.x > 80 && s.y > 80 && s.x < view.width - 80 && s.y < view.height - 80;
    if (inside && hitTest(t, placed, view, s) === target) return s;
    if (!inside) {
      const from = emptySpot(t, placed, view);
      const box = await canvasBox(page);
      const n = await frameCount(page);
      await page.mouse.move(box.x + from.x, box.y + from.y);
      await page.mouse.down();
      const dx = view.width / 2 - s.x;
      const dy = view.height / 2 - s.y;
      for (let k = 1; k <= 10; k++) await page.mouse.move(box.x + from.x + (dx * k) / 10, box.y + from.y + (dy * k) / 10);
      await page.mouse.up();
      await waitFrame(page, n, '(f) => true');
    } else {
      await wheelAt(page, s, -2);
    }
  }
  throw new Error(`node ${id} could not be brought under the pointer`);
};

// PassiveTreeView.lua:511-519
test('the Tree socket jump lands on the Items socket slot (0008)', async ({ page }) => {
  test.setTimeout(300_000);
  await installTreeProbe(page);
  const file = '0008-necromancer-vaal-summon-skeletons';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await tree.prepareTree();
  const active = go.fresh(build, (b) => [...(b.Build.PassiveNodes ?? [])]);
  await app.goTo('Tree');
  await waitFrame(page, 0, `(f) => f.className !== undefined && f.active === ${active.length}`, 60_000);
  const m = await go.model();
  const socket = 2311;
  const at = await nodeOnScreen(page, socket, build);
  const box = await canvasBox(page);
  await page.mouse.click(box.x + at.x, box.y + at.y, { button: 'right' });
  await page.getByRole('menu').getByRole('menuitem', { name: 'Socket a jewel…' }).click();
  await expect(page).toHaveURL(new RegExp(`/items\\?socket=${socket}$`));
  const select = page.locator(`[data-socket-node="${socket}"] [role="combobox"]`);
  await expect(select).toBeFocused();
  await expect(select).toHaveAttribute('aria-expanded', 'true');
  await select.press('Escape');
  // ItemSlotControl.lua:132-149
  await page.locator(`[data-socket-node="${socket}"]`).hover();
  await expect(page.getByRole('img', { name: /^Passive tree around socket #/ })).toBeVisible();
  await expect(page.locator(`circle[data-node="${socket}"][data-alloc="true"]`)).toHaveCount(1);
  // ItemSlotControl.lua:139-146
  await expect(page.locator('path[data-edge]').first()).toBeAttached();
  const slotBox = (await page.locator(`[data-socket-node="${socket}"]`).boundingBox())!;
  const viewBox = (await page.locator('[data-socket-preview]').boundingBox())!;
  expect(viewBox.y + viewBox.height <= slotBox.y || viewBox.y >= slotBox.y + slotBox.height).toBe(true);
  expect(viewBox.y).toBeGreaterThanOrEqual(0);
  await app.act(
    'unsocket-jewel',
    () => choose(select, 'None'),
    async () => {
      (await m.exposition.SocketTreeJewel(build, socket, 0)).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await app.act(
    'socket-jewel',
    async () => {
      const text = go.fresh(build, (b) => (b.Items.Items ?? []).find((it) => it.ID === 12)?.Text ?? '');
      const p = (await m.exposition.ParseItem(build, text.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&'), go.NO_TIP_STATS)).unwrap();
      const list = page.locator(`[id="${await select.getAttribute('aria-controls')}"]`);
      await select.click();
      await list
        .getByRole('option', { name: p.Title || p.Name, exact: true })
        .first()
        .click();
    },
    async () => {
      (await m.exposition.SocketTreeJewel(build, socket, 12)).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  app.assertNoErrors();
});
