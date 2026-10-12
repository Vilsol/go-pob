import { test, expect, chromium, type BrowserContext, type Page } from '@playwright/test';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

import { App } from './support/app';
import { buildXml } from './support/builds';
import { DATA_HOST } from './support/data_cache';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { isTreeAsset, TREE_ASSET_PATTERN } from './support/tree_assets';

test.describe.configure({ mode: 'default' });

const BASE = process.env.E2E_BASE ?? '';
const FONT_HOSTS = ['https://fonts.googleapis.com/', 'https://fonts.gstatic.com/'];

const saveAs = async (page: Page, name: string) => {
  await page.getByRole('button', { name: /^Save \(/ }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox', { name: 'Build name' }).fill(name);
  await dialog.getByRole('button', { name: 'Save', exact: true }).click();
  await expect(dialog).toBeHidden();
};

const goOffline = async (context: BrowserContext, origin: string) => {
  const refused: string[] = [];
  await context.unrouteAll({ behavior: 'ignoreErrors' });
  for (const host of [`${origin}/`, DATA_HOST, ...FONT_HOSTS]) {
    await context.route(`${host}**`, (route) => {
      refused.push(route.request().url());
      return route.abort('internetdisconnected');
    });
  }
  await context.setOffline(true);
  return refused;
};

test('offline after one online visit: reload, open a saved build, edit it', async ({ page, context, baseURL }) => {
  const file = '0280-elementalist-fireball';
  const name = `e2e-offline-${file}`;
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const want = async () => expectedLines(await go.outputs(build));

  if (BASE) {
    await page.route(`${baseURL}/`, (route) => route.fulfill({ status: 302, headers: { location: `${BASE}/` } }));
  }

  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), want);
  await saveAs(page, name);
  await app.openTree();
  await page.waitForFunction(
    (pattern) => {
      const rs = performance.getEntriesByType('resource').filter((r) => new RegExp(pattern).test(r.name));
      return rs.length > 0 && rs.every((r) => (r as PerformanceResourceTiming).responseEnd > 0);
    },
    TREE_ASSET_PATTERN,
    { timeout: 60_000 }
  );
  const sw = await page.evaluate(async () => {
    const reg = await navigator.serviceWorker?.getRegistration();
    if (!reg) return 'none';
    await Promise.race([navigator.serviceWorker.ready, new Promise((r) => setTimeout(r, 120_000))]);
    return reg.active?.state ?? 'not active';
  });
  console.log(`service worker after the online visit: ${sw}`);
  if (sw === 'activated') {
    await expect
      .poll(
        () =>
          page.evaluate(async (pattern) => {
            const urls = performance
              .getEntriesByType('resource')
              .map((r) => r.name)
              .filter((u) => new RegExp(pattern).test(u));
            const missing = [];
            for (const u of new Set(urls)) if (!(await caches.match(u))) missing.push(u);
            return missing;
          }, TREE_ASSET_PATTERN),
        { timeout: 60_000, message: 'tree assets the service worker never cached' }
      )
      .toEqual([]);
    const kept = await page.waitForFunction(
      () =>
        performance
          .getEntriesByName('pob:offline-kept')
          .map((m) => (m as PerformanceMark).detail as { tag: string; count: number; missing: string[] })
          .find((d) => d.tag === 'tree' && d.count > 0),
      undefined,
      { timeout: 120_000 }
    );
    const answer = (await kept.jsonValue()) as { count: number; missing: string[] };
    console.log(`tree sheets kept for offline: ${answer.count}, missing ${answer.missing.length}`);
    expect(answer.missing, 'tree sheets the service worker could not keep').toEqual([]);
  }
  if (BASE) await page.unroute(`${baseURL}/`);

  const refused = await goOffline(context, new URL(baseURL!).origin);
  const sheetErrors: string[] = [];
  page.on('console', (m) => {
    if (m.type() === 'error' && m.text().startsWith('Failed to load resource') && isTreeAsset(m.location().url)) sheetErrors.push(m.location().url);
  });
  await page.goto(`${BASE}/`);
  try {
    await app.savedBuild(name).waitFor({ timeout: 180_000 });
    await page.waitForFunction(() => performance.getEntriesByName('pob:data-loaded').length > 0, undefined, { timeout: 180_000 });
  } catch (e) {
    throw new Error(
      `the offline reload did not reach the build list with game data (refused: ${JSON.stringify(refused)}; errors: ${JSON.stringify(app.errors)})`,
      {
        cause: e
      }
    );
  }
  await expect(page.getByRole('status').filter({ hasText: /^Offline/ }), 'no offline indicator').toBeVisible();
  await app.act('open-saved-offline', () => app.openSaved(name), want);

  await app.openBuildControls();
  const level = page.getByRole('spinbutton', { name: 'Character level' });
  const n = Math.max(1, build.Build.Level - 20);
  await app.act(
    'set-level-offline',
    () => level.fill(String(n)),
    async () => {
      // Build.lua:218-224
      build.SetCharacterLevelAutoMode(false);
      build.SetLevel(n);
      return want();
    }
  );
  await expect(level).toHaveValue(String(n));
  await expect(page.getByRole('img', { name: 'Passive skill tree' })).toBeVisible();

  expect(refused, 'requests that needed the network while offline').toEqual([]);
  expect(sheetErrors, 'tree sheets that failed to load offline').toEqual([]);
  expect(app.errors, `console errors / unhandled exceptions on ${file}`).toEqual([]);
});

