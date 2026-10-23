import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { record } from './support/perf';
import { expectedLines } from './support/stats';
import { displayStats } from '../src/lib/display/stats';
import { formatNumSep, luaFormat } from '../src/lib/display/format_stat';
import { signed } from '../src/lib/design/compare';
import { calcSections } from '../src/lib/calcs/calc_sections';
import { calcElements, colPropsOf } from '../src/lib/display/calcs';
import { compareCalcsView, modEntries, type CompareCalcSection } from '../src/lib/display/compare_calcs';
import { pobPlain } from '../src/lib/design/pob_text';

const A = '0280-elementalist-fireball';
const B = '0075-elementalist-winter-orb';

const openCompare = async (page: Page) => {
  const app = await App.open(page, A);
  const aXml = buildXml(A);
  const a = await go.loadXml(aXml);
  await app.importCode(await go.encode(aXml), async () => expectedLines(await go.outputs(a)));
  return { app, a };
};

const importComparison = async (app: App) => {
  await app.goTo('Import / Export');
  const region = app.page.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(buildXml(B)));
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('radio', { name: /Import as a comparison/ }).click();
  const t0 = await app.now();
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await app.page.waitForURL((u) => u.pathname.endsWith('/compare'));
  await expect(app.page.getByRole('region', { name: 'Stat comparison' })).toBeVisible({ timeout: 60_000 });
  record({ kind: 'interaction', build: A, phase: 'compare', name: 'compare-import', durationMs: +((await app.now()) - t0).toFixed(1) });
};

const fmt = (stat: string, v: number) => {
  const s = displayStats.find((d) => d.stat === stat)!;
  return formatNumSep(luaFormat(s.fmt ?? 'd', v * (s.pc || s.mod ? 100 : 1)));
};

