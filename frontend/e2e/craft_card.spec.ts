import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { layoutFindings } from './support/layout';
import { plainText, statTemplate } from '../src/lib/design/crafting';

const BUILD = '0280-elementalist-fireball';
const RANGED = `Rarity: RARE
Gale Pendant
Onyx Amulet
Implicits: 1
+(10-16) to all Attributes
+(80-100) to maximum Life
+(30-40)% to Fire Resistance`;

const setup = async (page: Page, layout: 'stepped' | 'tabs' = 'stepped') => {
  await page.addInitScript((l) => localStorage.setItem('options:craftCardLayout', l), layout);
  const xml = buildXml(BUILD);
  const build = await go.loadXml(xml);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await expect(page.getByRole('region', { name: 'Equipped items' })).toBeVisible();
  return app;
};

const craftedHelmet = async () => {
  const m = await go.model();
  const [blank] = await m.exposition.CraftItemText(2, 'Doom Crown', 'Hubris Circlet');
  const controls = async (text: string) => (await m.exposition.ItemAffixControls(text))[0].Controls!;
  const pick = async (text: string, slot: number, want: (label: string) => boolean) => {
    const c = (await controls(text))[slot];
    const k = c.Series!.findIndex((s) => want(plainText(s.Label)));
    expect(k, `slot ${slot}`).toBeGreaterThan(0);
    const [t, ok] = await m.exposition.AffixEditText(text, slot, k, 0.7);
    expect(ok).toBe(true);
    return t;
  };
  let text = await pick(blank, 0, (l) => l === '+# to maximum Life');
  text = await pick(text, 1, (l) => l.includes('/'));
  const suffix = (await controls(text)).findIndex((c) => c.Type === 'Suffix');
  text = await pick(text, suffix, (l) => l.includes('Resistance'));
  return { m, text, controls: await controls(text) };
};

for (const layout of ['stepped', 'tabs'] as const)
  test(`${layout}: the crafting card at 960×1080 has nothing clipped or overlapping (0280)`, async ({ page }) => {
    test.setTimeout(180_000);
    await page.setViewportSize({ width: 960, height: 1080 });
    const app = await setup(page, layout);
    const { text } = await craftedHelmet();
    await page.getByRole('textbox', { name: /Item text/ }).fill(text);
    const card = page.getByRole('group', { name: 'Crafting card' });
    await expect(
      card
        .getByRole('combobox', { name: 'Affix 1' })
        .or(card.getByRole('tab', { name: /Base/ }))
        .first()
    ).toBeVisible();
    const tabs = await card.getByRole('tab').all();
    for (const tab of tabs.length ? tabs : [undefined]) {
      await tab?.click();
      expect(await layoutFindings(page, '[aria-label="Crafting card"]', []), `${layout} ${(await tab?.textContent()) ?? ''}`).toEqual([]);
    }
    app.assertNoErrors();
  });

test('the stepped card pairs Base | Implicits and Affixes | Result once it is 760px wide (0280)', async ({ page }) => {
  test.setTimeout(180_000);
  await page.setViewportSize({ width: 2560, height: 1440 });
  const app = await setup(page);
  const { text } = await craftedHelmet();
  await page.getByRole('textbox', { name: /Item text/ }).fill(text);
  const card = page.getByRole('group', { name: 'Crafting card' });
  await expect(card.getByRole('region', { name: 'Result' })).toBeVisible();
  const box = async (name: string) => (await card.getByRole('region', { name }).boundingBox())!;
  expect((await card.boundingBox())!.width).toBeGreaterThanOrEqual(760);
  const [base, implicits, affixes, result] = [await box('Base'), await box('Implicits'), await box('Affixes'), await box('Result')];
  expect(implicits.x).toBeGreaterThan(base.x + base.width);
  expect(implicits.y).toBe(base.y);
  expect(result.x).toBeGreaterThan(affixes.x + affixes.width);
  expect(result.y).toBe(affixes.y);
  expect(affixes.y).toBeGreaterThan(Math.max(base.y + base.height, implicits.y + implicits.height));

  await page.setViewportSize({ width: 960, height: 1080 });
  await expect.poll(async () => (await box('Implicits')).y).toBeGreaterThan((await box('Base')).y);
  app.assertNoErrors();
});

