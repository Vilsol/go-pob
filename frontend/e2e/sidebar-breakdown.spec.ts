import { test, expect, type Locator } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const steps = async (group: Locator) => (await group.locator('[data-step], .col-span-3').allTextContents()).map((s) => s.replace(/\s+/g, ' ').trim());

test('sidebar breakdown: a stat’s tooltip shows the steps the Calcs page shows for it', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  const cases: { sidebar: string; section: string; row: string }[] = [
    { sidebar: 'Total Life', section: 'Life', row: 'Total' },
    { sidebar: 'Hit DPS', section: 'Skill Hit Damage', row: 'Skill DPS' }
  ];
  const shown: string[][] = [];
  for (const c of cases) {
    await app
      .statsRegion()
      .locator('button')
      .filter({ hasText: new RegExp(`^${c.sidebar}`) })
      .first()
      .hover();
    const group = page.getByRole('tooltip').getByRole('group', { name: 'Calculation' });
    await expect(group).toBeVisible();
    shown.push(await steps(group));
    expect(shown.at(-1)!.length, `${c.sidebar}: no breakdown steps`).toBeGreaterThan(1);
    await page.mouse.move(0, 0);
  }

  await app.goTo('Calcs');
  const calcs = page.getByRole('region', { name: 'Calculations' });
  for (const [i, c] of cases.entries()) {
    const row = calcs.getByRole('region', { name: c.section, exact: true }).getByRole('button', { name: c.row, exact: true }).first();
    await row.hover();
    const group = page.getByRole('group', { name: 'Calculation' }).first();
    await expect(group).toBeVisible();
    expect(await steps(group), `${c.sidebar}: the sidebar's steps are not the Calcs page's`).toEqual(shown[i]);
  }
  app.assertNoErrors();
});