test('importing a comparison leaves A untouched, and the summary shows each build its own numbers', async ({ page }) => {
  const { app } = await openCompare(page);
  const codeBefore = await app.exportCode();
  await importComparison(app);

  const aOut = await go.outputs(await go.loadXml(buildXml(A)));
  const bOut = await go.outputs(await go.loadXml(buildXml(B)));
  const rows = page.getByRole('region', { name: 'Stat comparison' });
  for (const [label, stat] of [
    ['Total Life', 'Life'],
    ['Effective Hit Pool', 'TotalEHP'],
    ['Strength', 'Str']
  ] as const) {
    const want = `${label}: A ${fmt(stat, aOut.Output[stat])}, B ${fmt(stat, bOut.Output[stat])}`;
    await expect(rows.getByRole('group', { name: new RegExp(`^${want.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`) })).toHaveCount(1);
  }
  expect(await rows.getByRole('group').count()).toBeGreaterThan(20);
  // CompareTab.lua:3328-3341
  const [la, lb] = [aOut.Output.Life, bOut.Output.Life];
  const life = rows.getByRole('group', { name: /^Total Life: / }).locator('.butterfly-row');
  await life.hover();
  const diff = formatNumSep(luaFormat('+d', lb - la)) + (la ? ` (${luaFormat('+.1f%%', (lb / la) * 100 - 100)})` : '');
  await expect(page.getByRole('tooltip')).toHaveText(`Total Life · B − A: ${diff}`);

  await expect(page.getByRole('button', { name: 'Nothing to undo' })).toBeDisabled();
  expect(await app.exportCode()).toBe(codeBefore);

  const nextXml = buildXml(B);
  const next = await go.loadXml(nextXml);
  await app.goTo('Import / Export');
  const region = page.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(nextXml));
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('radio', { name: /Import to this build/ }).check();
  await app.act(
    'import',
    async () => {
      await region.getByRole('button', { name: 'Import', exact: true }).click();
      await page.getByRole('dialog', { name: 'Build Import' }).getByRole('button', { name: 'Import', exact: true }).click();
    },
    async () => expectedLines(await go.outputs(next))
  );
  await app.goTo('Compare');
  await expect(page.getByRole('region', { name: 'Choose a build to compare' })).toBeVisible();
  await expect(page.getByRole('region', { name: 'Stat comparison' })).toHaveCount(0);

  // CompareTab.lua:1462-1525
  const aCode = await go.encode(buildXml(A));
  // BuildSiteTools.lua:88-113
  await page.route(
    (u) => u.pathname.endsWith('/api/build') && u.searchParams.get('site') === 'POBBin' && u.searchParams.get('id') === 'e2e-compare',
    (route) => route.fulfill({ status: 200, body: aCode, headers: { 'content-type': 'text/plain' } })
  );
  const choose = page.getByRole('region', { name: 'Choose a build to compare' });
  await choose.getByRole('textbox', { name: 'Comparison build code' }).fill('https://pobb.in/e2e-compare');
  await choose.getByRole('button', { name: 'Compare' }).click();
  const linked = page.getByRole('region', { name: 'Stat comparison' });
  await expect(linked).toBeVisible({ timeout: 60_000 });
  const want = `Total Life: A ${fmt('Life', bOut.Output.Life)}, B ${fmt('Life', aOut.Output.Life)}`;
  await expect(linked.getByRole('group', { name: new RegExp(`^${want.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`) })).toHaveCount(1);

  // CompareTab.lua:227-242, 1240-1276
  const lifeRow = (a: number, b: number) =>
    linked.getByRole('group', { name: new RegExp(`^Total Life: A ${fmt('Life', a)}, B ${fmt('Life', b)}`.replace(/[.*+?$()|[\]\\]/g, '\\$&')) });
  await page.getByRole('button', { name: 'Import…' }).click();
  await choose.getByRole('textbox', { name: 'Comparison build code' }).fill(await go.encode(nextXml));
  await choose.getByRole('textbox', { name: 'Comparison name' }).fill('Next build');
  await choose.getByRole('button', { name: 'Compare' }).click();
  const withSel = page.getByRole('combobox', { name: 'Compare with' });
  // CompareTab.lua:1485-1511
  await expect(withSel.locator('option')).toHaveText(['Imported from pobb.in', 'Next build']);
  await expect(withSel).toHaveValue('1');
  await expect(lifeRow(bOut.Output.Life, bOut.Output.Life)).toHaveCount(1, { timeout: 60_000 });
  await withSel.selectOption('0');
  await expect(lifeRow(bOut.Output.Life, aOut.Output.Life)).toHaveCount(1, { timeout: 60_000 });
  await page.getByRole('button', { name: 'Remove comparison' }).click();
  await expect(withSel.locator('option')).toHaveCount(1);
  await expect(lifeRow(bOut.Output.Life, bOut.Output.Life)).toHaveCount(1, { timeout: 60_000 });
  expect(app.errors).toEqual([]);
});

