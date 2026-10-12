import { test, expect } from '@playwright/test';

import { App } from './support/app';
import { BUILD_SET, buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';

for (const { file, why } of BUILD_SET) {
  test(`${file} — ${why}`, async ({ page }) => {
    const xml = buildXml(file);
    const build = await go.loadXml(xml);
    // Build.lua:1014-1061
    const canonical = await go.serializeSaved(build);
    const want = async () => expectedLines(await go.outputs(build));

    const app = await App.open(page, file);
    await app.importCode(await go.encode(xml), want);
    await app.memory('after-import');

    const code = await app.exportCode();
    expect(await go.decode(code), 'export after import differs in Go').toBe(canonical);
    await app.importCode(code, want);
    expect(await go.decode(await app.exportCode()), 'export after re-import differs in Go').toBe(canonical);

    const name = `e2e-${file}`;
    await page.getByRole('button', { name: /^Save \(/ }).click();
    const dialog = page.getByRole('dialog');
    await dialog.getByRole('textbox', { name: 'Build name' }).fill(name);
    await dialog.getByRole('button', { name: 'Save', exact: true }).click();
    await expect(dialog).toBeHidden();

    await page.reload();
    await app.savedBuild(name).waitFor({ timeout: 180_000 });
    await app.act('open-saved', () => app.openSaved(name), want);
    expect(await go.decode(await app.exportCode()), 'saved-then-reloaded build differs in Go').toBe(canonical);

    app.assertNoErrors();
  });
}