test('installable: Chrome finds the manifest and reports no installability error', async ({ baseURL }, testInfo) => {
  const profile = mkdtempSync(join(tmpdir(), 'pob-install-'));
  const context = await chromium.launchPersistentContext(profile, { executablePath: testInfo.project.use.launchOptions?.executablePath, baseURL });
  try {
    for (const host of [DATA_HOST]) await context.route(`${host}**`, (route) => route.abort());
    const page = await context.newPage();
    await page.goto(`${BASE}/`);
    await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
    await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller), { timeout: 120_000 }).toBe(true);

    const cdp = await context.newCDPSession(page);
    const manifest = (await cdp.send('Page.getAppManifest')) as { url: string; errors: { message: string }[]; data?: string };
    expect(manifest.url, 'the page links no manifest').toMatch(new RegExp(`${BASE}/manifest\\.webmanifest$`));
    expect(manifest.errors, 'manifest parse errors').toEqual([]);
    const parsed = JSON.parse(manifest.data ?? '{}') as { display?: string; icons?: { sizes: string; purpose?: string }[] };
    expect(parsed.display).toBe('standalone');
    expect(parsed.icons?.map((i) => `${i.sizes} ${i.purpose ?? 'any'}`)).toEqual(['192x192 any', '512x512 any', '512x512 maskable']);

    console.log(`persistent storage in a fresh Chromium profile: ${await page.evaluate(() => navigator.storage.persisted())}`);
    const { installabilityErrors } = (await cdp.send('Page.getInstallabilityErrors')) as { installabilityErrors: { errorId: string }[] };
    expect(installabilityErrors.map((e) => e.errorId)).toEqual([]);
  } finally {
    await context.close();
    rmSync(profile, { recursive: true, force: true });
  }
});

const expectUpdate = async (page: Page, version: string, next: string, change?: string) => {
  const notReloaded = () => page.evaluate(() => (globalThis as { __pobSameDocument?: boolean }).__pobSameDocument === true).catch(() => undefined);
  await page.evaluate(() => ((globalThis as { __pobSameDocument?: boolean }).__pobSameDocument = true));
  await page.evaluate(async () => {
    await (await navigator.serviceWorker.getRegistration())?.update();
  });
  await expect
    .poll(
      () =>
        page.evaluate(async () => {
          const reg = await navigator.serviceWorker.getRegistration();
          return reg?.waiting ? 'waiting' : reg?.installing ? 'installing' : 'no new worker';
        }),
      { timeout: 120_000 }
    )
    .toBe('waiting');
  const prompt = page.getByRole('alert').filter({ hasText: 'A new version of go-pob is available' });
  await expect(prompt).toBeVisible({ timeout: 120_000 });
  await page.waitForTimeout(3_000);
  expect(await notReloaded(), 'the page reloaded without the click').toBe(true);
  expect(await page.evaluate(() => caches.keys())).toContain(`pob-app-${version}`);
  if (change) {
    await prompt.getByRole('button', { name: 'Changes…' }).click();
    await expect(prompt.getByText(change)).toBeVisible();
  }

  await prompt.getByRole('button', { name: 'Reload' }).click();
  await expect.poll(notReloaded, { timeout: 60_000 }).toBe(false);
  await expect.poll(() => page.evaluate(() => caches.keys()), { timeout: 60_000 }).toContain(`pob-app-${next}`);
  expect(await page.evaluate(() => caches.keys()), 'the old deployment cache outlived the update').not.toContain(`pob-app-${version}`);
  await expect(prompt).toBeHidden();
};

