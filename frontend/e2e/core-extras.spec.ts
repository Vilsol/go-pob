import { test, expect } from '@playwright/test';
import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

// SkillsTab.lua:357-383
const BUILD = '0005-occultist-vaal-double-strike';

test(`node-granted and explode group notes on ${BUILD}`, async ({ page }) => {
  const source = buildXml(BUILD);
  const model = await go.loadXml(source);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(source), async () => expectedLines(await go.outputs(model)));

  const set = model.Skills.SkillSets[model.ActiveSkillSetIndex()];
  const node = set.Skills.findIndex((g) => g.SourceAttr === 'Tree:35011');
  const explode = set.Skills.findIndex((g) => g.SourceAttr === 'Explode');
  expect(node, 'no node-granted group').toBeGreaterThanOrEqual(0);
  expect(explode, 'no explode group').toBeGreaterThanOrEqual(0);

  await app.goTo('Skills');
  const list = page.getByRole('region', { name: 'Socket groups' });
  const editor = page.getByRole('region', { name: 'Selected group' });
  await list.locator('button[aria-current]').nth(node).click();
  await expect(editor).toContainText("which is being provided by 'Huntleader'.");
  await expect(editor).toContainText('it will disappear if you un-allocate the node.');
  await list.locator('button[aria-current]').nth(explode).click();
  await expect(editor).toContainText('This is a special group created for the enemy explosion effect,');
  await expect(editor).toContainText('it will disappear if you lose the above sources.');
  await expect(editor).not.toContainText('???');

  // SkillListControl.lua:234-260
  const body = set.Skills.findIndex((g) => g.Slot === 'Body Armour');
  expect(body, 'no body armour group').toBeGreaterThanOrEqual(0);
  await expect(list.locator('button[aria-current]').nth(body).locator('img[data-slot-icon]')).toHaveAttribute('src', /slot-icons\/icon_body_armour\.png$/);
  app.assertNoErrors();
});

// ItemTools.lua:380-409, ItemDBControl.lua:376-380, Main.lua:439-448
test('F1 opens the wiki under the pointer, else the help (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await page.evaluate(() => {
    (window as unknown as { opened: string[] }).opened = [];
    window.open = (url?: string | URL) => {
      (window as unknown as { opened: string[] }).opened.push(String(url));
      return null;
    };
  });
  await app.goTo('Items');
  await page.getByRole('tablist', { name: 'Item views' }).getByRole('tab', { name: 'Browse items' }).click();
  const row = page.getByRole('table', { name: 'Unique list' }).getByRole('row').getByRole('cell').locator('button[data-pob-color]').first();
  const title = (await row.textContent())!.trim();
  await row.hover();
  await page.keyboard.press('F1');
  await expect
    .poll(() => page.evaluate(() => (window as unknown as { opened: string[] }).opened))
    .toEqual([`https://www.poewiki.net/wiki/${title.replaceAll(' ', '_')}`]);
  await expect(page.getByRole('dialog')).toHaveCount(0);

  await page.mouse.move(1, 1);
  await page.keyboard.press('F1');
  const help = page.getByRole('heading', { name: 'Items Tab' });
  await expect(help).toBeInViewport();
  app.assertNoErrors();
});

// SkillsTab.lua:242-288
test('Optimise Sockets on 0005-occultist-vaal-double-strike', async ({ page }) => {
  const file = '0005-occultist-vaal-double-strike';
  const xml = buildXml(file);
  const model = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(model)));

  const set = model.Skills.SkillSets[model.ActiveSkillSetIndex()];
  const index = set.Skills.findIndex((g) => g.Slot === 'Weapon 1');
  expect(index, 'no Weapon 1 group').toBeGreaterThanOrEqual(0);
  const m = await go.model();
  expect(await m.exposition.OptimiseSockets(model, 'Weapon 1'), 'Go refused the Weapon 1 item').toBe(true);
  const itemSet = model.Items.ItemSets.find((s) => s.ID === String(model.Items.ActiveItemSet)) ?? model.Items.ItemSets[0];
  const id = itemSet.Slots!.find((s) => s.Name === 'Weapon 1')!.ItemID;
  const want = model.Items.Items.find((it) => it.ID === id)!.Text;

  await app.goTo('Skills');
  await page.getByRole('region', { name: 'Socket groups' }).locator('button[aria-current]').nth(index).click();
  await page.getByRole('region', { name: 'Selected group' }).getByRole('button', { name: 'Match item sockets to gems' }).click();
  await expect.poll(async () => (await go.loadXml(await go.decode(await app.exportCode()))).Items.Items.find((it) => it.ID === id)?.Text).toBe(want);
  app.assertNoErrors();
});

// ItemsTab.lua:342-350, CompareBuySimilar.lua:44-203
test('Buy similar opens the trade search for the pasted item (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await page.evaluate(() => {
    (window as unknown as { opened: string[] }).opened = [];
    window.open = (url?: string | URL) => {
      (window as unknown as { opened: string[] }).opened.push(String(url));
      return null;
    };
  });
  await app.goTo('Items');
  await page.getByRole('textbox', { name: 'Item text' }).fill("Rarity: Unique\nKaom's Heart\nGlorious Plate\nHas no Sockets\n+500 to maximum Life");
  await page.getByRole('button', { name: 'Buy similar' }).click();
  const dialog = page.getByRole('dialog', { name: 'Buy Similar' });
  await dialog.getByRole('combobox', { name: 'Realm' }).selectOption({ label: 'PS4' });
  await dialog.getByRole('button', { name: 'Search' }).click();
  const opened = await page.evaluate(() => (window as unknown as { opened: string[] }).opened);
  expect(opened).toHaveLength(1);
  expect(opened[0].startsWith('https://www.pathofexile.com/trade/search/sony/Standard?q=')).toBe(true);
  const q = JSON.parse(decodeURIComponent(opened[0].split('?q=')[1])) as { query: Record<string, unknown> };
  expect(q.query).toMatchObject({ name: "Kaom's Heart", type: 'Glorious Plate', status: { option: 'securable' } });
  app.assertNoErrors();
});