test('the power report lists what the Go model measures, and Equip writes B’s item into A as one undoable step', async ({ page }) => {
  const { app, a } = await openCompare(page);
  await importComparison(app);
  await page.getByRole('tab', { name: 'Power report' }).click();
  await page.getByRole('combobox', { name: 'Metric' }).first().selectOption('life');
  const table = page.getByRole('table', { name: 'Power report rows' });
  await expect(page.getByRole('region', { name: 'Compare power report' }).getByRole('status')).toHaveCount(0, { timeout: 120_000 });

  const m = await go.model();
  const bModel = await go.loadXml(buildXml(B));
  const pb = m.calculator.NewComparePowerBuilder(
    a,
    bModel,
    ['FullDPS', 'TotalEHP', 'Life'],
    { TreeNodes: true, Items: true, SkillGems: true, SupportGems: true, Config: false },
    []
  )!;
  while (pb.StepMs(1000) < 1);
  // CompareTab.lua:2785-2868
  const pj = m.calculator.NewCompareJewelPowerBuilder(a, bModel, ['FullDPS', 'TotalEHP', 'Life'])!;
  while (pj.StepMs(1000) < 1);
  const want = [...(pb.Report() ?? []), ...(pj.Report() ?? [])]
    .map((e) => ({ what: e.Category === 'Tree' ? `Take ${e.Name}` : e.Name, v: Math.trunc(e.Impact![2]) === 0 ? 0 : e.Impact![2] }))
    .filter((e) => e.v !== 0)
    .map((e) => `${e.what}\t${signed(e.v)}`)
    .sort();
  pb.release();
  pj.release();
  expect(want.length, 'the model measured nothing, so a match would be vacuous').toBeGreaterThan(5);

  const shown = await table.evaluate((el) =>
    [...el.querySelectorAll('[role=row]')].slice(1, -1).map((r) => {
      const c = r.querySelectorAll('[role=cell]');
      return { cat: c[0].textContent?.trim(), what: c[1].firstElementChild?.textContent?.trim() ?? '', value: c[3].textContent?.trim() ?? '' };
    })
  );
  const got = shown
    .filter((r) => r.cat !== 'Config')
    .map((r) => `${r.what}\t${/^[▲▼] (\S+)/.exec(r.value)?.[1] ?? r.value}`)
    .sort();
  expect(got).toEqual(want);

  // ComparePowerReportListControl.lua:124-132
  // PassiveTreeView.lua:1695-1705
  const treeRow = table.locator('[role=row]').filter({ hasText: /^Tree/ }).first();
  const points = (await treeRow.locator('[role=cell]').nth(4).textContent())?.trim() ?? '';
  expect(points, 'no Tree row with points, so the path line is not graded').toMatch(/^\d+$/);
  await treeRow.getByRole('cell').nth(1).getByRole('button').hover();
  const tip = page.locator('#compare-power-tip');
  await expect(tip).toContainText(/Allocating this node will give you:|No changes from allocating this node/, { timeout: 60_000 });
  await expect(tip).toContainText(`${points} points to node`);
  // PassiveTreeView.lua:1706
  await expect(tip).toContainText('Tip: Press Ctrl+D to disable the display of stat differences.');

  // CompareTab.lua:968-974
  await page.getByRole('combobox', { name: 'Metric' }).first().selectOption('Mana');
  const pm = m.calculator.NewComparePowerBuilder(a, bModel, ['Mana'], { TreeNodes: true, Items: true, SkillGems: true, SupportGems: true, Config: false }, [])!;
  while (pm.StepMs(1000) < 1);
  const pjm = m.calculator.NewCompareJewelPowerBuilder(a, bModel, ['Mana'])!;
  while (pjm.StepMs(1000) < 1);
  const wantMana = [...(pm.Report() ?? []), ...(pjm.Report() ?? [])]
    .map((e) => ({ what: e.Category === 'Tree' ? `Take ${e.Name}` : e.Name, v: Math.trunc(e.Impact![0]) === 0 ? 0 : e.Impact![0] }))
    .filter((e) => e.v !== 0)
    .map((e) => `${e.what}\t${signed(e.v)}`)
    .sort();
  pm.release();
  pjm.release();
  expect(wantMana.length, 'the model measured no Mana change, so a match would be vacuous').toBeGreaterThan(0);
  const shownMana = await table.evaluate((el) =>
    [...el.querySelectorAll('[role=row]')].slice(1, -1).map((r) => {
      const c = r.querySelectorAll('[role=cell]');
      return { cat: c[0].textContent?.trim(), what: c[1].firstElementChild?.textContent?.trim() ?? '', value: c[3].textContent?.trim() ?? '' };
    })
  );
  expect(
    shownMana
      .filter((r) => r.cat !== 'Config')
      .map((r) => `${r.what}\t${/^[▲▼] (\S+)/.exec(r.value)?.[1] ?? r.value}`)
      .sort()
  ).toEqual(wantMana);
  await page.getByRole('combobox', { name: 'Metric' }).first().selectOption('life');

  const beltRow = table.getByRole('row').filter({ hasText: ', Belt' });
  const items = (await m.exposition.EquippedItems(bModel)) ?? [];
  const belt = items.find((i) => i.SlotName === 'Belt')!;
  await app.act(
    'equip B’s belt',
    () => beltRow.getByRole('button', { name: 'Equip' }).click(),
    async () => {
      (await m.exposition.EquipItem(a, belt.Text, 'Belt')).unwrap();
      return expectedLines(await go.outputs(a));
    }
  );
  await expect(page.getByRole('button', { name: /^Undo: / })).toBeEnabled();
  expect(app.errors).toEqual([]);
});

