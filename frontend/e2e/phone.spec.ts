import { test, expect } from '@playwright/test';

import { App } from './support/app';
import * as go from './support/go_model';
import { BUILD, PHONE, bar, hold, openMore, openPhoneBuild, routeOf } from './support/phone';
import { buildXml } from './support/builds';
import { diffLines, expectedLines } from './support/stats';
import * as tree from './support/tree_model';
import { frameCount, lastFrame, waitFrame, waitLastFrame } from './support/tree_probe';
import { canvasBox, hitTest, placedNodes, screenOf } from './support/tree_view';
import { buildSidebar } from '../src/lib/display/sidebar';
import { editDiff } from '../src/lib/display/stat_delta';
import { headlines } from '../src/lib/design/mobile/mobile';
import type { Page } from '@playwright/test';

// each strip cell's delta as Go's outputs before and after an edit give it, and as the strip shows it
const wantDelta = (before: go.ModelOutputs, after: go.ModelOutputs) =>
  headlines(buildSidebar(after), [], editDiff(before, after)).map((h) => h.delta?.value ?? '');
const shownDelta = (page: Page) =>
  page.locator('[data-stats-strip] [data-strip-cell]').evaluateAll((cs) => cs.map((c) => c.querySelector('[data-delta]')?.textContent?.trim() ?? ''));

