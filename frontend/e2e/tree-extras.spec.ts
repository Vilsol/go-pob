// TreeTab.lua:1161-2863

import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { expectedLines } from './support/stats';
import * as tree from './support/tree_model';
import { installTreeProbe, lastFrame, waitFrame } from './support/tree_probe';
import { canvasBox, emptySpot, placedNodes } from './support/tree_view';
import { ascendancyOffsets, nodeCentre } from '../src/lib/skill_tree/geometry';

const FILE = '0005-occultist-vaal-double-strike';

test("the timeless search lists the Go model's rows, opens the trade search and adds the picked jewel", async ({ page, context }) => {
  test.setTimeout(240_000);
  const xml = buildXml(FILE);
  const build = await go.loadXml(xml);
  const app = await App.open(page, FILE);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Tree');
  await page.getByRole('button', { name: /^Timeless/ }).click();
  const dlg = page.getByTestId('timeless-dialog');
  await dlg
    .getByRole('group', { name: 'Jewels' })
    .getByRole('button', { name: /^Lethal Pride/ })
    .click();
  await dlg
    .getByRole('region', { name: 'Conqueror' })
    .getByRole('button', { name: /^Rakiata/ })
    .click();
  await dlg.getByRole('button', { name: 'Next' }).click();
  await expect(dlg.getByRole('region', { name: 'Passive tree around the build' })).toBeVisible();
  await dlg
    .getByRole('region', { name: 'Sockets' })
    .getByRole('button', { name: /: 61834(?!\d)/ })
    .click();
  const live = dlg.locator('[data-live-tree]');
  await expect(live.locator('.radius-ring[data-ring="61834"]')).toBeVisible();
  const box = (await live.locator('canvas').first().boundingBox())!;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await expect(live.getByRole('tooltip')).toContainText('61834');
  await dlg.getByRole('button', { name: 'Next' }).click();
  // TreeTab.lua:1893-1914
  await dlg.getByRole('combobox', { name: 'Add a required stat' }).fill('total strength');
  await page.keyboard.press('Enter');
  await dlg.getByText("Edit as text (PoB's desired nodes list)").click();
  await expect(dlg.getByLabel('Desired nodes', { exact: true })).toHaveValue('total_strength, 1, 0, 0');
  await dlg.getByLabel('Desired nodes', { exact: true }).fill('total_strength, 1, 0, 0\nkarui_notable_add_life, 2');
  await dlg.getByLabel('Desired nodes', { exact: true }).blur();
  await dlg.getByRole('button', { name: 'Next' }).click();
  await dlg.getByRole('button', { name: 'Search', exact: true }).click();

  const want = await go.timelessSearch(build, {
    JewelType: 2,
    ConquerorType: 3,
    SocketID: 61834,
    SocketFilter: false,
    SocketFilterDistance: 0,
    ProtectedNodes: [],
    SearchList: 'total_strength, 1, 0, 0\nkarui_notable_add_life, 2',
    SearchListFallback: '',
    TotalMinimumWeight: 0,
    HasTotalMinimumWeight: false
  });
  expect(want.count, 'the model found no rows').toBeGreaterThan(100);
  await expect(dlg.getByText(`${want.count} results`)).toBeVisible({ timeout: 120_000 });
  const options = dlg.getByRole('listbox', { name: 'Search results' }).getByRole('option');
  for (let i = 0; i < 3; i++) await expect(options.nth(i)).toHaveAttribute('aria-label', want.rows[i].Label.trim());

  const popup = context.waitForEvent('page');
  await dlg.getByRole('button', { name: 'Open Trade URL' }).click();
  const url = new URL((await popup).url());
  expect(url.pathname).toBe('/trade/search/Standard/');
  const q = JSON.parse(url.searchParams.get('q') ?? '{}') as { query: { stats: { filters: { id: string; value: { min: number } }[] }[] } };
  const seeds = q.query.stats[0].filters.map((f) => f.value.min);
  expect(seeds).toHaveLength(11);
  expect(seeds.slice(0, 5)).toEqual(want.rows.map((r) => r.Seed));
  expect(new Set(q.query.stats[0].filters.map((f) => f.id))).toEqual(new Set(['explicit.pseudo_timeless_jewel_rakiata']));
  await expect(dlg.getByRole('button', { name: 'Open Next Trade URL' })).toBeVisible();

  // TimelessJewelListControl.lua:98-311
  await options.first().dblclick();
  await expect(options.first()).toHaveClass(/text-code-grey/);
  await page.keyboard.press('Escape');
  const saved = await go.decode(await app.exportCode());
  expect(saved).toContain(`Lethal Pride [${want.rows[0].Seed}; ${want.rows[0].Total}; Ghost Dance]`);
  expect(saved).toContain(`Rakiata`);
  app.assertNoErrors();
});

