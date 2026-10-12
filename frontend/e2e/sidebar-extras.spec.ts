// Build.lua:1849-1867

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { record } from './support/perf';
import { expectedLines, modelLines, type StatLine } from './support/stats';
import { buildSidebar } from '../src/lib/display/sidebar';

type Row = { height: number; center: boolean; cells: string[] };
type PoBCase = { file: string; statBox: Row[]; warnings: string[] };

const ORACLE = resolve(import.meta.dirname, '../../testdata/oracle/sidebar_stat_box_expected.jsonl');
const pobCases = new Map(
  readFileSync(ORACLE, 'utf8')
    .trim()
    .split('\n')
    .map((l) => JSON.parse(l) as PoBCase)
    .map((c) => [c.file.replace(/\.xml$/, ''), c])
);

const plain = (s: string) => s.replace(/\^(x[0-9A-Fa-f]{6}|\d)/g, '');
const CUSTOM = '^x5CF0BB';

const pobLines = (c: PoBCase): StatLine[] => {
  const lines: StatLine[] = [];
  let afterSkill = false;
  for (const r of c.statBox) {
    if (r.cells.length === 0) continue;
    if (r.center) {
      if (afterSkill) lines.push({ label: r.cells[0].startsWith('^8') ? 'Skill part' : 'Source', value: plain(r.cells[0]) });
      continue;
    }
    if (r.cells.length === 1) continue;
    afterSkill = r.cells[0].startsWith(CUSTOM);
    lines.push({ label: plain(r.cells[0]).replace(/:$/, ''), value: plain(r.cells[1]) });
  }
  return lines;
};

const NUMBER = /[+-]?\d[\d,]*(?:\.\d+)?/g;

const sameValue = (ours: string, pob: string) => {
  if (ours === pob) return true;
  if (ours.replace(NUMBER, '#') !== pob.replace(NUMBER, '#')) return false;
  const a = ours.match(NUMBER) ?? [];
  const b = pob.match(NUMBER) ?? [];
  return a.every((x, i) => {
    const [u, v] = [Number(x.replace(/,/g, '')), Number(b[i].replace(/,/g, ''))];
    const step = 10 ** -(b[i].split('.')[1]?.length ?? 0);
    return Math.abs(u - v) <= Math.max(0.005 * Math.abs(v), step);
  });
};

const diffAgainstPoB = (ours: StatLine[], pob: StatLine[]) => {
  const diffs: string[] = [];
  for (let i = 0; i < Math.max(ours.length, pob.length); i++) {
    const [o, p] = [ours[i], pob[i]];
    if (!o || !p || o.label !== p.label || !sameValue(o.value, p.value)) {
      diffs.push(`#${i}: ours ${o ? `${o.label}=${o.value}` : '<none>'}, PoB ${p ? `${p.label}=${p.value}` : '<none>'}`);
    }
  }
  return diffs;
};

const waitForWarnings = async (page: Page) => {
  await page.waitForFunction(() => {
    const d = document.documentElement.dataset;
    return d.warningsSeq !== undefined && d.warningsSeq === d.calcSeq && d.calcPending === '0';
  });
};

const readWarnings = async (page: Page): Promise<string[]> => {
  const button = page.getByRole('button', { name: /^\d+ build issues$/ });
  const count = Number((await button.getAttribute('aria-label'))?.split(' ')[0]);
  if (count === 0) return [];
  await button.click();
  const rows = await page.getByRole('list', { name: 'Build issues' }).getByRole('listitem').allInnerTexts();
  const items = rows.map((r) =>
    r
      .split('\n')
      .filter((l) => l !== '⚠')
      .slice(0, -1)
      .join('\n')
  );
  expect(count).toBe(items.length);
  await page.keyboard.press('Escape');
  return items;
};

const modelWarnings = async (build: Parameters<typeof go.outputs>[0]) => {
  const w = await go.warnings(build);
  return buildSidebar(await go.outputs(build), { seq: 0, ms: 0, ...w }).warnings.map((x) => x.text);
};

