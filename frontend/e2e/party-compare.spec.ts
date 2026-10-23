// CompareTab.lua:290-389, 2199-2230

import { test, expect } from './support/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { displayStats } from '../src/lib/display/stats';
import { formatNumSep, luaFormat } from '../src/lib/display/format_stat';

const A = '0280-elementalist-fireball';
const B = '0075-elementalist-winter-orb';

const fmt = (stat: string, v: number) => {
  const s = displayStats.find((d) => d.stat === stat)!;
  return formatNumSep(luaFormat(s.fmt ?? 'd', v * (s.pc || s.mod ? 100 : 1)));
};

test('B’s socket group select recalculates B as the model does', async ({ page }) => {
  test.setTimeout(300_000);
  const app = await App.open(page, A);
  const aXml = buildXml(A);
  const a = await go.loadXml(aXml);
  await app.importCode(await go.encode(aXml), async () => expectedLines(await go.outputs(a)));
  await app.goTo('Import / Export');
  const region = page.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(buildXml(B)));
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('radio', { name: /Import as a comparison/ }).click();
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await page.waitForURL((u) => u.pathname.endsWith('/compare'));
  const rows = page.getByRole('region', { name: 'Stat comparison' });
  await expect(rows).toBeVisible({ timeout: 60_000 });

  const sets = page.getByRole('group', { name: 'Comparison build sets' });
  const group = sets.getByRole('combobox', { name: 'Comparison socket group' });
  await expect(group).toBeVisible();
  const aHit = (await go.outputs(await go.loadXml(aXml))).Output.AverageHit;
  const labels = await group.locator('option').allTextContents();
  const current = Number(await group.inputValue());
  let pick = -1;
  let want = 0;
  const before = (await go.outputs(await go.loadXml(buildXml(B)))).Output.AverageHit;
  for (let i = 0; i < labels.length && pick < 0; i++) {
    if (i === current) continue;
    const m = await go.loadXml(buildXml(B));
    m.SetMainSocketGroup(i + 1);
    const hit = (await go.outputs(m)).Output.AverageHit;
    if (hit && fmt('AverageHit', hit) !== fmt('AverageHit', before)) {
      pick = i;
      want = hit;
    }
  }
  expect(pick, 'vacuity: no other group of B hits differently').toBeGreaterThanOrEqual(0);
  await group.selectOption(String(pick));
  const line = `Average Hit: A ${fmt('AverageHit', aHit)}, B ${fmt('AverageHit', want)}`;
  await expect(rows.getByRole('group', { name: new RegExp(`^${line.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`) })).toHaveCount(1);
  await expect(group).toHaveValue(String(pick));

  // CompareTab.lua:923-933, 2115
  await page.getByRole('tablist', { name: 'Compare views' }).getByRole('tab', { name: 'Config' }).click();
  const options = page.getByRole('table', { name: 'Config options' }).getByRole('row');
  await expect(options.nth(1)).toBeVisible();
  const relevant = await options.count();
  await page.getByRole('button', { name: 'Show All Configurations' }).click();
  await expect.poll(() => options.count()).toBeGreaterThan(relevant);
  await page.getByRole('button', { name: 'Hide Ineligible Configurations' }).click();
  await expect(options).toHaveCount(relevant);
  app.assertNoErrors();
});
