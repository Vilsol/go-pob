import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

test('skills: the gem search opens fast and walks every gem from the keyboard (0280)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Skills');

  const editor = page.getByRole('region', { name: 'Selected group' });
  const box = editor.getByRole('combobox', { name: 'Add a gem' });
  await expect(box).toBeVisible();
  const list = editor.getByRole('listbox', { name: 'Gem results' });

  const times: number[] = [];
  for (let k = 0; k < 3; k++) {
    times.push(
      await box.evaluate(
        (el: HTMLInputElement) =>
          new Promise<number>((done) => {
            const t = performance.now();
            el.focus();
            requestAnimationFrame(() => setTimeout(() => done(performance.now() - t)));
          })
      )
    );
    await expect(list).toBeVisible();
    await box.evaluate((el: HTMLInputElement) => el.blur());
    await expect(list).toBeHidden();
  }
  console.log(`gem search open (ms): ${times.map((t) => t.toFixed(0)).join(', ')}`);

  await box.focus();
  const rendered = await list.getByRole('option').count();
  console.log(`gem search options in the DOM: ${rendered}`);
  // GemSelectControl.lua:536-541
  expect(rendered).toBeLessThan(100);
  const total = Number((await list.locator('div').first().textContent())!.match(/^(\d+) gems/)![1]);
  expect(total).toBeGreaterThan(300);
  await box.press('ArrowUp');
  const lastId = (await box.getAttribute('aria-activedescendant'))!;
  const last = page.locator(`[id="${lastId}"]`);
  await expect(last).toHaveAttribute('aria-selected', 'true');
  await expect(last).toBeInViewport();
  await box.press('ArrowDown');
  const first = page.locator(`[id="${(await box.getAttribute('aria-activedescendant'))!}"]`);
  await expect(first).toBeInViewport();
  await expect(first).toHaveAttribute('aria-selected', 'true');
  app.assertNoErrors();
});
