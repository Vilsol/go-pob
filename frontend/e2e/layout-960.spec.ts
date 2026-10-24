import { mkdirSync, writeFileSync } from 'node:fs';

import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { layoutFindings, type Finding, type Scroller } from './support/layout';
import { expectedLines } from './support/stats';

const VIEWPORT = { width: 960, height: 1080 };

const SCROLLERS: Scroller[] = [
  { selector: '[aria-label="Build stats"]', axis: 'y', why: 'the sidebar stat list is as long as the build' },
  { selector: '[aria-label="Calculations"]', axis: 'y', why: 'the Calcs sections run past one screen' },
  { selector: '[data-calcs-wide]', axis: 'x', why: 'a Calcs damage-by-type table has a column per damage type' },
  { selector: 'main[aria-label="Config"] > .scroll', axis: 'y', why: 'the Config options run past one screen' },
  { selector: 'textarea[aria-label="Build code"]', axis: 'y', why: 'a read-only share code longer than its box' },
  { selector: 'main > [role=tabpanel]', axis: 'y', why: 'a Compare view runs past one screen' },
  { selector: 'main.scroll', axis: 'y', why: 'a page that scrolls as a whole (Items, Import / Export) runs past one screen' }
];

const BUILDS = ['0280-elementalist-fireball', '0008-necromancer-vaal-summon-skeletons', '0339-reliquarian-shield-crush-of-the-chieftain-2'];
const COMPARE = '0280-elementalist-fireball';
const ROUTES = ['Tree', 'Skills', 'Items', 'Calcs', 'Config', 'Party', 'Notes', 'Import / Export'];
const COMPARE_VIEWS = ['Summary', 'Power report', 'Tree', 'Items', 'Skills', 'Calcs', 'Config'];

const FIXME: Record<string, { owner: string; finding: string; builds?: string[] }> = {
  Tree: {
    owner: 'unowned, Tree page',
    finding: 'the tree toolbar does not fit 960: Timeless search cut 25px, the tree-version picker (160px) and a checkbox cut whole'
  },
  Items: {
    owner: 'unowned, Items page',
    finding:
      'once the board has filled in, the equipped-items board is wider than 960: Weapon 2 cut whole, Boots and Flask 5 cut by half, the tree-jewel socket selects cut'
  },
  Skills: {
    owner: 'unowned, Skills page',
    finding: "a gem row's level stepper spills 7px out of its column onto the row's Remove button (GemRow grid)"
  },
  Config: {
    owner: 'unowned, Config page',
    finding: 'a long custom modifier line cuts its input 76px and its Move later button (ConfigCustom)',
    builds: ['0008-necromancer-vaal-summon-skeletons']
  },
  'Compare › Power report': {
    owner: 'unowned, Compare page',
    finding: 'the rows’ Equip / Take / Use buttons sit wholly past the right edge of the report',
    builds: ['0008-necromancer-vaal-summon-skeletons', '0339-reliquarian-shield-crush-of-the-chieftain-2']
  },
  'Compare › Tree': {
    owner: 'unowned, Compare page',
    finding: 'the Node differences Drop / Take button is cut 13px',
    builds: ['0008-necromancer-vaal-summon-skeletons', '0339-reliquarian-shield-crush-of-the-chieftain-2']
  },
  'Compare › Config': {
    owner: 'unowned, Compare page',
    finding: 'the B WITH A’S VALUE column header is cut 9px'
  }
};

const settle = async (page: Page) => {
  let last = '';
  for (let i = 0; i < 240; i++) {
    await page.waitForTimeout(250);
    const now = await page.evaluate(() => {
      const root = document.querySelector('.page');
      if (!root) return '';
      let geo = 0;
      for (const el of root.querySelectorAll('*')) {
        const r = el.getBoundingClientRect();
        geo += r.left + 2 * r.top + 3 * r.width + 5 * r.height;
      }
      return `${root.textContent}|${Math.round(geo)}`;
    });
    if (now && now === last) return;
    last = now;
  }
  throw new Error('the page did not settle');
};
const rest = async (page: Page) => {
  await page.mouse.move(VIEWPORT.width - 1, VIEWPORT.height - 1);
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
};
const report = (fs: Finding[]) => fs.map((f) => `${f.check}: ${f.where} — ${f.detail}`).join('\n');