test('the tree, items, skills and config views read both builds; Use and Take write to A', async ({ page }) => {
  const { app, a } = await openCompare(page);
  await importComparison(app);
  const bModel = await go.loadXml(buildXml(B));

  await page.getByRole('tab', { name: 'Items' }).click();
  const items = page.getByRole('table', { name: 'Items by slot' });
  await expect(items.getByRole('row').filter({ hasText: 'Belt' })).toHaveCount(1);
  await expect(items.getByRole('row').filter({ hasText: 'Belt' }).getByRole('cell').last()).toHaveText(/[▲▼]/, { timeout: 60_000 });

  await page.getByRole('tab', { name: 'Skills' }).click();
  await expect(page.getByRole('region', { name: 'Skill setups' }).getByRole('table').first()).toBeVisible();

  await page.getByRole('tab', { name: 'Config' }).click();
  const options = page.getByRole('table', { name: 'Config options' });
  const first = options.locator('[role=row][data-diff=true]').first();
  await expect(first).toBeVisible();
  const label = (await first.getByRole('cell').first().textContent())!.split('·').slice(1).join('·').trim();
  const { reverseConfigOptions } = await import('../src/lib/display/configurations');
  const key = Object.values(reverseConfigOptions).find((v) => v.label.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '') === label)!.var;
  const bInput = go.fresh(bModel, (b) => (b.Config.Inputs ?? []).find((i) => i.Name === key));
  await app.act(
    'use B’s config value',
    () => first.getByRole('button', { name: 'Use B’s value' }).click(),
    async () => {
      if (bInput) a.SetConfigOption(bInput);
      else a.RemoveConfigOption(key);
      return expectedLines(await go.outputs(a));
    }
  );
  await expect(
    page.getByRole('button', {
      name: new RegExp(`^Undo: Used ${label.replace(/[\s:?]+$/, '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')} from .+ \\(Ctrl\\+Z\\)$`)
    })
  ).toBeEnabled();

  await page.getByRole('tab', { name: 'Tree' }).click();
  const diffs = page.getByRole('region', { name: 'Node differences' });
  const take = diffs.getByRole('button', { name: 'Take' }).first();
  await expect(take).toBeVisible();
  const before = go.fresh(a, (b) => [...(b.Build.PassiveNodes ?? [])]).length;
  const seq = await app.seq();
  await take.click();
  await app.waitSettled(seq);
  const after = await go.loadXml(await go.decode(await app.exportCode()));
  expect(go.fresh(after, (b) => b.Build.PassiveNodes ?? []).length).toBeGreaterThan(before);

  // CompareTab.lua:1331-1368
  const backTo = async (tab: string) => {
    await app.goTo('Compare');
    await page.getByRole('tab', { name: tab }).click();
  };
  const exported = async () => go.loadXml(await go.decode(await app.exportCode()));
  const treesBefore = go.fresh(after, (b) => b.Tree.Specs?.length ?? 0);
  await backTo('Tree');
  await diffs.getByRole('button', { name: 'Copy tree' }).click();
  await expect.poll(async () => go.fresh(await exported(), (b) => b.Tree.Specs?.length ?? 0)).toBe(treesBefore + 1);
  const mine = await exported();
  await backTo('Tree');
  await app.act(
    'copy and use B’s tree',
    () => diffs.getByRole('button', { name: 'Copy and use' }).click(),
    async () => {
      mine.CopyCompareSpec(bModel, true);
      return expectedLines(await go.outputs(mine));
    }
  );

  // CompareTab.lua:1439-1459
  const itemsBefore = go.fresh(mine, (b) => b.Items.Items?.length ?? 0);
  await backTo('Items');
  await items.getByRole('row').filter({ hasText: 'Belt' }).getByRole('button', { name: 'Copy' }).click();
  await expect.poll(async () => go.fresh(await exported(), (b) => b.Items.Items?.length ?? 0)).toBe(itemsBefore + 1);
  expect(app.errors).toEqual([]);
});

