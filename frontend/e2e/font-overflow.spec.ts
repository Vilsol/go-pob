import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';

import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { overflowing, setFontScaling, type Overflow } from './support/overflow';
import { expectedLines } from './support/stats';

const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2';
const COMPARE = '0280-elementalist-fireball';
const POINTS = Array.from({ length: 25 }, (_, i) => 6 + i / 2);
const BASELINE = new URL('./font-overflow-baseline.json', import.meta.url);

const MAIN = '.page-main';
const SIDEBAR = '.page-body > aside';
const TOPBAR = '.page > :first-child';

test('font scaling: no page lets more boxes overflow than its baseline (0339)', { tag: '@slow' }, async ({ page }) => {
  test.setTimeout(20 * 60_000);
  const xml = buildXml(BUILD);
  const build = await go.loadXml(xml);
  const app = await App.open(page, BUILD);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));

  const found: Record<string, Map<string, Overflow & { pt: number[] }>> = {};
  const census = async (name: string, root: string, skip: string[] = []) => {
    const seen = (found[name] ??= new Map());
    // a tooltip under the pointer the last click left behind is not the page's overflow
    await page.mouse.move(0, 0);
    for (const pt of POINTS) {
      await setFontScaling(page, pt);
      // a weight first used at this size loads now (display=swap); measure the face, not its fallback
      const failed = await page.evaluate(async () => {
        await document.fonts.ready;
        return [...document.fonts].filter((f) => f.status === 'error').map((f) => `${f.family} ${f.weight}`);
      });
      expect(failed, `${name} at ${pt}pt: font faces failed to load`).toEqual([]);
      for (const o of await overflowing(page, root, skip)) {
        const had = seen.get(o.key);
        if (had) had.pt.push(pt);
        else seen.set(o.key, { ...o, pt: [pt] });
      }
    }
    await setFontScaling(page, 12);
  };
  const settle = async () => {
    let last = '';
    for (let i = 0; i < 40; i++) {
      await page.waitForTimeout(250);
      const now = await page.evaluate((root) => document.querySelector(root)?.textContent ?? '', MAIN);
      if (now && now === last) return;
      last = now;
    }
    throw new Error('the page did not settle');
  };

  const ready: Record<string, () => Promise<void>> = {
    Tree: () => expect(page.getByRole('region', { name: 'Passive tree' })).toBeVisible(),
    Skills: () =>
      expect(
        page
          .getByRole('group', { name: 'Gems' })
          .getByRole('button', { name: /^Remove / })
          .first()
      ).toBeVisible(),
    Items: () => expect(page.getByRole('region', { name: 'Equipped items' })).toBeVisible(),
    Calcs: () => expect(page.getByRole('region', { name: 'Calculations' })).toBeVisible()
  };
  for (const link of ['Tree', 'Skills', 'Items', 'Calcs', 'Config', 'Party', 'Notes', 'Import / Export']) {
    const t0 = await app.now();
    await app.goTo(link);
    // the page fills from the worker, which may still be on its idle job (the item DBs) when the tab opens
    if (ready[link]) await app.followWorker(t0, ready[link]);
    await settle();
    if (link === 'Tree') {
      await census('Topbar', TOPBAR);
      await census('Sidebar', SIDEBAR);
    }
    await census(link, MAIN);
  }

  await page.getByRole('button', { name: 'Settings', exact: true }).click();
  const sheet = page.getByRole('dialog', { name: 'Settings' });
  for (const tab of await sheet.getByRole('tab').all()) {
    await tab.click();
    await census('Settings', 'dialog[open]');
  }
  await page.keyboard.press('Escape');
  await expect(sheet).toBeHidden();

  await app.goTo('Import / Export');
  const region = page.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(buildXml(COMPARE)));
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('radio', { name: /Import as a comparison/ }).click();
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await page.waitForURL((u) => u.pathname.endsWith('/compare'));
  await expect(page.getByRole('region', { name: 'Stat comparison' })).toBeVisible({ timeout: 60_000 });
  const views = page.getByRole('tablist', { name: 'Compare views' }).getByRole('tab');
  for (const view of await views.all()) {
    await view.click();
    await settle();
    if ((await view.textContent()) === 'Power report')
      await expect(page.getByRole('region', { name: 'Compare power report' }).getByRole('status')).toHaveCount(0, { timeout: 180_000 });
    await census('Compare', MAIN);
  }

  const counts = Object.fromEntries(Object.entries(found).map(([k, v]) => [k, v.size]));
  mkdirSync(new URL('./.results/', import.meta.url), { recursive: true });
  writeFileSync(
    new URL('./.results/font-overflow.json', import.meta.url),
    JSON.stringify(Object.fromEntries(Object.entries(found).map(([k, v]) => [k, [...v.values()]])), null, 1)
  );
  console.log('font-overflow census:', JSON.stringify(counts));
  if (process.env.E2E_FONT_BASELINE === '1') {
    writeFileSync(BASELINE, JSON.stringify(counts, null, 2) + '\n');
    return;
  }
  const ceiling = JSON.parse(readFileSync(BASELINE, 'utf8')) as Record<string, number>;
  expect(Object.keys(counts).sort(), 'the census covers the pages its baseline names').toEqual(Object.keys(ceiling).sort());
  for (const [name, n] of Object.entries(counts)) expect(n, `${name}: overflowing boxes (lower the baseline when it falls)`).toBeLessThanOrEqual(ceiling[name]);
  app.assertNoErrors();
});
