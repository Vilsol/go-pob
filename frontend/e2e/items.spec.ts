// ItemsTab.lua:5003-5158

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { test, expect, type Page } from '@playwright/test';

import { App } from './support/app';
import { buildXml } from './support/builds';
import * as go from './support/go_model';
import { record } from './support/perf';
import { expectedBlock, type DiffBlock, type WhatIfOutputs } from './support/stat_diff';
import { expectedLines } from './support/stats';
import type * as Types from '../src/lib/types';

type Build = Awaited<ReturnType<typeof go.loadXml>>;
type Shown = DiffBlock & { slot?: string; readyAt: number; calcMs: number };

const fixture = (name: string) => readFileSync(resolve(import.meta.dirname, 'fixtures/items', `${name}.txt`), 'utf8');
const ITEMS = { helmet: fixture('helmet'), ring: fixture('ring'), staff: fixture('staff'), flask: fixture('flask') };

const board = (page: Page) => page.getByRole('region', { name: 'Equipped items' });
const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const tile = (page: Page, slot: string) => board(page).getByRole('button', { name: new RegExp(`^${escapeRe(slot)}: `) });

const setup = async (page: Page, file: string) => {
  const xml = buildXml(file);
  const build = await go.loadXml(xml);
  const app = await App.open(page, file);
  await app.importCode(await go.encode(xml), async () => expectedLines(await go.outputs(build)));
  await app.goTo('Items');
  await expect(board(page)).toBeVisible();
  return { app, build };
};