// CompareTab.lua:4655-4902
test('the Calcs view shows each build’s Calcs values as the Go model computes them, only the differences by default', async ({ page }) => {
  const { app, a } = await openCompare(page);
  await importComparison(app);
  const bModel = await go.loadXml(buildXml(B));
  const elements = colPropsOf(calcElements(calcSections));
  const [outA, outB] = [await go.calcsOutputs(a, elements), await go.calcsOutputs(bModel, elements)];
  const lines = (v: CompareCalcSection[]) =>
    v.flatMap((s) => [
      `${s.id}\tA\tB`,
      ...s.blocks.flatMap((b) =>
        [`${b.label}:${b.extra ? `\t${b.extra.a}\t${b.extra.b}` : ''}`, ...b.rows.map((r) => `${r.label}\t${r.a}\t${r.b}`)].map(pobPlain)
      )
    ]);
  const region = page.getByRole('region', { name: 'Calcs' });
  const shown = () =>
    region.evaluate((el) =>
      [...el.querySelectorAll('[role=table] [role=row]')].map((r) =>
        [...r.querySelectorAll('[role=columnheader],[role=rowheader],[role=cell]')].map((c) => c.textContent?.trim() ?? '').join('\t')
      )
    );

  await page.getByRole('tab', { name: 'Calcs' }).click();
  const modsEqual = await go.compareModsEqual(a, bModel, elements, modEntries(calcSections));
  const diff = lines(compareCalcsView(calcSections, outA, outB, true, 'A', 'B', modsEqual));
  const all = lines(compareCalcsView(calcSections, outA, outB, false));
  expect(all.length, 'the full view must hold rows the differences leave out').toBeGreaterThan(diff.length);
  await expect(region.getByRole('table').first()).toBeVisible({ timeout: 60_000 });
  await expect.poll(shown).toEqual(diff);
  await region.getByRole('checkbox', { name: 'Show only differences' }).uncheck();
  await expect.poll(shown).toEqual(all);
  expect(app.errors).toEqual([]);
});