type DrawSprite = { k: string; q: number[]; t?: number[] };
const drawnTints = async (page: Page, at: { x: number; y: number }) => {
  const list = await page.evaluate(() => (globalThis as unknown as { __treeProbe: { drawList?: () => { items: unknown[] } } }).__treeProbe.drawList?.());
  return ((list?.items ?? []) as DrawSprite[])
    .filter((i) => i.k === 'sprite' && Math.abs(i.q[0] + i.q[2] / 2 - at.x) < 1 && Math.abs(i.q[1] + i.q[5] / 2 - at.y) < 1)
    .map((i) => (i.t ?? [1, 1, 1]).map((v) => Math.round(v)).join(','));
};

const openTree = async (page: Page, file = FILE) => {
  await installTreeProbe(page);
  await page.addInitScript(() => {
    (globalThis as unknown as { __treeProbe: { renderer?: string } }).__treeProbe.renderer = 'webgl';
  });
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await tree.prepareTree();
  const active = go.fresh(build, (b) => [...(b.Build.PassiveNodes ?? [])]);
  await app.goTo('Tree');
  await waitFrame(page, 0, `(f) => f.className !== undefined && f.active === ${active.length}`, 60_000);
  return { app, build, active };
};

// TreeTab.lua:106-128, PassiveTreeView.lua:166-185
test('compare: the compared tree colours the nodes the two trees do not share', async ({ page }) => {
  test.setTimeout(240_000);
  const { app, build, active } = await openTree(page);
  const specs = go.fresh(build, (b) => (b.Tree.Specs ?? []).map((s) => s.NodesAttr.split(',').filter(Boolean).map(Number)));
  const activeSpec = go.fresh(build, (b) => b.Tree.ActiveSpec) - 1;
  const other = activeSpec === 0 ? 1 : 0;
  await page.getByRole('checkbox', { name: 'Compare:' }).check();
  await page.getByRole('combobox', { name: 'Compared tree' }).selectOption(String(other));
  const t = tree.rawTree();
  const offsets = ascendancyOffsets(t);
  const start = (id: number) => t.nodes[id]?.classStartIndex !== undefined || !!t.nodes[id]?.isAscendancyStart;
  const mine = new Set(active);
  const theirs = new Set(specs[other]);
  const green = specs[other].filter((id) => !mine.has(id) && !start(id) && t.nodes[id] && !t.nodes[id].isMastery);
  const red = active.filter((id) => !theirs.has(id) && !start(id) && t.nodes[id] && !t.nodes[id].isMastery);
  expect(green.length + red.length, 'the two trees are the same').toBeGreaterThan(0);
  for (const [ids, want] of [
    [green, '0,1,0'],
    [red, '1,0,0']
  ] as const) {
    for (const id of ids.slice(0, 3)) {
      const at = nodeCentre(t, offsets, t.nodes[id])!;
      await expect.poll(async () => drawnTints(page, at), `node ${id}`).toContain(want);
    }
  }
  const shared = active.find((id) => theirs.has(id) && !start(id) && t.nodes[id] && !t.nodes[id].isMastery);
  if (shared !== undefined) expect(await drawnTints(page, nodeCentre(t, offsets, t.nodes[shared])!)).not.toContain('0,1,0');
  await page.getByRole('checkbox', { name: 'Compare:' }).uncheck();
  if (green[0] !== undefined) await expect.poll(async () => drawnTints(page, nodeCentre(t, offsets, t.nodes[green[0]])!)).not.toContain('0,1,0');
  app.assertNoErrors();
});