test('an update waits for the click: the prompt shows, nothing reloads until Reload, then the new version runs', async ({ page, context }) => {
  test.skip(process.env.E2E_SERVER === 'dev', 'needs the production build that vite preview serves');
  for (const host of [DATA_HOST]) await context.route(`${host}**`, (route) => route.abort());
  await page.goto(`${BASE}/`);
  await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
  await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller), { timeout: 120_000 }).toBe(true);

  const client = resolve(import.meta.dirname, '../.svelte-kit/output/client');
  const script = resolve(client, 'service-worker.js');
  const original = readFileSync(script, 'utf8');
  const { version } = JSON.parse(readFileSync(resolve(client, '_app/version.json'), 'utf8')) as { version: string };
  expect(original, 'the service worker does not carry the build version').toContain(version);
  const next = version.slice(0, -1) + String((Number(version.at(-1)) + 1) % 10);
  // Main.lua:1300-1320
  const log = resolve(client, 'changelog.txt');
  const changelog = readFileSync(log, 'utf8');
  const [h1, h2] = [...changelog.matchAll(/^VERSION\[(.+)\]\[(.+)\]$/gm)];
  expect(h1[1].length, 'the first two versions have one length').toBe(h2[1].length);
  const newer = h1[1].slice(0, -1) + String((Number(h1[1].at(-1)) + 1) % 10);
  writeFileSync(script, original.replaceAll(version, next));
  writeFileSync(log, changelog.replace(h1[0], `VERSION[${newer}][${h1[2]}]`).replace(h2[0], `VERSION[${h1[1]}][${h2[2]}]`));
  try {
    await expectUpdate(page, version, next, `Version ${newer} (${h1[2]})`);
  } finally {
    writeFileSync(script, original);
    writeFileSync(log, changelog);
  }
});

test('kill switch: ?sw=off unregisters the service worker and deletes its caches until ?sw=on', async ({ page, context }) => {
  for (const host of [DATA_HOST]) await context.route(`${host}**`, (route) => route.abort());
  const state = () =>
    page.evaluate(async () => ({
      registered: (await navigator.serviceWorker.getRegistrations()).length,
      caches: (await caches.keys()).filter((k) => k.startsWith('pob-app-')).length
    }));
  await page.goto(`${BASE}/`);
  await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
  await expect.poll(state, { timeout: 120_000 }).toEqual({ registered: 1, caches: 1 });

  await page.goto(`${BASE}/?sw=off`);
  await expect.poll(state, { timeout: 30_000 }).toEqual({ registered: 0, caches: 0 });
  await page.goto(`${BASE}/`);
  await page.waitForTimeout(3_000);
  expect(await state(), 'the switch did not stay off').toEqual({ registered: 0, caches: 0 });

  await page.goto(`${BASE}/?sw=on`);
  await expect.poll(state, { timeout: 120_000 }).toEqual({ registered: 1, caches: 1 });
});

// Main.lua:212-229, 374-381, Launch.lua:158-161
test('Check for Update and Ctrl+U say there is no update when the host serves this version', async ({ page, context }) => {
  for (const host of [DATA_HOST]) await context.route(`${host}**`, (route) => route.abort());
  await page.goto(`${BASE}/`);
  await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
  await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller), { timeout: 120_000 }).toBe(true);
  const none = page.getByRole('status').filter({ hasText: 'No update available' });
  await page.getByRole('button', { name: 'Check for Update' }).click();
  await expect(none).toHaveCount(1, { timeout: 60_000 });
  await page.keyboard.press('Control+u');
  await expect(none).toHaveCount(2, { timeout: 60_000 });
});
