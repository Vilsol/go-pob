import { mkdirSync, writeFileSync } from 'node:fs';

import { test, expect, type Page } from '@playwright/test';

import { layoutFindings, type Finding, type Scroller } from './support/layout';
import { PHONE, bar, openMore, openPhoneBuild } from './support/phone';

const ROOT = '[data-mobile-frame]';

const SCROLLERS: Scroller[] = [
  { selector: 'main', axis: 'y', why: 'every page scrolls inside the frame, between the strip and the tab bar' },
  { selector: '[data-stats-strip]', axis: 'x', why: 'more than four pinned stats scroll sideways' },
  { selector: '[aria-label="Calculations"]', axis: 'y', why: 'the Calcs sections run past one screen' },
  { selector: '[data-calcs-wide]', axis: 'x', why: 'a Calcs damage-by-type table has a column per damage type' },
  { selector: 'main[aria-label="Config"] > .scroll', axis: 'y', why: 'the Config options run past one screen' },
  { selector: 'textarea', axis: 'y', why: 'a share code or a note longer than its box' },
  { selector: 'main [role=tabpanel]', axis: 'y', why: 'a Skills pane or a Compare view runs past one screen' },
  { selector: 'main [role=tabpanel]', axis: 'x', why: 'the Skills panes and Compare tables swipe sideways' },
  { selector: '.phone-grid', axis: 'x', why: 'a wide list as a sticky table scrolls sideways under its name column' },
  { selector: '[role=tablist]', axis: 'x', why: 'a tab strip wider than the phone scrolls sideways' },
  { selector: '.scroll', axis: 'y', why: 'a list (socket groups, items) that runs past its panel' },
  { selector: '[data-tree-bar]', axis: 'x', why: 'the tree’s bottom bar (search, filter, reset) scrolls sideways on a phone' }
];

const PAGES: { name: string; open: (page: Page) => Promise<void> }[] = [
  ...['Tree', 'Skills', 'Items', 'Calcs', 'Config'].map((t) => ({ name: t, open: (page: Page) => bar(page).getByRole('link', { name: t }).click() })),
  ...['Notes', 'Party', 'Compare', 'Import/Export'].map((t) => ({
    name: t,
    open: async (page: Page) => (await openMore(page)).getByRole('link', { name: t }).click()
  }))
];

const settle = async (page: Page) => {
  let last = '';
  for (let i = 0; i < 240; i++) {
    await page.waitForTimeout(250);
    const now = await page.evaluate((root) => {
      const el = document.querySelector(root);
      if (!el) return '';
      let geo = 0;
      for (const e of el.querySelectorAll('*')) {
        const r = e.getBoundingClientRect();
        geo += r.left + 2 * r.top + 3 * r.width + 5 * r.height;
      }
      return `${el.textContent}|${Math.round(geo)}`;
    }, ROOT);
    if (now && now === last) return;
    last = now;
  }
  throw new Error('the page did not settle');
};
// a mobile browser widens its layout viewport to fit content that is too wide, so window.innerWidth follows the
// defect; the page must stay at the device width the context was given
const phoneFindings = async (page: Page): Promise<Finding[]> => {
  const [inner, scroll] = await page.evaluate(() => [window.innerWidth, (document.scrollingElement ?? document.documentElement).scrollWidth]);
  const width = PHONE.viewport.width;
  const own: Finding[] =
    inner > width || scroll > width
      ? [{ check: 'page-scroll', where: 'document', detail: `layout viewport ${inner}, scrollWidth ${scroll} > device ${width}` }]
      : [];
  return [...own, ...(await layoutFindings(page, ROOT, SCROLLERS))];
};
const report = (fs: Finding[]) => fs.map((f) => `${f.check}: ${f.where} — ${f.detail}`).join('\n');

test.describe('390×769 layout', () => {
  test.use(PHONE);
  test.describe.configure({ mode: 'default' });
  let found: Record<string, Finding[]> | undefined;

  test.beforeAll(async ({ browser }, info) => {
    test.setTimeout(10 * 60_000);
    const context = await browser.newContext({ ...PHONE, baseURL: info.project.use.baseURL });
    const page = await context.newPage();
    try {
      const { app } = await openPhoneBuild(page);
      found = {};
      for (const p of PAGES) {
        await p.open(page);
        await expect(page.getByRole('dialog', { name: 'More' })).toBeHidden();
        await settle(page);
        await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
        found[p.name] = await phoneFindings(page);
      }
      app.assertNoErrors();
      mkdirSync(new URL('./.results/', import.meta.url), { recursive: true });
      writeFileSync(new URL('./.results/layout-phone.json', import.meta.url), JSON.stringify(found, null, 1));
    } finally {
      await context.close();
    }
  });

  for (const p of PAGES) {
    test(p.name, () => {
      const fs = found?.[p.name];
      expect(fs, `${p.name} was not measured`).toBeDefined();
      expect(fs, `${p.name} at 390×769:\n${report(fs!)}`).toEqual([]);
    });
  }
});

test.describe('390×769 layout checks', () => {
  test.use(PHONE);
  test('each check fails on its defect', async ({ page }) => {
    test.setTimeout(5 * 60_000);
    const { app } = await openPhoneBuild(page);
    await (await openMore(page)).getByRole('link', { name: 'Notes' }).click();
    await expect(page.getByRole('toolbar', { name: 'Formatting' })).toBeVisible();
    await settle(page);
    expect(await phoneFindings(page), 'the control page is clean').toEqual([]);

    const mutations: { check: Finding['check']; what: string; target: string; css: string }[] = [
      { check: 'page-scroll', what: 'the frame keeps a desktop minimum width', target: ROOT, css: '{ min-width: 800px }' },
      { check: 'clipped', what: 'Save pushed past the window edge', target: 'header [aria-label^="Save"]', css: '{ margin-right: -200px }' },
      { check: 'overlap', what: 'Redo drawn over Undo', target: 'header [aria-label="Redo"]', css: '{ position: relative; left: -24px }' }
    ];
    for (const m of mutations) {
      await expect(page.locator(m.target), `${m.what}: its target`).toHaveCount(1);
      const style = await page.addStyleTag({ content: `${m.target} ${m.css}` });
      await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
      expect(
        (await phoneFindings(page)).map((f) => f.check),
        `${m.what}: the ${m.check} check stays green`
      ).toContain(m.check);
      await style.evaluate((el: Element) => el.remove());
      await page.evaluate(() => {
        for (const el of document.querySelectorAll('*')) [el.scrollLeft, el.scrollTop] = [0, 0];
      });
      expect(await phoneFindings(page), `${m.what}: undone`).toEqual([]);
    }
    app.assertNoErrors();
  });
});
