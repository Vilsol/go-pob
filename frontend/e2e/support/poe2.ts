import type { Locator, Page } from '@playwright/test';

import { displayStats2 } from '../../src/lib/display/stats2';

// a PoE2 build's page body in the app frame; the Calcs and Config pages are the ones PoE1 builds show
export const poe2Page = (page: Page) => page.locator('[data-poe2-page], [aria-label="Calculations"], input[aria-label="Filter options"]');

// the frame sidebar's value for a PoE2 stat, found by the label(s) PoE2's stat list gives it
export const poe2Stat = (page: Page, stat: string): Locator => {
  const labels = [...new Set(displayStats2.filter((s) => s.stat === stat && s.label).map((s) => s.label!))];
  if (!labels.length) throw new Error(`PoE2's stat list has no row for ${stat}`);
  const label = new RegExp(`^(${labels.map((l) => l.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})$`);
  return page
    .getByRole('region', { name: 'Build stats' })
    .locator('.stat-row')
    .filter({ has: page.locator('.stat-label', { hasText: label }) })
    .locator('.stat-value');
};

// the frame's top bar link to a page
export const poe2Nav = (page: Page, label: string) => page.locator('nav a.tab').filter({ hasText: new RegExp(`^${label}\\d*$`) });
