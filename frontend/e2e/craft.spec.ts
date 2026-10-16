import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import type * as Types from '../src/lib/types';
import { plainText, sortByValue } from '../src/lib/design/crafting';
import { choose, entries, expectEntries } from './support/dropdown';

const setup = async (page: Page, file: string) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await expect(page.getByRole('region', { name: 'Equipped items' })).toBeVisible();
  return { app, build };
};

const AMULET = `Rarity: RARE
Gale Pendant
Onyx Amulet
Implicits: 1
+12 to all Attributes
+70 to maximum Life`;

test('enchant and anoint the display item (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const box = page.getByRole('textbox', { name: /Item text/ });
  const controls = page.getByRole('region', { name: 'Item controls' });
  const edit = async (text: string, e: Partial<Types.calculator.ItemEdit>) =>
    (await m.exposition.EditItem(text, { Kind: '', Index: 0, Value: 0, Second: 0, Float: 0, Text: '', ...e })).unwrap();
  const helmet = readFileSync(new URL('./fixtures/items/helmet.txt', import.meta.url), 'utf8');

  // ItemsTab.lua:509-520
  await box.fill(helmet);
  const buttons = controls.getByRole('group', { name: 'Enchant, anoint and corrupt' });
  await expect(buttons.getByRole('button')).toHaveText(['Change Enchantment...', 'Corrupt...', 'Add Implicit...']);

  // ItemsTab.lua:2815-2839
  await buttons.getByRole('button', { name: 'Change Enchantment...' }).click();
  const dialog = page.getByRole('dialog', { name: 'Enchant Item' });
  const [choices] = await m.exposition.ItemEnchantChoices(build, helmet);
  const skills = await dialog.getByRole('combobox', { name: 'Skill' }).locator('option').allTextContents();
  expect(skills).toEqual(choices.Used);
  const enchantment = dialog.getByRole('combobox', { name: 'Enchantment' });
  await expect(enchantment).not.toHaveText('');
  const line = (await entries(enchantment))[0];
  await expect(dialog.getByRole('group', { name: 'Enchanted item' })).toContainText(line);

  // ItemsTab.lua:2754-2814
  const pick = choices
    .Used!.flatMap((skill) =>
      choices
        .Skills!.find((s) => s.Skill === skill)!
        .Sources!.map((source, index) => {
          const lines = [...(source.Lines ?? [])];
          const values = [...(misc.EnchantSortValues(helmet, 1, lines, 'FullDPS') ?? [])];
          return { skill, index, lines, values, moved: sortByValue(values, lines.length).some((k, i) => k !== i) };
        })
    )
    .find((x) => x.moved);
  if (!pick) throw new Error('REFUSING: no used skill whose enchantments Full DPS reorders');
  await dialog.getByRole('combobox', { name: 'Skill' }).selectOption({ label: pick.skill });
  await dialog.getByRole('combobox', { name: 'Source' }).selectOption({ index: pick.index });
  const lines = await entries(enchantment);
  expect(lines).toEqual(pick.lines);
  await dialog.getByRole('combobox', { name: 'Sort by' }).selectOption({ label: 'Full DPS' });
  await expect(enchantment).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
  const values = pick.values;
  await expectEntries(
    enchantment,
    sortByValue(values, lines.length).map((k) => lines[k])
  );
  const top = (await entries(enchantment))[0];
  await dialog.getByRole('button', { name: 'Enchant', exact: true }).click();
  const enchanted = await edit(helmet, { Kind: 'enchant', Index: 1, Text: top });
  await expect(box).toHaveValue(enchanted);

  // ItemsTab.lua:2862-2865
  await buttons.getByRole('button', { name: 'Change Enchantment...' }).click();
  await dialog.getByRole('button', { name: 'Remove' }).click();
  await expect(box).toHaveValue(await edit(enchanted, { Kind: 'enchant', Index: 1, Value: 1 }));

  // ItemsTab.lua:521-526
  await box.fill(AMULET);
  await expect(buttons.getByRole('button')).toHaveText(['Anoint...', 'Corrupt...', 'Add Implicit...']);
  await buttons.getByRole('button', { name: 'Anoint...' }).click();
  const anoint = page.getByRole('dialog', { name: 'Anoint Item' });
  await expect(anoint.getByRole('button', { name: 'No Anoint' })).toBeVisible();
  const list = anoint.getByRole('listbox', { name: 'Notables' });

  // NotableDBControl.lua:50-79, 113-182
  await anoint.getByRole('textbox', { name: 'Search notables' }).fill('maximum life');
  await anoint.getByRole('combobox', { name: 'Search in' }).selectOption('Modifiers');
  await expect(list.getByRole('option').first()).toBeVisible();
  const names = await list.getByRole('option').allTextContents();
  expect(names.length).toBeGreaterThan(1);
  const notables = (await m.exposition.AnointNotables())!.filter((n) => names.includes(n.Name));
  await anoint.getByRole('combobox', { name: 'Sort notables' }).selectOption({ label: 'Sort by Life' });
  await expect(list).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
  const powers = [
    ...(misc.AnointPowers(
      AMULET,
      1,
      notables.map((n) => n.ID),
      'Life'
    ) ?? [])
  ];
  expect(new Set(powers).size).toBeGreaterThan(1);
  const byPower = notables.map((n, k) => ({ name: n.Name, p: powers[k] })).sort((a, b) => (a.p !== b.p ? b.p - a.p : a.name < b.name ? -1 : 1));
  await expect(list.getByRole('option')).toHaveText(byPower.map((x) => x.name));

  // ItemsTab.lua:2925-2963
  const best = byPower[0].name;
  await list.getByRole('option', { name: best, exact: true }).click();
  await expect(anoint.getByRole('region', { name: 'Anoint comparison' })).toContainText(`Anointing ${best} will give you:`);
  await anoint.getByRole('button', { name: `Anoint ${best}` }).click();
  const anointed = await edit(AMULET, { Kind: 'anoint', Index: 1, Text: best });
  await expect(box).toHaveValue(anointed);

  // NotableDBControl.lua:246-256
  const all = (await m.exposition.AnointNotables())!;
  const odd = all.find((n) => n.Unsupported?.some(Boolean))!;
  await buttons.getByRole('button', { name: 'Anoint...' }).click();
  await anoint.getByRole('combobox', { name: 'Search in' }).selectOption('Names');
  await anoint.getByRole('textbox', { name: 'Search notables' }).fill(odd.Name);
  await list.getByRole('option', { name: odd.Name, exact: true }).click();
  await expect(anoint.getByRole('tooltip', { name: 'Notable' }).locator('[data-unsupported]')).toHaveCount(odd.Unsupported!.filter(Boolean).length);
  await anoint.getByRole('button', { name: 'Cancel' }).click();

  // ItemsTab.lua:2989-2992
  await buttons.getByRole('button', { name: 'Anoint...' }).click();
  await anoint.getByRole('button', { name: `Remove ${best}` }).click();
  await expect(box).toHaveValue(await edit(anointed, { Kind: 'anoint', Index: 1 }));

  misc.release();
  app.assertNoErrors();
});

