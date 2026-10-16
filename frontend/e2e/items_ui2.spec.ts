import { test, expect, type Locator, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { choose, expectEntries } from './support/dropdown';

const BUILD = '0280-elementalist-fireball';
const helmet = readFileSync(new URL('./fixtures/items/helmet.txt', import.meta.url), 'utf8');

const setup = async (page: Page, file = BUILD) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await expect(page.getByRole('region', { name: 'Equipped items' })).toBeVisible();
  return { app, build };
};

const expectUnclipped = async (box: Locator) => {
  const page = box.page();
  await box.click();
  const list = page.locator(`[id="${await box.getAttribute('aria-controls')}"]`);
  await expect(list).toBeVisible();
  const geometry = () =>
    list.evaluate((el) => {
      el.scrollTop = el.scrollHeight;
      const r = el.getBoundingClientRect();
      const owns = (x: number, y: number) => el.contains(document.elementFromPoint(x, y));
      const last = el.querySelector('[role="option"]:last-of-type')!.getBoundingClientRect();
      return {
        tall: r.height > 60,
        inside: r.left >= 0 && r.top >= 0 && r.right <= innerWidth && r.bottom <= innerHeight,
        corners: [owns(r.left + 4, r.top + 4), owns(r.right - 4, r.top + 4), owns(r.left + 4, r.bottom - 4), owns(r.right - 4, r.bottom - 4)],
        last: owns(last.left + last.width / 2, last.top + last.height / 2)
      };
    });
  await expect.poll(geometry).toEqual({ tall: true, inside: true, corners: [true, true, true, true], last: true });
  await page.keyboard.press('Escape');
  await expect(box).toHaveAttribute('aria-expanded', 'false');
};

for (const [width, height] of [
  [900, 500],
  [1600, 1000]
])
  test(`crafting dropdowns in a dialog are not clipped by it at ${width}×${height} (0280)`, async ({ page }) => {
    test.setTimeout(180_000);
    await page.setViewportSize({ width, height });
    await setup(page);
    await page.getByRole('textbox', { name: /Item text/ }).fill(helmet);
    const controls = page.getByRole('region', { name: 'Item controls' });
    const buttons = controls.getByRole('group', { name: 'Enchant, anoint and corrupt' });

    await controls.getByRole('button', { name: 'Add modifier...' }).click();
    const add = page.getByRole('dialog', { name: 'Add Modifier to Item' });
    await expectUnclipped(add.getByRole('combobox', { name: 'Modifier' }));
    await add.getByRole('button', { name: 'Cancel' }).click();
    await expect(add).toBeHidden();

    await buttons.getByRole('button', { name: 'Corrupt...' }).click();
    const corrupt = page.getByRole('dialog', { name: 'Corrupted Item' });
    await expectUnclipped(corrupt.getByRole('combobox', { name: 'Implicit #1' }));
  });

test("a crafted item's Base dropdown moves it to another base of its type, as the model does (0280)", async ({ page }) => {
  test.setTimeout(180_000);
  const { app } = await setup(page);
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const [crafted] = await m.exposition.CraftItemText(2, 'Doom Crown', 'Hubris Circlet');
  const [c] = await m.exposition.ItemAffixControls(crafted);
  const at = (slot: number, label: string) => c.Controls![slot].Series!.findIndex((s) => s.Label === label);
  const [withLife] = await m.exposition.AffixEditText(crafted, 0, at(0, '+# to maximum Life'), 0.5);
  const [withES] = await m.exposition.AffixEditText(withLife, 1, at(1, '#% increased Energy Shield'), 0.5);
  await box.fill(withES);

  const base = page.getByRole('group', { name: 'Affixes' }).getByRole('combobox', { name: 'Item base' });
  await expect(base).toHaveText('Hubris Circlet');
  await expectEntries(base, [...((await m.exposition.RebaseChoices(withES)) ?? [])]);
  await choose(base, 'Royal Burgonet');
  const [want, ok] = await m.exposition.RebaseItemText(withES, 'Royal Burgonet');
  expect(ok).toBe(true);
  expect(want.Dropped!.length, 'vacuity: the new base cannot roll one affix').toBe(1);
  await expect(box).toHaveValue(want.Text);
  await expect(base).toHaveText('Royal Burgonet');
  await expect(page.getByRole('status', { name: 'Dropped affixes' })).toContainText(want.Dropped![0]);
  await expect(page.getByRole('article', { name: 'Item: Doom Crown' })).toContainText('Royal Burgonet');
  await expect(page.getByRole('button', { name: 'Equip in Helmet' })).toBeVisible();
  app.assertNoErrors();
});

