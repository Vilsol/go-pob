import { test, expect, type BrowserContext, type Page } from '@playwright/test';
import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { deflateSync } from 'node:zlib';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { DATA_HOST, cachedAsset } from './support/data_cache';
import * as go from './support/go_model';
import { poe2Stat } from './support/poe2';
import { expectedLines } from './support/stats';
import { routeTreeAssets } from './support/tree_assets';

// the TinyGo builds the bundle serves (mise run e2e builds them when it runs this spec); CI's shards build standard Go
// only and skip these, its TinyGo job sets E2E_TINYGO=1 so a missing build fails instead
const BUILT = ['go-pob-tiny.wasm.gz', 'go-pob2-tiny.wasm.gz'].every((f) => existsSync(resolve(import.meta.dirname, '../build', f)));
const needsTinyGo = () => {
  if (process.env.E2E_TINYGO === '1') expect(BUILT, 'the bundle has no TinyGo wasm: mise run e2e-bundle --tinygo').toBe(true);
  else test.skip(!BUILT, 'the bundle has no TinyGo wasm (mise run e2e-bundle --tinygo builds it)');
};

const POE2_FIXTURE = resolve(import.meta.dirname, '../../wasm2/exposition/testdata/sorceress_spark.xml');
const poe2Code = () => deflateSync(readFileSync(POE2_FIXTURE)).toString('base64').replaceAll('+', '-').replaceAll('/', '_');

// once per tab, so a reload keeps what the page itself chose since
const chooseTinyGo = (context: BrowserContext) =>
  context.addInitScript(() => {
    if (sessionStorage.getItem('e2e:engine') !== null) return;
    sessionStorage.setItem('e2e:engine', 'tinygo');
    localStorage.setItem('options:engine', 'tinygo');
  });

// the wasm files the page asked for, by name
const wasmRequests = (page: Page) => {
  const seen: string[] = [];
  page.on('request', (r) => {
    const name = new URL(r.url()).pathname.split('/').pop() ?? '';
    if (name.includes('.wasm')) seen.push(name);
  });
  return seen;
};

const routeData = (context: BrowserContext) =>
  context.route(`${DATA_HOST}**`, async (route) => {
    const asset = cachedAsset(route.request().url());
    return route.fulfill({
      status: asset.status,
      body: asset.status === 200 ? asset.body : undefined,
      headers: { 'access-control-allow-origin': '*', 'content-type': 'application/octet-stream' }
    });
  });

const importPoe2 = async (page: Page) => {
  await page.getByRole('button', { name: 'Import build', exact: true }).click();
  const region = page.getByRole('dialog', { name: 'Import build' }).getByRole('region', { name: 'Build import' });
  const box = region.getByRole('textbox', { name: /^To import a build/ });
  await box.fill(poe2Code());
  await expect(box).toHaveAccessibleDescription('Code is valid');
  await region.getByRole('button', { name: 'Import', exact: true }).click();
};

// the PoE2 fixture's sidebar Life on one engine, read from a fresh page
const poe2Life = async (context: BrowserContext) => {
  const page = await context.newPage();
  const wasm = wasmRequests(page);
  const errors: string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  await routeTreeAssets(page);
  await page.goto('/');
  await expect(page.getByRole('button', { name: 'Import build', exact: true })).toBeVisible({ timeout: 180_000 });
  await importPoe2(page);
  const life = poe2Stat(page, 'Life');
  await expect(life).toHaveText(/^\d/, { timeout: 180_000 });
  const value = await life.textContent();
  await page.close();
  return { value, wasm, errors };
};

// the app as the other App specs open it, its service worker running
test('with TinyGo chosen, a PoE1 build shows the sidebar standard Go computes, and only TinyGo wasm loads (0280)', async ({ context, page }) => {
  needsTinyGo();
  const file = '0280-elementalist-fireball';
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  await chooseTinyGo(context);
  const wasm = wasmRequests(page);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  expect(wasm.filter((w) => w.startsWith('go-pob.wasm'))).toEqual([]);
  expect(wasm.some((w) => w.startsWith('go-pob-tiny.wasm'))).toBe(true);
  app.assertNoErrors();
});

