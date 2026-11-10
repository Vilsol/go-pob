import { test } from '@playwright/test';

import { expectKept, report, revisitPoE1, revisitPoE2 } from './support/tree_revisit';

test.use({ serviceWorkers: 'block' });

test('PoE1 tree: Tree -> Skills -> Tree draws its art at once, chromium', async ({ page }) => {
  const r = await revisitPoE1(page, 'webgl');
  report(page, r);
  expectKept(r);
});

test('PoE2 tree: Tree -> Skills -> Tree draws its art at once, chromium', async ({ page }) => {
  const r = await revisitPoE2(page, 'webgl');
  report(page, r);
  expectKept(r);
});
