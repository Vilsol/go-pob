import { test, expect, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';
import { poe2Page } from './support/poe2';
import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

const routeAssets = async (page: Page) => {
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
};

const expectWebFontsOnly = async (page: Page) => {
  const failed = await page.evaluate(async () => {
    await document.fonts.ready;
    return [...document.fonts].filter((f) => f.status === 'error').map((f) => `${f.family} ${f.weight}`);
  });
  expect(failed, 'font faces failed to load').toEqual([]);

  const unstyled = await page.evaluate(() =>
    [document.documentElement, ...document.querySelectorAll('body *')]
      .filter((el) => getComputedStyle(el).display !== 'none' && !/^"?IBM Plex/.test(getComputedStyle(el).fontFamily))
      .map((el) => `<${el.tagName.toLowerCase()} class="${el.getAttribute('class') ?? ''}"> ${getComputedStyle(el).fontFamily}`)
  );
  expect(unstyled, 'elements on a font other than the app fonts').toEqual([]);

  const cdp = await page.context().newCDPSession(page);
  await cdp.send('DOM.enable');
  await cdp.send('CSS.enable');
  const { root } = await cdp.send('DOM.getDocument', { depth: -1, pierce: true });
  type Node = { nodeId: number; nodeType: number; nodeName: string; nodeValue?: string; children?: Node[]; shadowRoots?: Node[]; pseudoElements?: Node[] };
  const system: string[] = [];
  const walk = async (n: Node) => {
    if (n.nodeType === 1) {
      // a node the page removed while the walk ran draws nothing
      const fonts = await cdp
        .send('CSS.getPlatformFontsForNode', { nodeId: n.nodeId })
        .then((r) => r.fonts)
        .catch((e: Error) => {
          if (/Could not find node/.test(e.message)) return [];
          throw e;
        });
      const own = (n.children ?? [])
        .filter((c) => c.nodeType === 3)
        .map((c) => c.nodeValue)
        .join('')
        .trim();
      const text = n.nodeName.startsWith('::')
        ? ((await cdp.send('CSS.getComputedStyleForNode', { nodeId: n.nodeId }).catch(() => undefined))?.computedStyle.find((p) => p.name === 'content')
            ?.value ?? '')
        : own;
      for (const f of fonts)
        if (!f.isCustomFont) system.push(`<${n.nodeName.toLowerCase()}> ${JSON.stringify(text.slice(0, 40))} ${f.familyName} (${f.glyphCount} glyphs)`);
    }
    for (const c of [...(n.children ?? []), ...(n.shadowRoots ?? []), ...(n.pseudoElements ?? [])]) await walk(c);
  };
  await walk(root as Node);
  await cdp.detach();
  expect(system, 'glyphs drawn with a system font').toEqual([]);
};

// a system font on the first page (a control on the browser's default font, or a glyph the web fonts lack) costs
// Chromium on Linux a blocking font-config query per font and size inside the first layout
test('the build list draws every element and glyph with the web fonts, none with a system font', async ({ page }) => {
  await routeAssets(page);
  await page.goto('/');
  await page.getByRole('button', { name: 'New build', exact: true }).waitFor({ timeout: 120_000 });
  await expectWebFontsOnly(page);
});

const FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_items.xml');
const code = (path: string) => deflateSync(readFileSync(path)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

test('a PoE2 build opened on its Calcs page draws every element and glyph with the web fonts', async ({ page }) => {
  test.setTimeout(240_000);
  await page.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
  await routeAssets(page);
  await page.goto(`/calcs?import=${encodeURIComponent(code(FIXTURE))}`);
  await expect(poe2Page(page)).toBeVisible({ timeout: 180_000 });
  await expect(page.getByRole('region', { name: 'Calculations' }).locator('section').first()).toBeVisible();
  await expectWebFontsOnly(page);
});

test('a PoE1 build draws every page, and every Compare view, with the web fonts', async ({ page }) => {
  test.setTimeout(600_000);
  const xml = buildXml('0339-reliquarian-shield-crush-of-the-chieftain-2');
  const build = await go.loadXml(xml);
  const app = await App.open(page, '0339-reliquarian-shield-crush-of-the-chieftain-2');
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
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
  for (const link of ['Tree', 'Skills', 'Items', 'Calcs', 'Config', 'Party', 'Notes']) {
    const t0 = await app.now();
    await app.goTo(link);
    if (ready[link]) await app.followWorker(t0, ready[link]);
    await test.step(link, () => expectWebFontsOnly(page));
  }

  await app.openImport();
  const region = page.getByRole('dialog', { name: 'Import / Export' }).getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(await go.encode(buildXml('0280-elementalist-fireball')));
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('radio', { name: /Import as a comparison/ }).click();
  await region.getByRole('button', { name: 'Import', exact: true }).click();
  await page.waitForURL((u) => u.pathname.endsWith('/compare'));
  await expect(page.getByRole('region', { name: 'Stat comparison' })).toBeVisible({ timeout: 60_000 });
  for (const view of await page.getByRole('tablist', { name: 'Compare views' }).getByRole('tab').all()) {
    await view.click();
    if ((await view.textContent()) === 'Power report')
      await expect(page.getByRole('region', { name: 'Compare power report' }).getByRole('status')).toHaveCount(0, { timeout: 180_000 });
    await test.step(`Compare ${await view.textContent()}`, () => expectWebFontsOnly(page));
  }
  app.assertNoErrors();
});