// PassiveTreeView.lua:1601-1644, 1712-1746
test('node tooltip: granted skill, reminder text, points and gold', async ({ page }) => {
  test.setTimeout(240_000);
  const { app, build, active } = await openTree(page);
  const m = await go.model();
  const t = tree.rawTree();
  const search = page.getByRole('searchbox', { name: 'Search the tree' });
  const results = page.getByRole('region', { name: 'Search results' });
  const tip = page.getByRole('tooltip').filter({ has: page.getByRole('group', { name: 'Stat differences' }) });

  const skillNode = Object.values(t.nodes).find(
    (n) => (n.stats ?? []).some((l) => /^Grants Level \d+ .+ Skill$/.test(l)) && m.exposition.NodeSkillTooltips(build, n.skill!, go.NO_TIP_STATS)?.length
  )!;
  const wantSkill = m.exposition.NodeSkillTooltips(build, skillNode.skill!, go.NO_TIP_STATS)![0]![0].Text.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '');
  await search.fill(`#${skillNode.skill}`);
  await results.getByRole('button', { name: tree.resultName({ id: skillNode.skill!, name: skillNode.name! }) }).hover();
  await expect(tip.getByLabel('Granted skill')).toContainText(wantSkill);

  const state = tree.treeState(build);
  const reminder = Object.values(t.nodes).find(
    (n) => n.reminderText?.length && !n.ascendancyName && !n.isMastery && (active.includes(n.skill!) || state.path(n.skill!))
  )!;
  await search.fill(`#${reminder.skill}`);
  await results.getByRole('button', { name: tree.resultName({ id: reminder.skill!, name: reminder.name! }) }).hover();
  await expect(tip.getByLabel('Reminder text')).toContainText(reminder.reminderText![0]);

  const far = Object.values(t.nodes).find(
    (n) => n.skill !== undefined && !active.includes(n.skill) && !n.isMastery && !n.ascendancyName && (state.path(n.skill)?.length ?? 0) > 2
  )!;
  await search.fill(`#${far.skill}`);
  await results.getByRole('button', { name: tree.resultName({ id: far.skill!, name: far.name! }) }).hover();
  await expect(tip.getByLabel('Points and gold')).toContainText(`${state.path(far.skill!)!.length} points to node`);

  const gold = m.exposition.GoldRespecPrice(build);
  expect(gold).toBeGreaterThan(0);
  const own = active.map((id) => t.nodes[id]).find((n) => n && n.isNotable && !n.ascendancyName)!;
  await search.fill(`#${own.skill}`);
  await results.getByRole('button', { name: tree.resultName({ id: own.skill!, name: own.name! }) }).hover();
  await expect(tip.getByLabel('Points and gold')).toContainText(/Gold required to unallocate/);
  const text = (await tip.getByLabel('Points and gold').textContent()) ?? '';
  const amount = Number(/([\d,]+) Gold/.exec(text)![1].replace(/,/g, ''));
  expect(amount % gold, `${amount} is not a multiple of ${gold}`).toBe(0);
  app.assertNoErrors();
});

// TreeTab.lua:1929-2119
test('timeless search: Generate writes the fallback list the model weighs', async ({ page }) => {
  test.setTimeout(240_000);
  const xml = buildXml(FILE);
  const build = await go.loadXml(xml);
  const app = await App.open(page, FILE);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Tree');
  await page.getByRole('button', { name: /^Timeless/ }).click();
  const dlg = page.getByTestId('timeless-dialog');
  await dlg
    .getByRole('group', { name: 'Jewels' })
    .getByRole('button', { name: /^Lethal Pride/ })
    .click();
  await dlg.getByRole('list', { name: 'Steps' }).getByRole('button', { name: '4 Ranking' }).click();
  await dlg.getByLabel('Fallback Weight Mode').selectOption({ label: 'Sort by Life' });
  await dlg.getByRole('button', { name: 'Generate' }).click();
  const want = await go.timelessFallback(build, 2, 'Life');
  expect(want, 'the model weighs nothing').not.toBe('');
  await dlg.getByRole('button', { name: 'Fallback Nodes' }).click();
  await expect(dlg.getByLabel('Fallback nodes')).toHaveValue(want);
  app.assertNoErrors();
});

