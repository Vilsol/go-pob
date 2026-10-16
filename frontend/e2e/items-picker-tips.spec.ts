// ItemsTab.lua:818-930

import { test, expect, type Locator, type Page } from '@playwright/test';
import { readFileSync } from 'node:fs';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import { record } from './support/perf';
import { expectedBlock, type DiffBlock, type WhatIfOutputs } from './support/stat_diff';
import { plainText } from '../src/lib/design/crafting';

const HEAVY = '1591-necromancer-summon-holy-relic-of-conviction-15';
const HEADER = '\nAdding this mod will give: ';
type Misc = NonNullable<Awaited<ReturnType<Awaited<ReturnType<typeof go.model>>['exposition']['GetMiscCalculator']>>>;

const setup = async (page: Page, file: string) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await expect(page.getByRole('region', { name: 'Equipped items' })).toBeVisible();
  return { app, build };
};

const listOf = async (box: Locator) => {
  if ((await box.getAttribute('aria-expanded')) !== 'true') await box.click();
  const list = box.page().locator(`[id="${await box.getAttribute('aria-controls')}"]`);
  await expect(list).toBeVisible();
  return list;
};

const watch = (page: Page, name: string) =>
  page.evaluate((label) => {
    const w = globalThis as unknown as { __tipShownAt?: number };
    w.__tipShownAt = undefined;
    const seen = () => document.querySelector(`[role="tooltip"][aria-label="${label}"]`);
    const mo = new MutationObserver(() => {
      if (w.__tipShownAt === undefined && seen()) {
        w.__tipShownAt = performance.timeOrigin + performance.now();
        mo.disconnect();
      }
    });
    mo.observe(document.body, { childList: true, subtree: true });
  }, name);
const shownAt = (page: Page) => page.evaluate(() => (globalThis as unknown as { __tipShownAt?: number }).__tipShownAt ?? NaN);

const shownBlocks = (tip: Locator) =>
  tip.evaluate((el) => {
    const text = (e: Element | undefined) => (e?.textContent ?? '').replace(/\s+/g, ' ').trim();
    return [...el.querySelectorAll<HTMLElement>('[data-header]')].map((b) => ({
      header: b.dataset.header!,
      readyAt: performance.timeOrigin + Number(b.dataset.readyAt),
      sections: [...b.querySelectorAll<HTMLElement>('[data-section]')].map((s) => ({
        actor: s.dataset.actor || undefined,
        lines: [...s.querySelectorAll('[data-line]')].map((l) => {
          const m = /^(\S+)(?: (.*))?$/.exec(text(l.children[1]).replace(/^[▲▼] /, ''))!;
          return `${m[1]} ${text(l.children[0])}${m[2] ? ` ${m[2]}` : ''}`;
        })
      }))
    }));
  });

const expectedCompare = (misc: Misc, text: string, lines: string[]): DiffBlock[] => {
  const [r, ok] = misc.CompareAddedMod(text, lines, '');
  expect(ok).toBe(true);
  const b = expectedBlock(HEADER, { ...(r.Base as WhatIfOutputs), SkillFlags: r.Added.SkillFlags } as WhatIfOutputs, r.Added as WhatIfOutputs);
  return b ? [b] : [{ header: HEADER, sections: [] }];
};

test(`an affix entry's tiers and what adding it gives, on hover; latency on a heavy build (${HEAVY})`, async ({ page }) => {
  test.setTimeout(600_000);
  const { app, build } = await setup(page, HEAVY);
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const box = page.getByRole('textbox', { name: /Item text/ });
  const [crafted] = await m.exposition.CraftItemText(2, '', 'Hubris Circlet');
  await box.fill(crafted);
  const [controls] = await m.exposition.ItemAffixControls(crafted);
  const series = controls.Controls![0].Series!;
  const affix = page.getByRole('group', { name: 'Affixes' }).getByRole('combobox', { name: 'Affix 1' });
  const list = await listOf(affix);
  const tip = page.getByRole('tooltip', { name: 'Affix tooltip' });

  // ItemsTab.lua:820
  await list.getByRole('option').nth(0).hover();
  await expect(tip).toHaveCount(0);

  const picks = [1, 2, 3, 5, 8, 13].filter((k) => k < series.length);
  expect(picks.length).toBeGreaterThan(4);
  for (const k of picks) {
    const g0 = Date.now();
    const want = expectedCompare(misc, crafted, [...(series[k].Mid ?? [])]);
    record({ kind: 'tree-latency', build: HEAVY, what: 'affix-compare-go-model', ms: Date.now() - g0 });
    await list.getByRole('option').nth(0).hover();
    await watch(page, 'Affix tooltip');
    const t0 = await app.now();
    await list.getByRole('option').nth(k).hover();
    await expect(tip).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
    for (const l of series[k].Tooltip ?? []) if (l !== '--' && plainText(l).trim()) await expect(tip).toContainText(plainText(l).trim());
    const shown = await shownBlocks(tip);
    expect(shown.map(({ header, sections }) => ({ header, sections }))).toEqual(want);
    record({ kind: 'tree-latency', build: HEAVY, what: 'affix-hover-tooltip', ms: +((await shownAt(page)) - t0).toFixed(1) });
    record({ kind: 'tree-latency', build: HEAVY, what: 'affix-hover-compare', ms: +(shown[0].readyAt - t0).toFixed(1) });
  }

  await list.getByRole('option').nth(0).hover();
  await watch(page, 'Affix tooltip');
  const t0 = await app.now();
  await list.getByRole('option').nth(picks[0]).hover();
  await expect(tip).toHaveAttribute('aria-busy', 'false');
  record({ kind: 'tree-latency', build: HEAVY, what: 'affix-hover-again', ms: +((await shownAt(page)) - t0).toFixed(1) });

  const [a, b] = [(await list.boundingBox())!, (await tip.boundingBox())!];
  expect(a.x + a.width <= b.x || b.x + b.width <= a.x, 'the tooltip covers the list').toBe(true);

  await page.keyboard.press('Escape');
  await expect(tip).toHaveCount(0);
  misc.release();
  await app.flushPerf('picker-tips');
  app.assertNoErrors();
});