for (const [width, height, font] of [
  [1024, 700, 16],
  [1600, 1000, 0]
])
  test(`the stash row menu's entries do not wrap into each other at ${width}×${height}${font ? `, ${font}px` : ''} (0280)`, async ({ page }) => {
    test.setTimeout(180_000);
    await page.setViewportSize({ width, height });
    const { build } = await setup(page);
    if (font) await page.addStyleTag({ content: `[role="menuitem"] { font-size: ${font}px !important; }` });
    const m = await go.model();
    const ring = ((await m.exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === 'Ring 1');
    expect(ring, 'vacuity: 0280 wears a ring in Ring 1, whose menu has the longest entry').toBeTruthy();
    const name = (await m.exposition.ParseItem(build, ring!.Text.replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&'), go.NO_TIP_STATS)).unwrap();
    await page
      .getByRole('tablist', { name: 'Item views' })
      .getByRole('tab', { name: /^Stash/ })
      .click();
    const title = name.Title || name.Name;
    await page
      .getByRole('table', { name: 'Stash' })
      .getByRole('button', { name: `Actions for ${title}` })
      .click();
    const menu = page.getByRole('menu', { name: title });
    await expect(menu.getByRole('menuitem', { name: 'Equip or unequip in Ring 2 (Ctrl+Shift+click)' })).toBeVisible();
    const boxes = await menu.getByRole('menuitem').evaluateAll((els) =>
      els.map((el) => {
        const r = el.getBoundingClientRect();
        return {
          label: el.textContent,
          top: r.top,
          bottom: r.bottom,
          left: r.left,
          right: r.right,
          fits: el.scrollWidth <= el.clientWidth && el.scrollHeight <= el.clientHeight
        };
      })
    );
    expect(boxes.length).toBeGreaterThan(3);
    for (const [i, b] of boxes.entries()) {
      expect(b.fits, `"${b.label}" fits its entry`).toBe(true);
      expect([b.left >= 0, b.right <= width], `"${b.label}" inside the viewport`).toEqual([true, true]);
      for (const o of boxes.slice(i + 1)) expect(b.bottom <= o.top || o.bottom <= b.top, `"${b.label}" and "${o.label}" overlap`).toBe(true);
    }
  });

// ItemsTab.lua:3651-3654, 4368
test("an equipped item's Add modifier preview shows its slot's impact (0339)", async ({ page }) => {
  test.setTimeout(180_000);
  const { app } = await setup(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  await page
    .getByRole('region', { name: 'Equipped items' })
    .getByRole('button', { name: /^Helmet: / })
    .click();
  await page
    .getByRole('dialog', { name: /^Item / })
    .getByRole('button', { name: 'Mod table' })
    .click();
  const dialog = page.getByRole('dialog', { name: 'Add Modifier to Item' });
  await dialog.getByRole('combobox', { name: 'Source' }).selectOption({ label: 'Custom' });
  await dialog.getByRole('textbox', { name: 'Custom modifier' }).fill('+7 to Dexterity');
  const preview = dialog.getByRole('group', { name: 'New item' });
  await expect(preview.getByRole('group', { name: 'Stat differences' })).toContainText('Equipping this item in Helmet will give you:');
  await expect(preview.getByRole('group', { name: 'Stat differences' })).toContainText('Dexterity');
  await expect(preview).not.toContainText('Fits no equipment slot');
  app.assertNoErrors();
});