// PassiveTreeView.lua:814-816, PassiveSpec.lua:1034-1048
test('split personality: the path to the class start takes its colour', async ({ page }) => {
  test.setTimeout(240_000);
  const { app, build } = await openTree(page, '0052-juggernaut-cleave');
  const m = await go.model();
  const misc = await m.exposition.GetMiscCalculator(build);
  const path = [...(misc!.SplitPersonalityPath() ?? [])];
  misc!.release();
  expect(path.length, 'the model has no path').toBeGreaterThan(10);
  const t = tree.rawTree();
  const offsets = ascendancyOffsets(t);
  const onTree = path.filter((id) => t.nodes[id] && !t.nodes[id].isMastery && t.nodes[id].classStartIndex === undefined);
  for (const id of onTree.slice(0, 4)) {
    await expect.poll(async () => drawnTints(page, nodeCentre(t, offsets, t.nodes[id])!), `node ${id}`).toContain('1,1,0');
  }
  // PassiveTreeView.lua:679-681
  const styles = async () => {
    const list = await page.evaluate(() => (globalThis as unknown as { __treeProbe: { drawList?: () => { items: unknown[] } } }).__treeProbe.drawList?.());
    return ((list?.items ?? []) as { k: string; style?: string }[]).filter((i) => i.k === 'line' || i.k === 'arc').map((i) => i.style);
  };
  await expect.poll(styles).toContain('#e9ba1e');
  app.assertNoErrors();
});

// PassiveTreeView.lua:214-217, 259-263
test('zoom keys: Page Up / Page Down and Ctrl+click zoom the tree, Ctrl+click allocates nothing', async ({ page }) => {
  test.setTimeout(240_000);
  const { app, active } = await openTree(page);
  const scaling = async () => (await lastFrame(page))!.view.scaling;
  await page.locator('body').click({ position: { x: 1, y: 1 } });
  const s0 = await scaling();
  await page.keyboard.press('PageUp');
  await expect.poll(scaling).toBeLessThan(s0);
  const s1 = await scaling();
  await page.keyboard.press('PageDown');
  await expect.poll(scaling).toBeGreaterThan(s1);
  const t = tree.rawTree();
  const view = (await lastFrame(page))!.view;
  const spot = emptySpot(t, placedNodes(t), view);
  const box = await canvasBox(page);
  const s2 = await scaling();
  await page.keyboard.down('Control');
  await page.mouse.click(box.x + spot.x, box.y + spot.y);
  await page.keyboard.up('Control');
  await expect.poll(scaling).toBeLessThan(s2);
  expect((await lastFrame(page))!.active).toBe(active.length);
  app.assertNoErrors();
});

// PassiveSpec.lua:776-784, 1136-1172
test('intuitive leap: a node an Impossible Escape affects is allocated alone', async ({ page }) => {
  test.setTimeout(240_000);
  const { app, build, active } = await openTree(page);
  const m = await go.model();
  const held = new Set((await m.exposition.LeapHeldNodes(build)) ?? []);
  const state = tree.treeState(build);
  const t = tree.rawTree();
  const target = [...held].find((id) => !active.includes(id) && t.nodes[id] && !t.nodes[id].isMastery && (state.path(id)?.length ?? 0) > 1);
  expect(target, 'no unallocated held node with a longer path').toBeDefined();
  await page.getByRole('searchbox', { name: 'Search the tree' }).fill(`#${target}`);
  const result = page
    .getByRole('region', { name: 'Search results' })
    .getByRole('button', { name: tree.resultName({ id: target!, name: t.nodes[target!].name! }) });
  await result.hover();
  await expect(page.getByRole('tooltip').getByLabel('Points and gold')).toContainText('(Can be allocated without pathing to it)');
  await result.click();
  await expect(result).toHaveAttribute('aria-pressed', 'true');
  await expect.poll(async () => (await lastFrame(page))!.active).toBe(active.length + 1);
  app.assertNoErrors();
});