const open = async (page: Page, file: string) => {
  const build = await go.loadXml(buildXml(file));
  const app = await App.open(page, file);
  const want = async () => expectedLines(await go.outputs(build));
  await app.importCode(await go.encode(buildXml(file)), want);
  return { app, build, want };
};

const recordWarnings = async (app: App, page: Page, action: string) => {
  await waitForWarnings(page);
  const t = await page.evaluate(() => {
    const d = document.documentElement.dataset;
    const seq = Number(d.warningsSeq);
    const rendered = (globalThis as unknown as { __pobCalcTimings?: { seq: number; renderedAt: number }[] }).__pobCalcTimings?.find((x) => x.seq === seq);
    return { at: Number(d.warningsAt), ms: Number(d.warningsMs), renderedAt: rendered?.renderedAt ?? NaN };
  });
  record({
    kind: 'action',
    action,
    build: app.build,
    settleMs: t.at - t.renderedAt,
    ticks: 0,
    calcMs: t.ms,
    dumpMs: 0,
    messageMs: 0,
    renderMs: 0,
    dispatchMs: 0
  });
};

const BUILDS = [
  { file: '0008-necromancer-vaal-summon-skeletons', why: 'minion main skill, minion Skill DPS lines' },
  { file: '0011-saboteur-cyclone', why: 'Skill DPS with a count, an ailment source line, a mana cost warning' },
  { file: '0201-elementalist-vaal-cold-snap-2', why: 'cost lists, unreserved percent, negative unreserved mana, missing anoint' },
  { file: '5515-ladder-Morkork_2988.Keepo_Lumiere', why: 'attribute requirement warnings naming an item and a gem' }
];

for (const { file, why } of BUILDS) {
  test(`sidebar extras on ${file}: ${why}`, async ({ page }) => {
    const { app, build } = await open(page, file);

    const pob = pobCases.get(file);
    expect(pob, `${file} has no line in ${ORACLE}`).toBeDefined();
    const model = modelLines(await go.outputs(build));
    expect(diffAgainstPoB(model, pobLines(pob!)), `${file}: the model's stat box differs from PoB's`).toEqual([]);

    await recordWarnings(app, page, 'warnings-after-import');
    const shown = await readWarnings(page);
    expect(shown, `${file}: the warnings line differs from the model's`).toEqual(await modelWarnings(build));
    expect([...shown].sort(), `${file}: the warnings line differs from PoB's`).toEqual([...pob!.warnings].sort());
    app.assertNoErrors();
  });
}

test('minion section follows the main skill: switch away from the minion skill and back', async ({ page }) => {
  const file = '0008-necromancer-vaal-summon-skeletons';
  const { app, build, want } = await open(page, file);
  const select = page.getByRole('combobox', { name: 'Main skill' });
  const groups = go.fresh(build, (b) => b.Skills.SkillSets[b.ActiveSkillSetIndex()].Skills.length);
  await expect(select.getByRole('option')).toHaveCount(groups);
  const original = await select.inputValue();
  const minionHeading = page.getByRole('region', { name: 'Build stats' }).getByRole('button', { name: 'Minion', exact: true });
  await expect(minionHeading).toBeVisible();

  let other = -1;
  for (let i = 0; i < groups && other < 0; i++) {
    if (String(i) === original) continue;
    build.SetMainSocketGroup(i + 1);
    if (!(await go.outputs(build)).Minion) other = i;
  }
  build.SetMainSocketGroup(Number(original) + 1);
  expect(other, 'the build needs a group without a minion').toBeGreaterThanOrEqual(0);

  await app.act(
    'set-main-skill-no-minion',
    () => select.selectOption(String(other)),
    async () => {
      build.SetMainSocketGroup(other + 1);
      return want();
    }
  );
  await expect(minionHeading).toHaveCount(0);
  await recordWarnings(app, page, 'warnings-after-main-skill');

  await app.act(
    'set-main-skill-minion',
    () => select.selectOption(original),
    async () => {
      build.SetMainSocketGroup(Number(original) + 1);
      return want();
    }
  );
  await expect(minionHeading).toBeVisible();
  expect(await go.outputs(build).then((o) => o.Minion !== undefined)).toBe(true);
  app.assertNoErrors();
});