const measure = async (page: Page, file: string) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(`layout 960 ${file}`);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
  await expect(page.getByRole('button', { name: 'Save As' })).toBeVisible();

  const found: Record<string, Finding[]> = {};
  for (const link of ROUTES) {
    await app.goTo(link);
    await settle(page);
    await rest(page);
    found[link] = await layoutFindings(page, '.page', SCROLLERS);
  }

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
  expect(await views.allTextContents(), 'the Compare views this spec names').toEqual(COMPARE_VIEWS);
  for (const view of await views.all()) {
    await view.click();
    await settle(page);
    if ((await view.textContent()) === 'Power report')
      await expect(page.getByRole('region', { name: 'Compare power report' }).getByRole('status')).toHaveCount(0, { timeout: 180_000 });
    await rest(page);
    found[`Compare › ${await view.textContent()}`] = await layoutFindings(page, '.page', SCROLLERS);
  }
  app.assertNoErrors();

  mkdirSync(new URL('./.results/', import.meta.url), { recursive: true });
  writeFileSync(new URL(`./.results/layout-960-${file}.json`, import.meta.url), JSON.stringify(found, null, 1));
  return found;
};

for (const file of BUILDS) {
  test.describe(`960px layout: ${file}`, () => {
    test.describe.configure({ mode: 'default' });
    let found: Record<string, Finding[]> | undefined;
    test.beforeAll(async ({ browser }, info) => {
      test.setTimeout(10 * 60_000);
      const page = await browser.newPage({ viewport: VIEWPORT, baseURL: info.project.use.baseURL });
      try {
        found = await measure(page, file);
      } finally {
        await page.close();
      }
    });
    for (const route of [...ROUTES, ...COMPARE_VIEWS.map((v) => `Compare › ${v}`)]) {
      const known = FIXME[route];
      if (known && (!known.builds || known.builds.includes(file))) {
        test.fixme(`${route} (${known.owner}: ${known.finding})`, () => {});
        continue;
      }
      test(route, () => {
        const fs = found?.[route];
        expect(fs, `${route} was not measured`).toBeDefined();
        expect(fs, `${route} at 960px:\n${report(fs!)}`).toEqual([]);
      });
    }
  });
}

test.describe('960px layout checks', () => {
  test.use({ viewport: VIEWPORT });
  test('each check fails on its defect', async ({ page }) => {
    test.setTimeout(5 * 60_000);
    const app = await App.open(page, 'new build');
    await app.goTo('Notes');
    await expect(page.getByRole('toolbar', { name: 'Formatting' })).toBeVisible();
    await settle(page);
    await rest(page);
    expect(await layoutFindings(page, '.page', SCROLLERS), 'the control page is clean').toEqual([]);

    const mutations: { check: Finding['check']; what: string; target: string; css: string }[] = [
      { check: 'page-scroll', what: 'the page keeps a 1280px minimum width', target: '.page', css: '{ min-width: 1280px }' },
      { check: 'clipped', what: 'Settings pushed past the window edge', target: 'header [aria-label="Settings"]', css: '{ margin-right: -200px }' },
      { check: 'scrollbar', what: 'the Formatting toolbar scrolls', target: '[aria-label="Formatting"]', css: '{ overflow-y: auto; max-height: 20px }' },
      { check: 'overlap', what: 'Settings drawn over About', target: 'header [aria-label="Settings"]', css: '{ position: relative; left: -24px }' }
    ];
    for (const m of mutations) {
      await expect(page.locator(m.target), `${m.what}: its target`).toHaveCount(1);
      const style = await page.addStyleTag({ content: `${m.target} ${m.css}` });
      await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
      const fs = await layoutFindings(page, '.page', SCROLLERS);
      expect(
        fs.map((f) => f.check),
        `${m.what}: the ${m.check} check stays green`
      ).toContain(m.check);
      await style.evaluate((el: Element) => el.remove());
      await page.evaluate(() => {
        for (const el of document.querySelectorAll('*')) [el.scrollLeft, el.scrollTop] = [0, 0];
      });
      expect(await layoutFindings(page, '.page', SCROLLERS), `${m.what}: undone`).toEqual([]);
    }
    app.assertNoErrors();
  });
});
