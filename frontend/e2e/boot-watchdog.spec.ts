import { test, expect } from '@playwright/test';

test.use({ serviceWorkers: 'block' });

const status = (page: import('@playwright/test').Page) => page.getByRole('status').filter({ hasText: 'go-pob' });

test('a wasm runtime that fails to boot shows why, with a retry', async ({ page }) => {
  // the wasm magic and version, then a section that is not one
  await page.route('**/go-pob.wasm.gz', (route) => route.fulfill({ body: Buffer.from([0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0xff, 0xff]) }));
  await page.goto('/');
  await expect(status(page)).toContainText('Booting failed:', { timeout: 60_000 });
  await expect(status(page).getByRole('button', { name: 'Retry' })).toBeVisible();
});

test('a boot that never finishes turns into an error with a retry that reloads', async ({ page }) => {
  // an upgrade of the builds database that never ends: the worker's own open of it waits behind it
  await page.addInitScript(() => {
    const req = indexedDB.open('builds', 1);
    req.onupgradeneeded = () => {
      const store = req.result.createObjectStore('builds');
      const spin = () => (store.get(0).onsuccess = spin);
      spin();
    };
  });
  await page.clock.install();
  await page.goto('/');
  await expect(status(page)).toContainText('Booting...', { timeout: 60_000 });
  await expect(status(page).getByRole('button', { name: 'Retry' })).toBeHidden();
  // a boot takes a few seconds; this one is held
  await page.waitForTimeout(8_000);
  await expect(status(page)).toContainText('Booting...');

  await page.clock.fastForward(31_000);
  await expect(status(page)).toContainText('Booting is taking much longer than it should.');
  await expect(status(page).getByRole('button', { name: 'Retry' })).toBeVisible();
  const reloaded = page.waitForEvent('framenavigated');
  await status(page).getByRole('button', { name: 'Retry' }).click();
  await reloaded;
});

test('a calculation worker that never answers turns into an error with a retry that reloads', async ({ page }) => {
  // a worker script that runs but never takes a message, so the page's first question to it goes unanswered
  await page.route(/\/workers\/sync_worker-[^/]*\.js$/, (route) => route.fulfill({ contentType: 'text/javascript', body: 'self.onmessage = () => {};' }));
  await page.clock.install();
  await page.goto('/');
  await expect(status(page)).toContainText('Initializing...', { timeout: 60_000 });
  await expect(status(page).getByRole('button', { name: 'Retry' })).toBeHidden();

  await page.clock.fastForward(31_000);
  await expect(status(page)).toContainText('The calculation worker did not start in 30 s.');
  await expect(status(page).getByRole('button', { name: 'Retry' })).toBeVisible();
  const reloaded = page.waitForEvent('framenavigated');
  await status(page).getByRole('button', { name: 'Retry' }).click();
  await reloaded;
});