test('each affix unit holds Go’s lines for its dropdown, and every roll slider is described by the lines it changes (0280)', async ({ page }) => {
  test.setTimeout(180_000);
  const app = await setup(page);
  const { m, text, controls } = await craftedHelmet();
  const box = page.getByRole('textbox', { name: /Item text/ });
  await box.fill(text);
  const affixes = page.getByRole('group', { name: 'Affixes' });
  await expect(affixes.getByRole('combobox', { name: 'Affix 1' })).toBeVisible();
  const described = (name: string) =>
    affixes.getByRole('slider', { name, exact: true }).evaluate((s) =>
      (s.getAttribute('aria-describedby') ?? '')
        .split(' ')
        .filter(Boolean)
        .map((id) => document.getElementById(id)?.textContent ?? '')
    );

  const info = (await m.exposition.DisplayItem(text))[0];
  const goLines = info.Mods!.filter((x) => x.Section === 'explicit').map((x) => plainText(x.Line));
  for (const [i, c] of controls.entries()) {
    if (!c.Sel) continue;
    const want = plainText(c.Series![c.Sel].Label).split('/').map(statTemplate);
    const lines = await described(`Roll of affix ${i + 1}`);
    expect(lines.map(statTemplate), `affix ${i + 1}`).toEqual(want);
    for (const l of lines) expect(goLines).toContain(l);
    await expect(affixes.getByRole('list', { name: `Lines of affix ${i + 1}` }).getByRole('listitem')).toHaveCount(want.length);
  }
  expect(controls[1].Series![controls[1].Sel].Label).toContain('/');

  // Item.lua:2085-2093
  const roll = affixes.getByRole('slider', { name: 'Roll of affix 2' });
  await roll.fill('0.95');
  const [rolled] = await m.exposition.AffixEditText(text, 1, controls[1].Sel, 0.95);
  await expect(box).toHaveValue(rolled);
  const preview = (await m.exposition.AffixRollTooltip(text, 1, 0.95)) ?? [];
  const now = await described('Roll of affix 2');
  const shared = await described('Roll of affix 1');
  expect(now.filter((l) => !shared.includes(l)).every((l) => preview.includes(l))).toBe(true);
  expect(now.filter((l) => !shared.includes(l)).length).toBeGreaterThan(0);

  const card = page.getByRole('region', { name: 'Item controls' });
  await box.fill(RANGED);
  const shown = (await m.exposition.DisplayItem(RANGED))[0].Mods!;
  const k = shown.findIndex((x) => x.Section === 'explicit' && x.HasRange);
  expect(k).toBeGreaterThanOrEqual(0);
  const lineOf = (line: string) =>
    card
      .getByRole('slider', { name: `Roll of ${line}`, exact: true })
      .evaluate((s) => document.getElementById(s.getAttribute('aria-describedby') ?? '')?.textContent);
  expect(await lineOf(shown[k].Line)).toBe(shown[k].Line);
  const edited = (await m.exposition.EditItem(RANGED, { Kind: 'range', Index: k, Value: 0, Second: 0, Float: 0.1, Text: '' })).unwrap();
  const after = (await m.exposition.DisplayItem(edited))[0].Mods![k].Line;
  expect(after).not.toBe(shown[k].Line);
  await card.getByRole('slider', { name: `Roll of ${shown[k].Line}`, exact: true }).fill('0.1');
  await expect(box).toHaveValue(edited);
  expect(await lineOf(after)).toBe(after);
  app.assertNoErrors();
});
