import { test } from '@playwright/test';

import { expectKept, report, revisitPoE1, revisitPoE2 } from './support/tree_revisit';

// E2E_FIREFOX: a Firefox build for Playwright to launch where the one it installed cannot find its libraries
// CI installs Chromium alone, so these run only where E2E_FIREFOX names a Firefox that launches
test.skip(!process.env.E2E_FIREFOX, 'needs E2E_FIREFOX: a Firefox build Playwright can launch here');
test.use({
  serviceWorkers: 'block',
  browserName: 'firefox',
  launchOptions: { executablePath: process.env.E2E_FIREFOX || undefined, args: [], firefoxUserPrefs: { 'webgl.force-enabled': true } }
});

test('PoE1 tree: Tree -> Skills -> Tree draws its art at once, firefox', async ({ page }) => {
  const r = await revisitPoE1(page, 'webgl');
  report(page, r);
  expectKept(r);
});

test('PoE2 tree: Tree -> Skills -> Tree draws its art at once, firefox', async ({ page }) => {
  const r = await revisitPoE2(page, 'webgl');
  report(page, r);
  expectKept(r);
});