const UNIQUE = `Rarity: UNIQUE
Test Crown
Hubris Circlet
Implicits: 0
{range:0.5}+(40-60) to maximum Life
Cannot be Frozen`;

const GLIMPSE = `Rarity: UNIQUE
Glimpse of Chaos
Vaal Mask
Implicits: 0
Can be modified while Corrupted
Can have up to 5 Implicit Modifiers while Item has this Modifier`;

test('corrupt the display item (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const box = page.getByRole('textbox', { name: /Item text/ });
  const controls = page.getByRole('region', { name: 'Item controls' });
  const buttons = controls.getByRole('group', { name: 'Enchant, anoint and corrupt' });
  const helmet = readFileSync(new URL('./fixtures/items/helmet.txt', import.meta.url), 'utf8');
  const corrupt = async (text: string, source: string, ids: string[], downside: string[], rolls?: [number[], number[]]) => {
    const [t, ok] = await m.exposition.CorruptItemText(text, source, ids, downside, !!rolls, rolls?.[0] ?? [], rolls?.[1] ?? []);
    expect(ok).toBe(true);
    return t;
  };

  // ItemsTab.lua:3027-3063
  await box.fill(helmet);
  await buttons.getByRole('button', { name: 'Corrupt...' }).click();
  const dialog = page.getByRole('dialog', { name: 'Corrupted Item' });
  const [choices] = await m.exposition.ItemCorruptChoices(helmet);
  const mods = choices.Corrupted!;
  const first = dialog.getByRole('combobox', { name: 'Implicit #1' });
  await expectEntries(first, ['None', ...mods.map((x) => x.Label)]);
  await choose(first, mods[0].Label);
  const second = dialog.getByRole('combobox', { name: 'Implicit #2' });
  await expectEntries(second, ['None', ...mods.filter((x) => x.Group !== mods[0].Group).map((x) => x.Label)]);
  // ItemsTab.lua:3301-3308, 2408-2422
  await expect(dialog.getByRole('group', { name: 'Implicit comparison' })).toContainText('Adding this mod will give');

  // ItemsTab.lua:3064-3135
  await choose(first, 'None');
  await dialog.getByRole('combobox', { name: 'Sort by' }).selectOption({ label: 'Life' });
  await expect(dialog.getByRole('combobox', { name: 'Sort by' })).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
  const values = [
    ...(misc.CorruptSortValues(
      helmet,
      'Corrupted',
      'Corrupted',
      mods.map((x) => x.ID),
      'Life'
    ) ?? [])
  ];
  const order = sortByValue(values, mods.length);
  expect(order.some((k, i) => k !== i)).toBe(true);
  await expectEntries(first, ['None', ...order.map((k) => mods[k].Label)]);
  const top = mods[order[0]];
  await choose(first, top.Label);
  await dialog.getByRole('button', { name: 'Corrupted', exact: true }).click();
  await expect(box).toHaveValue(await corrupt(helmet, 'Corrupted', [top.ID], []));

  // ItemsTab.lua:3176-3254
  await box.fill(UNIQUE);
  await buttons.getByRole('button', { name: 'Corrupt...' }).click();
  await dialog.getByRole('tab', { name: 'Roll Ranges' }).click();
  const rolls = dialog.getByRole('group', { name: 'Roll Ranges' });
  await rolls.getByRole('slider').fill('1.22');
  await expect(rolls).toContainText(await m.exposition.CorruptedRangeLine(UNIQUE, 0, 1.22));
  await dialog.getByRole('button', { name: 'Corrupted', exact: true }).click();
  await expect(box).toHaveValue(await corrupt(UNIQUE, 'Corrupted', [], [], [[0], [1.22]]));

  // ItemsTab.lua:3129-3155, 3190-3218
  await box.fill(GLIMPSE);
  await buttons.getByRole('button', { name: 'Corrupt...' }).click();
  const [glimpseChoices] = await m.exposition.ItemCorruptChoices(GLIMPSE);
  const special = glimpseChoices.Glimpse!;
  expect(glimpseChoices.MaxImplicits).toBe(5);
  await expect(dialog.getByRole('combobox', { name: 'Source' }).locator('option')).toHaveText(['Glimpse of Chaos', 'Corrupted', 'Scourge']);
  await expect(dialog.getByRole('combobox', { name: 'Implicit #5' })).toBeVisible();
  await expect(dialog.getByRole('combobox', { name: 'Implicit #6' })).toHaveCount(0);
  await choose(first, special[0].Label);
  await choose(dialog.getByRole('combobox', { name: 'Implicit #3' }), special[1].Label);
  await expectEntries(dialog.getByRole('combobox', { name: 'Implicit #5' }), [
    'None',
    ...special.filter((x) => x.Group !== special[0].Group && x.Group !== special[1].Group).map((x) => x.Label)
  ]);
  await dialog.getByRole('button', { name: 'Corrupted', exact: true }).click();
  await expect(box).toHaveValue(await corrupt(GLIMPSE, 'Glimpse of Chaos', [special[0].ID, special[1].ID], []));

  misc.release();
  app.assertNoErrors();
});

