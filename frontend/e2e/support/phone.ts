import { expect, type Page } from '@playwright/test';

import { App } from './app';
import { buildXml } from './builds';
import * as go from './go_model';
import { expectedLines } from './stats';
import { installTreeProbe } from './tree_probe';

// a phone-sized page with touch, the size the phone layout is built for
export const PHONE = { viewport: { width: 390, height: 769 }, hasTouch: true, isMobile: true };
export const BUILD = '0280-elementalist-fireball';

export const bar = (page: Page) => page.getByRole('navigation', { name: 'Pages', exact: true });
export const openMore = async (page: Page) => {
  await bar(page).getByRole('button', { name: 'More' }).click();
  return page.getByRole('dialog', { name: 'More' });
};
export const routeOf = (page: Page) => new URL(page.url()).pathname.split('/').filter(Boolean).at(-1);

// a press held past the hold time; Playwright's touchscreen only taps
export const hold = async (page: Page, x: number, y: number) => {
  const cdp = await page.context().newCDPSession(page);
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] });
  await page.waitForTimeout(700);
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await cdp.detach();
};

export const openPhoneBuild = async (page: Page, file = BUILD) => {
  await installTreeProbe(page);
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file, { phone: true });
  const more = await openMore(page);
  await more.getByRole('link', { name: 'Import/Export' }).click();
  await page.waitForURL((u) => u.pathname.endsWith('/import'));
  const region = page.getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(xml));
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await app.act(
    'import',
    async () => {
      await region.getByRole('button', { name: 'Import', exact: true }).click();
      await page.getByRole('dialog', { name: 'Build Import' }).getByRole('button', { name: 'Import', exact: true }).click();
    },
    async () => expectedLines(await go.outputs(build))
  );
  return { app, build };
};