test.describe(() => {
  test.use({ serviceWorkers: 'block' });

  test('with TinyGo chosen, a PoE2 build shows the sidebar Life the standard Go engine shows', async ({ browser }) => {
    needsTinyGo();
    const runs = [];
    for (const tiny of [false, true]) {
      const context = await browser.newContext({ serviceWorkers: 'block' });
      await context.addInitScript(() => localStorage.setItem('options:poe2', 'true'));
      if (tiny) await chooseTinyGo(context);
      await routeData(context);
      runs.push(await poe2Life(context));
      await context.close();
    }
    const [std, tiny] = runs;
    expect(std.wasm.some((w) => w.startsWith('go-pob2.wasm'))).toBe(true);
    expect(tiny.wasm.some((w) => w.startsWith('go-pob2-tiny.wasm'))).toBe(true);
    expect(tiny.wasm.filter((w) => w.startsWith('go-pob.wasm') || w.startsWith('go-pob2.wasm'))).toEqual([]);
    expect(tiny.value).toBe(std.value);
    expect([...std.errors, ...tiny.errors]).toEqual([]);
  });

  test('Settings switches the engine to TinyGo and back, each with a reload', async ({ page }) => {
    needsTinyGo();
    const wasm = wasmRequests(page);
    await routeData(page.context());
    await routeTreeAssets(page);
    await page.goto('/');
    await expect(page.getByRole('button', { name: 'Import build', exact: true })).toBeVisible({ timeout: 180_000 });

    const pick = async (engine: string) => {
      await page.getByRole('button', { name: 'Settings', exact: true }).click();
      const sheet = page.getByRole('dialog', { name: 'Settings' });
      await sheet.getByRole('combobox', { name: 'Engine', exact: true }).selectOption(engine);
      const reloaded = page.waitForEvent('load');
      await sheet.getByRole('button', { name: 'Save' }).click();
      await reloaded;
      await expect(page.getByRole('button', { name: 'Import build', exact: true })).toBeVisible({ timeout: 180_000 });
    };

    await pick('TinyGo (experimental)');
    expect(await page.evaluate(() => localStorage.getItem('options:engine'))).toBe('tinygo');
    expect(wasm.at(-1)).toMatch(/^go-pob-tiny\.wasm/);

    await pick('Go');
    expect(await page.evaluate(() => localStorage.getItem('options:engine'))).toBe('go');
    expect(wasm.at(-1)).toMatch(/^go-pob\.wasm/);

    // Cancel keeps the engine and does not reload
    await page.getByRole('button', { name: 'Settings', exact: true }).click();
    const sheet = page.getByRole('dialog', { name: 'Settings' });
    await sheet.getByRole('combobox', { name: 'Engine', exact: true }).selectOption('TinyGo (experimental)');
    const before = wasm.length;
    await sheet.getByRole('button', { name: 'Cancel' }).click();
    await page.waitForTimeout(1_000);
    expect(wasm.length).toBe(before);
    expect(await page.evaluate(() => localStorage.getItem('options:engine'))).toBe('go');
  });

  test('a TinyGo engine that does not boot switches back to Go, with a notice', async ({ context, page }) => {
    await chooseTinyGo(context);
    await page.route('**/go-pob-tiny.wasm*', (route) => route.fulfill({ body: Buffer.from([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0xff, 0xff]) }));
    const wasm = wasmRequests(page);
    await routeData(context);
    await routeTreeAssets(page);
    await page.goto('/');
    await expect(page.getByText('Switched back to the Go engine')).toBeVisible({ timeout: 180_000 });
    await expect(page.getByRole('button', { name: 'Import build', exact: true })).toBeVisible({ timeout: 180_000 });
    expect(await page.evaluate(() => localStorage.getItem('options:engine'))).toBe('go');
    expect(await page.evaluate(() => localStorage.getItem('options:engineFallback'))).toBeNull();
    expect(wasm.some((w) => w.startsWith('go-pob-tiny.wasm'))).toBe(true);
    expect(wasm.at(-1)).toMatch(/^go-pob\.wasm/);
  });
});