const equippedItem = async (build: Build, slot: string) => {
  const m = await go.model();
  const eq = ((await m.exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === slot);
  if (!eq) return undefined;
  const p = (await m.exposition.ParseItem(build, eq.Text, go.NO_TIP_STATS)).unwrap();
  return { name: p.Title || p.Name, base: p.BaseName || p.Type, text: eq.Text };
};

const expectTile = async (page: Page, build: Build, slot: string) => {
  const it = await equippedItem(build, slot);
  await expect(tile(page, slot)).toHaveAccessibleName(`${slot}: ${it ? it.name : 'Empty'}`);
};

const pageNow = (page: Page) => page.evaluate(() => performance.now());

const readBlocks = async (page: Page, region: ReturnType<Page['getByRole']>) => {
  await expect(region).toHaveAttribute('aria-busy', 'false', { timeout: 120_000 });
  return (await region.evaluate((el) => {
    const text = (e: Element | undefined) => (e?.textContent ?? '').replace(/\s+/g, ' ').trim();
    return [...el.querySelectorAll<HTMLElement>('[data-header]')].map((b) => ({
      header: b.dataset.header!,
      slot: b.dataset.slot,
      readyAt: Number(b.dataset.readyAt),
      calcMs: Number(b.dataset.calcMs),
      sections: [...b.querySelectorAll<HTMLElement>('[data-section]')].map((sec) => ({
        actor: sec.dataset.actor || undefined,
        lines: [...sec.querySelectorAll('[data-line]')].map((l) => {
          const m = /^(?:[▲▼] )?(\S+)(?: (.*))?$/.exec(text(l.children[1]))!;
          return `${m[1]} ${text(l.children[0])}${m[2] ? ` ${m[2]}` : ''}`;
        })
      }))
    }));
  })) as Shown[];
};

const pasteRegion = (page: Page) => page.getByRole('region', { name: 'Equip impact' });
const slotRegion = (page: Page) => page.getByRole('tooltip').getByRole('group', { name: 'Stat differences' });

const withMisc = async <T>(build: Build, fn: (misc: Types.exposition.MiscCalculator, base: WhatIfOutputs) => Promise<T>) => {
  const m = await go.model();
  const misc = (await m.exposition.GetMiscCalculator(build))!;
  try {
    return await fn(misc, misc.Base() as WhatIfOutputs);
  } finally {
    misc.release();
  }
};

/* ItemsTab.lua:5003-5158 */
const expectedPaste = (build: Build, text: string) =>
  withMisc(build, async (misc, base) => {
    const m = await go.model();
    const parsed = (await m.exposition.ParseItem(build, text, go.NO_TIP_STATS)).unwrap();
    if (parsed.Flask) {
      const f = (await misc.CalcToggleFlask(text)).unwrap();
      const header = `${f.Active ? 'Deactivating' : 'Activating'} this ${f.Tincture ? 'tincture' : 'flask'} will give you:`;
      return [{ ...(expectedBlock(header, base, f.Output as WhatIfOutputs) ?? { header, sections: [] }), slot: undefined as string | undefined }];
    }
    return ((await misc.CompareItem(text)).unwrap() ?? []).map((s) => {
      const header = `Equipping this item in ${s.Label} will give you:${s.Replacing ? `\n(replacing ${s.Replacing.Name})` : ''}`;
      return { ...(expectedBlock(header, base, s.Output as WhatIfOutputs) ?? { header, sections: [] }), slot: s.SlotName as string | undefined };
    });
  });

const strip = (b: Shown[]) => b.map(({ header, sections, slot }) => ({ header, sections, slot }));

const paste = async (page: Page, app: App, build: Build, file: string, name: string, text: string) => {
  const box = page.getByRole('textbox', { name: /Item text/ });
  const t0 = await pageNow(page);
  await box.fill(text);
  await expect(
    page.getByRole('article', { name: `Item: ${(await (await go.model()).exposition.ParseItem(build, text, go.NO_TIP_STATS)).unwrap().Name}` })
  ).toBeVisible();
  const shown = await readBlocks(page, pasteRegion(page));
  const want = await expectedPaste(build, text);
  expect(strip(shown), `${name} on ${file}: equip impact`).toEqual(want);
  expect(
    want.some((b) => b.sections.length > 0),
    `${name} on ${file}: the model shows no stat change in any slot, so nothing was graded`
  ).toBe(true);
  record({
    kind: 'tree-latency',
    build: file,
    what: `item-paste ${name} (${want.length} slot)`,
    ms: +(Math.max(...shown.map((b) => b.readyAt)) - t0).toFixed(1)
  });
  record({ kind: 'tree-latency', build: file, what: `item-paste ${name} go`, ms: +shown[0].calcMs.toFixed(1) });
  app.assertNoErrors();
  return shown;
};

const hoverSlot = async (page: Page, build: Build, file: string, slot: string, label = slot) => {
  const row = tile(page, slot);
  await page.mouse.move(0, 0);
  const t0 = await pageNow(page);
  await row.hover();
  const region = slotRegion(page);
  await expect(region).toHaveAttribute('data-slot', slot);
  const it = (await equippedItem(build, slot))!;
  await expect(page.getByRole('tooltip')).toContainText(it.name);
  await expect(page.getByRole('tooltip')).toContainText(it.base);
  const tip = (await (await go.model()).exposition.ParseItem(build, it.text, await go.itemTooltipStats(build))).unwrap().Tooltip;
  const lines = (tip.Lines ?? []).map((l) => (l.Separator ? '--' : l.Text.replace(/\^(?:[x#][0-9A-Fa-f]{6}|\d)/g, '')));
  while (lines[0] === '--') lines.shift();
  while (lines.at(-1) === '--') lines.pop();
  const drawn = await page
    .getByRole('tooltip')
    .locator('[data-item-lines] > *')
    .evaluateAll((els) => els.map((el) => (el.tagName === 'HR' ? '--' : (el.textContent ?? ''))));
  expect(drawn, `tooltip lines of ${slot} on ${file}`).toEqual(lines);
  expect(lines.length, `${slot} on ${file}: the model drew no tooltip lines`).toBeGreaterThan(0);
  const shown = await readBlocks(page, region);
  const want = await withMisc(build, async (misc, base) => {
    const eq = ((await (await go.model()).exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === slot)!;
    if (eq.Flask) {
      const f = (await misc.CalcToggleFlask(eq.Text)).unwrap();
      const header = `${f.Active ? 'Deactivating' : 'Activating'} this ${f.Tincture ? 'tincture' : 'flask'} will give you:`;
      // ItemsTab.lua:4803-5001
      const effective = (f.Effective ?? []).map((l) => l.replace(/\^(?:[x#][0-9A-Fa-f]{6}|\d)/g, ''));
      const drawnEffective = await region.getByRole('group', { name: 'Effective flask stats' }).locator('> *').allTextContents();
      expect(drawnEffective, `effective flask stats of ${slot} on ${file}`).toEqual(effective);
      return expectedBlock(header, base, f.Output as WhatIfOutputs) ?? { header, sections: [] };
    }
    const header = `Removing this item from ${label} will give you:`;
    return expectedBlock(header, base, (await misc.CalcUnequip(slot)) as WhatIfOutputs) ?? { header, sections: [] };
  });
  expect(
    strip(shown).map(({ header, sections }) => ({ header, sections })),
    `hover ${slot} on ${file}`
  ).toEqual([want]);
  record({ kind: 'tree-latency', build: file, what: `item-hover ${slot}`, ms: +(shown[0].readyAt - t0).toFixed(1) });
  return want;
};

test('paste, equip, hover, save/reload: 0280 (spell, small)', async ({ page }) => {
  const file = '0280-elementalist-fireball';
  const { app, build } = await setup(page, file);

  await paste(page, app, build, file, 'ring', ITEMS.ring);
  await paste(page, app, build, file, 'flask', ITEMS.flask);
  await paste(page, app, build, file, 'helmet', ITEMS.helmet);

  await app.act(
    'item-equip',
    () => pasteRegion(page).getByRole('button', { name: 'Equip in Helmet' }).click(),
    async () => {
      (await (await go.model()).exposition.EquipItem(build, ITEMS.helmet, 'Helmet')).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(tile(page, 'Helmet')).toHaveAccessibleName('Helmet: Doom Visor');
  await expectTile(page, build, 'Helmet');
  await hoverSlot(page, build, file, 'Helmet');
  await hoverSlot(page, build, file, 'Flask 2');

  await app.act(
    'flask-activate',
    () => page.getByRole('checkbox', { name: 'Flask 2 active' }).click(),
    async () => {
      build.SetFlaskActive('Flask 2', true);
      return expectedLines(await go.outputs(build));
    }
  );
  await hoverSlot(page, build, file, 'Flask 2');

  // ItemsTab.lua:1694-1701
  await page.getByRole('textbox', { name: /Item text/ }).fill(ITEMS.ring);
  await readBlocks(page, pasteRegion(page));
  await app.act(
    'item-save',
    () => page.getByRole('button', { name: 'Save to build' }).click(),
    async () => {
      await withMisc(build, async (misc) => (await misc.AddItem(ITEMS.ring)).unwrap());
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(page.getByText('Saved to the item list.')).toBeVisible();

  const exported = await go.loadXml(await go.decode(await app.exportCode()));
  const names = (b: Build) => go.model().then(async (m) => ((await m.exposition.EquippedItems(b)) ?? []).map((e) => `${e.SlotName}=${e.Name}:${e.Active}`));
  expect(await names(exported), 'the exported build equips what the model does').toEqual(await names(build));
  expect(go.fresh(exported, (b) => b.Items.Items?.length)).toBe(go.fresh(build, (b) => b.Items.Items?.length));
  await app.importCode(await app.exportCode(), async () => expectedLines(await go.outputs(build)));
  app.assertNoErrors();
});

test('minion build: paste and equip a helmet (0008)', async ({ page }) => {
  const file = '0008-necromancer-vaal-summon-skeletons';
  const { app, build } = await setup(page, file);
  const shown = await paste(page, app, build, file, 'helmet', ITEMS.helmet);
  expect(
    shown.every((b) => b.sections.every((s) => s.actor === 'Minion' || s.actor === 'Player')),
    'a minion main skill shows lines outside a Minion: / Player: section'
  ).toBe(true);
  await app.act(
    'item-equip',
    () => pasteRegion(page).getByRole('button', { name: 'Equip in Helmet' }).click(),
    async () => {
      (await (await go.model()).exposition.EquipItem(build, ITEMS.helmet, 'Helmet')).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await hoverSlot(page, build, file, 'Helmet');
  app.assertNoErrors();
});

test('dual wield: a staff empties Weapon 2 (13276)', async ({ page }) => {
  const file = '13276-ladder-cuba_5717.cubamrc';
  const { app, build } = await setup(page, file);
  await paste(page, app, build, file, 'staff', ITEMS.staff);
  await app.act(
    'item-equip',
    () => pasteRegion(page).getByRole('button', { name: 'Equip in Weapon 1' }).click(),
    async () => {
      (await (await go.model()).exposition.EquipItem(build, ITEMS.staff, 'Weapon 1')).unwrap();
      return expectedLines(await go.outputs(build));
    }
  );
  await expect(tile(page, 'Weapon 1')).toHaveAccessibleName('Weapon 1: Oblivion Pillar');
  await expectTile(page, build, 'Weapon 1');
  await expect(tile(page, 'Weapon 2')).toHaveAccessibleName('Weapon 2: Empty');
  await expectTile(page, build, 'Weapon 2');
  await hoverSlot(page, build, file, 'Flask 1');
  app.assertNoErrors();
});

// ItemSlotControl.lua:12-19, ItemsTab.lua:2457-2503
test('unequipping a bow drops its quiver (0241)', async ({ page }) => {
  const file = '0241-warden-galvanic-arrow';
  const { app, build } = await setup(page, file);
  expect((await equippedItem(build, 'Weapon 2'))?.base, 'vacuity: 0241 wears a quiver in Weapon 2').toMatch(/Quiver/);
  await expectTile(page, build, 'Weapon 1');
  await expectTile(page, build, 'Weapon 2');
  await board(page).getByRole('button', { name: 'Change Weapon 1' }).click();
  await app.act(
    'item-unequip',
    () => page.getByRole('menu', { name: 'Weapon 1' }).getByRole('menuitem', { name: 'Unequip' }).click(),
    async () => {
      expect((await (await go.model()).exposition.EquipItemID(build, 'Weapon 1', 0)).unwrap()).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  expect(await equippedItem(build, 'Weapon 2'), 'the model kept the quiver').toBeUndefined();
  await expect(tile(page, 'Weapon 1')).toHaveAccessibleName('Weapon 1: Empty');
  await expect(tile(page, 'Weapon 2')).toHaveAccessibleName('Weapon 2: Empty');
  app.assertNoErrors();
});

// ItemsTab.lua:1584-1610
// ItemSlotControl.lua:12-19
test('item sets: switch, equip an item of the build by id, switch back (0339)', async ({ page }) => {
  const file = '0339-reliquarian-shield-crush-of-the-chieftain-2';
  const { app, build } = await setup(page, file);
  const m = await go.model();
  const setSelect = page.getByRole('combobox', { name: 'Item set' });
  const slots = ['Helmet', 'Body Armour', 'Weapon 1', 'Weapon 2', 'Ring 1', 'Flask 1'];
  const expectTiles = async () => {
    for (const slot of slots) await expectTile(page, build, slot);
  };
  const activeSet = () => go.fresh(build, (b) => b.Items.ActiveItemSet);
  expect(activeSet()).toBe(2);
  await expect(setSelect).toHaveValue('2');
  await expectTiles();
  const enmityHelmet = await equippedItem(build, 'Helmet');

  await app.act(
    'item-set',
    () => setSelect.selectOption('1'),
    async () => {
      await m.exposition.SetActiveItemSet(build, 1);
      return expectedLines(await go.outputs(build));
    }
  );
  expect(activeSet()).toBe(1);
  await expect(setSelect).toHaveValue('1');
  await expectTiles();
  const setOneHelmet = await equippedItem(build, 'Helmet');
  expect(setOneHelmet?.name, 'vacuity: the two sets wear the same helmet').not.toBe(enmityHelmet?.name);

  const lists = (await m.exposition.SlotItemLists(build)) ?? [];
  const helmetIds = lists.find((l) => l.SlotName === 'Helmet')?.ItemIDs ?? [];
  const texts = new Map(go.fresh(build, (b) => (b.Items.Items ?? []).map((it) => [it.ID, it.Text] as const)));
  const equippedId = ((await m.exposition.EquippedItems(build)) ?? []).find((e) => e.SlotName === 'Helmet')?.ItemID;
  const offered: { id: number; name: string }[] = [];
  for (const id of helmetIds.filter((x) => x !== equippedId)) {
    const p = (await m.exposition.ParseItem(build, texts.get(id) ?? '', go.NO_TIP_STATS)).unwrap();
    offered.push({ id, name: p.Title || p.Name });
  }
  const target = offered.find((o) => o.name === enmityHelmet?.name);
  expect(target, `Go's Helmet list ${JSON.stringify(helmetIds)} lacks the enmity helmet`).toBeDefined();
  await board(page).getByRole('button', { name: 'Change Helmet' }).click();
  const menu = page.getByRole('menu', { name: 'Helmet' });
  // ItemListControl.lua:331-369
  await expect(menu.getByRole('menuitem')).toHaveText(['Unequip', ...offered.map((o) => `Equip ${o.name}`), 'Delete item…']);
  await app.act(
    'item-equip-id',
    () => menu.getByRole('menuitem', { name: `Equip ${target!.name}` }).click(),
    async () => {
      expect((await m.exposition.EquipItemID(build, 'Helmet', target!.id)).unwrap()).toBe(true);
      return expectedLines(await go.outputs(build));
    }
  );
  await expectTiles();
  await expect(tile(page, 'Helmet')).toHaveAccessibleName(`Helmet: ${target!.name}`);

  await app.act(
    'item-set',
    () => setSelect.selectOption('2'),
    async () => {
      await m.exposition.SetActiveItemSet(build, 2);
      return expectedLines(await go.outputs(build));
    }
  );
  expect(activeSet()).toBe(2);
  await expectTiles();
  await expect(tile(page, 'Helmet')).toHaveAccessibleName(`Helmet: ${enmityHelmet!.name}`);

  const exported = await go.loadXml(await go.decode(await app.exportCode()));
  const sets = (b: Build) =>
    go.fresh(b, (x) =>
      JSON.stringify([
        x.Items.ActiveItemSet,
        x.Items.ItemSets.map((s) =>
          (s.Slots ?? [])
            .filter((sl) => sl.ItemID > 0)
            .map((sl) => `${sl.Name}=${sl.ItemID}`)
            .sort()
        )
      ])
    );
  expect(sets(exported), 'the exported build holds the sets the model does').toBe(sets(build));
  app.assertNoErrors();
});

test('stale pastes are dropped, heavy build latency (1591)', async ({ page }) => {
  const file = '1591-necromancer-summon-holy-relic-of-conviction-15';
  const { app, build } = await setup(page, file);
  await paste(page, app, build, file, 'helmet', ITEMS.helmet);

  const box = page.getByRole('textbox', { name: /Item text/ });
  const t0 = await pageNow(page);
  await box.fill(ITEMS.staff);
  await box.fill(ITEMS.helmet);
  await box.fill(ITEMS.ring);
  await expect(page.getByRole('article', { name: 'Item: Storm Loop, Amethyst Ring' })).toBeVisible();
  const shown = await readBlocks(page, pasteRegion(page));
  expect(strip(shown), 'after three quick pastes the page shows the last').toEqual(await expectedPaste(build, ITEMS.ring));
  record({
    kind: 'tree-latency',
    build: file,
    what: 'item-paste settle after 3 quick pastes',
    ms: +(Math.max(...shown.map((b) => b.readyAt)) - t0).toFixed(1)
  });

  await hoverSlot(page, build, file, 'Helmet');
  await hoverSlot(page, build, file, 'Flask 2');
  app.assertNoErrors();
});
