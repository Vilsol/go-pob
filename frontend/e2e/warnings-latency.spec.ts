// Build.lua:1778-1831

import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { record } from './support/perf';
import { diffLines, expectedLines } from './support/stats';
import { buildSidebar } from '../src/lib/display/sidebar';

const SWITCHES = 6;

const BUILDS = [
  { file: '0201-elementalist-vaal-cold-snap-2', why: 'light calculation, cost lists over 17 skills' },
  { file: '0011-saboteur-cyclone', why: 'a mana cost warning on the main skill' },
  { file: '0001-necromancer-unknown', why: 'largest build, 24 active skills' }
];

const warningsSettled = (page: Page) =>
  page.waitForFunction(() => {
    const d = document.documentElement.dataset;
    return d.warningsSeq !== undefined && d.warningsSeq === d.calcSeq && d.calcPending === '0';
  });

const recordWarnings = async (page: Page, build: string, action: string) => {
  await warningsSettled(page);
  const t = await page.evaluate(() => {
    const d = document.documentElement.dataset;
    const seq = Number(d.warningsSeq);
    const rendered = (globalThis.__pobCalcTimings ?? []).find((x) => x.seq === seq);
    return { at: Number(d.warningsAt), ms: Number(d.warningsMs), renderedAt: rendered?.renderedAt ?? NaN };
  });
  expect(Number.isFinite(t.at - t.renderedAt), `${action}: no rendered Tick for the warnings' seq`).toBe(true);
  record({ kind: 'action', action, build, settleMs: t.at - t.renderedAt, ticks: 0, calcMs: t.ms, dumpMs: 0, messageMs: 0, renderMs: 0, dispatchMs: 0 });
};

const shownWarnings = async (page: Page): Promise<string[]> => {
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
  await page.keyboard.press('Escape');
  return items;
};

const modelWarnings = async (build: Parameters<typeof go.outputs>[0]) =>
  buildSidebar(await go.outputs(build), { seq: 0, ms: 0, ...(await go.warnings(build)) }).warnings.map((x) => x.text);

test('a main-skill switch right after the stats on 0001 waits behind no warnings step', async ({ page }) => {
  test.setTimeout(240_000);
  const file = '0001-necromancer-unknown';
  const build = await go.loadXml(buildXml(file));
  const app = await App.open(page, file);
  const want = async () => expectedLines(await go.outputs(build));
  await app.importCode(await go.encode(buildXml(file)), want);

  const select = page.getByRole('combobox', { name: 'Main skill' });
  const pick = () =>
    select.evaluate((el) => {
      const s = el as HTMLSelectElement;
      return { original: s.value, other: [...s.options].map((o) => o.value).find((v) => v !== s.value) };
    });
  await expect.poll(async () => (await pick()).other, { timeout: 30_000 }).toBeDefined();
  const { original, other: picked } = await pick();
  const other = picked!;

  const handle = await select.elementHandle();
  const attempt = async () => {
    const first = await app.seq();
    const armed = page.evaluate(
      ([el, b, to]) =>
        new Promise<{ at: number; seq: number; inFlight: boolean }>((resolve) => {
          const root = document.documentElement;
          const obs = new MutationObserver(() => {
            if (Number(root.dataset.calcSeq ?? 0) <= b || root.dataset.calcPending !== '0') return;
            obs.disconnect();
            const r = {
              at: performance.timeOrigin + performance.now(),
              seq: Number(root.dataset.calcSeq),
              inFlight: root.dataset.warningsSeq !== root.dataset.calcSeq
            };
            const s = el as HTMLSelectElement;
            s.value = to;
            s.dispatchEvent(new Event('change', { bubbles: true }));
            resolve(r);
          });
          obs.observe(root, { attributes: true, attributeFilter: ['data-calc-seq', 'data-calc-pending'] });
        }),
      [handle, first, original] as const
    );
    await select.selectOption(other);
    const r = await armed;
    await app.waitSettled(r.seq);
    return r;
  };
  let back = await attempt();
  for (let i = 1; i < 5 && !back.inFlight; i++) back = await attempt();
  expect(back.inFlight, "in 5 attempts the first switch's warnings had always landed already, so nothing was in flight to wait behind").toBe(true);
  build.SetMainSocketGroup(Number(original) + 1);
  expect(diffLines(await want(), await app.readStats()), 'displayed stats differ from Go after switching back').toEqual([]);
  const after = await app.timingsSince(back.seq);
  expect(after.length, 'the switch back rendered no calculation timing').toBeGreaterThan(0);
  const dispatchMs = after[0].startedAt - back.at;
  await recordWarnings(page, file, 'warnings-after-switch-during-run');
  const runMs = await page.evaluate(() => Number(document.documentElement.dataset.warningsMs));
  record({
    kind: 'action',
    action: 'switch-dispatch-during-warnings',
    build: file,
    settleMs: dispatchMs,
    ticks: 1,
    calcMs: 0,
    dumpMs: 0,
    messageMs: 0,
    renderMs: 0,
    dispatchMs
  });
  const steps = await page
    .workers()[0]
    .evaluate(() =>
      performance
        .getEntriesByName('pob:warnings-step')
        .map((e) => ({ start: performance.timeOrigin + e.startTime, end: performance.timeOrigin + e.startTime + e.duration }))
    );
  const blocking = steps.filter((s) => s.start < after[0].startedAt && s.end > back.at);
  expect(
    blocking.map((s) => +(s.end - s.start).toFixed(1)),
    `the switch (its calculation started ${dispatchMs.toFixed(0)} ms after it; the run takes ${runMs.toFixed(0)} ms) waited behind warnings steps`
  ).toEqual([]);
  app.assertNoErrors();
});

for (const { file, why } of BUILDS) {
  test(`warnings latency on ${file}: ${why}`, async ({ page }) => {
    test.setTimeout(240_000);
    const build = await go.loadXml(buildXml(file));
    const app = await App.open(page, file);
    const want = async () => expectedLines(await go.outputs(build));
    await app.importCode(await go.encode(buildXml(file)), want);
    await recordWarnings(page, file, 'warnings-after-import');

    const select = page.getByRole('combobox', { name: 'Main skill' });
    await expect(select.getByRole('option').first()).not.toHaveText('<No skills added yet>');
    const original = await select.inputValue();
    const other = (await select.getByRole('option').evaluateAll((os) => os.map((o) => (o as HTMLOptionElement).value))).find((v) => v !== original);
    expect(other, 'the build needs a second socket group').toBeDefined();

    for (let i = 0; i < SWITCHES; i++) {
      const to = i % 2 === 0 ? other! : original;
      await app.act(
        'warnings-latency-switch',
        () => select.selectOption(to),
        async () => {
          build.SetMainSocketGroup(Number(to) + 1);
          return want();
        }
      );
      await recordWarnings(page, file, 'warnings-after-switch');
    }
    expect(await shownWarnings(page), `${file}: the warnings line differs from the model's`).toEqual(await modelWarnings(build));
    app.assertNoErrors();
  });
}