test("a base entry's item on that base, on hover (0280)", async ({ page }) => {
  test.setTimeout(300_000);
  const { app } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const box = page.getByRole('textbox', { name: /Item text/ });
  const [crafted] = await m.exposition.CraftItemText(2, '', 'Hubris Circlet');
  await box.fill(crafted);
  const other = ((await m.exposition.RebaseChoices(crafted)) ?? []).find((b) => b !== 'Hubris Circlet')!;
  const list = await listOf(page.getByRole('group', { name: 'Affixes' }).getByRole('combobox', { name: 'Item base' }));
  await list.getByRole('option', { name: other, exact: true }).hover();
  const [rebased, ok] = await m.exposition.RebaseItemText(crafted, other);
  expect(ok).toBe(true);
  const tip = page.getByRole('group', { name: 'Base tooltip' });
  await expect(tip).toContainText(other);
  await expect(tip.locator('.fixed')).toBeVisible();
  await expect(tip).toContainText('Equipping this item in Helmet will give you');
  await expect(box).toHaveValue(crafted);
  expect(rebased.Text).not.toEqual(crafted);
  app.assertNoErrors();
});

test("Add Modifier's hovered modifier: its lines and what adding it gives (0280)", async ({ page }) => {
  test.setTimeout(300_000);
  const { app, build } = await setup(page, '0280-elementalist-fireball');
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  const box = page.getByRole('textbox', { name: /Item text/ });
  const helmet = readFileSync(new URL('./fixtures/items/helmet.txt', import.meta.url), 'utf8');
  const [choices] = await m.exposition.ItemCustomModChoices(helmet);
  await box.fill(helmet);
  await page.getByRole('region', { name: 'Item controls' }).getByRole('button', { name: 'Add modifier...' }).click();
  const dialog = page.getByRole('dialog', { name: 'Add Modifier to Item' });
  const list = await listOf(dialog.getByRole('combobox', { name: 'Modifier' }));
  const mod = choices.Mods!.MASTER![2];
  await list.getByRole('option').nth(2).hover();
  const tip = page.getByRole('tooltip', { name: 'Modifier tooltip' });
  await expect(tip).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
  for (const l of mod.Lines ?? []) await expect(tip).toContainText(plainText(l));
  const lines = (mod.Lines ?? []).map((l) => (mod.ModTags?.length ? `{tags:${mod.ModTags.join(',')}}` : '') + l);
  const shown = await shownBlocks(tip);
  expect(shown.map(({ header, sections }) => ({ header, sections }))).toEqual(expectedCompare(misc, helmet, lines));
  misc.release();
  app.assertNoErrors();
});

test("Craft item…'s hovered base stands beside the list, not under it (0280)", async ({ page }) => {
  test.setTimeout(300_000);
  const { app } = await setup(page, '0280-elementalist-fireball');
  await page.getByRole('button', { name: 'Craft item…' }).first().click();
  const craft = page.getByRole('dialog', { name: 'Craft Item' });
  const list = await listOf(craft.getByRole('combobox', { name: 'Base' }));
  const name = (await list.getByRole('option').nth(1).textContent())!.trim();
  await list.getByRole('option').nth(1).hover();
  const preview = craft.getByRole('group', { name: 'Base preview' }).locator('.fixed');
  await expect(preview).toContainText(name);
  const [a, b] = [(await list.boundingBox())!, (await preview.boundingBox())!];
  expect(b.width * b.height).toBeGreaterThan(0);
  expect(a.x + a.width <= b.x || b.x + b.width <= a.x, 'the preview covers the list').toBe(true);
  app.assertNoErrors();
});