// CompareTab.lua:3954-3963
test('items, Equip and Calcs controls show and write each build’s own, as the model has them', async ({ page }) => {
  const { app, a } = await openCompare(page);
  await importComparison(app);
  const bModel = await go.loadXml(buildXml(B));
  const m = await go.model();

  const version = (b: Parameters<typeof go.fresh>[0]) => go.fresh(b, (x) => (x.Tree.Specs ?? [])[x.ActiveSpecIndex()]?.TreeVersion ?? '');
  const sets = page.getByRole('group', { name: 'Comparison build sets' });
  await sets.getByRole('button', { name: /^This build's sets/ }).click();
  await sets.getByRole('button', { name: /^Comparison sets/ }).click();
  await expect(sets.getByRole('combobox', { name: 'Primary tree version' })).toHaveValue(version(a));
  await expect(sets.getByRole('combobox', { name: 'Comparison tree version' })).toHaveValue(version(bModel));

  await page.getByRole('tab', { name: 'Items' }).click();
  const belt = page.getByRole('table', { name: 'Items by slot' }).getByRole('row').filter({ hasText: 'Belt' });
  const bBelt = (await belt.getByRole('button').nth(1).textContent())!.trim();
  const [tip, ok] = await m.exposition.ItemSourceTooltip(bModel, bBelt, go.NO_TIP_STATS);
  expect(ok, `the model has B's ${bBelt}`).toBe(true);
  await belt.getByRole('button').nth(1).hover();
  await expect(page.locator('#compare-item-tip')).toContainText(tip.Title![0]);
  await expect(belt.getByRole('cell').last()).toHaveText(/[▲▼]/, { timeout: 60_000 });
  await belt.getByRole('button', { name: 'Equip' }).hover();
  await expect(page.locator('#compare-equip-tip')).toContainText('Equipping this item in Belt will give you:');

  await page.getByRole('tab', { name: 'Calcs' }).click();
  const theirs = page.getByRole('group', { name: /calcs controls$/ }).nth(1);
  const mode = theirs.getByRole('combobox', { name: 'Calc Mode' });
  await expect(mode).toHaveValue(go.fresh(bModel, (b) => b.CalcsInputString('misc_buffMode', 'EFFECTIVE')));
  await mode.selectOption('UNBUFFED');
  await expect(mode).toHaveValue('UNBUFFED');
  await expect(
    page
      .getByRole('group', { name: /calcs controls$/ })
      .first()
      .getByRole('combobox', { name: 'Calc Mode' })
  ).toHaveValue(go.fresh(a, (b) => b.CalcsInputString('misc_buffMode', 'EFFECTIVE')));
  expect(app.errors).toEqual([]);
});

// CompareTab.lua:766-773, 3466-3637, 3926-3940
test('Items: Expanded mode cards and the Jewels section are the model’s', { tag: '@smoke' }, async ({ page }) => {
  const { app, a } = await openCompare(page);
  await importComparison(app);
  const bModel = await go.loadXml(buildXml(B));
  const m = await go.model();
  await page.getByRole('tab', { name: 'Items' }).click();

  const jewels = (await m.calculator.CompareJewelSlots(a, bModel)) ?? [];
  expect(jewels.length, 'neither build holds a jewel, so the section would be vacuous').toBeGreaterThan(0);
  const shown = page
    .getByRole('rowgroup', { name: 'Jewels' })
    .getByRole('row')
    .filter({ has: page.getByRole('cell') });
  await expect(shown.locator('[role=cell]:first-child')).toHaveText(jewels.map((j) => j.Label));
  const missing = jewels.filter((j) => (j.PText && !j.PAllocated) || (j.CText && !j.CAllocated)).length;
  await expect(shown.filter({ hasText: '(tree missing allocated node)' })).toHaveCount(missing);

  const slot = async (b: typeof a, name: string) => ((await m.exposition.EquippedItems(b)) ?? []).find((e) => e.SlotName === name)?.Text ?? '';
  const [aHelm, bHelm] = [await slot(a, 'Helmet'), await slot(bModel, 'Helmet')];
  const want = [(await m.calculator.CompareItemExpanded(aHelm, bHelm)) ?? [], (await m.calculator.CompareItemExpanded(bHelm, aHelm)) ?? []];
  expect(want[0].length, 'the model drew no Helmet card').toBeGreaterThan(1);
  await page.getByRole('checkbox', { name: 'Expanded mode' }).check();
  const helmet = page.getByRole('group', { name: 'Helmet', exact: true });
  for (const [i, card] of want.entries()) {
    await expect(helmet.getByRole('list').nth(i).getByRole('listitem')).toHaveText(card.map((l) => pobPlain(l)));
  }

  // CompareTab.lua:3806-3809, 3950-3952
  await page.evaluate(() => {
    window.open = ((u: string) => ((window as unknown as { opened: string }).opened = u)) as typeof window.open;
  });
  await helmet.getByRole('button', { name: 'Buy' }).click();
  const dialog = page.getByRole('dialog', { name: 'Buy Similar' });
  await dialog.getByRole('button', { name: 'Search' }).click();
  const opened = await page.evaluate(() => (window as unknown as { opened?: string }).opened ?? '');
  expect(opened).toMatch(/^https:\/\/www\.pathofexile\.com\/trade\/search\/Standard\?q=/);
  expect(app.errors).toEqual([]);
});

// CompareTab.lua:1128-1224
test('Config: editing B’s option through its control writes B', async ({ page }) => {
  const { app } = await openCompare(page);
  await importComparison(app);
  await page.getByRole('tab', { name: 'Config' }).click();
  const options = page.getByRole('table', { name: 'Config options' });
  const box = options.getByRole('checkbox', { name: /\(B\)$/ }).first();
  await expect(box).toBeVisible({ timeout: 60_000 });
  const label = (await box.getAttribute('aria-label'))!;
  const was = await box.isChecked();
  await box.click();
  await expect(options.getByRole('checkbox', { name: label, exact: true })).toBeChecked({ checked: !was, timeout: 60_000 });
  await page.getByRole('tab', { name: 'Summary' }).click();
  await page.getByRole('tab', { name: 'Config' }).click();
  await expect(options.getByRole('checkbox', { name: label, exact: true })).toBeChecked({ checked: !was });
  expect(app.errors).toEqual([]);
});
