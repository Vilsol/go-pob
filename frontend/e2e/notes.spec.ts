import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { record } from './support/perf';
import { expectedLines } from './support/stats';
import { COLOR_MARK, codeToCss, plainToDoc } from '../src/lib/notes/pob_text';

const BUILDS = ['0758-necromancer-vaal-summon-skeletons-of-archers', '0046-elementalist-arc'];

const EDIT = ' e2e edit';
const TOKEN = /\^x[0-9A-Fa-f]{6}|\^[0-9]/g;

const importXml = async (app: App, xml: string) => {
  const build = await go.loadXml(xml);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
};

const editorBox = (page: Page) => page.getByRole('textbox', { name: 'Notes' });
const banner = (page: Page) => page.getByRole('status').filter({ hasText: 'CHANGED OUTSIDE GO-POB' });
const importBanner = (page: Page) => page.getByRole('status').filter({ hasText: 'IMPORTED FROM POB' });

const openNotes = async (app: App, lines: string[]) => {
  const t0 = await app.now();
  await app.goTo('Notes');
  await expect
    .poll(() => editorBox(app.page).evaluate((el) => Array.from(el.children, (c) => c.textContent ?? '').join('\n')), { timeout: 30_000 })
    .toBe(lines.join('\n'));
  record({ kind: 'interaction', build: app.build, phase: 'notes', name: 'notes-open', durationMs: +((await app.now()) - t0).toFixed(1) });
};

const exported = async (app: App) => {
  const xml = await go.decode(await app.exportCode());
  return { xml, build: await go.loadXml(xml) };
};

for (const file of BUILDS) {
  test(`rich notes on ${file}`, async ({ page }) => {
    const source = buildXml(file);
    const notes = (await go.loadXml(source)).Notes;
    expect(notes.match(TOKEN)?.length ?? 0, 'the build carries no colour codes, so the colour checks are vacuous').toBeGreaterThan(0);

    const app = await App.open(page, file);
    await importXml(app, source);
    await openNotes(
      app,
      notes.split('\n').map((l) => l.replace(TOKEN, ''))
    );
    await expect(importBanner(page)).toContainText(`Colour codes converted: ${notes.match(TOKEN)!.length} `);

    const first = plainToDoc(notes)
      .content!.flatMap((p) => p.content ?? [])
      .map((t) => ({ text: t.text!, code: t.marks?.find((m) => m.type === COLOR_MARK)?.attrs?.code as string | undefined }))
      .find((t) => t.code && codeToCss(t.code));
    expect(first, 'no coloured run to check').toBeDefined();
    const hex = codeToCss(first!.code!)!;
    const rgb = `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`;
    await expect(editorBox(page).locator(`span[data-pob-color="${first!.code}"]`).first()).toHaveCSS('color', rgb);

    const t0 = await app.now();
    await page.getByRole('checkbox', { name: 'View as PoB text' }).check();
    await expect.poll(() => page.getByRole('region', { name: 'PoB text' }).evaluate((el) => el.textContent)).toBe(notes);
    record({ kind: 'interaction', build: file, phase: 'notes', name: 'notes-view-pob-text', durationMs: +((await app.now()) - t0).toFixed(1) });
    await page.getByRole('checkbox', { name: 'View as PoB text' }).uncheck();

    const untouched = await exported(app);
    expect(untouched.build.Notes).toBe(notes);
    expect(untouched.build.RichNotesStatus()).toBe('none');

    await openNotes(
      app,
      notes.split('\n').map((l) => l.replace(TOKEN, ''))
    );
    await editorBox(page).click();
    await page.keyboard.press('Control+End');
    const t1 = await app.now();
    await page.keyboard.type(EDIT);
    await expect(editorBox(page)).toContainText(EDIT.trim());
    record({ kind: 'interaction', build: file, phase: 'notes', name: 'notes-type', durationMs: +((await app.now()) - t1).toFixed(1) });
    await app.flushPerf('notes-edit');

    const edited = await exported(app);
    expect(edited.build.RichNotesStatus(), 'the export carries no matching rich document').toBe('match');
    expect(edited.build.Notes).toContain(EDIT);
    expect(edited.build.Notes.replace(EDIT, ''), 'the edit changed more than the typed text').toBe(notes);

    await importXml(app, edited.xml);
    const editedLines = edited.build.Notes.split('\n').map((l) => l.replace(TOKEN, ''));
    await openNotes(app, editedLines);
    await expect(banner(page)).toHaveCount(0);
    await expect(importBanner(page)).toHaveCount(0);

    const foreign = 'edited in PoB';
    const tampered = edited.xml.replace(/<Notes>[\s\S]*?<\/Notes>/, `<Notes>${foreign}</Notes>`);
    expect(tampered).not.toBe(edited.xml);
    await importXml(app, tampered);
    await openNotes(app, [foreign]);
    await expect(banner(page)).toHaveCount(1);

    const t2 = await app.now();
    await page.getByRole('button', { name: 'Restore rich version' }).click();
    await expect(banner(page)).toHaveCount(0);
    await expect.poll(() => editorBox(page).evaluate((el) => Array.from(el.children, (c) => c.textContent ?? '').join('\n'))).toBe(editedLines.join('\n'));
    record({ kind: 'interaction', build: file, phase: 'notes', name: 'notes-restore-rich', durationMs: +((await app.now()) - t2).toFixed(1) });

    const restored = await exported(app);
    expect(restored.build.Notes).toBe(edited.build.Notes);
    expect(restored.build.RichNotesStatus()).toBe('match');
    expect(restored.build.RichNotesDoc()).toBe(edited.build.RichNotesDoc());

    app.assertNoErrors();
  });
}
