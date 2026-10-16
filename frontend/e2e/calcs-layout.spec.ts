import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const BUILD = '0339-reliquarian-shield-crush-of-the-chieftain-2';
const LABEL = 136;
const GAP = 8;

const geometry = (page: Page) =>
  page.getByRole('region', { name: 'Calculations' }).evaluate((root) => {
    const sections = [...root.querySelectorAll('section[aria-label]')].filter((s) => !s.querySelector('[role=table]'));
    const boxes = sections.map((s) => {
      const rows = [...s.querySelectorAll('[data-key]')].map((r) => {
        const row = r.getBoundingClientRect();
        const values = r.lastElementChild!;
        const cells = [...values.children].map((c) => c.getBoundingClientRect());
        return { left: row.left, right: values.getBoundingClientRect().right, cellsLeft: Math.min(...cells.map((c) => c.left)), width: values.scrollWidth };
      });
      return { name: s.getAttribute('aria-label')!, x: Math.round(s.getBoundingClientRect().left), rows };
    });
    const tables = [...root.querySelectorAll('[role=table]')].map((t) => {
      const head = t.querySelector('[role=row]:has([role=columnheader])');
      const heads = head ? [...head.querySelectorAll('[role=columnheader]')].slice(1).map((h) => Math.round(h.getBoundingClientRect().right)) : [];
      const rows = [...t.querySelectorAll('[role=row][data-key]')].map((r) =>
        [...r.querySelectorAll('[role=cell]')].map((c) => Math.round(c.getBoundingClientRect().right))
      );
      return { name: t.getAttribute('aria-label')!, heads, rows };
    });
    return { boxes, tables };
  });

for (const viewport of [
  { width: 1920, height: 1080, columns: 3 },
  { width: 960, height: 1080, columns: 1 }
]) {
  test(`calcs layout at ${viewport.width}px: compact boxes, numbers beside the labels, columns that fit`, async ({ page }) => {
    test.setTimeout(10 * 60_000);
    await page.setViewportSize(viewport);
    const xml = buildXml(BUILD);
    const build = await go.loadXml(xml);
    const app = await App.open(page, BUILD);
    await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
    await app.goTo('Calcs');
    await expect(page.getByRole('region', { name: 'Calculations' }).locator('[data-key]').first()).toBeVisible();

    const { boxes, tables } = await geometry(page);
    const open = boxes.filter((b) => b.rows.length);
    expect(open.length, 'open section boxes').toBeGreaterThan(5);
    for (const b of open) {
      const rights = new Set(b.rows.map((r) => Math.round(r.right)));
      expect(rights.size, `${b.name}: the numbers share one right edge`).toBe(1);
      const widest = Math.max(...b.rows.map((r) => r.width));
      for (const r of b.rows) {
        expect(r.right - r.left, `${b.name}: the numbers sit beside the label column`).toBeLessThanOrEqual(LABEL + GAP + widest + 1);
        expect(r.cellsLeft - r.left, `${b.name}: no number reaches into the label column`).toBeGreaterThanOrEqual(LABEL + GAP - 1);
      }
    }
    const columns = new Set(boxes.map((b) => b.x)).size;
    if (viewport.columns > 1) expect(columns, 'the boxes flow into several columns').toBeGreaterThanOrEqual(viewport.columns);
    else expect(columns, 'one column at 960').toBe(1);

    expect(tables.length, 'damage-by-type tables').toBeGreaterThan(0);
    for (const t of tables.filter((x) => x.heads.length))
      for (const r of t.rows) expect(r, `${t.name}: each cell under its column head`).toEqual(t.heads.slice(0, r.length));
    app.assertNoErrors();
  });
}
