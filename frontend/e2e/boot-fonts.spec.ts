import { test, expect } from '@playwright/test';

import { DATA_HOST, cachedAsset } from './support/data_cache';
import { routeTreeAssets } from './support/tree_assets';

// a system font on the first page (a control on the browser's default font, or a glyph the web fonts lack) costs
// Chromium on Linux a blocking font-config query per font and size inside the first layout
test('the build list draws every element and glyph with the web fonts, none with a system font', async ({ page }) => {
  await page.context().route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });
  await routeTreeAssets(page);
  await page.goto('/');
  await page.getByRole('button', { name: 'New build', exact: true }).waitFor({ timeout: 120_000 });

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
  type Node = { nodeId: number; nodeType: number; nodeName: string; children?: Node[]; shadowRoots?: Node[]; pseudoElements?: Node[] };
  const system: string[] = [];
  const walk = async (n: Node) => {
    if (n.nodeType === 1) {
      const { fonts } = await cdp.send('CSS.getPlatformFontsForNode', { nodeId: n.nodeId });
      for (const f of fonts) if (!f.isCustomFont) system.push(`<${n.nodeName.toLowerCase()}> ${f.familyName} (${f.glyphCount} glyphs)`);
    }
    for (const c of [...(n.children ?? []), ...(n.shadowRoots ?? []), ...(n.pseudoElements ?? [])]) await walk(c);
  };
  await walk(root as Node);
  expect(system, 'glyphs drawn with a system font').toEqual([]);
});
