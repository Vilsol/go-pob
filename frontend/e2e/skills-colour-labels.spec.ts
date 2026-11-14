// SkillListControl.lua:79-116, DropDownControl.lua:24-33

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { test, expect } from './support/test';

import { App } from './support/app';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const FIXTURE = resolve(import.meta.dirname, '../src/lib/design/fixtures/socket-group-colour-labels.xml');
const CODE = /\^(?:x[0-9A-Fa-f]{6}|\d)/;

test('skills: colour-coded group labels and set titles render as colour, label-only groups as headings and blank rows', async ({ page }) => {
  test.setTimeout(5 * 60_000);
  const xml = readFileSync(FIXTURE, 'utf8');
  const build = await go.loadXml(xml);
  const app = await App.open(page, 'socket-group-colour-labels');
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Skills');

  const groups = page.getByRole('region', { name: 'Socket groups' });
  await expect(groups.locator('[data-label-row="blank"]').first()).toBeVisible();
  await expect(groups.locator('[data-label-row="label"]', { hasText: /^<<Level \d+-\d+ ?>>$/ }).first()).toBeVisible();
  await expect(groups.locator('[data-label-row]', { hasText: 'No gems' })).toHaveCount(0);
  expect(await groups.innerText()).not.toMatch(CODE);
  expect(await page.locator('body').innerText()).not.toMatch(CODE);

  const set = page.getByRole('combobox', { name: 'Skill set' });
  await expect(set.locator('option:checked')).toHaveText(/^Level \d+-\d+ \{\d\}$/);
  expect(await set.locator('option').allTextContents()).not.toContainEqual(expect.stringMatching(CODE));

  const blank = groups.locator('[data-label-row="blank"]').first();
  await blank.click();
  await expect(blank).toHaveAttribute('aria-current', 'true');
  await expect(page.getByRole('region', { name: 'Selected group' }).getByRole('textbox', { name: 'Label' })).toHaveValue(CODE);
});