const KAOM = `Rarity: UNIQUE
Kaom's Heart
Glorious Plate
Implicits: 0
Has no Sockets
+500 to maximum Life`;

const COAT = `Rarity: RARE
Test Coat
Vaal Regalia
Searing Exarch Item
Eater of Worlds Item
Implicits: 0
+80 to maximum Life`;

test('add an implicit to the display item (0280)', async ({ page }) => {
  test.setTimeout(900_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const box = page.getByRole('textbox', { name: /Item text/ });
  const controls = page.getByRole('region', { name: 'Item controls' });
  const buttons = controls.getByRole('group', { name: 'Enchant, anoint and corrupt' });
  const [choices] = await m.exposition.ItemImplicitChoices(COAT);
  const text = async (source: string, id: string, custom: string) => {
    const [t, ok] = await m.exposition.ImplicitText(COAT, source, id, custom);
    expect(ok).toBe(true);
    return t;
  };

  // AddImplicitPopup.lua:173-181
  await box.fill(COAT);
  await buttons.getByRole('button', { name: 'Add Implicit...' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Implicit to Item' });
  await expect(dialog.getByRole('combobox', { name: 'Source' }).locator('option')).toHaveText(choices.Sources!.map((s) => s.Label));

  const groups = choices.Groups!.EXARCH!;
  await expect(dialog.getByRole('combobox', { name: 'Type' }).locator('option')).toHaveText(groups.map((g) => g.Label));
  await expectEntries(
    dialog.getByRole('combobox', { name: 'Modifier' }),
    groups[0].Mods!.map((x) => x.Label)
  );
  await expect(dialog.getByRole('group', { name: 'Implicit comparison' })).toContainText('Adding this mod will give');

  // ItemsTab.lua:3957-4040
  const ids = groups.flatMap((g) => g.Mods!.map((x) => x.ID));
  let pick: { stat: string; v: number[]; order: number[] } | undefined;
  for (const stat of ['EnergyShield', 'FullDPS', 'Mana', 'Life']) {
    const v = [...(misc.ImplicitSortValues(COAT, 'EXARCH', ids, stat) ?? [])];
    const best = (g: (typeof groups)[number]) => Math.max(...g.Mods!.map((x) => v[ids.indexOf(x.ID)]));
    const order = sortByValue(groups.map(best), groups.length);
    if (order.some((k, i) => k !== i)) {
      pick = { stat, v, order };
      break;
    }
  }
  if (!pick) throw new Error('REFUSING: no stat reorders the Exarch types');
  const { v, order } = pick;
  const label = (m.exposition.PowerStats() ?? []).find((x) => x.Stat === pick.stat)!.Label;
  await dialog.getByRole('combobox', { name: 'Sort by' }).selectOption({ label });
  await expect(dialog.getByRole('combobox', { name: 'Sort by' })).toHaveAttribute('aria-busy', 'false', { timeout: 600_000 });
  await expect(dialog.getByRole('combobox', { name: 'Type' }).locator('option')).toHaveText(order.map((k) => groups[k].Label));
  const top = groups[order[0]];
  const topMods = top.Mods!.map((x) => x.ID);
  const modOrder = sortByValue(
    topMods.map((id) => v[ids.indexOf(id)]),
    topMods.length
  );
  await dialog.getByRole('button', { name: 'Add', exact: true }).click();
  await expect(box).toHaveValue(await text('EXARCH', topMods[modOrder[0]], ''));

  // ItemsTab.lua:4045-4048
  await box.fill(COAT);
  await buttons.getByRole('button', { name: 'Add Implicit...' }).click();
  await dialog.getByRole('combobox', { name: 'Source' }).selectOption({ label: 'Custom' });
  await dialog.getByRole('textbox', { name: 'Custom implicit' }).fill('+5 to Strength');
  await dialog.getByRole('button', { name: 'Add', exact: true }).click();
  await expect(box).toHaveValue(await text('CUSTOM', '', '+5 to Strength'));

  // AddImplicitPopup.lua:121-172, 186-211, 365-377
  await box.fill(KAOM);
  await buttons.getByRole('button', { name: 'Add Implicit...' }).click();
  const [kaom] = await m.exposition.ItemImplicitChoices(KAOM);
  const vestigial = kaom.Groups!.VESTIGIAL!;
  await expect(dialog.getByRole('combobox', { name: 'Source' }).locator('option')).toHaveText(['Vestigial', 'Custom']);
  await expect(dialog.getByRole('combobox', { name: 'Type' }).locator('option')).toHaveText(vestigial.map((g) => g.Label.replace(/^\^x[0-9A-F]{6}/, '')));
  await expect(dialog.getByRole('combobox', { name: 'Modifier' })).toHaveCount(0);
  await dialog.getByRole('combobox', { name: 'Type' }).selectOption({ index: 1 });
  await expect(dialog.getByRole('group', { name: 'Implicit comparison' })).toContainText('Adding this mod will give');
  await dialog.getByRole('button', { name: 'Add', exact: true }).click();
  const [vestigialText] = await m.exposition.ImplicitText(KAOM, 'VESTIGIAL', vestigial[1].Mods![0].ID, '');
  await expect(box).toHaveValue(vestigialText);

  misc.release();
  app.assertNoErrors();
});

test('add a modifier to the display item (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const box = page.getByRole('textbox', { name: /Item text/ });
  const controls = page.getByRole('region', { name: 'Item controls' });
  const helmet = readFileSync(new URL('./fixtures/items/helmet.txt', import.meta.url), 'utf8');
  const [choices] = await m.exposition.ItemCustomModChoices(helmet);
  const text = async (source: string, id: string, custom: string) => {
    const [t, ok] = await m.exposition.CustomModText(helmet, source, id, custom);
    expect(ok).toBe(true);
    return t;
  };

  // ItemsTab.lua:3557-3589
  await box.fill(helmet);
  await controls.getByRole('button', { name: 'Add modifier...' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Modifier to Item' });
  await expect(dialog.getByRole('combobox', { name: 'Source' }).locator('option')).toHaveText(choices.Sources!.map((s) => s.Label));
  const bench = choices.Mods!.MASTER!;
  const modifier = dialog.getByRole('combobox', { name: 'Modifier' });
  await expectEntries(
    modifier,
    bench.map((x) => plainText(x.Label))
  );
  await expect(dialog.getByRole('group', { name: 'Modifier comparison' })).toContainText('Adding this mod will give');

  // ItemsTab.lua:3356-3406
  await dialog.getByRole('combobox', { name: 'Source' }).selectOption({ label: 'Prefix' });
  const prefixes = choices.Mods!.PREFIX!;
  await dialog.getByRole('combobox', { name: 'Sort by' }).selectOption({ label: 'Life' });
  await expect(dialog.getByRole('combobox', { name: 'Sort by' })).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
  const v = [
    ...(misc.CustomModSortValues(
      helmet,
      'PREFIX',
      prefixes.map((x) => x.ID),
      'Life'
    ) ?? [])
  ];
  const order = sortByValue(v, prefixes.length);
  expect(order.some((k, i) => k !== i)).toBe(true);
  await expectEntries(
    modifier,
    order.map((k) => plainText(prefixes[k].Label))
  );
  await dialog.getByRole('button', { name: 'Add', exact: true }).click();
  await expect(box).toHaveValue(await text('PREFIX', prefixes[order[0]].ID, ''));

  await box.fill(helmet);
  await controls.getByRole('button', { name: 'Add modifier...' }).click();
  await dialog.getByRole('combobox', { name: 'Source' }).selectOption({ label: 'Custom' });
  await dialog.getByRole('textbox', { name: 'Custom modifier' }).fill('+7 to Dexterity');
  await dialog.getByRole('button', { name: 'Add', exact: true }).click();
  await expect(box).toHaveValue(await text('CUSTOM', '', '+7 to Dexterity'));

  misc.release();
  app.assertNoErrors();
});

test('affix dropdowns on a crafted item (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const box = page.getByRole('textbox', { name: /Item text/ });
  const affixes = page.getByRole('group', { name: 'Affixes' });
  const [crafted] = await m.exposition.CraftItemText(2, '', 'Hubris Circlet');
  const controls = async (text: string) => (await m.exposition.ItemAffixControls(text))[0].Controls!;
  const edit = async (text: string, slot: number, series: number, slider: number) => {
    const [t, ok] = await m.exposition.AffixEditText(text, slot, series, slider);
    expect(ok).toBe(true);
    return t;
  };

  // ItemsTab.lua:2129-2217
  await box.fill(crafted);
  const first = affixes.getByRole('combobox', { name: 'Affix 1' });
  await expect(affixes.getByRole('combobox', { name: /^Affix \d$/ })).toHaveCount(6);
  let c = await controls(crafted);
  await expectEntries(
    first,
    c[0].Series!.map((s) => plainText(s.Label))
  );

  // ItemsTab.lua:796-811
  const life = c[0].Series!.findIndex((s) => s.Label === '+# to maximum Life');
  await choose(first, '+# to maximum Life');
  const withLife = await edit(crafted, 0, life, c[0].SliderVal);
  await expect(box).toHaveValue(withLife);
  await expect(affixes.getByRole('group', { name: 'Affix comparison' })).toContainText('Adding this mod will give');

  // ItemsTab.lua:931-968
  const slider = affixes.getByRole('slider', { name: 'Roll of affix 1' });
  await slider.fill('0.95');
  await expect(affixes.getByRole('tooltip', { name: 'Roll of affix 1' })).toContainText(((await m.exposition.AffixRollTooltip(withLife, 0, 0.95)) ?? [])[0]);
  const rolled = await edit(withLife, 0, life, 0.95);
  await expect(box).toHaveValue(rolled);

  // ItemsTab.lua:2219-2297
  await affixes.getByRole('combobox', { name: 'Modifier sorting' }).selectOption({ label: 'Life' });
  await expect(affixes.getByRole('combobox', { name: 'Modifier sorting' })).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
  c = await controls(rolled);
  const p = [...(misc.AffixPowers(rolled, 1, 'Life') ?? [])];
  const idx = c[1].Series!.map((_, k) => k);
  const order = [0, ...idx.slice(1).sort((a, b) => (p[b] ?? 0) - (p[a] ?? 0))];
  expect(order.some((k, i) => k !== i)).toBe(true);
  await expectEntries(
    affixes.getByRole('combobox', { name: 'Affix 2' }),
    order.map((k) => plainText(c[1].Series![k].Label))
  );

  misc.release();
  app.assertNoErrors();
});

test('cluster jewel crafting (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const cluster = page.getByRole('group', { name: 'Cluster jewel' });
  const [crafted] = await m.exposition.CraftItemText(2, '', 'Large Cluster Jewel');
  const edit = async (text: string, e: Partial<Types.calculator.ItemEdit>) =>
    (await m.exposition.EditItem(text, { Kind: '', Index: 0, Value: 0, Second: 0, Float: 0, Text: '', ...e })).unwrap();

  // ItemsTab.lua:2059-2084
  await box.fill(crafted);
  const [info] = await m.exposition.DisplayItem(crafted);
  const skills = info.Cluster!.Skills!;
  await expect(cluster.getByRole('combobox', { name: 'Cluster skill' }).locator('option')).toHaveText(skills.map((s) => s.Label));
  const pick = skills[3];
  await cluster.getByRole('combobox', { name: 'Cluster skill' }).selectOption({ label: pick.Label });
  const withSkill = await edit(crafted, { Kind: 'cluster', Text: pick.ID, Value: info.Cluster!.NodeCount });
  await expect(box).toHaveValue(withSkill);

  // ItemsTab.lua:709-715
  await cluster.getByRole('slider', { name: 'Added Passives' }).fill('11');
  const withCount = await edit(withSkill, { Kind: 'cluster', Text: pick.ID, Value: 11 });
  await expect(box).toHaveValue(withCount);

  // ItemsTab.lua:2150-2155
  const [controls] = await m.exposition.ItemAffixControls(withCount);
  await expectEntries(
    page.getByRole('group', { name: 'Affixes' }).getByRole('combobox', { name: 'Affix 1' }),
    controls.Controls![0].Series!.map((s) => plainText(s.Label))
  );
  app.assertNoErrors();
});

const BOW = `Rarity: RARE
Test Bow
Thicket Bow
Implicits: 0
+40 to maximum Life`;

test('crucible mods on a weapon (0280)', async ({ page }) => {
  test.setTimeout(300_000);
  const { app } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const controls = page.getByRole('region', { name: 'Item controls' });

  // ItemsTab.lua:997-1004, 3664-3709
  await box.fill(BOW);
  await controls.getByRole('button', { name: 'Add Crucible mod...' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Crucible Modifier to Item' });
  const [choices] = await m.exposition.ItemCrucibleChoices(BOW);
  const node1 = choices.Nodes![0]!;
  await expect(dialog.getByRole('combobox', { name: 'Node 1' }).locator('option')).toHaveText(['None', ...node1.map((e) => e.Label)]);
  await dialog.getByRole('combobox', { name: 'Node 1' }).selectOption({ label: node1[1].Label });
  await expect(dialog.getByRole('group', { name: 'Crucible comparison' })).toContainText('Adding this mod will give');

  // ItemsTab.lua:3710-3730
  await dialog.getByRole('button', { name: 'Add', exact: true }).click();
  const [want, ok] = await m.exposition.CrucibleText(BOW, [node1[1].ID, '', '', '', '']);
  expect(ok).toBe(true);
  await expect(box).toHaveValue(want);
  app.assertNoErrors();
});