test.describe('phone shell at 390×769', () => {
  test.use(PHONE);

  test('every tab, the More sheet and the stats sheet, on the real build state', async ({ page }) => {
    test.setTimeout(10 * 60_000);
    const { app, build } = await openPhoneBuild(page);
    await expect(page.getByRole('navigation', { name: 'Build pages' }), 'the desktop top bar is not mounted on a phone').toHaveCount(0);
    await expect(page.locator('.page-main'), 'nor the desktop page frame').toHaveCount(0);

    const tabs: [string, string, string][] = [
      ['Tree', 'tree', 'Tree'],
      ['Skills', 'skills', 'Skills'],
      ['Items', 'items', 'Items'],
      ['Calcs', 'calcs', 'Calcs'],
      ['Config', 'configuration', 'Config']
    ];
    for (const [link, route, main] of tabs) {
      await bar(page).getByRole('link', { name: link }).click();
      await page.waitForURL((u) => u.pathname.endsWith(`/${route}`));
      await expect(bar(page).getByRole('link', { name: link })).toHaveAttribute('aria-current', 'page');
      await expect(page.getByRole('main', { name: main })).toBeVisible();
    }
    await bar(page).getByRole('link', { name: 'Skills' }).click();
    const view = page.getByRole('tablist', { name: 'Skills view' });
    await expect(view.getByRole('tab', { name: 'Socket groups' })).toHaveAttribute('aria-selected', 'true');
    await page
      .getByRole('region', { name: 'Socket groups' })
      .getByRole('button', { name: /Flame Surge/ })
      .first()
      .click();
    await expect(view.getByRole('tab', { name: 'Selected group' }), 'a tapped group slides to its editor').toHaveAttribute('aria-selected', 'true');
    await view.getByRole('tab', { name: 'Socket groups' }).click();
    await expect(view.getByRole('tab', { name: 'Socket groups' })).toHaveAttribute('aria-selected', 'true');

    const morePages: [string, string, string][] = [
      ['Notes', 'notes', 'Notes'],
      ['Party', 'party', 'Party'],
      ['Compare', 'compare', 'Compare'],
      ['Import/Export', 'import', 'Import / Export']
    ];
    for (const [link, route, main] of morePages) {
      await (await openMore(page)).getByRole('link', { name: link }).click();
      await page.waitForURL((u) => u.pathname.endsWith(`/${route}`));
      await expect(page.getByRole('dialog', { name: 'More' })).toBeHidden();
      await expect(page.getByRole('main', { name: main })).toBeVisible();
      await expect(bar(page).getByRole('button', { name: 'More' }), 'More is the current tab').toHaveClass(/text-accent/);
    }
    await (await openMore(page)).getByRole('link', { name: 'Timeless' }).click();
    const timeless = page.getByRole('dialog', { name: 'Find a Timeless Jewel' });
    await expect(timeless).toBeVisible();
    expect(routeOf(page), 'Timeless is a dialog over the page, not a route').toBe('import');
    await timeless.getByRole('button', { name: 'Close' }).click();
    await expect(timeless).toBeHidden();

    for (const [action, dialog] of [
      ['Undo history', 'Undo history'],
      ['Build issues', 'Build issues'],
      ['Loadout', 'Loadout']
    ]) {
      await (await openMore(page)).getByRole('button', { name: new RegExp(`^${action}`) }).click();
      const sheet = page.getByRole('dialog', { name: dialog });
      await expect(sheet).toBeVisible();
      await sheet.getByRole('button', { name: 'Close' }).click();
      await expect(sheet).toBeHidden();
    }

    const cells = page.locator('[data-stats-strip] [data-strip-cell]');
    await expect(cells).toHaveCount(4);
    const want = expectedLines(await go.outputs(build));
    const life = want.find((l) => l.label === 'Total Life')!;
    await expect(cells.first(), 'the strip shows Go’s Life').toContainText(life.value);
    await page.locator('[data-stats-strip]').click();
    const stats = page.getByRole('dialog', { name: 'Stats' });
    await expect(stats.getByRole('region', { name: 'Build stats' })).toBeVisible();
    expect(diffLines(want, await app.readStats()), 'the stats sheet shows Go’s sidebar').toEqual([]);
    await stats
      .getByRole('button', { name: /^Pin Total Mana/ })
      .first()
      .click();
    await stats.getByRole('button', { name: 'Close' }).click();
    await expect(cells, 'a pin replaces the default cells').toHaveCount(1);
    await expect(cells.first()).toContainText('Mana');
    app.assertNoErrors();
  });

  test('tree: tap shows the tooltip, double tap allocates and Undo reverts, hold opens the node menu', async ({ page }) => {
    test.setTimeout(10 * 60_000);
    const { app, build } = await openPhoneBuild(page);
    await tree.prepareTree();
    const active = go.fresh(build, (b) => [...(b.Build.PassiveNodes ?? [])]);
    await bar(page).getByRole('link', { name: 'Tree' }).click();
    await waitLastFrame(page, `(f) => f.className !== undefined && f.active === ${active.length}`, 60_000);
    const beforeOut = await go.outputs(build);
    const before = expectedLines(beforeOut);

    const t = tree.rawTree();
    const state = tree.treeState(build);
    await page.waitForTimeout(1000);
    const view = (await lastFrame(page))!.view;
    const placed = placedNodes(t);
    const target = placed
      .map((p) => ({ p, s: screenOf(t, view, p.at) }))
      .filter(({ p }) => !active.includes(p.node.skill!) && (state.path(p.node.skill!)?.length ?? 0) > 0 && !p.node.isMastery && !!p.node.name)
      .filter(({ s }) => s.x > 40 && s.y > 40 && s.x < view.width - 60 && s.y < view.height - 40)
      .find(({ p, s }) => hitTest(t, placed, view, s) === p);
    expect(target, 'an unallocated node next to the tree, on screen').toBeDefined();
    const box = await canvasBox(page);
    const at = { x: box.x + target!.s.x, y: box.y + target!.s.y };
    const id = target!.p.node.skill!;

    const from = await frameCount(page);
    await page.touchscreen.tap(at.x, at.y);
    await waitFrame(page, from, `(f) => f.hovered === ${id}`);
    await expect(page.getByRole('tooltip').filter({ hasText: target!.p.node.name! })).toBeVisible();
    await page.waitForTimeout(400);
    expect((await lastFrame(page))!.active, 'a single tap allocates nothing').toBe(active.length);
    await expect(page.locator('[data-stats-strip]')).toContainText(before.find((l) => l.label === 'Total Life')!.value);
    expect(await shownDelta(page), 'no edit yet, so no delta').toEqual(['', '', '', '']);

    let afterOut!: go.ModelOutputs;
    await app.act(
      'phone-double-tap',
      async () => {
        await page.touchscreen.tap(at.x, at.y);
        await page.touchscreen.tap(at.x, at.y);
      },
      async () => {
        state.allocate(id);
        afterOut = await go.outputs(build);
        return expectedLines(afterOut);
      }
    );
    const allocated = wantDelta(beforeOut, afterOut);
    expect(allocated.some(Boolean), 'the allocation moves a strip stat, so the delta check is not vacuous').toBe(true);
    await expect.poll(() => shownDelta(page), { message: 'the strip shows Go’s change from the allocation' }).toEqual(allocated);
    const toast = page.getByRole('status').filter({ hasText: `Allocated ${target!.p.node.name}` });
    await expect(toast).toBeVisible();
    await app.act(
      'phone-undo-toast',
      () => toast.getByRole('button', { name: 'Undo' }).click(),
      () => Promise.resolve(before)
    );
    await waitLastFrame(page, `(f) => f.active === ${active.length}`);
    await expect.poll(() => shownDelta(page), { message: 'Undo replaces the delta with Go’s change back' }).toEqual(wantDelta(afterOut, beforeOut));
    await expect(page.getByRole('banner').getByRole('button', { name: 'Redo' }), 'the frame’s Redo follows the worker’s undo history').toBeEnabled();
    await (await openMore(page)).getByRole('button', { name: /^Undo history/ }).click();
    await expect(page.getByRole('dialog', { name: 'Undo history' }).getByRole('list', { name: 'Undo history' }).getByRole('listitem')).not.toHaveCount(0);
    await page.getByRole('dialog', { name: 'Undo history' }).getByRole('button', { name: 'Close' }).click();

    await hold(page, at.x, at.y);
    await expect(page.getByRole('menu')).toBeVisible();
    app.assertNoErrors();
  });

  test('the tree fitted on a phone leaves the build’s saved view alone', async ({ page }) => {
    test.setTimeout(10 * 60_000);
    const { app } = await openPhoneBuild(page);
    await bar(page).getByRole('link', { name: 'Tree' }).click();
    await waitLastFrame(page, '(f) => f.className !== undefined', 60_000);
    await page.waitForTimeout(1500);
    await (await openMore(page)).getByRole('link', { name: 'Import/Export' }).click();
    const box = page.getByRole('region', { name: 'Build sharing' }).getByRole('textbox', { name: 'Build code' });
    let code = '';
    await expect.poll(async () => (code = await box.inputValue())).not.toBe('');
    const zoom = (xml: string) =>
      /<TreeView[^>]*>/
        .exec(xml)?.[0]
        .match(/zoom(Level|X|Y)="[^"]*"/g)
        ?.sort();
    expect(zoom(await go.decode(code))).toEqual(zoom(buildXml(BUILD)));
    app.assertNoErrors();
  });

  test('the wide-list preference switches the Items list between cards and a table', async ({ page }) => {
    test.setTimeout(10 * 60_000);
    const { app } = await openPhoneBuild(page);
    await bar(page).getByRole('link', { name: 'Items' }).click();
    await page.getByRole('group', { name: 'Layout' }).getByRole('button', { name: 'List' }).click();
    const list = page.getByRole('table', { name: 'Equipped items' });
    await expect(list).toHaveAttribute('data-phone', 'cards');
    await (await openMore(page)).getByRole('button', { name: 'Settings' }).click();
    const sheet = page.getByRole('dialog', { name: 'Settings' });
    await sheet.getByRole('combobox', { name: 'Wide lists on a phone' }).selectOption('Table');
    await sheet.getByRole('button', { name: 'Save' }).click();
    await expect(sheet).toBeHidden();
    await expect(list).toHaveAttribute('data-phone', 'table');
    app.assertNoErrors();
  });
});

test.describe('desktop shell at 1600×1000 (control)', () => {
  test('has the top bar and no phone tab bar', async ({ page }) => {
    test.setTimeout(5 * 60_000);
    const app = await App.open(page, 'new build');
    await expect(page.getByRole('navigation', { name: 'Build pages' })).toBeVisible();
    await expect(page.locator('[data-mobile-frame]')).toHaveCount(0);
    await expect(bar(page)).toHaveCount(0);
    app.assertNoErrors();
  });
});
